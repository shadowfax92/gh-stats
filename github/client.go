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

const contributionsQuery = `query($login: String!, $from: DateTime!, $to: DateTime!) {
  user(login: $login) {
    contributionsCollection(from: $from, to: $to) {
      totalCommitContributions
      totalPullRequestContributions
      contributionCalendar {
        weeks {
          contributionDays {
            contributionCount
            date
          }
        }
      }
      commitContributionsByRepository(maxRepositories: 25) {
        repository { nameWithOwner }
        contributions { totalCount }
      }
      pullRequestContributionsByRepository(maxRepositories: 25) {
        repository { nameWithOwner }
        contributions { totalCount }
      }
      pullRequestContributions(first: 100) {
        nodes {
          occurredAt
        }
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
				TotalCommitContributions      int `json:"totalCommitContributions"`
				TotalPullRequestContributions int `json:"totalPullRequestContributions"`
				ContributionCalendar          struct {
					Weeks []struct {
						ContributionDays []struct {
							ContributionCount int    `json:"contributionCount"`
							Date              string `json:"date"`
						} `json:"contributionDays"`
					} `json:"weeks"`
				} `json:"contributionCalendar"`
				CommitContributionsByRepository []struct {
					Repository struct {
						NameWithOwner string `json:"nameWithOwner"`
					} `json:"repository"`
					Contributions struct {
						TotalCount int `json:"totalCount"`
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

	col := gqlResp.Data.User.ContributionsCollection
	result := &Contributions{
		TotalCommits: col.TotalCommitContributions,
		TotalPRs:     col.TotalPullRequestContributions,
	}

	for _, week := range col.ContributionCalendar.Weeks {
		for _, day := range week.ContributionDays {
			t, err := time.Parse("2006-01-02", day.Date)
			if err != nil {
				continue
			}
			if (t.Equal(from) || t.After(from)) && (t.Before(to) || t.Equal(to)) {
				result.Days = append(result.Days, DayContribution{
					Date:  t,
					Count: day.ContributionCount,
				})
			}
		}
	}

	prDayCounts := map[string]int{}
	for _, node := range col.PullRequestContributions.Nodes {
		t, err := time.Parse(time.RFC3339, node.OccurredAt)
		if err != nil {
			continue
		}
		dateKey := t.Format("2006-01-02")
		prDayCounts[dateKey]++
	}
	for dateKey, count := range prDayCounts {
		t, _ := time.Parse("2006-01-02", dateKey)
		if (t.Equal(from) || t.After(from)) && (t.Before(to) || t.Equal(to)) {
			result.PRDays = append(result.PRDays, DayContribution{Date: t, Count: count})
		}
	}
	sort.Slice(result.PRDays, func(i, j int) bool {
		return result.PRDays[i].Date.Before(result.PRDays[j].Date)
	})

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

	return result, nil
}
