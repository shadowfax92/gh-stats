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
	Short:       "Team contribution stats with per-member breakdown",
	Long:        "View team-level GitHub stats for an organization you belong to.\nShows window totals, trends, per-member breakdowns, and per-member sparklines for the last N days.",
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

		numWeeks := weeksForDays(days)
		stop = startSpinner(fmt.Sprintf("Fetching %d weeks for %d members from GitHub...", numWeeks, len(members)))

		type weekData struct {
			stats *gh.TeamStats
			start time.Time
			end   time.Time
		}
		allWeeks := make([]weekData, numWeeks)
		for i := 0; i < numWeeks; i++ {
			start, end := weekBounds(numWeeks - 1 - i)
			stats, _, err := client.FetchTeamStatsCached(org, members, start, end, fetchOpts())
			if err != nil {
				stop()
				return err
			}
			allWeeks[i] = weekData{stats: stats, start: start, end: end}
		}
		stop()

		this := allWeeks[numWeeks-1]
		now := time.Now()
		today := startOfDay(now)

		teamCommitDays := []gh.DayContribution{}
		teamPRDays := []gh.DayContribution{}
		memberCommitDays := map[string][]gh.DayContribution{}
		memberPRDays := map[string][]gh.DayContribution{}

		for _, w := range allWeeks {
			teamCommitDays = mergeDays(teamCommitDays, w.stats.Days)
			teamPRDays = mergeDays(teamPRDays, w.stats.PRDays)
			for _, m := range w.stats.Members {
				memberCommitDays[m.Username] = mergeDays(memberCommitDays[m.Username], m.Days)
				memberPRDays[m.Username] = mergeDays(memberPRDays[m.Username], m.PRDays)
			}
		}

		if jsonOutput {
			out := struct {
				Org     string                `json:"org"`
				Members []gh.MemberStats      `json:"members"`
				Repos   []gh.RepoContribution `json:"repos"`
			}{
				Org:     org,
				Members: this.stats.Members,
				Repos:   this.stats.OrgRepos,
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(out)
		}

		render.Bold.Print("Team")
		render.Dim.Printf("  ·  %s  ·  %s\n",
			render.CyanBold.Sprint(org),
			today.Format("Mon Jan 2"))
		fmt.Println()

		renderWindowSummary(windowSummaryLabel(days), "",
			summaryMetric{"Pull Requests", sumContributionDays(render.FillDays(teamPRDays, today, days))},
			summaryMetric{"Commits", sumContributionDays(render.FillDays(teamCommitDays, today, days))})
		renderTrendsSection(teamCommitDays, teamPRDays, today)
		renderWindowByMember(this.stats.Members, memberCommitDays, memberPRDays, today, days)
		renderTrendByMember(this.stats.Members, memberCommitDays, memberPRDays, today)

		renderDailyBars(fmt.Sprintf("Team Commits · last %d days", days),
			render.FillDays(teamCommitDays, today, days), today, color.New(color.FgCyan))
		renderDailyBars(fmt.Sprintf("Team PRs · last %d days", days),
			render.FillDays(teamPRDays, today, days), today, color.New(color.FgGreen))

		renderMemberSparklines("Last "+fmt.Sprintf("%d", days)+" days · commits per member",
			this.stats.Members, memberCommitDays, today, days, render.Cyan)
		renderMemberSparklines("Last "+fmt.Sprintf("%d", days)+" days · PRs per member",
			this.stats.Members, memberPRDays, today, days, render.Green)

		render.RepoBreakdown("Active Repos · this week", this.stats.OrgRepos, render.CyanBold, 10)

		return nil
	},
}

func mergeDays(existing, incoming []gh.DayContribution) []gh.DayContribution {
	byDate := map[string]int{}
	for _, d := range existing {
		byDate[d.Date.Format("2006-01-02")] = d.Count
	}
	for _, d := range incoming {
		byDate[d.Date.Format("2006-01-02")] += d.Count
	}
	out := make([]gh.DayContribution, 0, len(byDate))
	for k, v := range byDate {
		t, _ := time.Parse("2006-01-02", k)
		out = append(out, gh.DayContribution{Date: t, Count: v})
	}
	sortDays(out)
	return out
}

type memberWindowRow struct {
	name    string
	commits int
	prs     int
	total   int
}

// windowMemberRows totals each member over the active date window and sorts by activity.
func windowMemberRows(members []gh.MemberStats, commitDaysByMember, prDaysByMember map[string][]gh.DayContribution, today time.Time, days int) []memberWindowRow {
	rows := make([]memberWindowRow, 0, len(members))
	for _, m := range members {
		c := sumContributionDays(render.FillDays(commitDaysByMember[m.Username], today, days))
		p := sumContributionDays(render.FillDays(prDaysByMember[m.Username], today, days))
		if c == 0 && p == 0 {
			continue
		}
		rows = append(rows, memberWindowRow{m.Username, c, p, c + p})
	}
	for i := 1; i < len(rows); i++ {
		for j := i; j > 0 && rows[j].total > rows[j-1].total; j-- {
			rows[j], rows[j-1] = rows[j-1], rows[j]
		}
	}
	return rows
}

