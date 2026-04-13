package render

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/nickhudkins/gh-stats/github"
)

var (
	bold      = color.New(color.Bold)
	dim       = color.New(color.Faint)
	green     = color.New(color.FgGreen)
	greenBold = color.New(color.FgGreen, color.Bold)
	cyan      = color.New(color.FgCyan)
	cyanBold  = color.New(color.FgCyan, color.Bold)
	yellow    = color.New(color.FgYellow)
	red       = color.New(color.FgRed, color.Bold)
	magenta   = color.New(color.FgMagenta)
)

func WeekComparison(label string, thisWeek, lastWeek int, c *color.Color) {
	delta := thisWeek - lastWeek
	var arrow string
	var deltaColor *color.Color
	switch {
	case delta > 0:
		arrow = fmt.Sprintf("+%d", delta)
		deltaColor = greenBold
	case delta < 0:
		arrow = fmt.Sprintf("%d", delta)
		deltaColor = red
	default:
		arrow = "0"
		deltaColor = dim
	}

	c.Print(label)
	fmt.Printf("  This week: ")
	bold.Printf("%d", thisWeek)
	fmt.Printf("  Last week: ")
	dim.Printf("%d", lastWeek)
	fmt.Printf("  ")
	deltaColor.Printf("(%s)", arrow)
	fmt.Println()
}

func VerticalBars(values []int, labels []string, c *color.Color) {
	maxVal := 0
	for _, v := range values {
		if v > maxVal {
			maxVal = v
		}
	}

	if maxVal == 0 {
		fmt.Println("  (none)")
		return
	}

	chartHeight := min(8, maxVal)

	colWidth := 7
	for _, l := range labels {
		if len(l)+1 > colWidth {
			colWidth = len(l) + 1
		}
	}

	for row := chartHeight; row >= 1; row-- {
		threshold := float64(row) / float64(chartHeight) * float64(maxVal)
		fmt.Print("  ")
		for _, v := range values {
			if float64(v) >= threshold && v > 0 {
				bar := c.Sprint("██")
				pad := colWidth - 2
				fmt.Printf("%s%s", bar, strings.Repeat(" ", pad))
			} else {
				fmt.Print(strings.Repeat(" ", colWidth))
			}
		}
		fmt.Println()
	}

	fmt.Print("  ")
	for _, l := range labels {
		fmt.Printf("%-*s", colWidth, l)
	}
	fmt.Println()

	fmt.Print("  ")
	for _, v := range values {
		dim.Printf("%-*d", colWidth, v)
	}
	fmt.Println()
}

func RepoBreakdown(label string, repos []github.RepoContribution, c *color.Color, max int) {
	if len(repos) == 0 {
		return
	}

	c.Println(label)
	maxCount := 0
	for _, r := range repos {
		if r.Count > maxCount {
			maxCount = r.Count
		}
	}

	barWidth := 15
	shown := len(repos)
	if max > 0 && shown > max {
		shown = max
	}

	for _, r := range repos[:shown] {
		filled := 0
		if maxCount > 0 {
			filled = r.Count * barWidth / maxCount
		}
		if filled == 0 && r.Count > 0 {
			filled = 1
		}
		bar := c.Sprint(strings.Repeat("█", filled)) + dim.Sprint(strings.Repeat("░", barWidth-filled))

		repo := r.Repo
		if len(repo) > 35 {
			repo = repo[:32] + "..."
		}
		fmt.Printf("  %-37s %s %d\n", repo, bar, r.Count)
	}

	if len(repos) > shown {
		dim.Printf("  ... and %d more\n", len(repos)-shown)
	}
	fmt.Println()
}

