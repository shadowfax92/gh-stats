package cmd

import (
	"fmt"

	"github.com/fatih/color"
	gh "github.com/nickhudkins/gh-stats/github"
	"github.com/nickhudkins/gh-stats/render"
	"github.com/spf13/cobra"
)

var commitsCmd = &cobra.Command{
	Use:   "commits",
	Short: "Detailed commit stats",
	RunE: func(cmd *cobra.Command, args []string) error {
		bold := color.New(color.Bold)
		cyanBold := color.New(color.FgCyan, color.Bold)

		weekCommits := make([]int, weeks)
		weekLabels := make([]string, weeks)
		var thisWeekRepos []gh.RepoContribution

		for i := 0; i < weeks; i++ {
			start, end := weekBounds(weeks - 1 - i)
			contribs, err := client.FetchContributions(start, end)
			if err != nil {
				return err
			}
			weekCommits[i] = contribs.TotalCommits
			if weeks-1-i == 0 {
				weekLabels[i] = "This wk"
				thisWeekRepos = contribs.CommitRepos
			} else {
				weekLabels[i] = start.Format("Jan 02")
			}
		}

		thisStart, thisEnd := weekBounds(0)
		bold.Printf("Commits  %s — %s\n", thisStart.Format("Jan 2"), thisEnd.Format("Jan 2"))
		fmt.Println()

		totalThisWeek := weekCommits[weeks-1]
		totalLastWeek := 0
		if weeks > 1 {
			totalLastWeek = weekCommits[weeks-2]
		}
		render.WeekComparison("Summary", totalThisWeek, totalLastWeek, cyanBold)
		fmt.Println()

		bold.Printf("Weekly Trend (%d weeks)\n", weeks)
		render.VerticalBars(weekCommits, weekLabels, color.New(color.FgCyan))
		fmt.Println()

		render.RepoBreakdown("Commits by Repo (this week)", thisWeekRepos, cyanBold, 0)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(commitsCmd)
}
