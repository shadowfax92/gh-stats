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
	Bold        = color.New(color.Bold)
	Dim         = color.New(color.Faint)
	Green       = color.New(color.FgGreen)
	GreenBold   = color.New(color.FgGreen, color.Bold)
	Cyan        = color.New(color.FgCyan)
	CyanBold    = color.New(color.FgCyan, color.Bold)
	Yellow      = color.New(color.FgYellow)
	Red         = color.New(color.FgRed, color.Bold)
	Magenta     = color.New(color.FgMagenta)
	MagentaBold = color.New(color.FgMagenta, color.Bold)

	bold      = Bold
	dim       = Dim
	green     = Green
	greenBold = GreenBold
	cyan      = Cyan
	cyanBold  = CyanBold
	yellow    = Yellow
	red       = Red
	magenta   = Magenta
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

func CompactWeekRow(label string, thisWeek, lastWeek int, c *color.Color) {
	c.Printf("  %-15s ", label)
	bold.Printf("%6d  ", thisWeek)
	dim.Printf("vs %d last wk   ", lastWeek)
	pct := FormatPctInt(thisWeek, lastWeek)
	pctC := PctColorInt(thisWeek, lastWeek)
	pctC.Printf("%s\n", pct)
}

func FormatPctInt(current, previous int) string {
	if previous == 0 {
		if current == 0 {
			return "—"
		}
		return "new"
	}
	pct := (float64(current-previous) / float64(previous)) * 100
	sign := "+"
	if pct < 0 {
		sign = ""
	}
	return fmt.Sprintf("%s%.0f%%", sign, pct)
}

func PctColorInt(current, previous int) *color.Color {
	if previous == 0 {
		return dim
	}
	switch {
	case current > previous:
		return greenBold
	case current < previous:
		return red
	default:
		return dim
	}
}

func Sparkline(values []int) string {
	if len(values) == 0 {
		return ""
	}
	blocks := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}
	maxVal := 0
	for _, v := range values {
		if v > maxVal {
			maxVal = v
		}
	}
	if maxVal == 0 {
		return strings.Repeat(" ", len(values))
	}
	var b strings.Builder
	for _, v := range values {
		if v == 0 {
			b.WriteRune(' ')
			continue
		}
		idx := int((float64(v) / float64(maxVal)) * float64(len(blocks)))
		if idx >= len(blocks) {
			idx = len(blocks) - 1
		}
		b.WriteRune(blocks[idx])
	}
	return b.String()
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
	for _, v := range values {
		if w := len(fmt.Sprintf("%d", v)) + 1; w > colWidth {
			colWidth = w
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

