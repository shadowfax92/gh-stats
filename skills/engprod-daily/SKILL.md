---
name: engprod-daily
description: Generate a local per-engineer daily productivity digest from GitHub org activity using gh-stats team raw exports. Produces HTML + JSON only; does not publish or post to Slack.
argument-hint: "[yesterday | today | YYYY-MM-DD] <github-org> [config-yaml]"
---

# engprod-daily for gh-stats

Produces a local per-engineer productivity digest for one GitHub org and one target date. This is the `gh-stats`-native version of the original `engprod-daily` skill: raw PR and commit units come from `gh-stats team <org> --prs/--commits`, not from the old `engprod` CLI.

Output:

- `reports/<bundle>/<date>/prs.json` and `commits.json` — raw `gh-stats` dumps.
- `reports/<bundle>/<date>/pr-*.json` and `commit-*.json` — per-unit analysis sidecars.
- `reports/<bundle>/<date>/report.html` and `report.json` — final local report.

No Slack, no CDN upload, no `engprod share`.

## Inputs

1. **Date** — `yesterday`, `today`, or `YYYY-MM-DD`. Default: `yesterday` in local time.
2. **GitHub org** — required unless the config YAML has an `org:` field.
3. **Config YAML** — optional. Defaults to `./engprod.yaml`.

The config YAML is intentionally tiny:

```yaml
name: browseros
org: browseros-ai
engineers:
  - DaniAkash
  - felarof99
  - neel04
  - shadowfax92
  - shivammittal274
```

`name` controls the report directory. `org` controls the `gh-stats team` org. `engineers` controls final report ordering and zero-activity sections. If both a command argument org and config `org` exist, prefer the command argument.

## Process

### Step 1 — Resolve Inputs

- Resolve target date to `YYYY-MM-DD`.
- Resolve config path to an absolute path.
- Read `name`, optional `org`, and `engineers` from the config YAML.
- Resolve GitHub org from the explicit org argument or config `org`.
- Set report dir to `reports/<name>/<target-date>/` and create it.

Compute `lookback_days` as enough days for `gh-stats --days` to include the target date:

- `today` → `1`
- `yesterday` → `2`
- older dates → calendar day distance from today + `1`

If the target date is in the future, stop.

### Step 2 — Dump Raw Units With gh-stats

Run these from the repo root and redirect stdout to files:

```bash
gh-stats team <org> --prs --days <lookback_days> --json > reports/<bundle>/<date>/prs.json
gh-stats team <org> --commits --days <lookback_days> --json > reports/<bundle>/<date>/commits.json
```

If `gh-stats` is not on `PATH`, use the repo-local binary:

```bash
go run . team <org> --prs --days <lookback_days> --json > reports/<bundle>/<date>/prs.json
go run . team <org> --commits --days <lookback_days> --json > reports/<bundle>/<date>/commits.json
```

If either command exits non-zero, surface stderr and stop.

### Step 3 — Filter and Dedupe

Read both raw JSON files. They are day-grouped envelopes:

- PRs live at `days[].prs[]`; filter by `mergedAt` local date.
- Commits live at `days[].commits[]`; filter by `date` local date.

For dedupe:

1. Start with every target-day PR `sha`.
2. For each target-day PR, run `gh pr view <prNumber> --repo <repo> --json commits` and add every returned commit OID to the PR SHA set.
3. A target-day commit is loose only if its `sha` is not in that set.

If a PR commit lookup fails, record an error and still dedupe by the PR merge/squash SHA.

### Step 4 — Analyze Units

For each target-day PR and loose commit, produce one sidecar JSON. Run these analyses in parallel when your agent runtime supports it.

For a PR:

1. Fetch the diff:

```bash
gh pr diff <prNumber> --repo <repo>
```

2. Classify from the diff:
   - `type`: `feature`, `bugfix`, `refactor`, `chore`, `test`, or `docs`
   - `difficulty`: `trivial`, `easy`, `medium`, `hard`, or `complex`
   - `impact_score`: integer `0..10`
   - `judgement_multiplier`: `0.7`, `1.0`, or `1.3`
   - `reasoning`, `highlights`, `concerns`

3. Write `reports/<bundle>/<date>/pr-<number>.json`:

```json
{
  "unit_type": "pr",
  "status": "ok",
  "repo": "browseros-ai/app",
  "number": 123,
  "sha": "abc123",
  "branch": "main",
  "merged_at": "2026-05-03T18:03:11Z",
  "author": "DaniAkash",
  "subject": "Add auth",
  "pr_url": "https://github.com/browseros-ai/app/pull/123",
  "insertions": 300,
  "deletions": 40,
  "truncated": false,
  "type": "feature",
  "difficulty": "medium",
  "impact_score": 7,
  "judgement_multiplier": 1.0,
  "reasoning": "...",
  "highlights": ["..."],
  "concerns": []
}
```

For a loose commit:

1. Fetch commit detail:

```bash
gh api repos/<repo>/commits/<sha>
```

2. Classify from `files[].patch` and metadata:
   - `type`: `feature`, `bugfix`, `refactor`, `chore`, `test`, or `docs`
   - `summary`: one concise sentence

3. Write `reports/<bundle>/<date>/commit-<sha>.json`:

```json
{
  "unit_type": "commit",
  "status": "ok",
  "repo": "browseros-ai/app",
  "sha": "abc123def456",
  "sha_short": "abc123d",
  "branch": "main",
  "date": "2026-05-03T14:22:03Z",
  "author": "DaniAkash",
  "subject": "chore: bump dependency",
  "insertions": 4,
  "deletions": 4,
  "type": "chore",
  "summary": "Bump the dependency version in package metadata."
}
```

Always write an error sidecar if a diff/detail fetch fails.

### Step 5 — Render

Run the bundled renderer:

```bash
bun skills/engprod-daily/references/render.ts \
  reports/<bundle> \
  <config-path> \
  <target-date>
```

Then copy the rendered files into the date directory:

```bash
cp reports/<bundle>/report.html reports/<bundle>/<target-date>/report.html
cp reports/<bundle>/report.json reports/<bundle>/<target-date>/report.json
```

### Step 6 — Finish

Print the absolute paths for:

- `reports/<bundle>/<target-date>/report.html`
- `reports/<bundle>/<target-date>/report.json`

Stop there. Do not open a browser, publish, upload, or post to Slack.

## Rules

- Use `gh-stats` for raw PR and commit discovery.
- Do not use the old `engprod` CLI.
- Target-date filtering is explicit: PRs use `mergedAt`; commits use `date`.
- Dedupe loose commits against target-day PR commit SHAs.
- Analyzer sidecars are regenerated every run.
- Keep going with partial results when individual PR/commit analysis fails, but record errors in sidecars.

## References

- `references/render.ts` — assembles sidecars into final JSON and HTML.
- `references/score.ts` — deterministic final-score formulas.
- `references/template.html` — HTML/CSS shell.
