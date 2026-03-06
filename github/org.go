package github

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

type Org struct {
	Login       string `json:"login"`
	Description string `json:"description"`
}

type MemberStats struct {
	Username    string             `json:"username"`
	Commits     int                `json:"commits"`
	PRs         int                `json:"prs"`
	Total       int                `json:"total"`
	CommitRepos []RepoContribution `json:"commit_repos,omitempty"`
	PRRepos     []RepoContribution `json:"pr_repos,omitempty"`
}

type TeamStats struct {
	Org          string             `json:"org"`
	Members      []MemberStats      `json:"members"`
	TotalCommits int                `json:"total_commits"`
	TotalPRs     int                `json:"total_prs"`
	OrgRepos     []RepoContribution `json:"org_repos"`
	Days         []DayContribution  `json:"days,omitempty"`
	PRDays       []DayContribution  `json:"pr_days,omitempty"`
}

func (c *Client) restGet(path string) ([]byte, error) {
	url := "https://api.github.com" + path
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Accept", "application/vnd.github+json")

	httpClient := &http.Client{Timeout: 30 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API %s returned %d: %s", path, resp.StatusCode, string(body))
	}
	return body, nil
}

func (c *Client) ListOrgs() ([]Org, error) {
	body, err := c.restGet("/user/orgs?per_page=100")
	if err != nil {
		return nil, err
	}
	var orgs []Org
	if err := json.Unmarshal(body, &orgs); err != nil {
		return nil, err
	}
	return orgs, nil
}

func (c *Client) ListOrgMembers(org string) ([]string, error) {
	body, err := c.restGet(fmt.Sprintf("/orgs/%s/members?per_page=100", org))
	if err != nil {
		return nil, err
	}
	var members []struct {
		Login string `json:"login"`
	}
	if err := json.Unmarshal(body, &members); err != nil {
		return nil, err
	}
	var logins []string
	for _, m := range members {
		logins = append(logins, m.Login)
	}
	return logins, nil
}

func (c *Client) FetchTeamStats(org string, members []string, from, to time.Time) (*TeamStats, error) {
	type result struct {
		username string
		contribs *Contributions
		err      error
	}

	results := make([]result, len(members))
	var wg sync.WaitGroup

	for i, member := range members {
		wg.Add(1)
		go func(idx int, user string) {
			defer wg.Done()
			memberClient := &Client{Token: c.Token, Username: user}
			contribs, err := memberClient.FetchContributions(from, to)
			results[idx] = result{username: user, contribs: contribs, err: err}
		}(i, member)
	}
	wg.Wait()

	orgPrefix := strings.ToLower(org) + "/"
	stats := &TeamStats{Org: org}
	repoTotals := map[string]int{}
	dayTotals := map[string]int{}
	prDayTotals := map[string]int{}

	for _, r := range results {
		if r.err != nil {
			stats.Members = append(stats.Members, MemberStats{Username: r.username})
			continue
		}

		ms := MemberStats{Username: r.username}
		for _, repo := range r.contribs.CommitRepos {
			if strings.HasPrefix(strings.ToLower(repo.Repo), orgPrefix) {
				ms.Commits += repo.Count
				ms.CommitRepos = append(ms.CommitRepos, repo)
				repoTotals[repo.Repo] += repo.Count
			}
		}
		for _, repo := range r.contribs.PRRepos {
			if strings.HasPrefix(strings.ToLower(repo.Repo), orgPrefix) {
				ms.PRs += repo.Count
				ms.PRRepos = append(ms.PRRepos, repo)
				repoTotals[repo.Repo] += repo.Count
			}
		}
		ms.Total = ms.Commits + ms.PRs
		stats.TotalCommits += ms.Commits
		stats.TotalPRs += ms.PRs
		stats.Members = append(stats.Members, ms)

		for _, d := range r.contribs.Days {
			dayTotals[d.Date.Format("2006-01-02")] += d.Count
		}
		for _, d := range r.contribs.PRDays {
			prDayTotals[d.Date.Format("2006-01-02")] += d.Count
		}
	}

	sort.Slice(stats.Members, func(i, j int) bool {
		return stats.Members[i].Total > stats.Members[j].Total
	})

	for repo, count := range repoTotals {
		stats.OrgRepos = append(stats.OrgRepos, RepoContribution{Repo: repo, Count: count})
	}
	sort.Slice(stats.OrgRepos, func(i, j int) bool {
		return stats.OrgRepos[i].Count > stats.OrgRepos[j].Count
	})

	for dateStr, count := range dayTotals {
		t, _ := time.Parse("2006-01-02", dateStr)
		stats.Days = append(stats.Days, DayContribution{Date: t, Count: count})
	}
	sort.Slice(stats.Days, func(i, j int) bool {
		return stats.Days[i].Date.Before(stats.Days[j].Date)
	})

	for dateStr, count := range prDayTotals {
		t, _ := time.Parse("2006-01-02", dateStr)
		stats.PRDays = append(stats.PRDays, DayContribution{Date: t, Count: count})
	}
	sort.Slice(stats.PRDays, func(i, j int) bool {
		return stats.PRDays[i].Date.Before(stats.PRDays[j].Date)
	})

	return stats, nil
}
