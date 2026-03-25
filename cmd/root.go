package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/gstephens/poeusage/internal/api"
	"github.com/gstephens/poeusage/internal/config"
	"github.com/gstephens/poeusage/internal/output"
)

// Version and BuildDate are injected via ldflags.
var (
	Version   = "1.0.0"
	BuildDate = "unknown"
)

// UsageError represents an invalid-usage error (exit code 2).
type UsageError struct {
	Msg string
}

func (e *UsageError) Error() string {
	return e.Msg
}

// globalState holds shared state between subcommands.
var globalState struct {
	client  *api.Client
	outOpts output.Options
	cfg     config.Config
}

// global flag values
var (
	flagAPIKey   string
	flagJSON     bool
	flagPlain    bool
	flagNoColor  bool
	flagQuiet    bool
	flagVerbose  bool
	flagTimeout  int
)

var rootCmd = &cobra.Command{
	Use:   "poeusage",
	Short: "Monitor your Poe API point balance and usage history",
	Long: `poeusage is a CLI tool for monitoring your Poe API point balance and usage history.

Set POE_API_KEY or use --api-key to authenticate.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return initGlobalState(cmd)
	},
}

func initGlobalState(cmd *cobra.Command) error {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	globalState.cfg = cfg

	// Resolve timeout: flag > config
	timeout := flagTimeout
	if timeout == 30 && cfg.Timeout != 30 {
		timeout = cfg.Timeout
	}

	// Resolve API key: flag > env
	apiKey := flagAPIKey
	if apiKey == "" {
		apiKey = os.Getenv("POE_API_KEY")
	}
	if apiKey == "" {
		return &UsageError{Msg: "API key required: set POE_API_KEY env var or use --api-key flag"}
	}

	// Determine TTY
	isTTY := term.IsTerminal(int(os.Stdout.Fd()))

	// Determine no-color
	noColor := flagNoColor
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		noColor = true
	}

	// Determine format
	var fmt_ output.Format
	if flagJSON {
		fmt_ = output.FormatJSON
	} else if flagPlain {
		fmt_ = output.FormatPlain
	} else if isTTY {
		fmt_ = output.FormatTable
	} else {
		fmt_ = output.FormatPlain
	}

	globalState.outOpts = output.Options{
		Format:  fmt_,
		NoColor: noColor,
		Quiet:   flagQuiet,
		IsTTY:   isTTY,
	}

	globalState.client = api.NewClient(apiKey, timeout, flagVerbose, os.Stderr)

	return nil
}

// Execute runs the root command and returns an exit code.
func Execute() int {
	if err := rootCmd.Execute(); err != nil {
		var authErr *api.AuthError
		var usageErr *UsageError

		switch {
		case errors.As(err, &authErr):
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			return 3
		case errors.As(err, &usageErr):
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			return 2
		default:
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			return 1
		}
	}
	return 0
}

func init() {
	// Set version here so ldflags-injected values are used (not the zero values at struct init time).
	rootCmd.Version = fmt.Sprintf("%s (built %s)", Version, BuildDate)

	rootCmd.PersistentFlags().StringVar(&flagAPIKey, "api-key", "", "Poe API key (prefer POE_API_KEY env var)")
	rootCmd.PersistentFlags().BoolVar(&flagJSON, "json", false, "Output raw JSON to stdout")
	rootCmd.PersistentFlags().BoolVar(&flagPlain, "plain", false, "Stable line-based output (script-friendly)")
	rootCmd.PersistentFlags().BoolVar(&flagNoColor, "no-color", false, "Disable ANSI color")
	rootCmd.PersistentFlags().BoolVarP(&flagQuiet, "quiet", "q", false, "Suppress non-essential output")
	rootCmd.PersistentFlags().BoolVarP(&flagVerbose, "verbose", "v", false, "Show HTTP request details on stderr")
	rootCmd.PersistentFlags().IntVar(&flagTimeout, "timeout", 30, "HTTP timeout in seconds")
}
