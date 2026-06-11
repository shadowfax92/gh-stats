package cmd

import (
	"encoding/json"
	"testing"
	"time"

	gh "github.com/nickhudkins/gh-stats/github"
)

func TestActivityWindowIncludesTodayAndPreviousDays(t *testing.T) {
	now := time.Date(2026, 4, 24, 15, 30, 0, 0, time.UTC)
	since, until, err := activityWindow(7, now)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := since.Format(time.RFC3339), "2026-04-18T00:00:00Z"; got != want {
		t.Fatalf("since = %s, want %s", got, want)
	}
	if !until.Equal(now) {
		t.Fatalf("until = %s, want %s", until, now)
	}
}

func TestBuildPREnvelopeGroupsNewestFirst(t *testing.T) {
	since := time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC)
	until := time.Date(2026, 4, 24, 12, 0, 0, 0, time.UTC)
	envelope := buildPREnvelope([]gh.PullRequestDetail{
		{
			Repo:       "browseros-ai/app",
			Author:     "DaniAkash",
			MergedAt:   time.Date(2026, 4, 21, 10, 0, 0, 0, time.UTC),
			PRNumber:   101,
			Subject:    "feat: alpha",
			SHA:        "sha101",
			Branch:     "main",
			Insertions: 3,
			Deletions:  1,
			Net:        2,
		},
		{
			Repo:       "browseros-ai/app",
			Author:     "shadowfax92",
			MergedAt:   time.Date(2026, 4, 22, 10, 0, 0, 0, time.UTC),
			PRNumber:   102,
			Subject:    "fix: beta",
			SHA:        "sha102",
			Branch:     "dev",
			Insertions: 5,
			Deletions:  2,
			Net:        3,
		},
	}, since, until)

	if got, want := len(envelope.Days), 2; got != want {
		t.Fatalf("days = %d, want %d", got, want)
	}
	if got, want := envelope.Days[0].Date, "2026-04-22"; got != want {
		t.Fatalf("first day = %s, want %s", got, want)
	}
	if got, want := envelope.Days[0].Totals["prs"], 1; got != want {
		t.Fatalf("prs total = %d, want %d", got, want)
	}

	data, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(data) {
		t.Fatal("envelope did not marshal to valid JSON")
	}
}

func TestBuildCommitEnvelopeUsesEngprodFieldNames(t *testing.T) {
	since := time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC)
	until := time.Date(2026, 4, 24, 12, 0, 0, 0, time.UTC)
	envelope := buildCommitEnvelope([]gh.CommitDetail{
		{
			Repo:        "browseros-ai/app",
			SHA:         "abcdef123456",
			Author:      "DaniAkash",
			AuthorName:  "Dani",
			AuthorEmail: "dani@example.com",
			Date:        time.Date(2026, 4, 22, 10, 0, 0, 0, time.UTC),
			Subject:     "feat: alpha",
			Insertions:  7,
			Deletions:   3,
			Branch:      "main",
			IsMerge:     false,
			PRNumber:    nil,
		},
	}, since, until)

	data, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatal(err)
	}
	days := parsed["days"].([]any)
	commits := days[0].(map[string]any)["commits"].([]any)
	commit := commits[0].(map[string]any)
	for _, key := range []string{"authorName", "authorEmail", "isMerge", "prNumber"} {
		if _, ok := commit[key]; !ok {
			t.Fatalf("missing JSON key %q in %#v", key, commit)
		}
	}
}
