import type { Cycle, CycleInput, GoalStatus } from '../../types'
import { nextKey } from '../training/trainingForm'

// An editable goal of the cycle form; key keeps v-for stable while goals move.
export interface GoalRow {
  key: number
  title: string
  target: string
  status: GoalStatus
  notes: string
}

// At most this many goals, weeks 1-52, like the server.
export const MAX_GOALS = 20
export const MAX_CYCLE_WEEKS = 52

export const toGoalRows = (cycle: Cycle): GoalRow[] =>
  cycle.goals.map((g) => ({ key: nextKey(), title: g.title, target: g.target, status: g.status, notes: g.notes }))

export function toCycleInput(fields: Omit<CycleInput, 'goals'>, goals: GoalRow[]): CycleInput {
  return { ...fields, goals: goals.map(({ title, target, status, notes }) => ({ title, target, status, notes })) }
}

// Why the goals can't be saved yet, if anything; mirrors the server's checks.
export function goalsProblem(goals: GoalRow[]): string {
  if (goals.length > MAX_GOALS) return `A cycle has at most ${MAX_GOALS} goals.`
  const i = goals.findIndex((g) => !g.title.trim())
  return i >= 0 ? `Goal ${i + 1} needs a title.` : ''
}
