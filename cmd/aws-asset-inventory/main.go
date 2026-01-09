package main

import (
	"os"

	"github.com/spf13/cobra"
)

var (
	// Version information set via ldflags at build time
	gitBranch = "unknown"
	gitSHA    = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "aws-asset-inventory",
	Short: "Collect AWS Config resources across regions",
	Long: `A CLI tool that collects all resources AWS Config knows about
across specified regions and generates inventory reports.

Use subcommands to collect resources, generate reports, or view permissions.`,
}

func init() {
	rootCmd.AddCommand(collectCmd)
	rootCmd.AddCommand(reportCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(permissionsCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
