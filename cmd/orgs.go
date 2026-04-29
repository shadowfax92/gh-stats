package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/nickhudkins/gh-stats/render"
	"github.com/spf13/cobra"
)

var orgsCmd = &cobra.Command{
	Use:         "orgs",
	Short:       "List your GitHub organizations",
	Annotations: map[string]string{"group": groupTeam},
	RunE: func(cmd *cobra.Command, args []string) error {
		stop := startSpinner("Listing organizations...")
		orgs, _, err := client.ListOrgsCached(fetchOpts())
		stop()
		if err != nil {
			return err
		}

		if jsonOutput {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(orgs)
		}

		render.Bold.Println("Your Organizations")
		fmt.Println()

		if len(orgs) == 0 {
			render.Dim.Println("  No organizations found.")
			return nil
		}

		for _, org := range orgs {
			fmt.Printf("  %s", color.New(color.FgCyan, color.Bold).Sprint(org.Login))
			if org.Description != "" {
				render.Dim.Printf("  %s", org.Description)
			}
			fmt.Println()
		}

		fmt.Println()
		render.Dim.Println("Run: gh-stats team <org> to see team stats")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(orgsCmd)
}
