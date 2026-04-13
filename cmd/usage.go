package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/fatih/color"
	"github.com/nickhudkins/gh-stats/ccusage"
	"github.com/nickhudkins/gh-stats/render"
	"github.com/spf13/cobra"
)

var usageCmd = &cobra.Command{
	Use:   "usage",
	Short: "AI tool token usage stats (Claude Code + Codex)",
	Long:  "View daily token consumption and cost for Claude Code and OpenAI Codex.\nRequires npx (Node.js) and the ccusage packages.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if !jsonOutput {
			color.New(color.Faint).Println("Fetching usage data (this may take a few seconds)...")
		}

		data := ccusage.FetchAll()

		if data.Claude == nil && data.Codex == nil {
			for _, e := range data.Errors {
				fmt.Fprintf(os.Stderr, "  error: %s\n", e)
			}
			return fmt.Errorf("could not fetch usage data from any tool")
		}

		if jsonOutput {
			return usageJSON(data)
		}

		for _, e := range data.Errors {
			color.New(color.FgYellow).Fprintf(os.Stderr, "  warning: %s\n", e)
		}

		renderUsage(data)
		return nil
	},
}

func renderUsage(data *ccusage.UsageData) {
	bold := color.New(color.Bold)
	dim := color.New(color.Faint)
	cyanBold := color.New(color.FgCyan, color.Bold)
	greenBold := color.New(color.FgGreen, color.Bold)
	magentaBold := color.New(color.FgMagenta, color.Bold)

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	bold.Println("AI Usage Stats")
	dim.Printf("  %s\n", today.Format("Jan 2, 2006"))
	fmt.Println()

	// Per-tool summaries
	if data.Claude != nil && len(data.Claude.Daily) > 0 {
		renderToolSummary("Claude Code", data.Claude, cyanBold, today)
	}
	if data.Codex != nil && len(data.Codex.Daily) > 0 {
		renderToolSummary("Codex", data.Codex, greenBold, today)
	}

	// Combined daily token chart
	combined := combineDailyUsage(data)
	if len(combined) > 0 {
		last14 := filterLast14Days(combined, today)
		if len(last14) > 0 {
			tokenValues := make([]float64, len(last14))
			tokenLabels := make([]string, len(last14))
			costValues := make([]float64, len(last14))
			for i, d := range last14 {
				tokenValues[i] = float64(d.TotalTokens)
				costValues[i] = d.Cost
				if d.Date.Equal(today) {
					tokenLabels[i] = "Today"
				} else {
					tokenLabels[i] = d.Date.Format("Mon 02")
				}
			}

			bold.Println("Daily Tokens (last 14 days)")
			render.FloatBars(tokenValues, tokenLabels, func(v float64) string {
				return render.FormatTokens(int64(v))
			}, color.New(color.FgCyan))
			fmt.Println()

			bold.Println("Daily Cost (last 14 days)")
			render.FloatBars(costValues, tokenLabels, render.FormatCost, color.New(color.FgGreen))
			fmt.Println()
		}

		// Growth metrics
		bold.Println("Growth")
		todayEntry, yesterdayEntry := findDayEntries(combined, today)
		thisWeekTotal, lastWeekTotal := weekTotals(combined, today)

		render.GrowthLine(
			"  Day-over-Day Tokens ",
			float64(todayEntry.TotalTokens),
			float64(yesterdayEntry.TotalTokens),
			fmt.Sprintf("%s → %s", render.FormatTokens(yesterdayEntry.TotalTokens), render.FormatTokens(todayEntry.TotalTokens)),
			cyanBold,
		)
		render.GrowthLine(
			"  Day-over-Day Cost   ",
			todayEntry.Cost,
			yesterdayEntry.Cost,
			fmt.Sprintf("%s → %s", render.FormatCost(yesterdayEntry.Cost), render.FormatCost(todayEntry.Cost)),
			greenBold,
		)
		render.GrowthLine(
			"  Week-over-Week Tokens",
			float64(thisWeekTotal.TotalTokens),
			float64(lastWeekTotal.TotalTokens),
			fmt.Sprintf("%s → %s", render.FormatTokens(lastWeekTotal.TotalTokens), render.FormatTokens(thisWeekTotal.TotalTokens)),
			cyanBold,
		)
		render.GrowthLine(
			"  Week-over-Week Cost  ",
			thisWeekTotal.Cost,
			lastWeekTotal.Cost,
			fmt.Sprintf("%s → %s", render.FormatCost(lastWeekTotal.Cost), render.FormatCost(thisWeekTotal.Cost)),
			magentaBold,
		)
		fmt.Println()
	}
}

