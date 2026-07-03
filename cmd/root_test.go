package cmd

import (
	"testing"
	"time"

	gh "github.com/nickhudkins/gh-stats/github"
	"github.com/nickhudkins/gh-stats/render"
)

var testZone = time.FixedZone("TEST", -7*3600)

func testDay(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, testZone)
}

func TestWindowBoundsInclusiveEndingToday(t *testing.T) {
	today := testDay(2026, time.July, 3)

	start, end := windowBounds(today, 30)

	if got := start.Format("2006-01-02 15:04:05"); got != "2026-06-04 00:00:00" {
		t.Fatalf("start = %s, want 2026-06-04 00:00:00", got)
	}
	if got := end.Format("2006-01-02 15:04:05"); got != "2026-07-03 23:59:59" {
		t.Fatalf("end = %s, want 2026-07-03 23:59:59", got)
	}
}

func TestWindowTotalUsesOnlyFilledDaysWindow(t *testing.T) {
	today := testDay(2026, time.July, 3)
	days := []gh.DayContribution{
		{Date: testDay(2026, time.June, 30), Count: 100},
		{Date: testDay(2026, time.July, 1), Count: 2},
		{Date: testDay(2026, time.July, 3), Count: 4},
	}

	filled := render.FillDays(days, today, 3)

	if got := sumContributionDays(filled); got != 6 {
		t.Fatalf("window total = %d, want 6", got)
	}
}

func TestWindowMemberRowsSortByWindowTotal(t *testing.T) {
	today := testDay(2026, time.July, 3)
	members := []gh.MemberStats{
		{Username: "alice"},
		{Username: "bob"},
		{Username: "casey"},
	}
	commits := map[string][]gh.DayContribution{
		"alice": {
			{Date: testDay(2026, time.July, 1), Count: 1},
			{Date: testDay(2026, time.July, 3), Count: 1},
		},
		"bob": {
			{Date: testDay(2026, time.June, 30), Count: 100},
			{Date: testDay(2026, time.July, 2), Count: 5},
		},
	}
	prs := map[string][]gh.DayContribution{
		"alice": {{Date: testDay(2026, time.July, 3), Count: 2}},
	}

	rows := windowMemberRows(members, commits, prs, today, 3)

	if len(rows) != 2 {
		t.Fatalf("rows len = %d, want 2", len(rows))
	}
	if rows[0].name != "bob" || rows[0].commits != 5 || rows[0].prs != 0 || rows[0].total != 5 {
		t.Fatalf("first row = %+v, want bob with 5 commits", rows[0])
	}
	if rows[1].name != "alice" || rows[1].commits != 2 || rows[1].prs != 2 || rows[1].total != 4 {
		t.Fatalf("second row = %+v, want alice with 2 commits and 2 PRs", rows[1])
	}
}
