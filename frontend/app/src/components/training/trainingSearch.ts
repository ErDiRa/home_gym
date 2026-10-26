import { formatPeriod } from '../../dates'
import { fuzzySearch } from '../../fuzzySearch'
import type { Training } from '../../types'

// Finds trainings by name, cycle, period and "active" (only active ones have the word, so
// searching it skips the inactive ones).
export const searchTrainings = fuzzySearch<Training>([
  { name: 'name', weight: 3, get: (t) => t.name },
  { name: 'cycle', weight: 2, get: (t) => t.cycle_name ?? '' },
  { name: 'status', weight: 1, get: (t) => (t.active ? 'Active' : '') },
  { name: 'period', weight: 1, get: (t) => formatPeriod(t.start_date, t.end_date) },
])
