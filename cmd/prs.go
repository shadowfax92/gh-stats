package cmd

import (
	"github.com/fatih/color"
	gh "github.com/nickhudkins/gh-stats/github"
	"github.com/nickhudkins/gh-stats/render"
	"github.com/spf13/cobra"
)

var prsCmd = &cobra.Command{
	Use:         "prs",
	Short:       "Detailed pull request stats with weekly trend",
	Annotations: map[string]string{"group": groupViews},
	RunE: func(cmd *cobra.Command, args []string) error {
		return runWeeklyView("Pull Requests", color.New(color.FgGreen), render.GreenBold,
			func(c *gh.Contributions) (int, []gh.RepoContribution) {
				return c.TotalPRs, c.PRRepos
			})
	},
}

func init() {
	rootCmd.AddCommand(prsCmd)
}
