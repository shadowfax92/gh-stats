package github

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestFetchTeamPullRequestsNormalizesGraphQLRows(t *testing.T) {
	var body string
	client := &Client{
		Token: "token",
		HTTPClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.URL.Path != "/graphql" {
				t.Fatalf("path = %s, want /graphql", req.URL.Path)
			}
			raw, _ := io.ReadAll(req.Body)
			body = string(raw)
			return jsonResponse(`{
				"data": {
					"search": {
						"pageInfo": { "hasNextPage": false, "endCursor": "" },
						"nodes": [{
							"repository": { "nameWithOwner": "browseros-ai/app" },
							"author": { "login": "DaniAkash" },
							"mergedAt": "2026-04-22T10:00:00Z",
							"number": 101,
							"title": "feat: alpha",
							"baseRefName": "main",
							"additions": 12,
							"deletions": 4,
							"mergeCommit": { "oid": "sha101" },
							"body": "Alpha body",
							"labels": { "nodes": [{ "name": "feature" }] }
						}]
					}
				}
			}`), nil
		})},
	}

	rows, err := client.FetchTeamPullRequests(
		"browseros-ai",
		[]string{"DaniAkash"},
		time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 4, 24, 12, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body, "author:DaniAkash") || !strings.Contains(body, "merged:2026-04-20..2026-04-24") {
		t.Fatalf("query body did not include expected filters: %s", body)
	}
	if got, want := len(rows), 1; got != want {
		t.Fatalf("rows = %d, want %d", got, want)
	}
	row := rows[0]
	if row.Repo != "browseros-ai/app" || row.Author != "DaniAkash" || row.PRNumber != 101 {
		t.Fatalf("unexpected row: %#v", row)
	}
	if row.Net != 8 || row.SHA != "sha101" || len(row.Labels) != 1 || row.Labels[0] != "feature" {
		t.Fatalf("unexpected normalized fields: %#v", row)
	}
}

func TestFetchTeamCommitsListsReposAndCommitStats(t *testing.T) {
	client := &Client{
		Token: "token",
		HTTPClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			switch req.URL.Path {
			case "/orgs/browseros-ai/repos":
				return jsonResponse(`[{"full_name":"browseros-ai/app","default_branch":"main"}]`), nil
			case "/repos/browseros-ai/app/commits":
				if req.URL.Query().Get("author") != "DaniAkash" {
					t.Fatalf("author query = %s", req.URL.RawQuery)
				}
				return jsonResponse(`[{"sha":"abcdef123456"}]`), nil
			case "/repos/browseros-ai/app/commits/abcdef123456":
				return jsonResponse(`{
					"sha": "abcdef123456",
					"commit": {
						"author": {
							"name": "Dani",
							"email": "dani@example.com",
							"date": "2026-04-22T10:00:00Z"
						},
						"message": "feat: alpha\n\nbody"
					},
					"author": { "login": "DaniAkash" },
					"stats": { "additions": 9, "deletions": 2 },
					"parents": [{ "sha": "p1" }]
				}`), nil
			default:
				t.Fatalf("unexpected request path: %s?%s", req.URL.Path, req.URL.RawQuery)
			}
			return nil, nil
		})},
	}

	rows, err := client.FetchTeamCommits(
		"browseros-ai",
		[]string{"DaniAkash"},
		time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 4, 24, 12, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(rows), 1; got != want {
		t.Fatalf("rows = %d, want %d", got, want)
	}
	row := rows[0]
	if row.Repo != "browseros-ai/app" || row.Author != "DaniAkash" || row.Subject != "feat: alpha" {
		t.Fatalf("unexpected row: %#v", row)
	}
	if row.Insertions != 9 || row.Deletions != 2 || row.Branch != "main" || row.IsMerge {
		t.Fatalf("unexpected commit fields: %#v", row)
	}
}

func TestFetchTeamCommitsSkipsEmptyRepos(t *testing.T) {
	client := &Client{
		Token: "token",
		HTTPClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			switch req.URL.Path {
			case "/orgs/browseros-ai/repos":
				return jsonResponse(`[{"full_name":"browseros-ai/empty","default_branch":"main"}]`), nil
			case "/repos/browseros-ai/empty/commits":
				return &http.Response{
					StatusCode: http.StatusConflict,
					Body:       io.NopCloser(strings.NewReader(`{"message":"Git Repository is empty."}`)),
				}, nil
			default:
				t.Fatalf("unexpected request path: %s", req.URL.Path)
			}
			return nil, nil
		})},
	}

	rows, err := client.FetchTeamCommits(
		"browseros-ai",
		[]string{"DaniAkash"},
		time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 4, 24, 12, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Fatalf("rows = %#v, want empty", rows)
	}
}

func jsonResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: 200,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
