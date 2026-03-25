package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/gstephens/poeusage/internal/api"
)

// Format represents the output format.
type Format string

const (
	FormatTable Format = "table"
	FormatCSV   Format = "csv"
	FormatJSON  Format = "json"
	FormatPlain Format = "plain"
)

// Options holds output formatting options.
type Options struct {
	Format  Format
	NoColor bool
	Quiet   bool
	IsTTY   bool
}

// UseColor returns true if color output should be used.
func UseColor(opts Options) bool {
	if opts.NoColor {
		return false
	}
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	if os.Getenv("TERM") == "dumb" {
		return false
	}
	return opts.IsTTY
}

// FormatInt formats an integer with comma separators.
func FormatInt(n int) string {
	if n < 0 {
		return "-" + FormatInt(-n)
	}
	s := strconv.Itoa(n)
	if len(s) <= 3 {
		return s
	}
	var result []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}
	return string(result)
}

// ParseDate parses a date string in YYYY-MM-DD or unix seconds format.
func ParseDate(s string) (time.Time, error) {
	// Try unix seconds first
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		return time.Unix(n, 0).UTC(), nil
	}
	// Try YYYY-MM-DD
	t, err := time.ParseInLocation("2006-01-02", s, time.UTC)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date %q: expected YYYY-MM-DD or unix seconds", s)
	}
	return t, nil
}

// ParseUntilDate parses a date string for use as an until bound (adds 24h so the day is inclusive).
func ParseUntilDate(s string) (time.Time, error) {
	t, err := ParseDate(s)
	if err != nil {
		return t, err
	}
	// If it was a YYYY-MM-DD parse (not unix seconds), add 24h to include the full day
	if _, err2 := strconv.ParseInt(s, 10, 64); err2 != nil {
		t = t.Add(24 * time.Hour)
	}
	return t, nil
}

// normalizeUsageType maps user-friendly type strings to API values.
func normalizeUsageType(t string) string {
	switch strings.ToLower(t) {
	case "chat":
		return "Chat"
	case "api":
		return "API"
	case "canvas":
		return "Canvas App"
	default:
		return t
	}
}

// FilterRecords filters records by bot name, usage type, and date range.
func FilterRecords(records []api.UsageRecord, bot, usageType string, since, until time.Time) []api.UsageRecord {
	var out []api.UsageRecord
	normalizedType := ""
	if usageType != "" {
		normalizedType = normalizeUsageType(usageType)
	}

	for _, r := range records {
		if bot != "" && !strings.Contains(strings.ToLower(r.BotName), strings.ToLower(bot)) {
			continue
		}
		if normalizedType != "" && !strings.EqualFold(r.UsageType, normalizedType) {
			continue
		}
		t := r.Time()
		if !since.IsZero() && t.Before(since) {
			continue
		}
		if !until.IsZero() && !t.Before(until) {
			continue
		}
		out = append(out, r)
	}
	return out
}

// PrintBalance prints the balance in the appropriate format.
func PrintBalance(w io.Writer, bal int, opts Options) {
	switch opts.Format {
	case FormatJSON:
		fmt.Fprintf(w, `{"current_point_balance":%d}`+"\n", bal)
	case FormatPlain:
		fmt.Fprintf(w, "%d\n", bal)
	default:
		if opts.IsTTY {
			fmt.Fprintf(w, "Current balance: %s pts\n", FormatInt(bal))
		} else {
			fmt.Fprintf(w, "%d\n", bal)
		}
	}
}

// historyJSONRecord is the JSON representation of a usage record.
type historyJSONRecord struct {
	BotName      string            `json:"bot_name"`
	Time         string            `json:"time"`
	QueryID      string            `json:"query_id"`
	CostUSD      string            `json:"cost_usd"`
	CostPoints   int               `json:"cost_points"`
	UsageType    string            `json:"usage_type"`
	ChatName     *string           `json:"chat_name"`
	CostBreakdown breakdownJSON    `json:"cost_breakdown"`
}

type breakdownJSON struct {
	Input         int `json:"input"`
	Output        int `json:"output"`
	CacheWrite    int `json:"cache_write"`
	CacheDiscount int `json:"cache_discount"`
	Total         int `json:"total"`
}

// PrintHistory prints usage history in the appropriate format.
func PrintHistory(w io.Writer, records []api.UsageRecord, opts Options) {
	switch opts.Format {
	case FormatJSON:
		printHistoryJSON(w, records)
	case FormatCSV, FormatPlain:
		printHistoryCSV(w, records)
	default:
		// table
		if opts.IsTTY {
			printHistoryTable(w, records, opts)
		} else {
			printHistoryCSV(w, records)
		}
	}
}

