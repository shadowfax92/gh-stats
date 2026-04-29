<div align="center">

# 📊 gh-stats

**GitHub contribution stats in your terminal.**

*PRs, commits, repos, week-over-week trends — one command.*

</div>

You want to know how your week is going without leaving the terminal. `gh-stats` pulls your contribution data from the GitHub GraphQL API, caches it, and renders a one-screen dashboard with sparklines and growth percentages — for yourself or your entire org.

- 📈 **One-screen dashboard** — this week vs last, with explicit growth %
- 📅 **14-day sparklines** — commits and PRs in two compact lines
- 📦 **Top repos** — see where your work is concentrated
- 👥 **Team stats** — org-wide leaderboard with member breakdown
- ⚡ **Cached by default** — repeat runs are ~50ms instead of 5s
- 🔍 **Detailed mode** — `-d` adds full daily bar charts and weekly trends
- 🔧 **JSON output** — pipe data to `jq` or other tools with `--json`

> Looking for AI token usage stats? See [`tokens`](https://github.com/shadowfax92/tokens) — Claude Code + Codex daily spend in the same dashboard format.

---

<div align="center">

![gh-stats dashboard](assets/dashboard.svg)

</div>

## Install

Requires [Go 1.25+](https://go.dev/dl/) and [GitHub CLI (`gh`)](https://cli.github.com/) authenticated via `gh auth login`.

```sh
git clone https://github.com/shadowfax92/gh-stats.git
cd gh-stats
make install
```

No separate token configuration needed — `gh-stats` uses your `gh` auth token automatically.

## Quick Start

```sh
gh-stats              # one-screen dashboard
gh-stats -d           # detailed view with full daily bars
gh-stats commits      # weekly commit trend + repos
gh-stats team <org>   # team leaderboard for an org
```

## Commands

| Command | Description |
|---------|-------------|
| `gh-stats` | Default dashboard — this week vs last, sparklines, top repos |
| `gh-stats commits` | Weekly commit trend with full bar chart |
| `gh-stats prs` | Weekly PR trend with full bar chart |
| `gh-stats repos` | Repos ranked by combined commits + PRs |
| `gh-stats orgs` | List your GitHub organizations |
| `gh-stats team <org>` | Team-wide stats for an org with member leaderboard |
| `gh-stats team <org> --member <user>` | Filter team view to one member |
| `gh-stats refresh` | Bust the cache and re-fetch |
| `gh-stats cache` | Print cache path; `--clear` to delete it |

### Global flags

| Flag | Default | Description |
|------|---------|-------------|
| `--weeks N` | `4` | Number of weeks for trend charts |
| `-d, --detailed` | `false` | Show full daily bar charts and weekly trends |
| `--no-cache` | `false` | Bypass cache, force re-fetch |
| `--json` | `false` | JSON output |
| `--user <login>` | auto-detected | GitHub username |

## Detailed mode

`-d` switches the dashboard from sparklines to full vertical bar charts:

<div align="center">

![gh-stats detailed](assets/dashboard-detailed.svg)

</div>

## Per-area deep dives

`gh-stats commits` and `gh-stats prs` zoom in on one metric with the full weekly bar chart and a per-repo breakdown:

<div align="center">

![gh-stats commits](assets/commits.svg)

</div>

`gh-stats repos` ranks repos by combined commits + PRs:

<div align="center">

![gh-stats repos](assets/repos.svg)

</div>

## Config

Location: `~/.config/gh-stats/config.yaml` (or `$XDG_CONFIG_HOME/gh-stats/config.yaml`)

Your GitHub username is auto-detected from `gh` on first run and cached here. You can also set it manually:

```yaml
username: shadowfax92
```

The fetch cache lives at `~/.cache/gh-stats/cache.json`. Print or clear it with:

```sh
gh-stats cache         # print cache path
gh-stats cache --clear # delete the cache
gh-stats refresh       # re-fetch this/last week
```

## Shell completions

```sh
make completions    # installs fish completions
```

---

<div align="center">

> Personal tool I built for my own workflow. Feel free to fork and adapt.

</div>
