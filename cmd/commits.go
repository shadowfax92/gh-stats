package cmd

import (
	"fmt"
	"time"

	"github.com/fatih/color"
	gh "github.com/nickhudkins/gh-stats/github"
	"github.com/nickhudkins/gh-stats/render"
	"github.com/spf13/cobra"
)

var commitsCmd = &cobra.Command{
	Use:         "commits",
	Short:       "Detailed commit stats with daily breakdown",
	Annotations: map[string]string{"group": groupViews},
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDailyView("Commits",
			color.New(color.FgCyan), render.CyanBold,
			func(c *gh.Contributions) []gh.DayContribution { return c.Days },
			func(c *gh.Contributions) []gh.RepoContribution { return c.CommitRepos })
	},
}

func runDailyView(label string, barColor *color.Color, headerColor *color.Color,
	dayGetter func(*gh.Contributions) []gh.DayContribution,
	repoGetter func(*gh.Contributions) []gh.RepoContribution) error {

	numWeeks := weeksForDays(days)
	stop := startSpinner(fmt.Sprintf("Fetching %d weeks from GitHub...", numWeeks))
	weekly, _, err := fetchAllWeeks(numWeeks)
	stop()
	if err != nil {
		return err
	}

	now := time.Now()
	today := startOfDay(now)
	dayData := aggregateDays(weekly, dayGetter)

	headerColor.Print(label)
	render.Dim.Printf("  ·  %s\n", today.Format("Mon Jan 2"))
	fmt.Println()

	yest := today.AddDate(0, 0, -1)
	todayCount := render.CountOn(dayData, today)
	yestCount := render.CountOn(dayData, yest)

	thisMon, thisSun := render.WeekBounds(today)
	if thisSun.After(today) {
		thisSun = today
	}
	lastMon := thisMon.AddDate(0, 0, -7)
	lastSun := thisMon.AddDate(0, 0, -1)
	thisWk := render.SumDays(dayData, thisMon, thisSun)
	lastWk := render.SumDays(dayData, lastMon, lastSun)

	render.Bold.Println("Today")
	fmt.Printf("  %-15s ", label)
	render.Bold.Printf("%4d\n", todayCount)
	fmt.Println()

	render.Bold.Println("Trends")
	fmt.Printf("  %-16s ", "Day-over-Day")
	render.Dim.Printf("%d → %d   ", yestCount, todayCount)
	render.PctColorInt(todayCount, yestCount).Printf("%s\n", render.FormatPctInt(todayCount, yestCount))
	fmt.Printf("  %-16s ", "Week-over-Week")
	render.Dim.Printf("%d → %d   ", lastWk, thisWk)
	render.PctColorInt(thisWk, lastWk).Printf("%s\n", render.FormatPctInt(thisWk, lastWk))
	fmt.Println()

	filled := render.FillDays(dayData, today, days)
	values := make([]int, len(filled))
	labels := make([]string, len(filled))
	todayKey := today.Format("2006-01-02")
	yestKey := today.AddDate(0, 0, -1).Format("2006-01-02")
	for i, d := range filled {
		values[i] = d.Count
		key := d.Date.Format("2006-01-02")
		switch key {
		case todayKey:
			labels[i] = "Today"
		case yestKey:
			labels[i] = "Yest"
		default:
			labels[i] = d.Date.Format("Mon 02")
		}
	}
	render.Bold.Printf("Daily · last %d days\n", days)
	render.VerticalBars(values, labels, barColor)
	fmt.Println()

	repos := aggregateRepos(weekly, func(c *gh.Contributions) []gh.RepoContribution {
		if c == weekly[len(weekly)-1] {
			return repoGetter(c)
		}
		return nil
	})
	render.RepoBreakdown(fmt.Sprintf("%s by repo · this week", label), repos, headerColor, 0)
	return nil
}

func init() {
	rootCmd.AddCommand(commitsCmd)
}
