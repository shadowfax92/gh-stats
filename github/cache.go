package github

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

const cacheVersion = 3

type cacheEntry struct {
	FetchedAt    time.Time      `json:"fetched_at"`
	Contribs     *Contributions `json:"contribs,omitempty"`
	TeamStats    *TeamStats     `json:"team_stats,omitempty"`
	OrgList      []Org          `json:"org_list,omitempty"`
	OrgMembers   []string       `json:"org_members,omitempty"`
}

type cacheFile struct {
	Version int                   `json:"version"`
	Entries map[string]cacheEntry `json:"entries"`
}

type FetchOptions struct {
	NoCache  bool
	CacheTTL time.Duration
}

func cachePath() string {
	if xdg := os.Getenv("XDG_CACHE_HOME"); xdg != "" {
		return filepath.Join(xdg, "gh-stats", "cache.json")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cache", "gh-stats", "cache.json")
}

func CachePath() string {
	return cachePath()
}

func ClearCache() error {
	err := os.Remove(cachePath())
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func loadCache() *cacheFile {
	data, err := os.ReadFile(cachePath())
	if err != nil {
		return &cacheFile{Version: cacheVersion, Entries: map[string]cacheEntry{}}
	}
	var cf cacheFile
	if err := json.Unmarshal(data, &cf); err != nil || cf.Version != cacheVersion {
		return &cacheFile{Version: cacheVersion, Entries: map[string]cacheEntry{}}
	}
	if cf.Entries == nil {
		cf.Entries = map[string]cacheEntry{}
	}
	return &cf
}

func saveCache(cf *cacheFile) error {
	p := cachePath()
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(cf, "", "  ")
	if err != nil {
		return err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, b, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

func contribsKey(username string, from, to time.Time) string {
	return fmt.Sprintf("contribs|%s|%s|%s", username, from.Format("2006-01-02"), to.Format("2006-01-02"))
}

func teamKey(org string, members []string, from, to time.Time) string {
	sorted := append([]string{}, members...)
	sort.Strings(sorted)
	hash := fmt.Sprintf("%d", len(sorted))
	if len(sorted) > 0 {
		hash = fmt.Sprintf("%d|%s|%s", len(sorted), sorted[0], sorted[len(sorted)-1])
	}
	return fmt.Sprintf("team|%s|%s|%s|%s", org, hash, from.Format("2006-01-02"), to.Format("2006-01-02"))
}

func orgsKey(username string) string {
	return "orgs|" + username
}

func membersKey(org string) string {
	return "members|" + org
}

// FetchContributionsCached wraps FetchContributions with disk cache.
// Cache TTL applies only to weeks containing today; older weeks are cached forever.
func (c *Client) FetchContributionsCached(from, to time.Time, opts FetchOptions) (*Contributions, bool, error) {
	key := contribsKey(c.Username, from, to)
	cf := loadCache()

	if !opts.NoCache {
		if entry, ok := cf.Entries[key]; ok && entry.Contribs != nil {
			today := startOfDay(time.Now())
			isCurrent := !to.Before(today)
			if !isCurrent || time.Since(entry.FetchedAt) < opts.CacheTTL {
				return entry.Contribs, true, nil
			}
		}
	}

	contribs, err := c.FetchContributions(from, to)
	if err != nil {
		return nil, false, err
	}

	cf.Entries[key] = cacheEntry{FetchedAt: time.Now(), Contribs: contribs}
	_ = saveCache(cf)
	return contribs, false, nil
}

// FetchTeamStatsCached wraps FetchTeamStats with disk cache.
func (c *Client) FetchTeamStatsCached(org string, members []string, from, to time.Time, opts FetchOptions) (*TeamStats, bool, error) {
	key := teamKey(org, members, from, to)
	cf := loadCache()

	if !opts.NoCache {
		if entry, ok := cf.Entries[key]; ok && entry.TeamStats != nil {
			today := startOfDay(time.Now())
			isCurrent := !to.Before(today)
			if !isCurrent || time.Since(entry.FetchedAt) < opts.CacheTTL {
				return entry.TeamStats, true, nil
			}
		}
	}

	stats, err := c.FetchTeamStats(org, members, from, to)
	if err != nil {
		return nil, false, err
	}

	cf.Entries[key] = cacheEntry{FetchedAt: time.Now(), TeamStats: stats}
	_ = saveCache(cf)
	return stats, false, nil
}

// ListOrgsCached caches /user/orgs (rare changes, longer TTL is fine but reuse caller's TTL).
func (c *Client) ListOrgsCached(opts FetchOptions) ([]Org, bool, error) {
	key := orgsKey(c.Username)
	cf := loadCache()

	if !opts.NoCache {
		if entry, ok := cf.Entries[key]; ok && entry.OrgList != nil {
			if time.Since(entry.FetchedAt) < opts.CacheTTL {
				return entry.OrgList, true, nil
			}
		}
	}

	orgs, err := c.ListOrgs()
	if err != nil {
		return nil, false, err
	}
	cf.Entries[key] = cacheEntry{FetchedAt: time.Now(), OrgList: orgs}
	_ = saveCache(cf)
	return orgs, false, nil
}

// ListOrgMembersCached caches org membership lookups.
func (c *Client) ListOrgMembersCached(org string, opts FetchOptions) ([]string, bool, error) {
	key := membersKey(org)
	cf := loadCache()

	if !opts.NoCache {
		if entry, ok := cf.Entries[key]; ok && entry.OrgMembers != nil {
			if time.Since(entry.FetchedAt) < opts.CacheTTL {
				return entry.OrgMembers, true, nil
			}
		}
	}

	members, err := c.ListOrgMembers(org)
	if err != nil {
		return nil, false, err
	}
	cf.Entries[key] = cacheEntry{FetchedAt: time.Now(), OrgMembers: members}
	_ = saveCache(cf)
	return members, false, nil
}

func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// ErrNoCache is returned when a cache-only fetch finds nothing.
var ErrNoCache = errors.New("no cached data available")
