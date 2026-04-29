package cmd

import (
	"fmt"
	"strings"
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
	noCache    bool
	detailed   bool
	cacheTTL   = 5 * time.Minute
	client     *gh.Client
)

const (
	groupViews    = "Views:"
	groupTeam     = "Teams:"
	groupData     = "Data:"
	groupSettings = "Settings:"
)

var rootCmd = &cobra.Command{
	Use:           "gh-stats",
	Short:         "Personal GitHub contribution stats",
	Long:          "View your GitHub contribution stats — PRs, commits, repos, day-over-day and week-over-week trends.",
	SilenceUsage:  true,
	SilenceErrors: true,
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

func fetchOpts() gh.FetchOptions {
	return gh.FetchOptions{NoCache: noCache, CacheTTL: cacheTTL}
}

func weekBounds(weeksAgo int) (time.Time, time.Time) {
	now := time.Now()
	today := startOfDay(now)

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

func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func dashboard() error {
	thisStart, thisEnd := weekBounds(0)
	lastStart, lastEnd := weekBounds(1)

	stop := startSpinner("Fetching contributions from GitHub...")
	thisWeek, hit1, err := client.FetchContributionsCached(thisStart, thisEnd, fetchOpts())
	if err != nil {
		stop()
		return err
	}
	lastWeek, hit2, err := client.FetchContributionsCached(lastStart, lastEnd, fetchOpts())
	stop()
	if err != nil {
		return err
	}

	if jsonOutput {
		return render.ContributionsJSON(thisWeek, lastWeek)
	}

	cached := hit1 && hit2

	now := time.Now()
	today := startOfDay(now)

	render.Bold.Print("GitHub Stats")
	render.Dim.Printf("  ·  %s", username)
	render.Dim.Printf("  ·  %s", today.Format("Mon Jan 2"))
	if cached {
		render.Dim.Print("  ·  cached")
	}
	fmt.Println()
	fmt.Println()

	render.Bold.Println("This Week")
	render.Dim.Printf("  %s → %s\n", thisStart.Format("Jan 2"), thisEnd.Format("Jan 2"))
	render.CompactWeekRow("Pull Requests", thisWeek.TotalPRs, lastWeek.TotalPRs, render.GreenBold)
	render.CompactWeekRow("Commits", thisWeek.TotalCommits, lastWeek.TotalCommits, render.CyanBold)
	render.CompactWeekRow("Total", thisWeek.TotalPRs+thisWeek.TotalCommits, lastWeek.TotalPRs+lastWeek.TotalCommits, render.MagentaBold)
	fmt.Println()

	allDays := combineDays(thisWeek.Days, lastWeek.Days)
	allPRDays := combineDays(thisWeek.PRDays, lastWeek.PRDays)

	if detailed {
		renderDailyBars("Daily Commits (last 2 weeks)", allDays, today, color.New(color.FgCyan))
		renderDailyBars("Daily PRs (last 2 weeks)", allPRDays, today, color.New(color.FgGreen))
	} else {
		renderSparklines(allDays, allPRDays, today)
	}

	render.RepoBreakdown("Top Repos · commits this week", thisWeek.CommitRepos, render.CyanBold, 6)
	if len(thisWeek.PRRepos) > 0 {
		render.RepoBreakdown("Top Repos · PRs this week", thisWeek.PRRepos, render.GreenBold, 6)
	}

	return nil
}

func renderSparklines(commitDays, prDays []gh.DayContribution, today time.Time) {
	if len(commitDays) == 0 && len(prDays) == 0 {
		return
	}

	commitVals, prVals, first, last := alignDayValues(commitDays, prDays, today, 14)
	if commitVals == nil && prVals == nil {
		return
	}

	render.Bold.Println("Last 14 days")
	if commitVals != nil {
		render.Cyan.Printf("  %s", render.Sparkline(commitVals))
		render.Dim.Println("   commits")
	}
	if prVals != nil {
		render.Green.Printf("  %s", render.Sparkline(prVals))
		render.Dim.Println("   PRs")
	}
	pad := strings.Repeat(" ", maxInt(1, 14-len(first)-len(last)))
	render.Dim.Printf("  %s%s%s\n", first, pad, last)
	fmt.Println()
}

func alignDayValues(commitDays, prDays []gh.DayContribution, today time.Time, window int) ([]int, []int, string, string) {
	commitMap := map[string]int{}
	for _, d := range commitDays {
		commitMap[d.Date.Format("2006-01-02")] = d.Count
	}
	prMap := map[string]int{}
	for _, d := range prDays {
		prMap[d.Date.Format("2006-01-02")] = d.Count
	}

	commitVals := make([]int, window)
	prVals := make([]int, window)
	for i := 0; i < window; i++ {
		d := today.AddDate(0, 0, -(window - 1 - i))
		key := d.Format("2006-01-02")
		commitVals[i] = commitMap[key]
		prVals[i] = prMap[key]
	}

	first := today.AddDate(0, 0, -(window - 1)).Format("Jan 02")
	last := "Today"

	hasCommits := false
	for _, v := range commitVals {
		if v > 0 {
			hasCommits = true
			break
		}
	}
	hasPRs := false
	for _, v := range prVals {
		if v > 0 {
			hasPRs = true
			break
		}
	}

	if !hasCommits {
		commitVals = nil
	}
	if !hasPRs {
		prVals = nil
	}
	return commitVals, prVals, first, last
}

func renderDailyBars(label string, days []gh.DayContribution, today time.Time, c *color.Color) {
	if len(days) == 0 {
		return
	}
	todayKey := today.Format("2006-01-02")
	yestKey := today.AddDate(0, 0, -1).Format("2006-01-02")
	values := make([]int, len(days))
	labels := make([]string, len(days))
	for i, d := range days {
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
	render.Bold.Println(label)
	render.VerticalBars(values, labels, c)
	fmt.Println()
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

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

const usageTemplate = `{{helpHeader "Usage:"}}{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

{{helpHeader "Aliases:"}}
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

{{helpHeader "Examples:"}}
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}
{{groupedHelp .}}{{end}}{{if .HasAvailableLocalFlags}}

{{helpHeader "Flags:"}}
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

{{helpHeader "Global Flags:"}}
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableSubCommands}}

{{helpHint (printf "Use \"%s [command] --help\" for more information." .CommandPath)}}{{end}}
`

var (
	helpHeaderColor = color.New(color.Bold, color.FgCyan)
	helpCmdColor    = color.New(color.FgHiGreen)
	helpAliasColor  = color.New(color.FgYellow)
	helpHintColor   = color.New(color.Faint)
)

func helpHeader(s string) string { return helpHeaderColor.Sprint(s) }
func helpCmdCol(s string) string { return helpCmdColor.Sprint(s) }
func helpHint(s string) string   { return helpHintColor.Sprint(s) }
func helpAliases(aliases []string) string {
	return helpAliasColor.Sprintf("(aliases: %s)", strings.Join(aliases, ", "))
}

var groupOrder = []string{groupViews, groupTeam, groupData, groupSettings, "Other:"}

func groupedHelp(cmd *cobra.Command) string {
	groups := map[string][]*cobra.Command{}
	for _, c := range cmd.Commands() {
		if !c.IsAvailableCommand() && c.Name() != "help" {
			continue
		}
		g := c.Annotations["group"]
		if g == "" {
			g = "Other:"
		}
		groups[g] = append(groups[g], c)
	}
	var b strings.Builder
	for _, name := range groupOrder {
		cmds, ok := groups[name]
		if !ok {
			continue
		}
		b.WriteString("\n" + helpHeader(name) + "\n")
		for _, c := range cmds {
			line := "  " + helpCmdCol(fmt.Sprintf("%-12s", c.Name())) + " " + c.Short
			if len(c.Aliases) > 0 {
				line += " " + helpAliases(c.Aliases)
			}
			b.WriteString(line + "\n")
		}
	}
	return b.String()
}

func init() {
	cobra.AddTemplateFunc("helpHeader", helpHeader)
	cobra.AddTemplateFunc("helpCmdCol", helpCmdCol)
	cobra.AddTemplateFunc("helpAliases", helpAliases)
	cobra.AddTemplateFunc("helpHint", helpHint)
	cobra.AddTemplateFunc("groupedHelp", groupedHelp)

	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	rootCmd.PersistentFlags().IntVar(&weeks, "weeks", 4, "Number of weeks for trends")
	rootCmd.PersistentFlags().StringVar(&username, "user", "", "GitHub username (auto-detected from gh)")
	rootCmd.PersistentFlags().BoolVar(&noCache, "no-cache", false, "Bypass cache, force re-fetch")
	rootCmd.PersistentFlags().BoolVarP(&detailed, "detailed", "d", false, "Show full daily bar charts and per-week trends")
	rootCmd.SetUsageTemplate(usageTemplate)
}

func Execute() error {
	return rootCmd.Execute()
}
