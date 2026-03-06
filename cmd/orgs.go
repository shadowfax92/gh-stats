package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var orgsCmd = &cobra.Command{
	Use:   "orgs",
	Short: "List your GitHub organizations",
	RunE: func(cmd *cobra.Command, args []string) error {
		orgs, err := client.ListOrgs()
		if err != nil {
			return err
		}

		if jsonOutput {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(orgs)
		}

		bold := color.New(color.Bold)
		dim := color.New(color.Faint)

		bold.Println("Your Organizations")
		fmt.Println()

		if len(orgs) == 0 {
			dim.Println("  No organizations found.")
			return nil
		}

		for _, org := range orgs {
			fmt.Printf("  %s", color.New(color.FgCyan, color.Bold).Sprint(org.Login))
			if org.Description != "" {
				dim.Printf("  %s", org.Description)
			}
			fmt.Println()
		}

		fmt.Println()
		dim.Println("Run: gh-stats team <org> to see team stats")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(orgsCmd)
}
