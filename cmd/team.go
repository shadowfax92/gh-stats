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

		bold := color.New(color.Bold)
		dim := color.New(color.Faint)
		if !jsonOutput {
			bold.Printf("Fetching stats for %s", color.New(color.FgCyan, color.Bold).Sprint(org))
			dim.Printf(" (%d members, %d weeks)...\n", len(members), weeks)
		}

		// Fetch all weeks
		type weekData struct {
			stats *gh.TeamStats
			start time.Time
			end   time.Time
		}
		allWeeks := make([]weekData, weeks)
		for i := 0; i < weeks; i++ {
			start, end := weekBounds(weeks - 1 - i)
			stats, err := client.FetchTeamStats(org, members, start, end)
			if err != nil {
				return err
			}
			allWeeks[i] = weekData{stats: stats, start: start, end: end}
		}

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
				Org      string                    `json:"org"`
				ThisWeek weekJ                     `json:"this_week"`
				LastWeek weekJ                     `json:"last_week"`
				Members  []gh.MemberStats          `json:"members"`
				Repos    []gh.RepoContribution     `json:"repos"`
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

		greenBold := color.New(color.FgGreen, color.Bold)
		cyanBold := color.New(color.FgCyan, color.Bold)
		magentaBold := color.New(color.FgMagenta, color.Bold)

		fmt.Println()
		bold.Printf("Team Stats: %s", cyanBold.Sprint(org))
		dim.Printf("  (%s – %s)\n", thisWeek.start.Format("Jan 2"), thisWeek.end.Format("Jan 2"))
		fmt.Println()

		// Week-over-week comparison
		lastCommits, lastPRs := 0, 0
		if lastWeek.stats != nil {
			lastCommits = lastWeek.stats.TotalCommits
			lastPRs = lastWeek.stats.TotalPRs
		}
		render.WeekComparison("Pull Requests", thisWeek.stats.TotalPRs, lastPRs, greenBold)
		render.WeekComparison("Commits      ", thisWeek.stats.TotalCommits, lastCommits, cyanBold)
		fmt.Println()

		// Weekly trend charts
		if weeks > 1 {
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

			bold.Printf("Weekly Commits (%d weeks)\n", weeks)
			render.VerticalBars(weekCommits, weekLabels, color.New(color.FgCyan))
			fmt.Println()

			bold.Printf("Weekly PRs (%d weeks)\n", weeks)
			render.VerticalBars(weekPRs, weekLabels, color.New(color.FgGreen))
			fmt.Println()
		}

		// Daily activity charts (this week + last week)
		var allDays, allPRDays []gh.DayContribution
		if lastWeek.stats != nil {
			allDays = combineDays(thisWeek.stats.Days, lastWeek.stats.Days)
			allPRDays = combineDays(thisWeek.stats.PRDays, lastWeek.stats.PRDays)
		} else {
			allDays = thisWeek.stats.Days
			allPRDays = thisWeek.stats.PRDays
		}

		if len(allDays) > 0 {
			dayValues := make([]int, len(allDays))
			dayLabels := make([]string, len(allDays))
			now := time.Now()
			today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
			for i, d := range allDays {
				dayValues[i] = d.Count
				if d.Date.Equal(today) {
					dayLabels[i] = "Today"
				} else {
					dayLabels[i] = d.Date.Format("Mon 02")
				}
			}
			bold.Println("Team Daily Activity (last 2 weeks)")
			render.VerticalBars(dayValues, dayLabels, color.New(color.FgCyan))
			fmt.Println()
		}

		if len(allPRDays) > 0 {
			prValues := make([]int, len(allPRDays))
			prLabels := make([]string, len(allPRDays))
			now := time.Now()
			today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
			for i, d := range allPRDays {
				prValues[i] = d.Count
				if d.Date.Equal(today) {
					prLabels[i] = "Today"
				} else {
					prLabels[i] = d.Date.Format("Mon 02")
				}
			}
			bold.Println("Team Daily PRs (last 2 weeks)")
			render.VerticalBars(prValues, prLabels, color.New(color.FgGreen))
			fmt.Println()
		}

		// Member leaderboard
		var lastMembers []gh.MemberStats
		if lastWeek.stats != nil {
			lastMembers = lastWeek.stats.Members
		}
		render.MemberLeaderboard("Member Activity (this week)", thisWeek.stats.Members, lastMembers, magentaBold)

		// Repos
		render.RepoBreakdown("Active Repos", thisWeek.stats.OrgRepos, cyanBold, 10)

		return nil
	},
}

func init() {
	teamCmd.Flags().StringVar(&memberFilter, "member", "", "Filter to a specific team member")
	rootCmd.AddCommand(teamCmd)
}
