package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	gh "github.com/nickhudkins/gh-stats/github"
	"github.com/nickhudkins/gh-stats/render"
)

type teamActivityEnvelope struct {
	Window teamActivityWindow `json:"window"`
	Days   []teamActivityDay  `json:"days"`
}

type teamActivityWindow struct {
	Since string `json:"since"`
	Until string `json:"until"`
}

type teamActivityDay struct {
	Date    string                 `json:"date"`
	Totals  map[string]int         `json:"totals"`
	PRs     []gh.PullRequestDetail `json:"prs,omitempty"`
	Commits []gh.CommitDetail      `json:"commits,omitempty"`
}

func runTeamPullRequestDetails(org string, members []string) error {
	since, until, err := activityWindow(days, time.Now())
	if err != nil {
		return err
	}
	stop := startSpinner(fmt.Sprintf("Fetching PRs for %d members from GitHub...", len(members)))
	prs, err := client.FetchTeamPullRequests(org, members, since, until)
	stop()
	if err != nil {
		return err
	}

	envelope := buildPREnvelope(prs, since, until)
	if jsonOutput {
		return writeActivityJSON(envelope)
	}
	renderPullRequestDetails(org, envelope)
	return nil
}

func runTeamCommitDetails(org string, members []string) error {
	since, until, err := activityWindow(days, time.Now())
	if err != nil {
		return err
	}
	stop := startSpinner(fmt.Sprintf("Fetching commits for %d members from GitHub...", len(members)))
	commits, err := client.FetchTeamCommits(org, members, since, until)
	stop()
	if err != nil {
		return err
	}

	envelope := buildCommitEnvelope(commits, since, until)
	if jsonOutput {
		return writeActivityJSON(envelope)
	}
	renderCommitDetails(org, envelope)
	return nil
}

func activityWindow(dayCount int, now time.Time) (time.Time, time.Time, error) {
	if dayCount < 1 {
		return time.Time{}, time.Time{}, fmt.Errorf("--days must be at least 1")
	}
	since := startOfDay(now).AddDate(0, 0, -(dayCount - 1))
	return since, now, nil
}

func buildPREnvelope(prs []gh.PullRequestDetail, since, until time.Time) teamActivityEnvelope {
	byDate := map[string][]gh.PullRequestDetail{}
	for _, pr := range prs {
		key := pr.MergedAt.Local().Format("2006-01-02")
		byDate[key] = append(byDate[key], pr)
	}

	days := make([]teamActivityDay, 0, len(byDate))
	for date, rows := range byDate {
		sort.Slice(rows, func(i, j int) bool {
			if !rows[i].MergedAt.Equal(rows[j].MergedAt) {
				return rows[i].MergedAt.After(rows[j].MergedAt)
			}
			return rows[i].Repo < rows[j].Repo
		})
		insertions, deletions := sumPRChanges(rows)
		days = append(days, teamActivityDay{
			Date: date,
			Totals: map[string]int{
				"prs":        len(rows),
				"insertions": insertions,
				"deletions":  deletions,
				"net":        insertions - deletions,
			},
			PRs: rows,
		})
	}
	sortActivityDays(days)
	return teamActivityEnvelope{Window: newActivityWindow(since, until), Days: days}
}

func buildCommitEnvelope(commits []gh.CommitDetail, since, until time.Time) teamActivityEnvelope {
	byDate := map[string][]gh.CommitDetail{}
	for _, commit := range commits {
		key := commit.Date.Local().Format("2006-01-02")
		byDate[key] = append(byDate[key], commit)
	}

	days := make([]teamActivityDay, 0, len(byDate))
	for date, rows := range byDate {
		sort.Slice(rows, func(i, j int) bool {
			if !rows[i].Date.Equal(rows[j].Date) {
				return rows[i].Date.After(rows[j].Date)
			}
			return rows[i].Repo < rows[j].Repo
		})
		insertions, deletions := sumCommitChanges(rows)
		days = append(days, teamActivityDay{
			Date: date,
			Totals: map[string]int{
				"commits":    len(rows),
				"insertions": insertions,
				"deletions":  deletions,
				"net":        insertions - deletions,
			},
			Commits: rows,
		})
	}
	sortActivityDays(days)
	return teamActivityEnvelope{Window: newActivityWindow(since, until), Days: days}
}

func newActivityWindow(since, until time.Time) teamActivityWindow {
	return teamActivityWindow{
		Since: since.Format(time.RFC3339),
		Until: until.Format(time.RFC3339),
	}
}

func writeActivityJSON(envelope teamActivityEnvelope) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(envelope)
}

func renderPullRequestDetails(org string, envelope teamActivityEnvelope) {
	render.Bold.Print("Team PRs")
	render.Dim.Printf("  ·  %s  ·  %s → %s\n", render.CyanBold.Sprint(org), envelope.Window.Since[:10], envelope.Window.Until[:10])
	fmt.Println()
	if len(envelope.Days) == 0 {
		render.Dim.Println("  No merged PRs in this window.")
		return
	}
	for _, day := range envelope.Days {
		fmt.Printf("%s  %d PRs  %+d / -%d  net %+d\n",
			day.Date, day.Totals["prs"], day.Totals["insertions"], day.Totals["deletions"], day.Totals["net"])
		for _, pr := range day.PRs {
			fmt.Printf("  %-32s %-14s #%d %-10s %+d -%d  %s\n",
				truncateRight(pr.Repo, 32), truncateRight(pr.Author, 14), pr.PRNumber,
				truncateRight(pr.Branch, 10), pr.Insertions, pr.Deletions, pr.Subject)
		}
	}
}

func renderCommitDetails(org string, envelope teamActivityEnvelope) {
	render.Bold.Print("Team Commits")
	render.Dim.Printf("  ·  %s  ·  %s → %s\n", render.CyanBold.Sprint(org), envelope.Window.Since[:10], envelope.Window.Until[:10])
	fmt.Println()
	if len(envelope.Days) == 0 {
		render.Dim.Println("  No commits in this window.")
		return
	}
	for _, day := range envelope.Days {
		fmt.Printf("%s  %d commits  %+d / -%d  net %+d\n",
			day.Date, day.Totals["commits"], day.Totals["insertions"], day.Totals["deletions"], day.Totals["net"])
		for _, commit := range day.Commits {
			fmt.Printf("  %-32s %-14s %s %-10s %+d -%d  %s\n",
				truncateRight(commit.Repo, 32), truncateRight(commit.Author, 14),
				shortSHA(commit.SHA), truncateRight(commit.Branch, 10),
				commit.Insertions, commit.Deletions, commit.Subject)
		}
	}
}

func sumPRChanges(rows []gh.PullRequestDetail) (int, int) {
	insertions, deletions := 0, 0
	for _, row := range rows {
		insertions += row.Insertions
		deletions += row.Deletions
	}
	return insertions, deletions
}

func sumCommitChanges(rows []gh.CommitDetail) (int, int) {
	insertions, deletions := 0, 0
	for _, row := range rows {
		insertions += row.Insertions
		deletions += row.Deletions
	}
	return insertions, deletions
}

func sortActivityDays(days []teamActivityDay) {
	sort.Slice(days, func(i, j int) bool {
		return days[i].Date > days[j].Date
	})
}

func shortSHA(sha string) string {
	if len(sha) <= 7 {
		return sha
	}
	return sha[:7]
}

func truncateRight(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return strings.TrimSpace(s[:max-3]) + "..."
}