func printHistoryJSON(w io.Writer, records []api.UsageRecord) {
	out := make([]historyJSONRecord, 0, len(records))
	for _, r := range records {
		out = append(out, historyJSONRecord{
			BotName:    r.BotName,
			Time:       r.Time().Format(time.RFC3339),
			QueryID:    r.QueryID,
			CostUSD:    r.CostUSD,
			CostPoints: r.CostPoints,
			UsageType:  r.UsageType,
			ChatName:   r.ChatName,
			CostBreakdown: breakdownJSON{
				Input:         int(r.Breakdown.Input),
				Output:        int(r.Breakdown.Output),
				CacheWrite:    int(r.Breakdown.CacheWrite),
				CacheDiscount: int(r.Breakdown.CacheDiscount),
				Total:         int(r.Breakdown.Total),
			},
		})
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(out)
}

func printHistoryCSV(w io.Writer, records []api.UsageRecord) {
	cw := csv.NewWriter(w)
	cw.Write([]string{
		"time", "bot_name", "usage_type", "cost_points", "cost_usd",
		"query_id", "chat_name", "input_pts", "output_pts", "cache_write_pts", "cache_discount_pts",
	})
	for _, r := range records {
		chatName := ""
		if r.ChatName != nil {
			chatName = *r.ChatName
		}
		cw.Write([]string{
			r.Time().Format(time.RFC3339),
			r.BotName,
			r.UsageType,
			strconv.Itoa(r.CostPoints),
			r.CostUSD,
			r.QueryID,
			chatName,
			strconv.Itoa(int(r.Breakdown.Input)),
			strconv.Itoa(int(r.Breakdown.Output)),
			strconv.Itoa(int(r.Breakdown.CacheWrite)),
			strconv.Itoa(int(r.Breakdown.CacheDiscount)),
		})
	}
	cw.Flush()
}

func printHistoryTable(w io.Writer, records []api.UsageRecord, opts Options) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "TIME\tBOT\tTYPE\tPOINTS\tCOST (USD)")

	var totalPoints int
	var totalCostUSD float64

	for _, r := range records {
		t := r.Time().Format("2006-01-02 15:04")
		costUSD, _ := strconv.ParseFloat(r.CostUSD, 64)
		totalPoints += r.CostPoints
		totalCostUSD += costUSD
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t$%s\n",
			t, r.BotName, r.UsageType, FormatInt(r.CostPoints), r.CostUSD)
	}
	tw.Flush()

	// Footer
	fmt.Fprintf(w, "%s records shown. Total: %s pts / $%.5f\n",
		FormatInt(len(records)), FormatInt(totalPoints), totalCostUSD)
}

// summaryKey is used for grouping.
type summaryKey struct {
	Primary   string
	Secondary string
}

type summaryRow struct {
	Key     summaryKey
	Queries int
	Points  int
	CostUSD float64
}

// PrintSummary prints a summary of usage records grouped by the specified field.
func PrintSummary(w io.Writer, records []api.UsageRecord, groupBy string, opts Options) {
	switch opts.Format {
	case FormatJSON:
		printSummaryJSON(w, records, groupBy)
	case FormatCSV, FormatPlain:
		printSummaryCSV(w, records, groupBy)
	default:
		if opts.IsTTY {
			printSummaryTable(w, records, groupBy, opts)
		} else {
			printSummaryCSV(w, records, groupBy)
		}
	}
}

func aggregateRecords(records []api.UsageRecord, groupBy string) ([]summaryRow, int, float64) {
	aggMap := map[summaryKey]*summaryRow{}
	var order []summaryKey

	for _, r := range records {
		var key summaryKey
		switch groupBy {
		case "type":
			key = summaryKey{Primary: r.UsageType}
		case "day":
			key = summaryKey{Primary: r.Time().Format("2006-01-02")}
		case "bot,type":
			key = summaryKey{Primary: r.BotName, Secondary: r.UsageType}
		default: // bot
			key = summaryKey{Primary: r.BotName}
		}

		row, exists := aggMap[key]
		if !exists {
			row = &summaryRow{Key: key}
			aggMap[key] = row
			order = append(order, key)
		}
		costUSD, _ := strconv.ParseFloat(r.CostUSD, 64)
		row.Queries++
		row.Points += r.CostPoints
		row.CostUSD += costUSD
	}

	// Sort by points descending
	sort.Slice(order, func(i, j int) bool {
		ki, kj := order[i], order[j]
		if aggMap[ki].Points != aggMap[kj].Points {
			return aggMap[ki].Points > aggMap[kj].Points
		}
		if ki.Primary != kj.Primary {
			return ki.Primary < kj.Primary
		}
		return ki.Secondary < kj.Secondary
	})

	rows := make([]summaryRow, 0, len(order))
	var totalPoints int
	var totalCostUSD float64
	for _, k := range order {
		rows = append(rows, *aggMap[k])
		totalPoints += aggMap[k].Points
		totalCostUSD += aggMap[k].CostUSD
	}

	return rows, totalPoints, totalCostUSD
}

