/**
 * Deterministic PR/commit scoring formula. Single source of truth for
 * `final_score`. The LLM analyzers produce rubric judgements
 * (`type`, `difficulty`, `impact_score`, `judgement_multiplier`); this module
 * blends those into the integer 0..100 dashboard score.
 *
 * PR formula:
 *   loc          = insertions + deletions
 *   loc_factor   = log2(loc + 1) / log2(10001)              // 0..~1
 *   formula      = loc_factor * DIFF_WEIGHT[difficulty] * TYPE_WEIGHT[type]
 *   impact_norm  = clamp(0, 1, impact_score / 10)
 *   blended      = 0.6 * formula + 0.4 * impact_norm
 *   final_score  = clamp(0, 100, round(blended * judgement_multiplier * 100))
 *
 * Commit formula (no difficulty, impact, or multiplier — by design minimal):
 *   final_score  = clamp(0, 100, round(loc_factor * TYPE_WEIGHT[type] * 100))
 */

export type PrType =
  | 'feature'
  | 'bugfix'
  | 'refactor'
  | 'docs'
  | 'test'
  | 'chore'

export type Difficulty = 'trivial' | 'easy' | 'medium' | 'hard' | 'complex'

export const TYPE_WEIGHT: Record<PrType, number> = {
  feature: 1.0,
  bugfix: 0.8,
  refactor: 0.7,
  docs: 0.6,
  test: 0.5,
  chore: 0.3,
}

export const DIFF_WEIGHT: Record<Difficulty, number> = {
  trivial: 0.2,
  easy: 0.5,
  medium: 1.0,
  hard: 1.5,
  complex: 2.0,
}

const LOC_LOG_DENOM = Math.log2(10001)
const FORMULA_WEIGHT = 0.6
const IMPACT_WEIGHT = 0.4

export type PrScoreInput = {
  type: PrType
  difficulty: Difficulty
  impact_score: number
  insertions: number
  deletions: number
  judgement_multiplier: number
}

export type PrScoreBreakdown = {
  loc: number
  loc_factor: number
  diff_weight: number
  type_weight: number
  formula_score: number
  impact_normalized: number
  blended: number
  multiplier: number
  final_score: number
}

export type CommitScoreInput = {
  type: PrType
  insertions: number
  deletions: number
}

export type CommitScoreBreakdown = {
  loc: number
  loc_factor: number
  type_weight: number
  final_score: number
}

export function computePrScore(input: PrScoreInput): PrScoreBreakdown {
  const tw = TYPE_WEIGHT[input.type]
  const dw = DIFF_WEIGHT[input.difficulty]
  const loc = input.insertions + input.deletions
  const lf = locFactor(loc)
  const formula = lf * dw * tw
  const impact = clamp01(input.impact_score / 10)
  const blended = FORMULA_WEIGHT * formula + IMPACT_WEIGHT * impact
  const final = clamp(
    0,
    100,
    Math.round(blended * input.judgement_multiplier * 100),
  )
  return {
    loc,
    loc_factor: round2(lf),
    diff_weight: dw,
    type_weight: tw,
    formula_score: round2(formula),
    impact_normalized: round2(impact),
    blended: round2(blended),
    multiplier: input.judgement_multiplier,
    final_score: final,
  }
}

export function computeCommitScore(
  input: CommitScoreInput,
): CommitScoreBreakdown {
  const tw = TYPE_WEIGHT[input.type]
  const loc = input.insertions + input.deletions
  const lf = locFactor(loc)
  const final = clamp(0, 100, Math.round(lf * tw * 100))
  return {
    loc,
    loc_factor: round2(lf),
    type_weight: tw,
    final_score: final,
  }
}

function locFactor(loc: number): number {
  return Math.log2(loc + 1) / LOC_LOG_DENOM
}

function clamp(min: number, max: number, n: number): number {
  return Math.max(min, Math.min(max, n))
}

function clamp01(n: number): number {
  return clamp(0, 1, n)
}

function round2(n: number): number {
  return Math.round(n * 100) / 100
}
