package cmd

import (
	"github.com/fatih/color"
	gh "github.com/nickhudkins/gh-stats/github"
	"github.com/nickhudkins/gh-stats/render"
	"github.com/spf13/cobra"
)

var prsCmd = &cobra.Command{
	Use:         "prs",
	Short:       "Detailed pull request stats with daily breakdown",
	Annotations: map[string]string{"group": groupViews},
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDailyView("Pull Requests",
			color.New(color.FgGreen), render.GreenBold,
			func(c *gh.Contributions) []gh.DayContribution { return c.PRDays },
			func(c *gh.Contributions) []gh.RepoContribution { return c.PRRepos })
	},
}

func init() {
	rootCmd.AddCommand(prsCmd)
}
