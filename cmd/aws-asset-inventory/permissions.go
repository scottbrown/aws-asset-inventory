package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var permissionsCmd = &cobra.Command{
	Use:   "permissions",
	Short: "Print required AWS IAM permissions",
	Long:  `Display the AWS IAM permissions required to use this tool with AWS Config.`,
	Run: func(cmd *cobra.Command, args []string) {
		for _, perm := range requiredPermissions() {
			fmt.Fprintln(cmd.OutOrStdout(), perm)
		}
	},
}

func requiredPermissions() []string {
	return []string{
		"config:GetDiscoveredResourceCounts",
		"config:ListDiscoveredResources",
		"config:BatchGetResourceConfig",
	}
}
