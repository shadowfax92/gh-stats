#!/usr/bin/env bun

/**
 * engprod-daily renderer.
 *
 * Assembles per-unit sidecar JSONs (pr-<num>.json, commit-<sha>.json) into a
 * single multi-day HTML + JSON report. Designed to be shelled by the skill's
 * orchestrator after analyzer agents have written their sidecars.
 *
 * Usage:
 *   bun render.ts <reports-root> <config-yaml> [date1,date2,...]
 *
 *   reports-root   directory containing <date>/pr-*.json sidecars
 *   config-yaml    the engprod.yaml to read `name` + `engineers` from
 *   dates          optional comma-separated list of YYYY-MM-DD; if omitted,
 *                  auto-detects from subdirectories of reports-root
 *
 * Outputs written to <reports-root>/:
 *   report.json, report.html
 */

import { existsSync } from 'node:fs'
import { readdir, readFile, writeFile } from 'node:fs/promises'
import { join, resolve } from 'node:path'
import {
  type CommitScoreBreakdown,
  computeCommitScore,
  computePrScore,
  type Difficulty,
  type PrScoreBreakdown,
  type PrType,
} from './score'

// Tiny YAML subset parser: enough for engprod.yaml (name: str, engineers: [str]).
function parseConfigYaml(text: string): { name: string; engineers: string[] } {
  const nameMatch = text.match(/^name:\s*([^\n#]+?)\s*$/m)
  const name = nameMatch
    ? nameMatch[1].trim().replace(/^["']|["']$/g, '')
    : 'unknown'
  const engineers: string[] = []
  const engSection = text.match(/^engineers:\s*\n((?:\s*-\s*.+\n?)+)/m)
  if (engSection) {
    for (const line of engSection[1].split('\n')) {
      const m = line.match(/^\s*-\s*(.+?)\s*$/)
      if (m) engineers.push(m[1].replace(/^["']|["']$/g, ''))
    }
  }
  return { name, engineers }
}

const DISPLAY_NAMES: Record<string, string> = {
  DaniAkash: 'Dani',
  felarof99: 'Nithin',
  shivammittal274: 'Shivam',
  shadowfax92: 'shadowfax92',
  neel04: 'Neel',
}

type PrCard = {
  unit_type: 'pr'
  repo: string
  number: number
  sha: string
  branch: string
  merged_at: string
  author: string
  subject: string
  pr_url: string
  insertions: number
  deletions: number
  truncated: boolean
  type: string
  difficulty: string
  impact_score: number
  judgement_multiplier: number
  final_score: number
  breakdown: PrScoreBreakdown
  reasoning: string
  highlights: string[]
  concerns: string[]
}

type LooseCommitCard = {
  unit_type: 'commit'
  repo: string
  sha: string
  sha_short: string
  branch: string
  date: string
  author: string
  subject: string
  insertions: number
  deletions: number
  type: PrType
  summary: string
  final_score: number
  breakdown: CommitScoreBreakdown
}

type EngineerBlock = {
  github: string
  display_name: string | null
  has_activity: boolean
  totals: {
    pr_count: number
    commit_count: number
    loc_added: number
    loc_removed: number
    median_score: number | null
    top3_mean: number | null
  }
  prs: PrCard[]
  loose_commits: LooseCommitCard[]
}

type DayBlock = {
  date: string
  totals: {
    pr_count: number
    commit_count: number
    loc_added: number
    loc_removed: number
  }
  engineers: EngineerBlock[]
}

const [reportsRootArg, configPathArg, datesArg] = process.argv.slice(2)
if (!reportsRootArg || !configPathArg) {
  console.error(
    'usage: bun render.ts <reports-root> <config-yaml> [date1,date2,...]',
  )
  process.exit(2)
}
const reportsRoot = resolve(reportsRootArg)
const config = parseConfigYaml(await readFile(resolve(configPathArg), 'utf8'))
const configuredEngineers = config.engineers
const configName = config.name

const dates = datesArg
  ? datesArg
      .split(',')
      .map((s) => s.trim())
      .filter(Boolean)
  : (await readdir(reportsRoot, { withFileTypes: true }))
      .filter((d) => d.isDirectory() && /^\d{4}-\d{2}-\d{2}$/.test(d.name))
      .map((d) => d.name)
      .sort()
      .reverse()

const normalizePr = (raw: Record<string, unknown>): PrCard | null => {
  if (raw.status === 'error') return null
  const insertions = Number(raw.insertions)
  const deletions = Number(raw.deletions)
  const type = raw.type as PrType
  const difficulty = raw.difficulty as Difficulty
  const impact_score = Number(raw.impact_score)
  const judgement_multiplier = Number(raw.judgement_multiplier)
  const breakdown = computePrScore({
    type,
    difficulty,
    impact_score,
    insertions,
    deletions,
    judgement_multiplier,
  })
  return {
    unit_type: 'pr',
    repo: String(raw.repo ?? 'unknown'),
    number: Number(raw.number ?? raw.pr_number ?? 0),
    sha: String(raw.sha ?? ''),
    branch: String(raw.branch ?? ''),
    merged_at: String(raw.merged_at ?? ''),
    author: String(raw.author ?? ''),
    subject: String(raw.subject ?? ''),
    pr_url: String(raw.pr_url ?? ''),
    insertions,
    deletions,
    truncated: Boolean(raw.truncated ?? false),
    type,
    difficulty,
    impact_score,
    judgement_multiplier: breakdown.multiplier,
    final_score: breakdown.final_score,
    breakdown,
    reasoning: String(raw.reasoning ?? ''),
    highlights: Array.isArray(raw.highlights) ? raw.highlights.map(String) : [],
    concerns: Array.isArray(raw.concerns) ? raw.concerns.map(String) : [],
  }
}

const normalizeCommit = (
  raw: Record<string, unknown>,
): LooseCommitCard | null => {
  if (raw.status === 'error') return null
  const insertions = Number(raw.insertions ?? 0)
  const deletions = Number(raw.deletions ?? 0)
  const type = raw.type as PrType
  const breakdown = computeCommitScore({ type, insertions, deletions })
  const sha = String(raw.sha ?? '')
  return {
    unit_type: 'commit',
    repo: String(raw.repo ?? 'unknown'),
    sha,
    sha_short: String(raw.sha_short ?? sha.slice(0, 7)),
    branch: String(raw.branch ?? ''),
    date: String(raw.date ?? ''),
    author: String(raw.author ?? ''),
    subject: String(raw.subject ?? ''),
    insertions,
    deletions,
    type,
    summary: String(raw.summary ?? ''),
    final_score: breakdown.final_score,
    breakdown,
  }
}

function median(scores: number[]): number | null {
  if (scores.length === 0) return null
  const sorted = scores.slice().sort((a, b) => a - b)
  const middle = Math.floor(sorted.length / 2)
  if (sorted.length % 2 === 1) return sorted[middle] ?? null
  return (
    Math.round(
      (((sorted[middle - 1] ?? 0) + (sorted[middle] ?? 0)) / 2) * 100,
    ) / 100
  )
}

function top3Mean(scores: number[]): number | null {
  if (scores.length < 3) return null
  const top3 = scores
    .slice()
    .sort((a, b) => b - a)
    .slice(0, 3)
  return (
    Math.round((top3.reduce((sum, score) => sum + score, 0) / 3) * 100) / 100
  )
}

function compareNullableDesc(a: number | null, b: number | null): number {
  if (a === null && b === null) return 0
  if (a === null) return 1
  if (b === null) return -1
  return b - a
}

function formatNullableScore(score: number | null): string {
  return score === null ? 'null' : String(score)
}

const buildDay = async (date: string): Promise<DayBlock | null> => {
  const dir = join(reportsRoot, date)
  if (!existsSync(dir)) return null
  const allFiles = await readdir(dir)
  const prFiles = allFiles.filter(
    (f) => f.startsWith('pr-') && f.endsWith('.json'),
  )
  const commitFiles = allFiles.filter(
    (f) => f.startsWith('commit-') && f.endsWith('.json'),
  )

  const prs: PrCard[] = []
  for (const f of prFiles) {
    const raw = JSON.parse(await readFile(join(dir, f), 'utf8')) as Record<
      string,
      unknown
    >
    const pr = normalizePr(raw)
    if (pr) prs.push(pr)
  }

  const commits: LooseCommitCard[] = []
  for (const f of commitFiles) {
    const raw = JSON.parse(await readFile(join(dir, f), 'utf8')) as Record<
      string,
      unknown
    >
    const commit = normalizeCommit(raw)
    if (commit) commits.push(commit)
  }

  const prsByAuthor = new Map<string, PrCard[]>()
  for (const pr of prs) {
    const arr = prsByAuthor.get(pr.author) ?? []
    arr.push(pr)
    prsByAuthor.set(pr.author, arr)
  }

  const commitsByAuthor = new Map<string, LooseCommitCard[]>()
  for (const c of commits) {
    const arr = commitsByAuthor.get(c.author) ?? []
    arr.push(c)
    commitsByAuthor.set(c.author, arr)
  }

  const engineerBlocks: EngineerBlock[] = []
  for (const github of configuredEngineers) {
    const authorPrs = (prsByAuthor.get(github) ?? []).sort(
      (a, b) => b.final_score - a.final_score,
    )
    const authorCommits = (commitsByAuthor.get(github) ?? []).sort(
      (a, b) => b.final_score - a.final_score,
    )
    const hasActivity = authorPrs.length > 0 || authorCommits.length > 0
    const locAdded =
      authorPrs.reduce((s, p) => s + p.insertions, 0) +
      authorCommits.reduce((s, c) => s + c.insertions, 0)
    const locRemoved =
      authorPrs.reduce((s, p) => s + p.deletions, 0) +
      authorCommits.reduce((s, c) => s + c.deletions, 0)
    const scores = authorPrs.map((pr) => pr.final_score)
    engineerBlocks.push({
      github,
      display_name: DISPLAY_NAMES[github] ?? null,
      has_activity: hasActivity,
      totals: {
        pr_count: authorPrs.length,
        commit_count: authorCommits.length,
        loc_added: locAdded,
        loc_removed: locRemoved,
        median_score: median(scores),
        top3_mean: top3Mean(scores),
      },
      prs: authorPrs,
      loose_commits: authorCommits,
    })
  }
  engineerBlocks.sort((a, b) => {
    const medianSort = compareNullableDesc(
      a.totals.median_score,
      b.totals.median_score,
    )
    if (medianSort !== 0) return medianSort
    const top3Sort = compareNullableDesc(a.totals.top3_mean, b.totals.top3_mean)
    if (top3Sort !== 0) return top3Sort
    if (b.totals.pr_count !== a.totals.pr_count)
      return b.totals.pr_count - a.totals.pr_count
    return a.github.localeCompare(b.github)
  })

  return {
    date,
    totals: {
      pr_count: prs.length,
      commit_count: commits.length,
      loc_added:
        prs.reduce((s, p) => s + p.insertions, 0) +
        commits.reduce((s, c) => s + c.insertions, 0),
      loc_removed:
        prs.reduce((s, p) => s + p.deletions, 0) +
        commits.reduce((s, c) => s + c.deletions, 0),
    },
    engineers: engineerBlocks,
  }
}

const days: DayBlock[] = []
for (const date of dates) {
  const day = await buildDay(date)
  if (day) days.push(day)
}

const overall = {
  config_name: configName,
  target_dates: days.map((d) => d.date),
  generated_at: new Date().toISOString(),
  totals: {
    pr_count: days.reduce((s, d) => s + d.totals.pr_count, 0),
    commit_count: days.reduce((s, d) => s + d.totals.commit_count, 0),
    loc_added: days.reduce((s, d) => s + d.totals.loc_added, 0),
    loc_removed: days.reduce((s, d) => s + d.totals.loc_removed, 0),
  },
  days,
}

await writeFile(
  join(reportsRoot, 'report.json'),
  JSON.stringify(overall, null, 2),
)

const esc = (s: string) =>
  s.replace(
    /[&<>"']/g,
    (c) =>
      ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[
        c
      ] as string,
  )

const templatePath = resolve(import.meta.dir, 'template.html')
const template = await readFile(templatePath, 'utf8')
const cssMatch = template.match(/<style>([\s\S]*?)<\/style>/)
const css = cssMatch ? cssMatch[1] : ''

const renderPr = (pr: PrCard): string => {
  const bk = pr.breakdown
  const multChip =
    bk.multiplier !== 1.0
      ? `<span class="chip">mult ${bk.multiplier}</span>`
      : ''
  const breakdownLine =
    `loc ${bk.loc_factor} · diff ${bk.diff_weight} · type ${bk.type_weight}` +
    ` → formula ${bk.formula_score} · impact ${bk.impact_normalized}` +
    ` · mult ${bk.multiplier} → ${bk.final_score}`
  return `
  <article class="pr-card type-${esc(pr.type)}">
    <header>
      <span class="subject">
        <a class="pr-link" href="${esc(pr.pr_url)}">#${pr.number}</a>
        ${esc(pr.subject)}
      </span>
      <span class="score">${pr.final_score}</span>
    </header>
    <div class="meta">
      <span class="chip">${esc(pr.repo)}</span>
      <span class="chip">${esc(pr.branch)}</span>
      <span class="chip">+${pr.insertions} / −${pr.deletions}</span>
      <span class="chip">${esc(pr.type)}</span>
      <span class="chip">${esc(pr.difficulty)}</span>
      <span class="chip">impact ${pr.impact_score}</span>
      ${multChip}
      ${pr.truncated ? '<span class="chip">diff truncated</span>' : ''}
    </div>
    <div class="breakdown">${breakdownLine}</div>
    <div class="reasoning">${esc(pr.reasoning)}</div>
    ${pr.highlights.length > 0 ? `<ul class="highlights">${pr.highlights.map((h) => `<li>${esc(h)}</li>`).join('')}</ul>` : ''}
    ${pr.concerns.length > 0 ? `<ul class="concerns">${pr.concerns.map((c) => `<li>${esc(c)}</li>`).join('')}</ul>` : ''}
  </article>`
}

const renderLooseCommit = (c: LooseCommitCard): string =>
  `<li><code>${esc(c.sha_short)}</code> ${esc(c.subject)}` +
  `<span class="type">— ${esc(c.type)} · +${c.insertions} / −${c.deletions} · score ${c.final_score}</span></li>`

const renderLooseCommits = (commits: LooseCommitCard[]): string =>
  commits.length === 0
    ? ''
    : `<div class="loose-commits">
        <span class="label">Loose commits (${commits.length})</span>
        <ul>${commits.map(renderLooseCommit).join('')}</ul>
      </div>`

const renderEngineer = (e: EngineerBlock): string => `
  <details class="engineer" ${e.has_activity ? 'open' : ''}>
    <summary>
      <span class="github-name">${esc(e.github)}</span>
      ${e.display_name && e.display_name !== e.github ? `<span class="display-name">· ${esc(e.display_name)}</span>` : ''}
      ${
        e.has_activity
          ? `<span class="pills">${e.totals.pr_count} PRs<span class="sep">·</span>${e.totals.commit_count} commits<span class="sep">·</span>+${e.totals.loc_added} / −${e.totals.loc_removed}<span class="sep">·</span>median ${formatNullableScore(e.totals.median_score)}<span class="sep">·</span>top-3 ${formatNullableScore(e.totals.top3_mean)}</span>`
          : '<span class="no-activity">(no activity)</span>'
      }
    </summary>
    <div class="engineer-body">
      ${e.prs.map(renderPr).join('')}
      ${renderLooseCommits(e.loose_commits)}
    </div>
  </details>`

const renderDay = (d: DayBlock, openByDefault: boolean): string => `
  <details class="day" ${openByDefault ? 'open' : ''}>
    <summary>
      ${esc(d.date)}
      <span class="pills">${d.totals.pr_count} PRs<span class="sep">·</span>${d.totals.commit_count} commits<span class="sep">·</span>+${d.totals.loc_added} / −${d.totals.loc_removed}</span>
    </summary>
    ${d.engineers.map(renderEngineer).join('')}
  </details>`

// ── charts ──────────────────────────────────────────────────────────────────

// Render charts for exactly the report period passed to this renderer.
const chartDays = days.slice().sort((a, b) => a.date.localeCompare(b.date))

type EngineerPoint = { engineer: string; value: number }

function niceCeilInt(n: number): number {
  if (n <= 0) return 1
  const exp = 10 ** Math.floor(Math.log10(n))
  const scaled = n / exp
  const niceScaled = scaled <= 1 ? 1 : scaled <= 2 ? 2 : scaled <= 5 ? 5 : 10
  return niceScaled * exp
}

function buildEngineerPoints(
  valueForEngineer: (engineer: EngineerBlock) => number,
): EngineerPoint[] {
  const totals = new Map(configuredEngineers.map((github) => [github, 0]))
  for (const day of chartDays) {
    for (const engineer of day.engineers) {
      totals.set(
        engineer.github,
        (totals.get(engineer.github) ?? 0) + valueForEngineer(engineer),
      )
    }
  }

  const active = [...totals.entries()]
    .map(([engineer, value]) => ({ engineer, value }))
    .filter((point) => point.value > 0)
    .sort((a, b) => b.value - a.value || a.engineer.localeCompare(b.engineer))

  return active.length > 0
    ? active
    : configuredEngineers.map((engineer) => ({ engineer, value: 0 }))
}

function renderEngineerBarChart(
  title: string,
  unit: string,
  points: EngineerPoint[],
  color: string,
): string {
  if (points.length === 0) return ''
  const W = 1040
  const rowH = 34
  const H = Math.max(220, 90 + points.length * rowH)
  const M = { top: 54, right: 64, bottom: 42, left: 190 }
  const innerW = W - M.left - M.right
  const niceMax = niceCeilInt(Math.max(1, ...points.map((p) => p.value)))
  const xScale = (v: number) => M.left + (v / niceMax) * innerW

  const xTicks = Array.from({ length: 5 }, (_, i) =>
    Math.round((niceMax * i) / 4),
  )
  const xGrid = xTicks
    .map(
      (t) =>
        `<line x1="${xScale(t).toFixed(1)}" y1="${M.top - 8}" x2="${xScale(t).toFixed(1)}" y2="${H - M.bottom}" stroke="#e8edf2" stroke-dasharray="2,3"/>` +
        `<text x="${xScale(t).toFixed(1)}" y="${H - 18}" text-anchor="middle" font-size="10" fill="#667085">${t}</text>`,
    )
    .join('')

  const bars = points
    .map(
      (p, i) =>
        `<text x="${M.left - 12}" y="${M.top + i * rowH + 18}" text-anchor="end" font-size="12" fill="#344054">${esc(p.engineer)}</text>` +
        `<rect class="bar" x="${M.left}" y="${M.top + i * rowH}" width="${Math.max(2, xScale(p.value) - M.left).toFixed(1)}" height="20" rx="3" fill="${color}"><title>${esc(p.engineer)} · ${p.value} ${esc(unit)}</title></rect>` +
        `<text x="${Math.min(W - M.right + 8, xScale(p.value) + 8).toFixed(1)}" y="${M.top + i * rowH + 15}" font-size="11" fill="#475467">${p.value}</text>`,
    )
    .join('')

  return `<svg viewBox="0 0 ${W} ${H}" class="chart" preserveAspectRatio="xMidYMid meet" xmlns="http://www.w3.org/2000/svg">
    <rect x="0" y="0" width="${W}" height="${H}" fill="#ffffff"/>
    <text x="${M.left}" y="22" font-size="14" font-weight="600" fill="#1a1a1a">${esc(title)}</text>
    <text x="${M.left}" y="40" font-size="11" fill="#667085">Report period by engineer</text>
    ${xGrid}
    <line x1="${M.left}" y1="${M.top - 8}" x2="${M.left}" y2="${H - M.bottom}" stroke="#cbd5e1"/>
    <line x1="${M.left}" y1="${H - M.bottom}" x2="${M.left + innerW}" y2="${H - M.bottom}" stroke="#cbd5e1"/>
    ${bars}
  </svg>`
}

const prCountPoints = buildEngineerPoints(
  (engineer) => engineer.totals.pr_count,
)

const chartsBlock = `
<section class="charts">
  <div class="chart-wrap">${renderEngineerBarChart('PRs merged', 'PRs', prCountPoints, '#2b6cb0')}</div>
</section>`

const html = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>engprod report · ${esc(overall.config_name)} · ${esc(overall.target_dates[0] ?? '')}</title>
  <style>${css}
    details.day > summary .pills { margin-left: auto; color: var(--muted); font-size: 0.95rem; }
    details.day > summary .pills .sep { color: var(--faint); margin: 0 0.35rem; }
    section.charts { display: flex; flex-direction: column; gap: 1rem; margin-bottom: 1.5rem; }
    section.charts .chart-wrap { background: var(--surface); border: 1px solid var(--border); border-radius: 6px; padding: 0.75rem; width: 100%; overflow-x: auto; }
    section.charts svg.chart { display: block; width: 100%; height: auto; }
    .pr-card .breakdown { margin-top: 0.3rem; font-size: 0.78rem; color: var(--faint); font-family: ui-monospace, Menlo, Monaco, monospace; }
  </style>
</head>
<body>
  <header class="top">
    <h1>engprod · ${esc(overall.config_name)}</h1>
    <div class="subtitle">${
      overall.target_dates.length > 1
        ? `${esc(overall.target_dates[overall.target_dates.length - 1] ?? '')} → ${esc(overall.target_dates[0] ?? '')} · ${overall.target_dates.length} days`
        : esc(overall.target_dates[0] ?? '')
    }</div>
    <div class="totals">
      <span>${overall.totals.pr_count} PRs</span><span class="sep">·</span>
      <span>${overall.totals.commit_count} commits</span><span class="sep">·</span>
      <span>+${overall.totals.loc_added} / −${overall.totals.loc_removed}</span>
    </div>
  </header>
  ${chartsBlock}
  <main>
    ${days.map((d, i) => renderDay(d, i === 0)).join('')}
  </main>
  <footer class="meta">
    Generated ${esc(overall.generated_at)} · engprod-daily
  </footer>
</body>
</html>
`

await writeFile(join(reportsRoot, 'report.html'), html)

const activeEngineers = new Set<string>()
for (const d of days) {
  for (const e of d.engineers) {
    if (e.has_activity) activeEngineers.add(e.github)
  }
}
console.log(`wrote ${join(reportsRoot, 'report.json')}`)
console.log(`wrote ${join(reportsRoot, 'report.html')}`)
console.log(
  `days: ${days.length} · prs: ${overall.totals.pr_count} · commits: ${overall.totals.commit_count} · loc: +${overall.totals.loc_added}/-${overall.totals.loc_removed}`,
)
console.log(
  `engineers with activity: ${[...activeEngineers].join(', ') || '(none)'}`,
)
