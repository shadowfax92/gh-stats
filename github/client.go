package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"
)

type Client struct {
	Token    string
	Username string
}

type DayContribution struct {
	Date  time.Time
	Count int
}

type RepoContribution struct {
	Repo  string `json:"repo"`
	Count int    `json:"count"`
}

type Contributions struct {
	TotalCommits int
	TotalPRs     int
	Days         []DayContribution
	PRDays       []DayContribution
	CommitRepos  []RepoContribution
	PRRepos      []RepoContribution
}

const graphqlEndpoint = "https://api.github.com/graphql"

// Per-day commit counts come from commitContributionsByRepository's commitCount
// nodes (true commits), NOT contributionCalendar — the calendar's contributionCount
// folds in PRs, issues and reviews, which would double-count the PR row.
const contributionsQuery = `query($login: String!, $from: DateTime!, $to: DateTime!) {
  user(login: $login) {
    contributionsCollection(from: $from, to: $to) {
      totalCommitContributions
      totalPullRequestContributions
      commitContributionsByRepository(maxRepositories: 25) {
        repository { nameWithOwner }
        contributions(first: 100) {
          totalCount
          nodes { occurredAt commitCount }
        }
      }
      pullRequestContributionsByRepository(maxRepositories: 25) {
        repository { nameWithOwner }
        contributions { totalCount }
      }
      pullRequestContributions(first: 100) {
        nodes { occurredAt }
      }
    }
  }
}`

type graphqlRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables"`
}

type graphqlResponse struct {
	Data struct {
		User struct {
			ContributionsCollection struct {
				TotalCommitContributions        int `json:"totalCommitContributions"`
				TotalPullRequestContributions   int `json:"totalPullRequestContributions"`
				CommitContributionsByRepository []struct {
					Repository struct {
						NameWithOwner string `json:"nameWithOwner"`
					} `json:"repository"`
					Contributions struct {
						TotalCount int `json:"totalCount"`
						Nodes      []struct {
							OccurredAt  string `json:"occurredAt"`
							CommitCount int    `json:"commitCount"`
						} `json:"nodes"`
					} `json:"contributions"`
				} `json:"commitContributionsByRepository"`
				PullRequestContributionsByRepository []struct {
					Repository struct {
						NameWithOwner string `json:"nameWithOwner"`
					} `json:"repository"`
					Contributions struct {
						TotalCount int `json:"totalCount"`
					} `json:"contributions"`
				} `json:"pullRequestContributionsByRepository"`
				PullRequestContributions struct {
					Nodes []struct {
						OccurredAt string `json:"occurredAt"`
					} `json:"nodes"`
				} `json:"pullRequestContributions"`
			} `json:"contributionsCollection"`
		} `json:"user"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

func (c *Client) FetchContributions(from, to time.Time) (*Contributions, error) {
	reqBody := graphqlRequest{
		Query: contributionsQuery,
		Variables: map[string]any{
			"login": c.Username,
			"from":  from.Format(time.RFC3339),
			"to":    to.Format(time.RFC3339),
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", graphqlEndpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")

	httpClient := &http.Client{Timeout: 30 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API returned %d: %s", resp.StatusCode, string(respBody))
	}

	var gqlResp graphqlResponse
	if err := json.Unmarshal(respBody, &gqlResp); err != nil {
		return nil, err
	}

	if len(gqlResp.Errors) > 0 {
		return nil, fmt.Errorf("GraphQL error: %s", gqlResp.Errors[0].Message)
	}

	return buildContributions(&gqlResp, from, to), nil
}

// buildContributions assembles a Contributions from a decoded GraphQL response.
// It is the pure, HTTP-free seam that owns all day bucketing: every contribution
// day is attributed to the user's *local* calendar day (from.Location(), which is
// time.Local in real runs), so "today" lines up with what cmd computes from
// time.Now(). Commit days come from per-repo commitCount nodes (true commits), not
// the contributionCalendar, which would fold PRs/issues/reviews into "Commits".
func buildContributions(resp *graphqlResponse, from, to time.Time) *Contributions {
	loc := from.Location()
	fromKey := from.Format("2006-01-02")
	toKey := to.Format("2006-01-02")

	col := resp.Data.User.ContributionsCollection
	// Period totals over GitHub's from/to window — NOT the sum of the local-bucketed
	// daily arrays below; the two can differ at the local-vs-UTC day boundary.
	result := &Contributions{
		TotalCommits: col.TotalCommitContributions,
		TotalPRs:     col.TotalPullRequestContributions,
	}

	commitDayCounts := map[string]int{}
	for _, repo := range col.CommitContributionsByRepository {
		for _, node := range repo.Contributions.Nodes {
			t, err := time.Parse(time.RFC3339, node.OccurredAt)
			if err != nil {
				continue
			}
			commitDayCounts[t.In(loc).Format("2006-01-02")] += node.CommitCount
		}
	}
	result.Days = daysInRange(commitDayCounts, loc, fromKey, toKey)

	prDayCounts := map[string]int{}
	for _, node := range col.PullRequestContributions.Nodes {
		t, err := time.Parse(time.RFC3339, node.OccurredAt)
		if err != nil {
			continue
		}
		prDayCounts[t.In(loc).Format("2006-01-02")]++
	}
	result.PRDays = daysInRange(prDayCounts, loc, fromKey, toKey)

	for _, r := range col.CommitContributionsByRepository {
		result.CommitRepos = append(result.CommitRepos, RepoContribution{
			Repo:  r.Repository.NameWithOwner,
			Count: r.Contributions.TotalCount,
		})
	}
	sort.Slice(result.CommitRepos, func(i, j int) bool {
		return result.CommitRepos[i].Count > result.CommitRepos[j].Count
	})

	for _, r := range col.PullRequestContributionsByRepository {
		result.PRRepos = append(result.PRRepos, RepoContribution{
			Repo:  r.Repository.NameWithOwner,
			Count: r.Contributions.TotalCount,
		})
	}
	sort.Slice(result.PRRepos, func(i, j int) bool {
		return result.PRRepos[i].Count > result.PRRepos[j].Count
	})

	return result
}

// daysInRange turns a local-date-keyed count map into a sorted slice, keeping only
// days within [fromKey, toKey] inclusive. The bound check is a string compare (same
// as render.SumDays) so a day at the window edge isn't dropped by a UTC-vs-local
// offset; dates are parsed in loc so they round-trip to the key cmd/render formats.
func daysInRange(counts map[string]int, loc *time.Location, fromKey, toKey string) []DayContribution {
	out := make([]DayContribution, 0, len(counts))
	for key, count := range counts {
		if key < fromKey || key > toKey {
			continue
		}
		t, err := time.ParseInLocation("2006-01-02", key, loc)
		if err != nil {
			continue
		}
		out = append(out, DayContribution{Date: t, Count: count})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Date.Before(out[j].Date) })
	return out
}
