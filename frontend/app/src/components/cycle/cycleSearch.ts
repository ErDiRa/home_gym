import { formatPeriod } from '../../dates'
import { fuzzySearch } from '../../fuzzySearch'
import type { CycleSummary } from '../../types'
import { cycleStatus } from './cycleStatus'

// Finds cycles by name, where today is in them ("Week 3 of 10", "Ended") and period.
export const searchCycles = fuzzySearch<CycleSummary>([
  { name: 'name', weight: 3, get: (c) => c.name },
  { name: 'status', weight: 1, get: (c) => cycleStatus(c) },
  { name: 'period', weight: 1, get: (c) => formatPeriod(c.start_date, c.end_date) },
])
