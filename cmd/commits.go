package cmd

import (
	"fmt"

	"github.com/fatih/color"
	gh "github.com/nickhudkins/gh-stats/github"
	"github.com/nickhudkins/gh-stats/render"
	"github.com/spf13/cobra"
)

var commitsCmd = &cobra.Command{
	Use:         "commits",
	Short:       "Detailed commit stats with weekly trend",
	Annotations: map[string]string{"group": groupViews},
	RunE: func(cmd *cobra.Command, args []string) error {
		return runWeeklyView("Commits", color.New(color.FgCyan), render.CyanBold,
			func(c *gh.Contributions) (int, []gh.RepoContribution) {
				return c.TotalCommits, c.CommitRepos
			})
	},
}

func runWeeklyView(label string, barColor *color.Color, headerColor *color.Color, extract func(*gh.Contributions) (int, []gh.RepoContribution)) error {
	stop := startSpinner(fmt.Sprintf("Fetching %d weeks from GitHub...", weeks))

	weekValues := make([]int, weeks)
	weekLabels := make([]string, weeks)
	var thisWeekRepos []gh.RepoContribution

	for i := 0; i < weeks; i++ {
		start, end := weekBounds(weeks - 1 - i)
		contribs, _, err := client.FetchContributionsCached(start, end, fetchOpts())
		if err != nil {
			stop()
			return err
		}
		val, repos := extract(contribs)
		weekValues[i] = val
		if i == weeks-1 {
			weekLabels[i] = "This wk"
			thisWeekRepos = repos
		} else {
			weekLabels[i] = start.Format("Jan 02")
		}
	}
	stop()

	thisStart, thisEnd := weekBounds(0)
	render.Bold.Printf("%s", label)
	render.Dim.Printf("  ·  %s → %s\n", thisStart.Format("Jan 2"), thisEnd.Format("Jan 2"))
	fmt.Println()

	totalThis := weekValues[weeks-1]
	totalLast := 0
	if weeks > 1 {
		totalLast = weekValues[weeks-2]
	}
	render.CompactWeekRow("This week", totalThis, totalLast, headerColor)
	fmt.Println()

	render.Bold.Printf("Weekly trend · last %d weeks\n", weeks)
	render.VerticalBars(weekValues, weekLabels, barColor)
	fmt.Println()

	render.RepoBreakdown(fmt.Sprintf("%s by repo (this week)", label), thisWeekRepos, headerColor, 0)
	return nil
}

func init() {
	rootCmd.AddCommand(commitsCmd)
}