func renderToolSummary(name string, usage *ccusage.ToolUsage, c *color.Color, today time.Time) {
	bold := color.New(color.Bold)
	dim := color.New(color.Faint)

	c.Printf("  %s", name)
	fmt.Println()

	todayEntry := findEntryForDate(usage.Daily, today)

	bold.Printf("    Today: ")
	if todayEntry.TotalTokens > 0 {
		fmt.Printf("%s tokens  ", render.FormatTokens(todayEntry.TotalTokens))
		dim.Printf("(in: %s  out: %s  cache: %s)  ", render.FormatTokens(todayEntry.InputTokens), render.FormatTokens(todayEntry.OutputTokens), render.FormatTokens(todayEntry.CacheTokens))
		color.New(color.FgGreen).Printf("%s", render.FormatCost(todayEntry.Cost))
	} else {
		dim.Print("no usage yet")
	}
	fmt.Println()

	bold.Printf("    Total: ")
	fmt.Printf("%s tokens  ", render.FormatTokens(usage.Total.TotalTokens))
	dim.Printf("(in: %s  out: %s  cache: %s)  ", render.FormatTokens(usage.Total.InputTokens), render.FormatTokens(usage.Total.OutputTokens), render.FormatTokens(usage.Total.CacheTokens))
	color.New(color.FgGreen).Printf("%s", render.FormatCost(usage.Total.Cost))
	fmt.Printf("  ")
	dim.Printf("(%d days)", len(usage.Daily))
	fmt.Println()
	fmt.Println()
}

func findEntryForDate(daily []ccusage.DailyEntry, date time.Time) ccusage.DailyEntry {
	dateStr := date.Format("2006-01-02")
	for _, d := range daily {
		if d.Date.Format("2006-01-02") == dateStr {
			return d
		}
	}
	return ccusage.DailyEntry{}
}

func combineDailyUsage(data *ccusage.UsageData) []ccusage.DailyEntry {
	byDate := map[string]*ccusage.DailyEntry{}

	addEntries := func(daily []ccusage.DailyEntry) {
		for _, d := range daily {
			key := d.Date.Format("2006-01-02")
			if existing, ok := byDate[key]; ok {
				existing.InputTokens += d.InputTokens
				existing.OutputTokens += d.OutputTokens
				existing.CacheTokens += d.CacheTokens
				existing.TotalTokens += d.TotalTokens
				existing.Cost += d.Cost
			} else {
				entry := d
				byDate[key] = &entry
			}
		}
	}

	if data.Claude != nil {
		addEntries(data.Claude.Daily)
	}
	if data.Codex != nil {
		addEntries(data.Codex.Daily)
	}

	var result []ccusage.DailyEntry
	for _, e := range byDate {
		result = append(result, *e)
	}

	for i := 1; i < len(result); i++ {
		for j := i; j > 0 && result[j].Date.Before(result[j-1].Date); j-- {
			result[j], result[j-1] = result[j-1], result[j]
		}
	}

	return result
}

