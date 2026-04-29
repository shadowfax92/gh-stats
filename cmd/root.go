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
	days       int
	username   string
	noCache    bool
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
	Long:          "View your GitHub contribution stats — today's PRs and commits, day-over-day and week-over-week trends, sparklines.",
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

// weeksForDays returns how many week-blocks we need to fetch to cover `days`
// days back from today. Slightly conservative.
func weeksForDays(days int) int {
	w := days/7 + 2
	if w < 2 {
		return 2
	}
	return w
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

// fetchAllWeeks fetches the past N weeks (from oldest to current) of personal contribs.
func fetchAllWeeks(numWeeks int) ([]*gh.Contributions, bool, error) {
	all := make([]*gh.Contributions, numWeeks)
	allCached := true
	for i := 0; i < numWeeks; i++ {
		start, end := weekBounds(numWeeks - 1 - i)
		c, hit, err := client.FetchContributionsCached(start, end, fetchOpts())
		if err != nil {
			return nil, false, err
		}
		all[i] = c
		if !hit {
			allCached = false
		}
	}
	return all, allCached, nil
}

// aggregateDays merges per-week Days slices into a single ordered list (oldest → newest).
func aggregateDays(weekly []*gh.Contributions, getter func(*gh.Contributions) []gh.DayContribution) []gh.DayContribution {
	seen := map[string]gh.DayContribution{}
	for _, w := range weekly {
		for _, d := range getter(w) {
			seen[d.Date.Format("2006-01-02")] = d
		}
	}
	out := make([]gh.DayContribution, 0, len(seen))
	for _, d := range seen {
		out = append(out, d)
	}
	sortDays(out)
	return out
}

// aggregateRepos merges per-week repo contributions into a single ranked list.
func aggregateRepos(weekly []*gh.Contributions, getter func(*gh.Contributions) []gh.RepoContribution) []gh.RepoContribution {
	totals := map[string]int{}
	for _, w := range weekly {
		for _, r := range getter(w) {
			totals[r.Repo] += r.Count
		}
	}
	out := make([]gh.RepoContribution, 0, len(totals))
	for repo, count := range totals {
		out = append(out, gh.RepoContribution{Repo: repo, Count: count})
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].Count > out[j-1].Count; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

func dashboard() error {
	numWeeks := weeksForDays(days)

	stop := startSpinner(fmt.Sprintf("Fetching %d weeks from GitHub...", numWeeks))
	weekly, cached, err := fetchAllWeeks(numWeeks)
	stop()
	if err != nil {
		return err
	}

	if jsonOutput {
		// Reuse the latest week vs prior week comparison for JSON.
		this := weekly[len(weekly)-1]
		var last *gh.Contributions
		if len(weekly) > 1 {
			last = weekly[len(weekly)-2]
		}
		return render.ContributionsJSON(this, last)
	}

	now := time.Now()
	today := startOfDay(now)

	commitDays := aggregateDays(weekly, func(c *gh.Contributions) []gh.DayContribution { return c.Days })
	prDays := aggregateDays(weekly, func(c *gh.Contributions) []gh.DayContribution { return c.PRDays })

	render.Bold.Print("GitHub Stats")
	render.Dim.Printf("  ·  %s  ·  %s", username, today.Format("Mon Jan 2"))
	if cached {
		render.Dim.Print("  ·  cached")
	}
	fmt.Println()
	fmt.Println()

	renderTodaySection(commitDays, prDays, today, "")
	renderTrendsSection(commitDays, prDays, today)

	renderDailyBars(fmt.Sprintf("Commits · last %d days", days),
		render.FillDays(commitDays, today, days), today, color.New(color.FgCyan))
	renderDailyBars(fmt.Sprintf("PRs · last %d days", days),
		render.FillDays(prDays, today, days), today, color.New(color.FgGreen))

	thisStart, _ := weekBounds(0)
	commitRepos := aggregateRepos(weekly, func(c *gh.Contributions) []gh.RepoContribution {
		// Show repos for the current week only on the dashboard.
		if c == weekly[len(weekly)-1] {
			return c.CommitRepos
		}
		return nil
	})
	prRepos := aggregateRepos(weekly, func(c *gh.Contributions) []gh.RepoContribution {
		if c == weekly[len(weekly)-1] {
			return c.PRRepos
		}
		return nil
	})

	render.Bold.Print("Top Repos")
	render.Dim.Printf("  ·  this week (%s → today)\n", thisStart.Format("Jan 2"))
	render.RepoBreakdown("Commits", commitRepos, render.CyanBold, 6)
	if len(prRepos) > 0 {
		render.RepoBreakdown("PRs", prRepos, render.GreenBold, 6)
	}

	return nil
}

func renderTodaySection(commitDays, prDays []gh.DayContribution, today time.Time, indent string) {
	commitsToday := render.CountOn(commitDays, today)
	prsToday := render.CountOn(prDays, today)

	render.Bold.Println(indent + "Today")
	fmt.Printf(indent+"  %-15s ", "Pull Requests")
	render.Bold.Printf("%4d\n", prsToday)
	fmt.Printf(indent+"  %-15s ", "Commits")
	render.Bold.Printf("%4d\n", commitsToday)
	fmt.Println()
}

func renderTrendsSection(commitDays, prDays []gh.DayContribution, today time.Time) {
	yest := today.AddDate(0, 0, -1)
	commitsToday := render.CountOn(commitDays, today)
	commitsYest := render.CountOn(commitDays, yest)
	prsToday := render.CountOn(prDays, today)
	prsYest := render.CountOn(prDays, yest)

	thisMon, thisSun := render.WeekBounds(today)
	if thisSun.After(today) {
		thisSun = today
	}
	lastMon := thisMon.AddDate(0, 0, -7)
	lastSun := thisMon.AddDate(0, 0, -1)

	commitsThisWk := render.SumDays(commitDays, thisMon, thisSun)
	commitsLastWk := render.SumDays(commitDays, lastMon, lastSun)
	prsThisWk := render.SumDays(prDays, thisMon, thisSun)
	prsLastWk := render.SumDays(prDays, lastMon, lastSun)

	render.Bold.Println("Trends")
	printTrendDualRow("Day-over-Day", prsToday, prsYest, commitsToday, commitsYest)
	printTrendDualRow("Week-over-Week", prsThisWk, prsLastWk, commitsThisWk, commitsLastWk)
	fmt.Println()
}

func printTrendDualRow(label string, prsCur, prsPrev, comCur, comPrev int) {
	fmt.Printf("  %-16s ", label)
	render.Dim.Print("PRs ")
	render.PctColorInt(prsCur, prsPrev).Printf("%-7s", render.FormatPctInt(prsCur, prsPrev))
	render.Dim.Print("   commits ")
	render.PctColorInt(comCur, comPrev).Printf("%s", render.FormatPctInt(comCur, comPrev))
	fmt.Println()
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
	rootCmd.PersistentFlags().IntVar(&days, "days", 14, "Window in days for trends and charts")
	rootCmd.PersistentFlags().StringVar(&username, "user", "", "GitHub username (auto-detected from gh)")
	rootCmd.PersistentFlags().BoolVar(&noCache, "no-cache", false, "Bypass cache, force re-fetch")
	rootCmd.SetUsageTemplate(usageTemplate)
}

func Execute() error {
	return rootCmd.Execute()
}