func MemberLeaderboard(label string, thisWeek, lastWeek []github.MemberStats, c *color.Color) {
	if len(thisWeek) == 0 {
		return
	}

	c.Println(label)
	fmt.Println()

	lastWeekMap := map[string]github.MemberStats{}
	for _, m := range lastWeek {
		lastWeekMap[m.Username] = m
	}

	maxTotal := 0
	for _, m := range thisWeek {
		if m.Total > maxTotal {
			maxTotal = m.Total
		}
	}

	barWidth := 20
	for _, m := range thisWeek {
		filled := 0
		if maxTotal > 0 {
			filled = m.Total * barWidth / maxTotal
		}
		if filled == 0 && m.Total > 0 {
			filled = 1
		}

		bar := cyan.Sprint(strings.Repeat("█", filled)) + dim.Sprint(strings.Repeat("░", barWidth-filled))

		counts := fmt.Sprintf("%d commits, %d PRs", m.Commits, m.PRs)

		prev := lastWeekMap[m.Username]
		delta := m.Total - prev.Total
		var deltaStr string
		switch {
		case delta > 0:
			deltaStr = greenBold.Sprintf(" (+%d)", delta)
		case delta < 0:
			deltaStr = red.Sprintf(" (%d)", delta)
		default:
			deltaStr = dim.Sprint(" (=)")
		}

		fmt.Printf("  %-18s %s %s%s\n", bold.Sprint(m.Username), bar, counts, deltaStr)
	}
	fmt.Println()
}

func FormatTokens(n int64) string {
	switch {
	case n >= 1_000_000_000:
		return fmt.Sprintf("%.1fB", float64(n)/1_000_000_000)
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	default:
		return fmt.Sprintf("%d", n)
	}
}

func FormatCost(f float64) string {
	if f >= 100 {
		return fmt.Sprintf("$%.0f", f)
	}
	return fmt.Sprintf("$%.2f", f)
}

func GrowthLine(label string, current, previous float64, unit string, c *color.Color) {
	delta := current - previous
	var pct float64
	if previous > 0 {
		pct = (delta / previous) * 100
	}

	c.Print(label)
	fmt.Printf("  ")
	bold.Printf("%s", unit)

	if previous == 0 {
		dim.Printf("  (no previous data)")
	} else {
		var deltaColor *color.Color
		var sign string
		switch {
		case delta > 0:
			deltaColor = greenBold
			sign = "+"
		case delta < 0:
			deltaColor = red
			sign = ""
		default:
			deltaColor = dim
			sign = ""
		}
		deltaColor.Printf("  (%s%.1f%%)", sign, pct)
	}
	fmt.Println()
}

func FloatBars(values []float64, labels []string, formatter func(float64) string, c *color.Color) {
	maxVal := 0.0
	for _, v := range values {
		if v > maxVal {
			maxVal = v
		}
	}

	if maxVal == 0 {
		fmt.Println("  (none)")
		return
	}

	chartHeight := 8

	colWidth := 7
	for _, l := range labels {
		if len(l)+1 > colWidth {
			colWidth = len(l) + 1
		}
	}

	for row := chartHeight; row >= 1; row-- {
		threshold := float64(row) / float64(chartHeight) * maxVal
		fmt.Print("  ")
		for _, v := range values {
			if v >= threshold && v > 0 {
				bar := c.Sprint("██")
				pad := colWidth - 2
				fmt.Printf("%s%s", bar, strings.Repeat(" ", pad))
			} else {
				fmt.Print(strings.Repeat(" ", colWidth))
			}
		}
		fmt.Println()
	}

	fmt.Print("  ")
	for _, l := range labels {
		fmt.Printf("%-*s", colWidth, l)
	}
	fmt.Println()

	fmt.Print("  ")
	for _, v := range values {
		s := formatter(v)
		dim.Printf("%-*s", colWidth, s)
	}
	fmt.Println()
}

func ContributionsJSON(thisWeek, lastWeek *github.Contributions) error {
	type jsonOutput struct {
		ThisWeek struct {
			Commits int `json:"commits"`
			PRs     int `json:"prs"`
		} `json:"this_week"`
		LastWeek struct {
			Commits int `json:"commits"`
			PRs     int `json:"prs"`
		} `json:"last_week"`
		CommitRepos []github.RepoContribution `json:"commit_repos"`
		PRRepos     []github.RepoContribution `json:"pr_repos"`
	}

	out := jsonOutput{}
	out.ThisWeek.Commits = thisWeek.TotalCommits
	out.ThisWeek.PRs = thisWeek.TotalPRs
	out.LastWeek.Commits = lastWeek.TotalCommits
	out.LastWeek.PRs = lastWeek.TotalPRs
	out.CommitRepos = thisWeek.CommitRepos
	out.PRRepos = thisWeek.PRRepos

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

