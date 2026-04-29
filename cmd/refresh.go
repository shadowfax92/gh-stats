package cmd

import (
	"fmt"
	"time"

	gh "github.com/nickhudkins/gh-stats/github"
	"github.com/nickhudkins/gh-stats/render"
	"github.com/spf13/cobra"
)

var refreshCmd = &cobra.Command{
	Use:         "refresh",
	Short:       "Bust the cache and re-fetch this/last week's contributions",
	Annotations: map[string]string{"group": groupData},
	RunE: func(cmd *cobra.Command, args []string) error {
		thisStart, thisEnd := weekBounds(0)
		lastStart, lastEnd := weekBounds(1)

		stop := startSpinner("Refreshing contributions from GitHub...")
		start := time.Now()
		if _, _, err := client.FetchContributionsCached(thisStart, thisEnd, gh.FetchOptions{NoCache: true, CacheTTL: cacheTTL}); err != nil {
			stop()
			return err
		}
		if _, _, err := client.FetchContributionsCached(lastStart, lastEnd, gh.FetchOptions{NoCache: true, CacheTTL: cacheTTL}); err != nil {
			stop()
			return err
		}
		stop()

		render.Bold.Printf("Done in %s.\n", time.Since(start).Round(time.Millisecond))
		render.Dim.Printf("Cache: %s\n", gh.CachePath())
		_ = fmt.Println
		return nil
	},
}

func init() {
	rootCmd.AddCommand(refreshCmd)
}
