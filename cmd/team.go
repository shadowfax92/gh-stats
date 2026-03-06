package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/nickhudkins/gh-stats/github"
	"github.com/nickhudkins/gh-stats/render"
	"github.com/spf13/cobra"
)

var memberFilter string

var teamCmd = &cobra.Command{
	Use:   "team <org>",
	Short: "Team contribution stats for an organization",
	Long:  "View team-level GitHub stats for an organization you belong to.\nShows member leaderboard, team totals, and org repo activity.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		org := args[0]

		members, err := client.ListOrgMembers(org)
		if err != nil {
			return fmt.Errorf("could not list members for %s: %w", org, err)
		}
		if len(members) == 0 {
			return fmt.Errorf("no members found for org %s (check access)", org)
		}

		if memberFilter != "" {
			found := false
			for _, m := range members {
				if strings.EqualFold(m, memberFilter) {
					members = []string{m}
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("member %q not found in org %s", memberFilter, org)
			}
		}

		thisStart, thisEnd := weekBounds(0)
		lastStart, lastEnd := weekBounds(1)

		bold := color.New(color.Bold)
		dim := color.New(color.Faint)
		if !jsonOutput {
			bold.Printf("Fetching stats for %s", color.New(color.FgCyan, color.Bold).Sprint(org))
			dim.Printf(" (%d members)...\n", len(members))
		}

		thisWeek, err := client.FetchTeamStats(org, members, thisStart, thisEnd)
		if err != nil {
			return err
		}
		lastWeek, err := client.FetchTeamStats(org, members, lastStart, lastEnd)
		if err != nil {
			return err
		}

		if jsonOutput {
			type weekJ struct {
				Commits int `json:"commits"`
				PRs     int `json:"prs"`
			}
			out := struct {
				Org      string                `json:"org"`
				ThisWeek weekJ                 `json:"this_week"`
				LastWeek weekJ                 `json:"last_week"`
				Members  []github.MemberStats  `json:"members"`
				Repos    []github.RepoContribution `json:"repos"`
			}{
				Org:      org,
				ThisWeek: weekJ{Commits: thisWeek.TotalCommits, PRs: thisWeek.TotalPRs},
				LastWeek: weekJ{Commits: lastWeek.TotalCommits, PRs: lastWeek.TotalPRs},
				Members:  thisWeek.Members,
				Repos:    thisWeek.OrgRepos,
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(out)
		}

		greenBold := color.New(color.FgGreen, color.Bold)
		cyanBold := color.New(color.FgCyan, color.Bold)
		magentaBold := color.New(color.FgMagenta, color.Bold)

		fmt.Println()
		bold.Printf("Team Stats: %s", cyanBold.Sprint(org))
		dim.Printf("  (%s – %s)\n", thisStart.Format("Jan 2"), thisEnd.Format("Jan 2"))
		fmt.Println()

		render.WeekComparison("Pull Requests", thisWeek.TotalPRs, lastWeek.TotalPRs, greenBold)
		render.WeekComparison("Commits      ", thisWeek.TotalCommits, lastWeek.TotalCommits, cyanBold)
		fmt.Println()

		render.MemberLeaderboard("Member Activity (this week)", thisWeek.Members, lastWeek.Members, magentaBold)

		render.RepoBreakdown("Active Repos", thisWeek.OrgRepos, cyanBold, 10)

		return nil
	},
}

func init() {
	teamCmd.Flags().StringVar(&memberFilter, "member", "", "Filter to a specific team member")
	rootCmd.AddCommand(teamCmd)
}
