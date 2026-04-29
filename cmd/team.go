package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	gh "github.com/nickhudkins/gh-stats/github"
	"github.com/nickhudkins/gh-stats/render"
	"github.com/spf13/cobra"
)

var memberFilter string

var teamCmd = &cobra.Command{
	Use:         "team <org>",
	Short:       "Team contribution stats for an organization",
	Long:        "View team-level GitHub stats for an organization you belong to.\nShows member leaderboard, team totals, and org repo activity.",
	Annotations: map[string]string{"group": groupTeam},
	Args:        cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		org := args[0]

		stop := startSpinner(fmt.Sprintf("Listing %s members...", org))
		members, _, err := client.ListOrgMembersCached(org, fetchOpts())
		stop()
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

		stop = startSpinner(fmt.Sprintf("Fetching %d weeks for %d members from GitHub...", weeks, len(members)))

		type weekData struct {
			stats *gh.TeamStats
			start time.Time
			end   time.Time
		}
		allWeeks := make([]weekData, weeks)
		for i := 0; i < weeks; i++ {
			start, end := weekBounds(weeks - 1 - i)
			stats, _, err := client.FetchTeamStatsCached(org, members, start, end, fetchOpts())
			if err != nil {
				stop()
				return err
			}
			allWeeks[i] = weekData{stats: stats, start: start, end: end}
		}
		stop()

		thisWeek := allWeeks[weeks-1]
		var lastWeek weekData
		if weeks > 1 {
			lastWeek = allWeeks[weeks-2]
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
				Members  []gh.MemberStats      `json:"members"`
				Repos    []gh.RepoContribution `json:"repos"`
			}{
				Org:      org,
				ThisWeek: weekJ{Commits: thisWeek.stats.TotalCommits, PRs: thisWeek.stats.TotalPRs},
				Members:  thisWeek.stats.Members,
				Repos:    thisWeek.stats.OrgRepos,
			}
			if lastWeek.stats != nil {
				out.LastWeek = weekJ{Commits: lastWeek.stats.TotalCommits, PRs: lastWeek.stats.TotalPRs}
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(out)
		}

		render.Bold.Print("Team Stats")
		render.Dim.Printf("  ·  %s  ·  %s → %s\n",
			render.CyanBold.Sprint(org),
			thisWeek.start.Format("Jan 2"), thisWeek.end.Format("Jan 2"))
		fmt.Println()

		lastCommits, lastPRs := 0, 0
		if lastWeek.stats != nil {
			lastCommits = lastWeek.stats.TotalCommits
			lastPRs = lastWeek.stats.TotalPRs
		}
		render.Bold.Println("This Week")
		render.CompactWeekRow("Pull Requests", thisWeek.stats.TotalPRs, lastPRs, render.GreenBold)
		render.CompactWeekRow("Commits", thisWeek.stats.TotalCommits, lastCommits, render.CyanBold)
		render.CompactWeekRow("Total",
			thisWeek.stats.TotalPRs+thisWeek.stats.TotalCommits,
			lastPRs+lastCommits,
			render.MagentaBold)
		fmt.Println()

		var allDays, allPRDays []gh.DayContribution
		if lastWeek.stats != nil {
			allDays = combineDays(thisWeek.stats.Days, lastWeek.stats.Days)
			allPRDays = combineDays(thisWeek.stats.PRDays, lastWeek.stats.PRDays)
		} else {
			allDays = thisWeek.stats.Days
			allPRDays = thisWeek.stats.PRDays
		}

		now := time.Now()
		today := startOfDay(now)

		if detailed {
			renderDailyBars("Team Daily Commits (last 2 weeks)", allDays, today, color.New(color.FgCyan))
			renderDailyBars("Team Daily PRs (last 2 weeks)", allPRDays, today, color.New(color.FgGreen))
		} else {
			renderSparklines(allDays, allPRDays, today)
		}

		if detailed && weeks > 1 {
			weekCommits := make([]int, weeks)
			weekPRs := make([]int, weeks)
			weekLabels := make([]string, weeks)
			for i, w := range allWeeks {
				weekCommits[i] = w.stats.TotalCommits
				weekPRs[i] = w.stats.TotalPRs
				if i == weeks-1 {
					weekLabels[i] = "This wk"
				} else {
					weekLabels[i] = w.start.Format("Jan 02")
				}
			}
			render.Bold.Printf("Weekly Commits · last %d weeks\n", weeks)
			render.VerticalBars(weekCommits, weekLabels, color.New(color.FgCyan))
			fmt.Println()
			render.Bold.Printf("Weekly PRs · last %d weeks\n", weeks)
			render.VerticalBars(weekPRs, weekLabels, color.New(color.FgGreen))
			fmt.Println()
		}

		var lastMembers []gh.MemberStats
		if lastWeek.stats != nil {
			lastMembers = lastWeek.stats.Members
		}
		render.MemberLeaderboard("Member Activity (this week)", thisWeek.stats.Members, lastMembers, render.MagentaBold)

		render.RepoBreakdown("Active Repos", thisWeek.stats.OrgRepos, render.CyanBold, 10)

		return nil
	},
}

func init() {
	teamCmd.Flags().StringVar(&memberFilter, "member", "", "Filter to a specific team member")
	rootCmd.AddCommand(teamCmd)
}
