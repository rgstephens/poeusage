package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/gstephens/poeusage/internal/output"
)

var (
	historyLimit      int
	historyPageSize   int
	historyNoPaginate bool
	historyCursor     string
	historyBot        string
	historyType       string
	historySince      string
	historyUntil      string
	historyOutputFile string
	historyFormat     string
)

var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "Fetch usage history",
	RunE:  runHistory,
}

func runHistory(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Resolve page size: flag > config default
	pageSize := historyPageSize
	if !cmd.Flags().Changed("page-size") && globalState.cfg.PageSize != 0 {
		pageSize = globalState.cfg.PageSize
	}
	if pageSize > 100 {
		pageSize = 100
	}
	if pageSize < 1 {
		pageSize = 1
	}

	// Validate format
	opts := globalState.outOpts
	if historyFormat != "" {
		switch strings.ToLower(historyFormat) {
		case "table":
			opts.Format = output.FormatTable
		case "csv":
			opts.Format = output.FormatCSV
		case "json":
			opts.Format = output.FormatJSON
		default:
			return &UsageError{Msg: fmt.Sprintf("unknown format %q: must be table, csv, or json", historyFormat)}
		}
	}
	// Format resolution: explicit --format > --json global > --plain global > TTY→table / non-TTY→csv
	// (already handled by opts from globalState, --format flag overrides)

	// Parse date filters
	var sinceTime, untilTime time.Time
	if historySince != "" {
		t, err := output.ParseDate(historySince)
		if err != nil {
			return &UsageError{Msg: err.Error()}
		}
		sinceTime = t
	}
	if historyUntil != "" {
		t, err := output.ParseUntilDate(historyUntil)
		if err != nil {
			return &UsageError{Msg: err.Error()}
		}
		untilTime = t
	}

	// Determine writer
	var w io.Writer = os.Stdout
	if historyOutputFile != "" && historyOutputFile != "-" {
		f, err := os.Create(historyOutputFile)
		if err != nil {
			return fmt.Errorf("failed to open output file: %w", err)
		}
		defer f.Close()
		w = f
		// File output is not a TTY
		if opts.Format == output.FormatTable {
			opts.Format = output.FormatCSV
		}
		opts.IsTTY = false
	}

	if historyNoPaginate {
		// Fetch one page
		page, err := globalState.client.GetHistoryPage(ctx, pageSize, historyCursor)
		if err != nil {
			return err
		}
		filtered := output.FilterRecords(page.Data, historyBot, historyType, sinceTime, untilTime)
		output.PrintHistory(w, filtered, opts)
		if page.HasMore && len(page.Data) > 0 {
			lastQueryID := page.Data[len(page.Data)-1].QueryID
			fmt.Fprintf(os.Stderr, "next-cursor: %s\n", lastQueryID)
		}
	} else {
		// Auto-paginate
		var progressFn func(int)
		if flagVerbose {
			progressFn = func(n int) {
				fmt.Fprintf(os.Stderr, "Fetching... (%d records)\n", n)
			}
		}

		allRecords, err := globalState.client.GetAllHistory(ctx, pageSize, historyLimit, progressFn)
		if err != nil {
			return err
		}

		filtered := output.FilterRecords(allRecords, historyBot, historyType, sinceTime, untilTime)
		output.PrintHistory(w, filtered, opts)
	}

	return nil
}

func init() {
	historyCmd.Flags().IntVarP(&historyLimit, "limit", "n", 0, "Max records to fetch total (0 = fetch all)")
	historyCmd.Flags().IntVar(&historyPageSize, "page-size", 100, "Records per API call (max 100)")
	historyCmd.Flags().BoolVar(&historyNoPaginate, "no-paginate", false, "Fetch only one page, print cursor for next")
	historyCmd.Flags().StringVar(&historyCursor, "cursor", "", "Resume pagination from this query_id")
	historyCmd.Flags().StringVarP(&historyBot, "bot", "b", "", "Filter by bot name (substring match, case-insensitive)")
	historyCmd.Flags().StringVarP(&historyType, "type", "t", "", "Filter by usage type: chat, api, canvas")
	historyCmd.Flags().StringVar(&historySince, "since", "", "Only show records after this date (YYYY-MM-DD or unix timestamp)")
	historyCmd.Flags().StringVar(&historyUntil, "until", "", "Only show records before this date (YYYY-MM-DD or unix timestamp)")
	historyCmd.Flags().StringVarP(&historyOutputFile, "output", "o", "", "Write output to file (use - for stdout)")
	historyCmd.Flags().StringVarP(&historyFormat, "format", "f", "", "Output format: table, csv, json")

	rootCmd.AddCommand(historyCmd)
}
