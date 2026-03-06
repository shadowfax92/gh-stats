package cmd

import (
	"fmt"
	"time"

	"github.com/fatih/color"
	"github.com/nickhudkins/gh-stats/config"
	gh "github.com/nickhudkins/gh-stats/github"
	"github.com/nickhudkins/gh-stats/render"
	"github.com/spf13/cobra"
)

var (
	jsonOutput bool
	weeks      int
	username   string
	client     *gh.Client
)

var rootCmd = &cobra.Command{
	Use:   "gh-stats",
	Short: "Personal GitHub contribution stats",
	Long:  "View your GitHub contribution stats — PRs, commits, repos, day-over-day and week-over-week trends.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		if username == "" {
			username = cfg.Username
		}
		if username == "" {
			username, err = config.DetectUsername()
			if err != nil {
				return fmt.Errorf("could not detect GitHub username: %w\nRun: gh auth login", err)
			}
			cfg.Username = username
			_ = config.Save(cfg)
		}

		token, err := config.GetToken()
		if err != nil {
			return fmt.Errorf("could not get GitHub token: %w\nRun: gh auth login", err)
		}

		client = &gh.Client{Token: token, Username: username}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return dashboard()
	},
}

func weekBounds(weeksAgo int) (time.Time, time.Time) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	daysFromMonday := (int(today.Weekday()) + 6) % 7
	thisMonday := today.AddDate(0, 0, -daysFromMonday)
	startMonday := thisMonday.AddDate(0, 0, -7*weeksAgo)

	end := startMonday.AddDate(0, 0, 6)
	if end.After(today) {
		end = today
	}

	endTime := time.Date(end.Year(), end.Month(), end.Day(), 23, 59, 59, 0, end.Location())
	return startMonday, endTime
}

func dashboard() error {
	thisStart, thisEnd := weekBounds(0)
	lastStart, lastEnd := weekBounds(1)

	thisWeek, err := client.FetchContributions(thisStart, thisEnd)
	if err != nil {
		return err
	}
	lastWeek, err := client.FetchContributions(lastStart, lastEnd)
	if err != nil {
		return err
	}

	if jsonOutput {
		return render.ContributionsJSON(thisWeek, lastWeek)
	}

	bold := color.New(color.Bold)
	dim := color.New(color.Faint)
	greenBold := color.New(color.FgGreen, color.Bold)
	cyanBold := color.New(color.FgCyan, color.Bold)

	bold.Printf("GitHub Stats for %s", username)
	dim.Printf("  (%s – %s)\n", thisStart.Format("Jan 2"), thisEnd.Format("Jan 2"))
	fmt.Println()

	render.WeekComparison("Pull Requests", thisWeek.TotalPRs, lastWeek.TotalPRs, greenBold)
	render.WeekComparison("Commits      ", thisWeek.TotalCommits, lastWeek.TotalCommits, cyanBold)
	fmt.Println()

	// Daily activity chart
	allDays := combineDays(thisWeek.Days, lastWeek.Days)
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
		bold.Println("Daily Activity (last 2 weeks)")
		render.VerticalBars(dayValues, dayLabels, color.New(color.FgCyan))
		fmt.Println()
	}

	render.RepoBreakdown("Commits by Repo", thisWeek.CommitRepos, cyanBold, 8)
	render.RepoBreakdown("PRs by Repo", thisWeek.PRRepos, greenBold, 8)

	return nil
}

func combineDays(thisWeek, lastWeek []gh.DayContribution) []gh.DayContribution {
	seen := map[string]gh.DayContribution{}
	for _, d := range lastWeek {
		seen[d.Date.Format("2006-01-02")] = d
	}
	for _, d := range thisWeek {
		seen[d.Date.Format("2006-01-02")] = d
	}

	var all []gh.DayContribution
	for _, d := range seen {
		all = append(all, d)
	}

	sortDays(all)
	return all
}

func sortDays(days []gh.DayContribution) {
	for i := 1; i < len(days); i++ {
		for j := i; j > 0 && days[j].Date.Before(days[j-1].Date); j-- {
			days[j], days[j-1] = days[j-1], days[j]
		}
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	rootCmd.PersistentFlags().IntVar(&weeks, "weeks", 4, "Number of weeks for trends")
	rootCmd.PersistentFlags().StringVar(&username, "user", "", "GitHub username (auto-detected from gh)")
}

func Execute() error {
	return rootCmd.Execute()
}
