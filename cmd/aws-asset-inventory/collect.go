package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/configservice"
	"github.com/scottbrown/aws-asset-inventory/awsassetinventory"
	"github.com/spf13/cobra"
)

var (
	collectProfile     string
	collectRegions     string
	collectOutput      string
	collectVerbose     bool
	collectConcurrency int
)

var collectCmd = &cobra.Command{
	Use:   "collect",
	Short: "Collect AWS resources from AWS Config",
	Long: `Collect all resources that AWS Config knows about across specified regions.
Outputs the inventory as JSON to stdout or a file.`,
	RunE: runCollect,
}

func init() {
	collectCmd.Flags().StringVarP(&collectProfile, "profile", "p", "", "AWS profile name (uses default credential chain if omitted)")
	collectCmd.Flags().StringVarP(&collectRegions, "regions", "r", "", "Comma-separated list of AWS regions (required)")
	collectCmd.Flags().StringVarP(&collectOutput, "output", "o", "", "Output file path (default: stdout)")
	collectCmd.Flags().BoolVarP(&collectVerbose, "verbose", "v", false, "Show detailed progress during collection")
	collectCmd.Flags().IntVar(&collectConcurrency, "concurrency", 0, "Max concurrent region collections (default 5)")

	_ = collectCmd.MarkFlagRequired("regions")
}

func runCollect(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	regionList := parseRegions(collectRegions)
	if len(regionList) == 0 {
		return fmt.Errorf("at least one region must be specified")
	}

	for _, r := range regionList {
		if !r.IsValid() {
			return fmt.Errorf("invalid region: %s", r)
		}
	}

	if collectProfile != "" {
		fmt.Fprintf(os.Stderr, "Collecting resources from %d region(s) using profile '%s'...\n", len(regionList), collectProfile)
	} else {
		fmt.Fprintf(os.Stderr, "Collecting resources from %d region(s) using default credentials...\n", len(regionList))
	}

	clientFactory := func(region awsassetinventory.Region) awsassetinventory.ConfigClient {
		opts := []func(*config.LoadOptions) error{
			config.WithRegion(region.String()),
		}
		if collectProfile != "" {
			opts = append(opts, config.WithSharedConfigProfile(collectProfile))
		}
		cfg, err := config.LoadDefaultConfig(ctx, opts...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to load config for region %s: %v\n", region, err)
			return nil
		}
		return configservice.NewFromConfig(cfg)
	}

	collector := awsassetinventory.NewCollector(collectProfile, clientFactory)
	if collectConcurrency > 0 {
		collector.MaxConcurrency = collectConcurrency
	}
	if collectVerbose {
		collector.Logger = func(format string, args ...any) {
			fmt.Fprintf(os.Stderr, format+"\n", args...)
		}
	}

	inventory, err := collector.Collect(ctx, regionList)
	if err != nil {
		var collectErrs awsassetinventory.CollectErrors
		if errors.As(err, &collectErrs) {
			failedRegions := collectErrs.Regions()
			fmt.Fprintf(os.Stderr, "Warning: %d region(s) failed: %s\n",
				len(failedRegions), strings.Join(regionStrings(failedRegions), ", "))
			for _, re := range collectErrs.Errors {
				fmt.Fprintf(os.Stderr, "  [%s] %v\n", re.Region, re.Err)
			}
		} else {
			fmt.Fprintf(os.Stderr, "Warning: collection completed with errors: %v\n", err)
		}
	}

	fmt.Fprintf(os.Stderr, "Collected %d resources\n", inventory.ResourceCount())

	data, err := inventory.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize JSON: %w", err)
	}

	if collectOutput == "" || collectOutput == "-" {
		fmt.Println(string(data))
	} else {
		if err := os.WriteFile(collectOutput, data, 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Inventory written to: %s\n", collectOutput)
	}

	return nil
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

func regionStrings(regions []awsassetinventory.Region) []string {
	strs := make([]string, len(regions))
	for i, r := range regions {
		strs[i] = r.String()
	}
	return strs
}
