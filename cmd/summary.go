package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/gstephens/poeusage/internal/output"
)

var (
	summaryBot     string
	summaryType    string
	summarySince   string
	summaryUntil   string
	summaryGroupBy string
	summaryFormat  string
)

var summaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Aggregate usage history and display a cost breakdown",
	RunE:  runSummary,
}

func runSummary(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Resolve page size from config
	pageSize := globalState.cfg.PageSize
	if pageSize < 1 {
		pageSize = 100
	}
	if pageSize > 100 {
		pageSize = 100
	}

	// Validate format
	opts := globalState.outOpts
	if summaryFormat != "" {
		switch strings.ToLower(summaryFormat) {
		case "table":
			opts.Format = output.FormatTable
		case "csv":
			opts.Format = output.FormatCSV
		case "json":
			opts.Format = output.FormatJSON
		default:
			return &UsageError{Msg: fmt.Sprintf("unknown format %q: must be table, csv, or json", summaryFormat)}
		}
	}

	// Validate group-by
	switch summaryGroupBy {
	case "bot", "type", "day", "bot,type":
		// valid
	default:
		return &UsageError{Msg: fmt.Sprintf("unknown group-by %q: must be bot, type, day, or bot,type", summaryGroupBy)}
	}

	// Parse date filters
	var sinceTime, untilTime time.Time
	if summarySince != "" {
		t, err := output.ParseDate(summarySince)
		if err != nil {
			return &UsageError{Msg: err.Error()}
		}
		sinceTime = t
	}
	if summaryUntil != "" {
		t, err := output.ParseUntilDate(summaryUntil)
		if err != nil {
			return &UsageError{Msg: err.Error()}
		}
		untilTime = t
	}

	// Fetch all records
	var progressFn func(int)
	if flagVerbose {
		progressFn = func(n int) {
			fmt.Fprintf(os.Stderr, "Fetching... (%d records)\n", n)
		}
	}

	allRecords, err := globalState.client.GetAllHistory(ctx, pageSize, 0, progressFn)
	if err != nil {
		return err
	}

	filtered := output.FilterRecords(allRecords, summaryBot, summaryType, sinceTime, untilTime)
	output.PrintSummary(os.Stdout, filtered, summaryGroupBy, opts)

	return nil
}

func init() {
	summaryCmd.Flags().StringVarP(&summaryBot, "bot", "b", "", "Filter by bot name (substring match)")
	summaryCmd.Flags().StringVarP(&summaryType, "type", "t", "", "Filter by usage type: chat, api, canvas")
	summaryCmd.Flags().StringVar(&summarySince, "since", "", "Only include records after this date (YYYY-MM-DD or unix timestamp)")
	summaryCmd.Flags().StringVar(&summaryUntil, "until", "", "Only include records before this date (YYYY-MM-DD or unix timestamp)")
	summaryCmd.Flags().StringVarP(&summaryGroupBy, "group-by", "g", "bot", "Group aggregation: bot, type, day, bot,type")
	summaryCmd.Flags().StringVarP(&summaryFormat, "format", "f", "", "Output format: table, csv, json")

	rootCmd.AddCommand(summaryCmd)
}
