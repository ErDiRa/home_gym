// A unit of a training, e.g. "Day A": done on the period's matching weekdays and on its dates.
export interface TrainingUnit {
  id: string
  name: string
  notes: string
  weekdays: number[] // ISO, 1 = Monday
  dates: string[] // YYYY-MM-DD
}

// A training the client follows. The API also sends each unit's blocks and exercises, which
// this app doesn't show yet.
export interface Training {
  id: string
  name: string
  description: string
  start_date: string // YYYY-MM-DD, first day of the period
  end_date: string // YYYY-MM-DD, last day of the period
  units: TrainingUnit[]
}
