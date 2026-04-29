package render

import (
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/nickhudkins/gh-stats/github"
)

// FillDays returns exactly N entries ending at today (oldest → newest), filling
// missing days with zero counts.
func FillDays(days []github.DayContribution, today time.Time, n int) []github.DayContribution {
	byDate := map[string]int{}
	for _, d := range days {
		byDate[d.Date.Format("2006-01-02")] = d.Count
	}
	out := make([]github.DayContribution, n)
	for i := 0; i < n; i++ {
		d := today.AddDate(0, 0, -(n - 1 - i))
		key := d.Format("2006-01-02")
		out[i] = github.DayContribution{Date: d, Count: byDate[key]}
	}
	return out
}

// CountOn returns the count for a specific date (0 if missing).
func CountOn(days []github.DayContribution, date time.Time) int {
	key := date.Format("2006-01-02")
	for _, d := range days {
		if d.Date.Format("2006-01-02") == key {
			return d.Count
		}
	}
	return 0
}

// SumDays returns the sum of counts in [from, to] inclusive (date comparison).
func SumDays(days []github.DayContribution, from, to time.Time) int {
	fromKey := from.Format("2006-01-02")
	toKey := to.Format("2006-01-02")
	sum := 0
	for _, d := range days {
		key := d.Date.Format("2006-01-02")
		if key >= fromKey && key <= toKey {
			sum += d.Count
		}
	}
	return sum
}

// WeekBounds returns Monday and Sunday for the calendar week containing `day`.
func WeekBounds(day time.Time) (time.Time, time.Time) {
	t := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location())
	daysFromMonday := (int(t.Weekday()) + 6) % 7
	mon := t.AddDate(0, 0, -daysFromMonday)
	sun := mon.AddDate(0, 0, 6)
	return mon, sun
}

// SparklineRow renders one row of label + sparkline + total. Used for per-member tables.
func SparklineRow(label string, values []int, total int, c *color.Color, labelWidth int) {
	fmt.Printf("  %-*s ", labelWidth, label)
	c.Print(Sparkline(values))
	Dim.Printf("  %d\n", total)
}

// MaxLabelWidth returns the max of the printable label widths, capped at maxCap.
func MaxLabelWidth(labels []string, maxCap int) int {
	w := 0
	for _, l := range labels {
		if len(l) > w {
			w = len(l)
		}
	}
	if w > maxCap {
		w = maxCap
	}
	return w
}

// TruncateLeft truncates s from the right to width w, adding ellipsis if cut.
func TruncateLeft(s string, w int) string {
	if len(s) <= w {
		return s
	}
	if w <= 1 {
		return strings.Repeat(".", w)
	}
	return s[:w-1] + "…"
}
