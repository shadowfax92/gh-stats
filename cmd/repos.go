package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/fatih/color"
	gh "github.com/nickhudkins/gh-stats/github"
	"github.com/spf13/cobra"
)

var reposCmd = &cobra.Command{
	Use:   "repos",
	Short: "Contribution breakdown by repository",
	RunE: func(cmd *cobra.Command, args []string) error {
		thisStart, thisEnd := weekBounds(0)

		contribs, err := client.FetchContributions(thisStart, thisEnd)
		if err != nil {
			return err
		}

		merged := mergeRepos(contribs.CommitRepos, contribs.PRRepos)

		if jsonOutput {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(merged)
		}

		bold := color.New(color.Bold)
		dim := color.New(color.Faint)
		cyan := color.New(color.FgCyan)
		green := color.New(color.FgGreen)

		bold.Printf("Repos  %s — %s\n", thisStart.Format("Jan 2"), thisEnd.Format("Jan 2"))
		fmt.Println()

		if len(merged) == 0 {
			dim.Println("  No contributions this week.")
			return nil
		}

		maxTotal := 0
		for _, r := range merged {
			if r.Total > maxTotal {
				maxTotal = r.Total
			}
		}

		barWidth := 20
		for _, r := range merged {
			filled := 0
			if maxTotal > 0 {
				filled = r.Total * barWidth / maxTotal
			}
			if filled == 0 && r.Total > 0 {
				filled = 1
			}

			bar := cyan.Sprint(strings.Repeat("█", filled)) + dim.Sprint(strings.Repeat("░", barWidth-filled))

			repo := r.Repo
			if len(repo) > 35 {
				repo = repo[:32] + "..."
			}

			counts := fmt.Sprintf("%d commits", r.Commits)
			if r.PRs > 0 {
				counts += green.Sprintf(", %d PRs", r.PRs)
			}
			fmt.Printf("  %-37s %s %s\n", repo, bar, counts)
		}

		return nil
	},
}

type mergedRepo struct {
	Repo    string `json:"repo"`
	Commits int    `json:"commits"`
	PRs     int    `json:"prs"`
	Total   int    `json:"total"`
}

func mergeRepos(commitRepos, prRepos []gh.RepoContribution) []mergedRepo {
	m := map[string]*mergedRepo{}
	for _, r := range commitRepos {
		if _, ok := m[r.Repo]; !ok {
			m[r.Repo] = &mergedRepo{Repo: r.Repo}
		}
		m[r.Repo].Commits = r.Count
		m[r.Repo].Total += r.Count
	}
	for _, r := range prRepos {
		if _, ok := m[r.Repo]; !ok {
			m[r.Repo] = &mergedRepo{Repo: r.Repo}
		}
		m[r.Repo].PRs = r.Count
		m[r.Repo].Total += r.Count
	}

	var result []mergedRepo
	for _, r := range m {
		result = append(result, *r)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Total > result[j].Total
	})
	return result
}

func init() {
	rootCmd.AddCommand(reposCmd)
}