// renderWindowByMember ranks members by total activity in the active date window.
func renderWindowByMember(members []gh.MemberStats, commitDaysByMember, prDaysByMember map[string][]gh.DayContribution, today time.Time, days int) {
	rows := windowMemberRows(members, commitDaysByMember, prDaysByMember, today, days)
	label := windowSummaryLabel(days) + " by Member"
	if len(rows) == 0 {
		render.Bold.Println(label)
		render.Dim.Println("  No activity in this window.")
		fmt.Println()
		return
	}

	maxTotal := rows[0].total
	labelW := 0
	for _, r := range rows {
		if len(r.name) > labelW {
			labelW = len(r.name)
		}
	}
	if labelW > 20 {
		labelW = 20
	}

	render.Bold.Println(label)
	barWidth := 16
	for _, r := range rows {
		filled := r.total * barWidth / maxInt(1, maxTotal)
		if filled == 0 && r.total > 0 {
			filled = 1
		}
		bar := render.Cyan.Sprint(strings.Repeat("█", filled)) + render.Dim.Sprint(strings.Repeat("░", barWidth-filled))
		fmt.Printf("  %-*s %s ", labelW, render.TruncateLeft(r.name, labelW), bar)
		render.Dim.Printf("%d commits, %d PRs\n", r.commits, r.prs)
	}
	fmt.Println()
}

func renderTrendByMember(members []gh.MemberStats, commitDaysByMember, prDaysByMember map[string][]gh.DayContribution, today time.Time) {
	yest := today.AddDate(0, 0, -1)
	thisMon, thisSun := render.WeekBounds(today)
	if thisSun.After(today) {
		thisSun = today
	}
	lastMon := thisMon.AddDate(0, 0, -7)
	lastSun := thisMon.AddDate(0, 0, -1)

	type row struct {
		name              string
		todayTot, yestTot int
		thisWk, lastWk    int
	}

	var rows []row
	for _, m := range members {
		cDays := commitDaysByMember[m.Username]
		pDays := prDaysByMember[m.Username]
		todayT := render.CountOn(cDays, today) + render.CountOn(pDays, today)
		yestT := render.CountOn(cDays, yest) + render.CountOn(pDays, yest)
		twTot := render.SumDays(cDays, thisMon, thisSun) + render.SumDays(pDays, thisMon, thisSun)
		lwTot := render.SumDays(cDays, lastMon, lastSun) + render.SumDays(pDays, lastMon, lastSun)
		if todayT == 0 && yestT == 0 && twTot == 0 && lwTot == 0 {
			continue
		}
		rows = append(rows, row{m.Username, todayT, yestT, twTot, lwTot})
	}
	if len(rows) == 0 {
		return
	}
	for i := 1; i < len(rows); i++ {
		for j := i; j > 0 && rows[j].thisWk > rows[j-1].thisWk; j-- {
			rows[j], rows[j-1] = rows[j-1], rows[j]
		}
	}

	labelW := 0
	for _, r := range rows {
		if len(r.name) > labelW {
			labelW = len(r.name)
		}
	}
	if labelW > 20 {
		labelW = 20
	}

	render.Bold.Println("Trend by Member")
	fmt.Printf("  %-*s   %-13s     %s\n", labelW, "", "DoD (today)", "WoW (this wk)")
	for _, r := range rows {
		fmt.Printf("  %-*s   ", labelW, render.TruncateLeft(r.name, labelW))
		render.Dim.Printf("%d → %-3d ", r.yestTot, r.todayTot)
		render.PctColorInt(r.todayTot, r.yestTot).Printf("%-7s", render.FormatPctInt(r.todayTot, r.yestTot))
		render.Dim.Printf("   %d → %-4d ", r.lastWk, r.thisWk)
		render.PctColorInt(r.thisWk, r.lastWk).Printf("%s", render.FormatPctInt(r.thisWk, r.lastWk))
		fmt.Println()
	}
	fmt.Println()
}

func renderMemberSparklines(label string, members []gh.MemberStats, perMember map[string][]gh.DayContribution, today time.Time, n int, c *color.Color) {
	type row struct {
		name   string
		values []int
		total  int
	}
	var rows []row
	for _, m := range members {
		dayList := perMember[m.Username]
		filled := render.FillDays(dayList, today, n)
		values := make([]int, len(filled))
		total := 0
		for i, d := range filled {
			values[i] = d.Count
			total += d.Count
		}
		if total == 0 {
			continue
		}
		rows = append(rows, row{m.Username, values, total})
	}
	if len(rows) == 0 {
		return
	}
	for i := 1; i < len(rows); i++ {
		for j := i; j > 0 && rows[j].total > rows[j-1].total; j-- {
			rows[j], rows[j-1] = rows[j-1], rows[j]
		}
	}

	labelW := 0
	for _, r := range rows {
		if len(r.name) > labelW {
			labelW = len(r.name)
		}
	}
	if labelW > 20 {
		labelW = 20
	}

	render.Bold.Println(label)
	for _, r := range rows {
		fmt.Printf("  %-*s ", labelW, render.TruncateLeft(r.name, labelW))
		c.Print(render.Sparkline(r.values))
		render.Dim.Printf("  %d\n", r.total)
	}
	fmt.Println()
}

func init() {
	teamCmd.Flags().StringVar(&memberFilter, "member", "", "Filter to a specific team member")
	rootCmd.AddCommand(teamCmd)
}
