<div align="center">

# 📊 gh-stats

**GitHub contribution stats in your terminal.**

*PRs, commits, repos, trends, and team analytics — one command.*

</div>

You want to know how your week is going without leaving the terminal. gh-stats pulls your contribution data from the GitHub GraphQL API and renders dashboards with bar charts, repo breakdowns, and week-over-week comparisons — for yourself or your entire org.

- 📈 **Personal dashboard** — week-over-week PRs and commits with trend indicators
- 📅 **Daily activity charts** — terminal bar charts for commits and PRs over the last 2 weeks
- 📦 **Repo breakdown** — see which repositories you're most active in
- 👥 **Team stats** — org-wide leaderboard, member activity, and concurrent multi-member fetching
- 📉 **Weekly trends** — multi-week bar charts for commits and PRs
- 🔧 **JSON output** — pipe data to `jq` or other tools with `--json`

---

## Install

Requires [Go 1.25+](https://go.dev/dl/) and [GitHub CLI (`gh`)](https://cli.github.com/) authenticated via `gh auth login`.

```sh
git clone https://github.com/shadowfax92/gh-stats.git
cd gh-stats
make install    # builds and copies to $GOPATH/bin/
```

No separate token configuration needed — gh-stats uses your `gh` auth token automatically.

## CLI

```sh
gh-stats                          # personal dashboard (default)
gh-stats commits                  # detailed commit stats with weekly trends
gh-stats prs                      # detailed PR stats with weekly trends
gh-stats repos                    # contribution breakdown by repository
gh-stats orgs                     # list your organizations
gh-stats team <org>               # team stats for an organization
gh-stats team <org> --member bob  # filter to a specific team member
```

### Global Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--json` | `false` | Output as JSON |
| `--weeks` | `4` | Number of weeks for trend charts |
| `--user` | auto-detected | GitHub username |

## Config

Location: `~/.config/gh-stats/config.yaml` (or `$XDG_CONFIG_HOME/gh-stats/config.yaml`)

Your GitHub username is auto-detected from `gh` on first run and cached here. You can also set it manually:

```yaml
username: shadowfax92
```

## Shell Completions

```sh
make completions    # installs fish completions
```

---

> This is a personal tool I built for my own workflow. Feel free to fork and adapt it to your needs.
