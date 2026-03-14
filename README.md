# gh-stats

A terminal dashboard for your GitHub contribution stats. View PRs, commits, repo breakdowns, daily activity charts, and team-level analytics — all from the command line.

## Features

- **Personal dashboard** — week-over-week PRs and commits with trend indicators
- **Daily activity charts** — terminal bar charts for commits and PRs over the last 2 weeks
- **Repo breakdown** — see which repositories you're most active in
- **Team stats** — org-wide leaderboard, member activity, and repo breakdown with concurrent fetching
- **Weekly trends** — multi-week bar charts for commits and PRs
- **JSON output** — pipe data to `jq` or other tools with `--json`

## Prerequisites

- [Go 1.25+](https://go.dev/dl/)
- [GitHub CLI (`gh`)](https://cli.github.com/) — authenticated via `gh auth login`

`gh-stats` uses your `gh` auth token automatically. No separate token configuration needed.

## Install

```bash
git clone https://github.com/shadowfax92/gh-stats.git
cd gh-stats
make install
```

This builds the binary and copies it to `$GOPATH/bin`.

## Usage

```bash
# Personal dashboard (default)
gh-stats

# Detailed commit stats with weekly trends
gh-stats commits

# Detailed PR stats with weekly trends
gh-stats prs

# Contribution breakdown by repository
gh-stats repos

# List your organizations
gh-stats orgs

# Team stats for an organization
gh-stats team <org>

# Filter to a specific team member
gh-stats team <org> --member <username>
```

### Global Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--json` | `false` | Output as JSON |
| `--weeks` | `4` | Number of weeks for trend charts |
| `--user` | auto-detected | GitHub username |

## Configuration

Config is stored at `~/.config/gh-stats/config.yaml` (or `$XDG_CONFIG_HOME/gh-stats/config.yaml`). Your GitHub username is auto-detected from `gh` on first run and cached there.

## Shell Completions

```bash
# Fish
make completions
```
