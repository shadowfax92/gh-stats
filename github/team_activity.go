package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

type PullRequestDetail struct {
	Repo       string    `json:"repo"`
	Author     string    `json:"author"`
	MergedAt   time.Time `json:"mergedAt"`
	PRNumber   int       `json:"prNumber"`
	Subject    string    `json:"subject"`
	SHA        string    `json:"sha"`
	Branch     string    `json:"branch"`
	Insertions int       `json:"insertions"`
	Deletions  int       `json:"deletions"`
	Net        int       `json:"net"`
	Body       string    `json:"body,omitempty"`
	Labels     []string  `json:"labels,omitempty"`
}

type CommitDetail struct {
	Repo        string    `json:"repo"`
	SHA         string    `json:"sha"`
	Author      string    `json:"author"`
	AuthorName  string    `json:"authorName"`
	AuthorEmail string    `json:"authorEmail,omitempty"`
	Date        time.Time `json:"date"`
	Subject     string    `json:"subject"`
	Insertions  int       `json:"insertions"`
	Deletions   int       `json:"deletions"`
	Branch      string    `json:"branch"`
	IsMerge     bool      `json:"isMerge"`
	PRNumber    *int      `json:"prNumber"`
}

const teamPullRequestsQuery = `query($query: String!, $after: String) {
  search(query: $query, type: ISSUE, first: 100, after: $after) {
    pageInfo { hasNextPage endCursor }
    nodes {
      ... on PullRequest {
        repository { nameWithOwner }
        author { login }
        mergedAt
        number
        title
        baseRefName
        additions
        deletions
        mergeCommit { oid }
        body
        labels(first: 20) { nodes { name } }
      }
    }
  }
}`

func (c *Client) FetchTeamPullRequests(org string, members []string, since, until time.Time) ([]PullRequestDetail, error) {
	var all []PullRequestDetail
	seen := map[string]bool{}
	for _, member := range members {
		rows, err := c.fetchMemberPullRequests(org, member, since, until)
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			key := fmt.Sprintf("%s#%d", row.Repo, row.PRNumber)
			if !seen[key] {
				all = append(all, row)
				seen[key] = true
			}
		}
	}
	sort.Slice(all, func(i, j int) bool {
		if !all[i].MergedAt.Equal(all[j].MergedAt) {
			return all[i].MergedAt.After(all[j].MergedAt)
		}
		return all[i].Repo < all[j].Repo
	})
	return all, nil
}

func (c *Client) FetchTeamCommits(org string, members []string, since, until time.Time) ([]CommitDetail, error) {
	repos, err := c.listOrgRepos(org)
	if err != nil {
		return nil, err
	}
	var all []CommitDetail
	seen := map[string]bool{}
	for _, repo := range repos {
		for _, member := range members {
			rows, err := c.fetchRepoCommits(repo, member, since, until)
			if err != nil {
				return nil, err
			}
			for _, row := range rows {
				key := row.Repo + "@" + row.SHA
				if !seen[key] {
					all = append(all, row)
					seen[key] = true
				}
			}
		}
	}
	sort.Slice(all, func(i, j int) bool {
		if !all[i].Date.Equal(all[j].Date) {
			return all[i].Date.After(all[j].Date)
		}
		return all[i].Repo < all[j].Repo
	})
	return all, nil
}

func (c *Client) fetchMemberPullRequests(org, member string, since, until time.Time) ([]PullRequestDetail, error) {
	query := fmt.Sprintf("org:%s is:pr is:merged author:%s merged:%s..%s",
		org, member, since.Format("2006-01-02"), until.Format("2006-01-02"))
	var all []PullRequestDetail
	var after *string
	for {
		var resp teamPullRequestsResponse
		if err := c.postGraphQL(teamPullRequestsQuery, map[string]any{
			"query": query,
			"after": after,
		}, &resp); err != nil {
			return nil, err
		}
		for _, node := range resp.Data.Search.Nodes {
			if node.Repository.NameWithOwner == "" || node.Number == 0 {
				continue
			}
			mergedAt, err := time.Parse(time.RFC3339, node.MergedAt)
			if err != nil || mergedAt.Before(since) || mergedAt.After(until) {
				continue
			}
			labels := make([]string, 0, len(node.Labels.Nodes))
			for _, label := range node.Labels.Nodes {
				if label.Name != "" {
					labels = append(labels, label.Name)
				}
			}
			all = append(all, PullRequestDetail{
				Repo:       node.Repository.NameWithOwner,
				Author:     node.Author.Login,
				MergedAt:   mergedAt,
				PRNumber:   node.Number,
				Subject:    node.Title,
				SHA:        node.MergeCommit.OID,
				Branch:     node.BaseRefName,
				Insertions: node.Additions,
				Deletions:  node.Deletions,
				Net:        node.Additions - node.Deletions,
				Body:       node.Body,
				Labels:     labels,
			})
		}
		if !resp.Data.Search.PageInfo.HasNextPage || resp.Data.Search.PageInfo.EndCursor == "" {
			break
		}
		cursor := resp.Data.Search.PageInfo.EndCursor
		after = &cursor
	}
	return all, nil
}

func (c *Client) listOrgRepos(org string) ([]orgRepo, error) {
	var repos []orgRepo
	for page := 1; ; page++ {
		path := fmt.Sprintf("/orgs/%s/repos?type=all&per_page=100&page=%d", url.PathEscape(org), page)
		body, err := c.restGet(path)
		if err != nil {
			return nil, err
		}
		var batch []orgRepo
		if err := json.Unmarshal(body, &batch); err != nil {
			return nil, err
		}
		repos = append(repos, batch...)
		if len(batch) < 100 {
			break
		}
	}
	return repos, nil
}

