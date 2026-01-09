package main

import (
	"fmt"
	"os"

	"github.com/scottbrown/aws-asset-inventory/awsassetinventory"
	"github.com/spf13/cobra"
)

var (
	reportInput          string
	reportOutput         string
	reportIncludeDetails bool
)

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Generate a markdown report from inventory JSON",
	Long: `Generate a markdown report from a previously collected inventory JSON file.
The report includes resource counts by type and region.`,
	RunE: runReport,
}

func init() {
	reportCmd.Flags().StringVarP(&reportInput, "input", "i", "", "Input JSON inventory file (required)")
	reportCmd.Flags().StringVarP(&reportOutput, "output", "o", "", "Output file path (default: stdout)")
	reportCmd.Flags().BoolVar(&reportIncludeDetails, "include-details", false, "Include resource details in report")

	_ = reportCmd.MarkFlagRequired("input")
}

func runReport(cmd *cobra.Command, args []string) error {
	data, err := os.ReadFile(reportInput)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	inventory, err := awsassetinventory.LoadFromJSON(data)
	if err != nil {
		return fmt.Errorf("failed to parse inventory JSON: %w", err)
	}

	rg := awsassetinventory.NewReportGenerator(inventory)
	rg.IncludeDetails = reportIncludeDetails

	if reportOutput == "" || reportOutput == "-" {
		return rg.Generate(os.Stdout)
	}

	f, err := os.Create(reportOutput)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer f.Close()

	if err := rg.Generate(f); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Report written to: %s\n", reportOutput)
	return nil
}
