package github

import (
	"encoding/json"
	"testing"
	"time"
)

// build decodes a raw GraphQL response body and runs it through buildContributions
// for the given window, mirroring what FetchContributions does after the HTTP call.
func build(t *testing.T, body string, from, to time.Time) *Contributions {
	t.Helper()
	var resp graphqlResponse
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}
	return buildContributions(&resp, from, to)
}

// pdt is a fixed UTC-7 zone so tests don't depend on the machine's TZ.
var pdt = time.FixedZone("PDT", -7*3600)

func dayCount(days []DayContribution, date string) (int, bool) {
	for _, d := range days {
		if d.Date.Format("2006-01-02") == date {
			return d.Count, true
		}
	}
	return 0, false
}

// A PR whose occurredAt is just past UTC midnight belongs to the *previous* local
// day for a UTC-7 user. This is the exact bug: 2026-06-09T00:08:57Z is June 8 PDT.
func TestPRBucketedOnLocalDay(t *testing.T) {
	body := `{"data":{"user":{"contributionsCollection":{
		"totalCommitContributions":0,"totalPullRequestContributions":3,
		"commitContributionsByRepository":[],
		"pullRequestContributionsByRepository":[],
		"pullRequestContributions":{"nodes":[
			{"occurredAt":"2026-06-09T00:08:57Z"},
			{"occurredAt":"2026-06-08T23:58:38Z"},
			{"occurredAt":"2026-06-08T18:23:44Z"}
		]}
	}}}}`
	from := time.Date(2026, 6, 8, 0, 0, 0, 0, pdt)
	to := time.Date(2026, 6, 8, 23, 59, 59, 0, pdt)

	c := build(t, body, from, to)

	got, ok := dayCount(c.PRDays, "2026-06-08")
	if !ok || got != 3 {
		t.Fatalf("June 8 PRDays = %d (present=%v), want 3", got, ok)
	}
	if _, ok := dayCount(c.PRDays, "2026-06-09"); ok {
		t.Fatalf("June 9 should not appear in a June 8 window")
	}
}

// The window's first day (today / the week's Monday) must not be dropped by a
// UTC-vs-local-offset mismatch in the range filter. Regression for "Mon 01 → 0".
func TestEdgeDayNotDropped(t *testing.T) {
	body := `{"data":{"user":{"contributionsCollection":{
		"totalCommitContributions":5,"totalPullRequestContributions":0,
		"commitContributionsByRepository":[
			{"repository":{"nameWithOwner":"o/r"},"contributions":{"totalCount":5,"nodes":[
				{"occurredAt":"2026-06-07T07:00:00Z","commitCount":5}
			]}}
		],
		"pullRequestContributionsByRepository":[],
		"pullRequestContributions":{"nodes":[]}
	}}}}`
	// Window starts on June 7 (the edge); from carries the -07:00 offset.
	from := time.Date(2026, 6, 7, 0, 0, 0, 0, pdt)
	to := time.Date(2026, 6, 8, 23, 59, 59, 0, pdt)

	c := build(t, body, from, to)

	got, ok := dayCount(c.Days, "2026-06-07")
	if !ok || got != 5 {
		t.Fatalf("June 7 (edge) Days = %d (present=%v), want 5", got, ok)
	}
}

// "Commits" must be commit-only (sum of per-repo commitCount), not the
// all-contributions calendar total.
func TestCommitsAreCommitOnly(t *testing.T) {
	body := `{"data":{"user":{"contributionsCollection":{
		"totalCommitContributions":27,"totalPullRequestContributions":2,
		"commitContributionsByRepository":[
			{"repository":{"nameWithOwner":"browseros-ai/BrowserOS"},"contributions":{"totalCount":14,"nodes":[{"occurredAt":"2026-06-08T07:00:00Z","commitCount":14}]}},
			{"repository":{"nameWithOwner":"shadowfax92/grove"},"contributions":{"totalCount":7,"nodes":[{"occurredAt":"2026-06-08T07:00:00Z","commitCount":7}]}},
			{"repository":{"nameWithOwner":"shadowfax92/wrapux"},"contributions":{"totalCount":4,"nodes":[{"occurredAt":"2026-06-08T07:00:00Z","commitCount":4}]}},
			{"repository":{"nameWithOwner":"shadowfax92/tmx"},"contributions":{"totalCount":1,"nodes":[{"occurredAt":"2026-06-08T07:00:00Z","commitCount":1}]}},
			{"repository":{"nameWithOwner":"shadowfax92/skl"},"contributions":{"totalCount":1,"nodes":[{"occurredAt":"2026-06-08T07:00:00Z","commitCount":1}]}}
		],
		"pullRequestContributionsByRepository":[
			{"repository":{"nameWithOwner":"shadowfax92/grove"},"contributions":{"totalCount":2}}
		],
		"pullRequestContributions":{"nodes":[
			{"occurredAt":"2026-06-08T20:00:00Z"},
			{"occurredAt":"2026-06-08T21:00:00Z"}
		]}
	}}}}`
	from := time.Date(2026, 6, 8, 0, 0, 0, 0, pdt)
	to := time.Date(2026, 6, 8, 23, 59, 59, 0, pdt)

	c := build(t, body, from, to)

	got, ok := dayCount(c.Days, "2026-06-08")
	if !ok || got != 27 {
		t.Fatalf("June 8 Days (commits) = %d (present=%v), want 27 (commit sum, not 29 w/ PRs)", got, ok)
	}
	// CommitRepos still from per-repo totalCount, sorted desc.
	if len(c.CommitRepos) != 5 || c.CommitRepos[0].Repo != "browseros-ai/BrowserOS" || c.CommitRepos[0].Count != 14 {
		t.Fatalf("CommitRepos not populated/sorted: %+v", c.CommitRepos)
	}
	if c.TotalCommits != 27 || c.TotalPRs != 2 {
		t.Fatalf("period totals wrong: commits=%d prs=%d", c.TotalCommits, c.TotalPRs)
	}
}

// A commit node at midnight-profile-tz (07:00Z for PDT) buckets to the local day.
func TestCommitDayBucketedLocally(t *testing.T) {
	body := `{"data":{"user":{"contributionsCollection":{
		"totalCommitContributions":3,"totalPullRequestContributions":0,
		"commitContributionsByRepository":[
			{"repository":{"nameWithOwner":"o/r"},"contributions":{"totalCount":3,"nodes":[
				{"occurredAt":"2026-06-08T07:00:00Z","commitCount":3}
			]}}
		],
		"pullRequestContributionsByRepository":[],
		"pullRequestContributions":{"nodes":[]}
	}}}}`
	from := time.Date(2026, 6, 8, 0, 0, 0, 0, pdt)
	to := time.Date(2026, 6, 8, 23, 59, 59, 0, pdt)

	c := build(t, body, from, to)

	if got, ok := dayCount(c.Days, "2026-06-08"); !ok || got != 3 {
		t.Fatalf("June 8 commits = %d (present=%v), want 3", got, ok)
	}
	// Stored date must round-trip to the same key the cmd layer looks up.
	if len(c.Days) != 1 {
		t.Fatalf("want exactly one commit day, got %d", len(c.Days))
	}
}
