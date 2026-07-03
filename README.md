<div align="center">

# 📊 gh-stats

**Windowed GitHub contribution stats in your terminal.**

*Totals, day-over-day, week-over-week — for you and your whole team.*

</div>

You want to know "how many PRs did I land this window?" and "what's my team doing?" without leaving the terminal. `gh-stats` pulls your contribution data from the GitHub GraphQL API, caches it, and renders a windowed dashboard with daily charts, growth percentages, and per-member breakdowns.

- 📅 **Window totals first** — top-of-screen answer to "what did I/we ship over `--days`?"
- 📈 **Growth %** — explicit day-over-day and week-over-week deltas, color-coded
- 📊 **Daily bar charts** — 14 days of commits and PRs with per-day labels and values
- 👥 **Team breakdown** — per-member sparklines and DoD/WoW trends
- ⚡ **Cached by default** — repeat runs are ~50ms instead of 5s
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
gh-stats                       # totals, trends, and daily charts for the last 14 days
gh-stats team <org>            # team totals, trends, and per-member breakdown
gh-stats commits               # daily commit chart with DoD / WoW
gh-stats --days 30             # show totals and charts for the last 30 days
```

## Commands

| Command | Description |
|---------|-------------|
| `gh-stats` | Default dashboard — window totals, trends, daily charts, top repos |
| `gh-stats commits` | Daily commits — window total, DoD, WoW, daily bar chart, repos |
| `gh-stats prs` | Daily PRs — same shape as commits |
| `gh-stats repos` | Repos ranked by combined commits + PRs over `--days` |
| `gh-stats orgs` | List your GitHub organizations |
| `gh-stats team <org>` | Team-wide stats — window totals, DoD/WoW, per-member breakdown + sparklines |
| `gh-stats team <org> --member <user>` | Filter team view to one member |
| `gh-stats refresh` | Bust the cache and re-fetch |
| `gh-stats cache` | Print cache path; `--clear` to delete it |

### Global flags

| Flag | Default | Description |
|------|---------|-------------|
| `--days N` | `14` | Window in days for totals, repo rankings, and charts |
| `--no-cache` | `false` | Bypass cache, force re-fetch |
| `--json` | `false` | JSON output |
| `--user <login>` | auto-detected | GitHub username |

## Team dashboard

`gh-stats team <org>` is the killer view: window totals, per-member breakdown, day-over-day and week-over-week trends per person, plus a sparkline row for each member:

<div align="center">

![gh-stats team](assets/team.svg)

</div>

## Per-area deep dives

`gh-stats commits` and `gh-stats prs` zoom in on one metric with a daily bar chart over the `--days` window:

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