func filterLast14Days(entries []ccusage.DailyEntry, today time.Time) []ccusage.DailyEntry {
	cutoff := today.AddDate(0, 0, -13)
	var filtered []ccusage.DailyEntry
	for _, e := range entries {
		if !e.Date.Before(cutoff) && !e.Date.After(today) {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

func findDayEntries(entries []ccusage.DailyEntry, today time.Time) (todayEntry, yesterdayEntry ccusage.DailyEntry) {
	yesterday := today.AddDate(0, 0, -1)
	for _, e := range entries {
		d := e.Date.Format("2006-01-02")
		if d == today.Format("2006-01-02") {
			todayEntry = e
		}
		if d == yesterday.Format("2006-01-02") {
			yesterdayEntry = e
		}
	}
	return
}

func weekTotals(entries []ccusage.DailyEntry, today time.Time) (thisWeek, lastWeek ccusage.DailyEntry) {
	daysFromMonday := (int(today.Weekday()) + 6) % 7
	thisMonday := today.AddDate(0, 0, -daysFromMonday)
	lastMonday := thisMonday.AddDate(0, 0, -7)

	for _, e := range entries {
		if !e.Date.Before(thisMonday) && !e.Date.After(today) {
			thisWeek.TotalTokens += e.TotalTokens
			thisWeek.InputTokens += e.InputTokens
			thisWeek.OutputTokens += e.OutputTokens
			thisWeek.Cost += e.Cost
		} else if !e.Date.Before(lastMonday) && e.Date.Before(thisMonday) {
			lastWeek.TotalTokens += e.TotalTokens
			lastWeek.InputTokens += e.InputTokens
			lastWeek.OutputTokens += e.OutputTokens
			lastWeek.Cost += e.Cost
		}
	}
	return
}

func usageJSON(data *ccusage.UsageData) error {
	type dailyJSON struct {
		Date         string  `json:"date"`
		InputTokens  int64   `json:"input_tokens"`
		OutputTokens int64   `json:"output_tokens"`
		CacheTokens  int64   `json:"cache_tokens"`
		TotalTokens  int64   `json:"total_tokens"`
		Cost         float64 `json:"cost"`
	}

	type toolJSON struct {
		Tool  string      `json:"tool"`
		Daily []dailyJSON `json:"daily"`
		Total dailyJSON   `json:"total"`
	}

	type output struct {
		Tools    []toolJSON  `json:"tools"`
		Combined []dailyJSON `json:"combined_daily"`
		Errors   []string    `json:"errors,omitempty"`
	}

	toDailyJSON := func(entries []ccusage.DailyEntry) []dailyJSON {
		out := make([]dailyJSON, len(entries))
		for i, e := range entries {
			out[i] = dailyJSON{
				Date:         e.Date.Format("2006-01-02"),
				InputTokens:  e.InputTokens,
				OutputTokens: e.OutputTokens,
				CacheTokens:  e.CacheTokens,
				TotalTokens:  e.TotalTokens,
				Cost:         e.Cost,
			}
		}
		return out
	}

	toTotalJSON := func(t ccusage.DailyEntry) dailyJSON {
		return dailyJSON{
			InputTokens:  t.InputTokens,
			OutputTokens: t.OutputTokens,
			CacheTokens:  t.CacheTokens,
			TotalTokens:  t.TotalTokens,
			Cost:         t.Cost,
		}
	}

	out := output{Errors: data.Errors}

	if data.Claude != nil {
		out.Tools = append(out.Tools, toolJSON{
			Tool:  data.Claude.Tool,
			Daily: toDailyJSON(data.Claude.Daily),
			Total: toTotalJSON(data.Claude.Total),
		})
	}
	if data.Codex != nil {
		out.Tools = append(out.Tools, toolJSON{
			Tool:  data.Codex.Tool,
			Daily: toDailyJSON(data.Codex.Daily),
			Total: toTotalJSON(data.Codex.Total),
		})
	}

	combined := combineDailyUsage(data)
	out.Combined = toDailyJSON(combined)

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func init() {
	rootCmd.AddCommand(usageCmd)
}