func (c *Client) fetchRepoCommits(repo orgRepo, member string, since, until time.Time) ([]CommitDetail, error) {
	owner, name, ok := strings.Cut(repo.FullName, "/")
	if !ok {
		return nil, nil
	}
	var rows []CommitDetail
	for page := 1; ; page++ {
		values := url.Values{}
		values.Set("author", member)
		values.Set("since", since.Format(time.RFC3339))
		values.Set("until", until.Format(time.RFC3339))
		values.Set("per_page", "100")
		values.Set("page", fmt.Sprintf("%d", page))
		path := fmt.Sprintf("/repos/%s/%s/commits?%s", url.PathEscape(owner), url.PathEscape(name), values.Encode())
		body, err := c.restGet(path)
		if err != nil {
			if apiErr, ok := err.(*APIError); ok && apiErr.StatusCode == http.StatusConflict {
				return rows, nil
			}
			return nil, err
		}
		var batch []commitListItem
		if err := json.Unmarshal(body, &batch); err != nil {
			return nil, err
		}
		for _, item := range batch {
			detail, err := c.fetchCommitDetail(repo, item.SHA, member)
			if err != nil {
				return nil, err
			}
			rows = append(rows, detail)
		}
		if len(batch) < 100 {
			break
		}
	}
	return rows, nil
}

func (c *Client) fetchCommitDetail(repo orgRepo, sha, requestedAuthor string) (CommitDetail, error) {
	owner, name, ok := strings.Cut(repo.FullName, "/")
	if !ok {
		return CommitDetail{}, fmt.Errorf("invalid repo name %q", repo.FullName)
	}
	path := fmt.Sprintf("/repos/%s/%s/commits/%s", url.PathEscape(owner), url.PathEscape(name), url.PathEscape(sha))
	body, err := c.restGet(path)
	if err != nil {
		return CommitDetail{}, err
	}
	var detail commitDetailResponse
	if err := json.Unmarshal(body, &detail); err != nil {
		return CommitDetail{}, err
	}
	date, _ := time.Parse(time.RFC3339, detail.Commit.Author.Date)
	author := requestedAuthor
	if detail.Author.Login != "" {
		author = detail.Author.Login
	}
	return CommitDetail{
		Repo:        repo.FullName,
		SHA:         detail.SHA,
		Author:      author,
		AuthorName:  detail.Commit.Author.Name,
		AuthorEmail: detail.Commit.Author.Email,
		Date:        date,
		Subject:     firstLine(detail.Commit.Message),
		Insertions:  detail.Stats.Additions,
		Deletions:   detail.Stats.Deletions,
		Branch:      repo.DefaultBranch,
		IsMerge:     len(detail.Parents) > 1,
		PRNumber:    nil,
	}, nil
}

func (c *Client) postGraphQL(query string, variables map[string]any, out any) error {
	reqBody := graphqlRequest{Query: query, Variables: variables}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", graphqlEndpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("GitHub API returned %d: %s", resp.StatusCode, string(respBody))
	}
	if err := json.Unmarshal(respBody, out); err != nil {
		return err
	}
	if e, ok := out.(interface{ graphqlErrors() []graphqlError }); ok {
		if errors := e.graphqlErrors(); len(errors) > 0 {
			return fmt.Errorf("GraphQL error: %s", errors[0].Message)
		}
	}
	return nil
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

type graphqlError struct {
	Message string `json:"message"`
}

type teamPullRequestsResponse struct {
	Data struct {
		Search struct {
			PageInfo struct {
				HasNextPage bool   `json:"hasNextPage"`
				EndCursor   string `json:"endCursor"`
			} `json:"pageInfo"`
			Nodes []struct {
				Repository struct {
					NameWithOwner string `json:"nameWithOwner"`
				} `json:"repository"`
				Author struct {
					Login string `json:"login"`
				} `json:"author"`
				MergedAt    string `json:"mergedAt"`
				Number      int    `json:"number"`
				Title       string `json:"title"`
				BaseRefName string `json:"baseRefName"`
				Additions   int    `json:"additions"`
				Deletions   int    `json:"deletions"`
				MergeCommit struct {
					OID string `json:"oid"`
				} `json:"mergeCommit"`
				Body   string `json:"body"`
				Labels struct {
					Nodes []struct {
						Name string `json:"name"`
					} `json:"nodes"`
				} `json:"labels"`
			} `json:"nodes"`
		} `json:"search"`
	} `json:"data"`
	Errors []graphqlError `json:"errors"`
}

func (r *teamPullRequestsResponse) graphqlErrors() []graphqlError {
	return r.Errors
}

type orgRepo struct {
	FullName      string `json:"full_name"`
	DefaultBranch string `json:"default_branch"`
}

type commitListItem struct {
	SHA string `json:"sha"`
}

type commitDetailResponse struct {
	SHA    string `json:"sha"`
	Commit struct {
		Author struct {
			Name  string `json:"name"`
			Email string `json:"email"`
			Date  string `json:"date"`
		} `json:"author"`
		Message string `json:"message"`
	} `json:"commit"`
	Author struct {
		Login string `json:"login"`
	} `json:"author"`
	Stats struct {
		Additions int `json:"additions"`
		Deletions int `json:"deletions"`
	} `json:"stats"`
	Parents []struct {
		SHA string `json:"sha"`
	} `json:"parents"`
}
