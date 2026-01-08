package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/configservice"
	"github.com/scottbrown/aws-asset-inventory/awsassetinventory"
	"github.com/spf13/cobra"
)

var (
	profile        string
	regions        string
	outputFile     string
	reportFile     string
	permissionsOnly bool
	noReport       bool
	includeDetails bool
	verbose        bool
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "aws-asset-inventory",
	Short: "Collect AWS Config resources across regions",
	Long: `A CLI tool that collects all resources AWS Config knows about
across specified regions and generates an inventory report.`,
	RunE: run,
}

func init() {
	rootCmd.Flags().StringVarP(&profile, "profile", "p", "", "AWS profile name (uses default credential chain if omitted)")
	rootCmd.Flags().StringVarP(&regions, "regions", "r", "", "Comma-separated list of AWS regions (required)")
	rootCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Path for JSON inventory output (use '-' for stdout)")
	rootCmd.Flags().StringVar(&reportFile, "report", "", "Path for markdown report (use '-' for stdout)")
	rootCmd.Flags().BoolVar(&noReport, "no-report", false, "Skip markdown report generation")
	rootCmd.Flags().BoolVar(&includeDetails, "include-details", false, "Include resource details in report")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed progress during collection")
	rootCmd.Flags().BoolVar(&permissionsOnly, "permissions", false, "Print required AWS Config permissions and exit")
}

func run(cmd *cobra.Command, args []string) error {
	if permissionsOnly {
		for _, perm := range requiredPermissions() {
			fmt.Fprintln(os.Stdout, perm)
		}
		return nil
	}

	ctx := context.Background()

	if regions == "" {
		return fmt.Errorf("--regions is required")
	}

	regionList := parseRegions(regions)
	if len(regionList) == 0 {
		return fmt.Errorf("at least one region must be specified")
	}

	for _, r := range regionList {
		if !r.IsValid() {
			return fmt.Errorf("invalid region: %s", r)
		}
	}

	if outputFile == "-" && !noReport && (reportFile == "" || reportFile == "-") {
		return fmt.Errorf("cannot write both JSON and report to stdout; use --no-report or specify --report <file>")
	}

	if profile != "" {
		fmt.Fprintf(os.Stderr, "Collecting resources from %d region(s) using profile '%s'...\n", len(regionList), profile)
	} else {
		fmt.Fprintf(os.Stderr, "Collecting resources from %d region(s) using default credentials...\n", len(regionList))
	}

	clientFactory := func(region awsassetinventory.Region) awsassetinventory.ConfigClient {
		opts := []func(*config.LoadOptions) error{
			config.WithRegion(region.String()),
		}
		if profile != "" {
			opts = append(opts, config.WithSharedConfigProfile(profile))
		}
		cfg, err := config.LoadDefaultConfig(ctx, opts...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to load config for region %s: %v\n", region, err)
			return nil
		}
		return configservice.NewFromConfig(cfg)
	}

	collector := awsassetinventory.NewCollector(profile, clientFactory)
	if verbose {
		collector.Logger = func(format string, args ...any) {
			fmt.Fprintf(os.Stderr, format+"\n", args...)
		}
	}
	inventory, err := collector.Collect(ctx, regionList)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: collection completed with errors: %v\n", err)
	}

	fmt.Fprintf(os.Stderr, "Collected %d resources\n", inventory.ResourceCount())

	if outputFile != "" {
		if outputFile == "-" {
			data, err := inventory.ToJSON()
			if err != nil {
				return fmt.Errorf("failed to serialize JSON: %w", err)
			}
			fmt.Fprintln(os.Stdout, string(data))
		} else {
			if err := writeJSONOutput(inventory, outputFile); err != nil {
				return fmt.Errorf("failed to write JSON output: %w", err)
			}
			fmt.Fprintf(os.Stderr, "JSON inventory written to: %s\n", outputFile)
		}
	}

	if !noReport {
		if err := writeReport(inventory, reportFile, includeDetails); err != nil {
			return fmt.Errorf("failed to write report: %w", err)
		}
	}

	return nil
}

func requiredPermissions() []string {
	return []string{
		"config:GetDiscoveredResourceCounts",
		"config:ListDiscoveredResources",
		"config:BatchGetResourceConfig",
	}
}

func parseRegions(input string) []awsassetinventory.Region {
	parts := strings.Split(input, ",")
	regions := make([]awsassetinventory.Region, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			regions = append(regions, awsassetinventory.Region(trimmed))
		}
	}
	return regions
}

func writeJSONOutput(inv *awsassetinventory.Inventory, path string) error {
	data, err := inv.ToJSON()
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func writeReport(inv *awsassetinventory.Inventory, path string, includeDetails bool) error {
	rg := awsassetinventory.NewReportGenerator(inv)
	rg.IncludeDetails = includeDetails

	if path == "" || path == "-" {
		return rg.Generate(os.Stdout)
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := rg.Generate(f); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Report written to: %s\n", path)
	return nil
}