func printSummaryTable(w io.Writer, records []api.UsageRecord, groupBy string, opts Options) {
	rows, totalPoints, totalCostUSD := aggregateRecords(records, groupBy)
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)

	isBotType := groupBy == "bot,type"
	if isBotType {
		fmt.Fprintln(tw, "BOT\tTYPE\tQUERIES\tPOINTS\tCOST (USD)")
	} else {
		var header string
		switch groupBy {
		case "type":
			header = "TYPE"
		case "day":
			header = "DAY"
		default:
			header = "BOT"
		}
		fmt.Fprintf(tw, "%s\tQUERIES\tPOINTS\tCOST (USD)\n", header)
	}

	for _, row := range rows {
		if isBotType {
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t$%.5f\n",
				row.Key.Primary, row.Key.Secondary,
				FormatInt(row.Queries), FormatInt(row.Points), row.CostUSD)
		} else {
			fmt.Fprintf(tw, "%s\t%s\t%s\t$%.5f\n",
				row.Key.Primary, FormatInt(row.Queries), FormatInt(row.Points), row.CostUSD)
		}
	}

	// TOTAL row
	if isBotType {
		fmt.Fprintf(tw, "TOTAL\t\t%s\t%s\t$%.5f\n",
			FormatInt(len(records)), FormatInt(totalPoints), totalCostUSD)
	} else {
		fmt.Fprintf(tw, "TOTAL\t%s\t%s\t$%.5f\n",
			FormatInt(len(records)), FormatInt(totalPoints), totalCostUSD)
	}

	tw.Flush()
}

func printSummaryCSV(w io.Writer, records []api.UsageRecord, groupBy string) {
	rows, totalPoints, totalCostUSD := aggregateRecords(records, groupBy)
	cw := csv.NewWriter(w)

	isBotType := groupBy == "bot,type"
	if isBotType {
		cw.Write([]string{"bot", "type", "queries", "points", "cost_usd"})
	} else {
		var groupHeader string
		switch groupBy {
		case "type":
			groupHeader = "type"
		case "day":
			groupHeader = "day"
		default:
			groupHeader = "bot"
		}
		cw.Write([]string{groupHeader, "queries", "points", "cost_usd"})
	}

	for _, row := range rows {
		if isBotType {
			cw.Write([]string{
				row.Key.Primary, row.Key.Secondary,
				strconv.Itoa(row.Queries), strconv.Itoa(row.Points),
				fmt.Sprintf("%.5f", row.CostUSD),
			})
		} else {
			cw.Write([]string{
				row.Key.Primary, strconv.Itoa(row.Queries), strconv.Itoa(row.Points),
				fmt.Sprintf("%.5f", row.CostUSD),
			})
		}
	}

	// Total row
	if isBotType {
		cw.Write([]string{"TOTAL", "", strconv.Itoa(len(records)), strconv.Itoa(totalPoints), fmt.Sprintf("%.5f", totalCostUSD)})
	} else {
		cw.Write([]string{"TOTAL", strconv.Itoa(len(records)), strconv.Itoa(totalPoints), fmt.Sprintf("%.5f", totalCostUSD)})
	}

	cw.Flush()
}

type summaryJSONRow struct {
	Group   string  `json:"group"`
	Group2  string  `json:"group2,omitempty"`
	Queries int     `json:"queries"`
	Points  int     `json:"points"`
	CostUSD float64 `json:"cost_usd"`
}

func printSummaryJSON(w io.Writer, records []api.UsageRecord, groupBy string) {
	rows, totalPoints, totalCostUSD := aggregateRecords(records, groupBy)

	isBotType := groupBy == "bot,type"
	out := make([]summaryJSONRow, 0, len(rows)+1)
	for _, row := range rows {
		r := summaryJSONRow{
			Group:   row.Key.Primary,
			Queries: row.Queries,
			Points:  row.Points,
			CostUSD: math.Round(row.CostUSD*1e7) / 1e7,
		}
		if isBotType {
			r.Group2 = row.Key.Secondary
		}
		out = append(out, r)
	}
	out = append(out, summaryJSONRow{
		Group:   "TOTAL",
		Queries: len(records),
		Points:  totalPoints,
		CostUSD: math.Round(totalCostUSD*1e7) / 1e7,
	})

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(out)
}
