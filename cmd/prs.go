package cmd

import (
	"fmt"

	"github.com/fatih/color"
	gh "github.com/nickhudkins/gh-stats/github"
	"github.com/nickhudkins/gh-stats/render"
	"github.com/spf13/cobra"
)

var prsCmd = &cobra.Command{
	Use:   "prs",
	Short: "Detailed pull request stats",
	RunE: func(cmd *cobra.Command, args []string) error {
		bold := color.New(color.Bold)
		greenBold := color.New(color.FgGreen, color.Bold)

		weekPRs := make([]int, weeks)
		weekLabels := make([]string, weeks)
		var thisWeekRepos []gh.RepoContribution

		for i := 0; i < weeks; i++ {
			start, end := weekBounds(weeks - 1 - i)
			contribs, err := client.FetchContributions(start, end)
			if err != nil {
				return err
			}
			weekPRs[i] = contribs.TotalPRs
			if weeks-1-i == 0 {
				weekLabels[i] = "This wk"
				thisWeekRepos = contribs.PRRepos
			} else {
				weekLabels[i] = start.Format("Jan 02")
			}
		}

		thisStart, thisEnd := weekBounds(0)
		bold.Printf("Pull Requests  %s — %s\n", thisStart.Format("Jan 2"), thisEnd.Format("Jan 2"))
		fmt.Println()

		totalThisWeek := weekPRs[weeks-1]
		totalLastWeek := 0
		if weeks > 1 {
			totalLastWeek = weekPRs[weeks-2]
		}
		render.WeekComparison("Summary", totalThisWeek, totalLastWeek, greenBold)
		fmt.Println()

		bold.Printf("Weekly Trend (%d weeks)\n", weeks)
		render.VerticalBars(weekPRs, weekLabels, color.New(color.FgGreen))
		fmt.Println()

		render.RepoBreakdown("PRs by Repo (this week)", thisWeekRepos, greenBold, 0)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(prsCmd)
}
