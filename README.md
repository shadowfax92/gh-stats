<div align="center">

# 📊 gh-stats

**Daily GitHub contribution stats in your terminal.**

*Today, day-over-day, week-over-week — for you and your whole team.*

</div>

You want to know "how many PRs did I land today?" and "what's my team doing this week?" without leaving the terminal. `gh-stats` pulls your contribution data from the GitHub GraphQL API, caches it, and renders a daily dashboard with sparklines, growth percentages, and per-member breakdowns.

- 📅 **Today first** — top-of-screen answer to "what did I/we ship today?"
- 📈 **Growth %** — explicit day-over-day and week-over-week deltas, color-coded
- ✨ **14-day sparklines** — commits and PRs per member at a glance
- 👥 **Team breakdown** — per-member trends so you can spot velocity changes
- ⚡ **Cached by default** — repeat runs are ~50ms instead of 5s
- 🔍 **Detailed mode** — `-d` swaps sparklines for full daily bar charts
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
gh-stats                       # how am I doing today + last 14 days
gh-stats team <org>            # how is my team doing today + per-member sparklines
gh-stats commits               # daily commit chart with DoD / WoW
gh-stats -d                    # detailed: full daily bars instead of sparklines
gh-stats --days 30             # extend the trend window to 30 days
```

## Commands

| Command | Description |
|---------|-------------|
| `gh-stats` | Default dashboard — today, trends, 14-day sparklines, top repos |
| `gh-stats commits` | Daily commits — today, DoD, WoW, daily bar chart, repos |
| `gh-stats prs` | Daily PRs — same shape as commits |
| `gh-stats repos` | Repos ranked by combined commits + PRs (this week) |
| `gh-stats orgs` | List your GitHub organizations |
| `gh-stats team <org>` | Team-wide stats — today, DoD/WoW, per-member breakdown + sparklines |
| `gh-stats team <org> --member <user>` | Filter team view to one member |
| `gh-stats refresh` | Bust the cache and re-fetch |
| `gh-stats cache` | Print cache path; `--clear` to delete it |

### Global flags

| Flag | Default | Description |
|------|---------|-------------|
| `--days N` | `14` | Window in days for trends and charts |
| `-d, --detailed` | `false` | Show full daily bar charts instead of sparklines |
| `--no-cache` | `false` | Bypass cache, force re-fetch |
| `--json` | `false` | JSON output |
| `--user <login>` | auto-detected | GitHub username |

## Team dashboard

`gh-stats team <org>` is the killer view — today's totals, per-member breakdown, day-over-day and week-over-week trends per person, plus a 14-day sparkline row for each member:

<div align="center">

![gh-stats team](assets/team.svg)

</div>

## Detailed mode

`-d` switches sparklines for full vertical bar charts (works on dashboard, `commits`, `prs`, `team`):

<div align="center">

![gh-stats detailed](assets/dashboard-detailed.svg)

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
