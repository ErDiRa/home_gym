<template>
  <section>
    <div class="flex items-center justify-between gap-3">
      <h1 class="text-2xl font-bold tracking-tight text-slate-900">History</h1>
      <SegmentedControl v-model="view" :options="VIEWS" label="Show as" />
    </div>

    <!-- The calendar: a month (?month=, none = the current one) and its picked day (?day=). -->
    <template v-if="view === 'calendar'">
      <div v-if="monthError" class="mt-6 rounded-2xl border border-red-200 bg-red-50 p-6 text-center">
        <p class="font-medium text-red-800">{{ monthError }}</p>
        <AppButton variant="secondary" size="touch" class="mt-3" @click="loadMonth">Try again</AppButton>
      </div>
      <HistoryCalendar
        v-model:month="month"
        v-model:selected="day"
        :workouts="monthWorkouts"
        :loading="monthLoading"
        :macros-of="macrosOf"
        :training-of="trainingOf"
      />
    </template>

    <template v-else>
      <p v-if="loading && !workouts.length" class="mt-6 text-center text-sm text-slate-500">Loading your workouts…</p>
      <div v-else-if="error" class="mt-6 rounded-2xl border border-red-200 bg-red-50 p-6 text-center">
        <p class="font-medium text-red-800">{{ error }}</p>
        <AppButton variant="secondary" size="touch" class="mt-3" @click="load">Try again</AppButton>
      </div>
      <p
        v-else-if="!workouts.length"
        class="mt-6 rounded-2xl border border-dashed border-slate-300 bg-white p-6 text-center text-sm text-slate-500"
      >
        No finished workouts in the last {{ weeks }} weeks.
      </p>

      <section v-for="group in groups" :key="group.start" class="mt-6" :aria-label="group.label">
        <div class="flex items-baseline justify-between">
          <h2 class="font-semibold text-slate-900">{{ group.label }}</h2>
          <span class="text-sm text-slate-500">{{ group.workouts.length }} {{ group.workouts.length === 1 ? 'session' : 'sessions' }}</span>
        </div>
        <ul class="mt-2 space-y-2">
          <li v-for="w in group.workouts" :key="w.id">
            <WorkoutRow :workout="w" :macros="macrosOf(w)" />
          </li>
        </ul>
      </section>

      <AppButton v-if="weeks < MAX_WEEKS" variant="secondary" size="touch" class="mt-6 w-full" :disabled="loading" @click="loadMore">
        Show earlier weeks
      </AppButton>
    </template>
  </section>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AppButton from '../../components/button/AppButton.vue'
import SegmentedControl, { type Segment } from '../../components/form/SegmentedControl.vue'
import { macrosLine } from '../../components/nutrition/macros'
import HistoryCalendar from '../../components/workout/HistoryCalendar.vue'
import WorkoutRow from '../../components/workout/WorkoutRow.vue'
import { addDays, formatDay, isMonth, monthGrid, monthOf, nextWeek, startOf, toIso, today, weekStart } from '../../dates'
import { useNutritionStore } from '../../stores/nutrition'
import { useTrainingsStore } from '../../stores/trainings'
import { useWorkoutsStore } from '../../stores/workouts'
import type { DayMacros, WorkoutSummary } from '../../types'

const route = useRoute()
const router = useRouter()
const store = useWorkoutsStore()
const nutrition = useNutritionStore()
const trainingsStore = useTrainingsStore()
const STEP = 8 // weeks loaded at a time
const MAX_WEEKS = 52 // the server lists at most a year at once

type View = 'list' | 'calendar'
const VIEWS: Segment<View>[] = [
  { value: 'calendar', label: 'Calendar' },
  { value: 'list', label: 'List' },
]

// The view (the calendar unless ?view=list), month and day live in the URL, so coming back from
// a workout shows the same ones.
const view = computed<View>({
  get: () => (route.query.view === 'list' ? 'list' : 'calendar'),
  set: (v) =>
    router.replace({
      query: v === 'list' ? { ...route.query, view: v, month: undefined, day: undefined } : { ...route.query, view: undefined },
    }),
})

const thisMonth = monthOf(today())
const month = computed<string>({
  get: () => {
    const m = route.query.month
    return typeof m === 'string' && isMonth(m) && m <= thisMonth ? m : thisMonth
  },
  set: (m) => router.replace({ query: { ...route.query, month: m === thisMonth ? undefined : m, day: undefined } }),
})

// The picked day: one of the month's grid from the URL, else today in the current month, else the
// month's last day trained.
const day = computed<string | null>({
  get: () => {
    const d = route.query.day
    if (typeof d === 'string' && monthGrid(month.value).includes(d)) return d
    if (month.value === thisMonth) return today()
    const trained = monthWorkouts.value.map((w) => toIso(new Date(w.completed_at))).filter((d) => monthOf(d) === month.value)
    return trained[0] ?? null // newest first
  },
  set: (d) => router.replace({ query: { ...route.query, day: d ?? undefined } }),
})

const weeks = ref(STEP)
const workouts = ref<WorkoutSummary[]>([])
const loading = ref(false)
const error = ref('')
// The macros logged per day (YYYY-MM-DD) of the weeks or months shown.
const macros = ref(new Map<string, DayMacros>())

// The macros of a workout's day, as a compact line.
function macrosOf(w: WorkoutSummary): string {
  const m = macros.value.get(toIso(new Date(w.completed_at)))
  return m ? `Ate ${macrosLine(m)}` : ''
}

// The names of the user's trainings by id; a workout of a training deleted since shows without one.
const trainingNames = computed(() => new Map(trainingsStore.trainings.map((t) => [t.id, t.name])))

function trainingOf(w: WorkoutSummary): string {
  return (w.training_id && trainingNames.value.get(w.training_id)) || ''
}

// The local Monday a moment's week starts on, as a timestamp.
function startOfWeek(at: string): number {
  const d = new Date(at)
  d.setHours(0, 0, 0, 0)
  d.setDate(d.getDate() - ((d.getDay() + 6) % 7))
  return d.getTime()
}

// The workouts by week, newest first (the server sends them newest first).
const groups = computed(() => {
  const thisWeek = weekStart().getTime()
  const lastWeek = weekStart(-1).getTime()
  const byWeek = new Map<number, WorkoutSummary[]>()
  for (const w of workouts.value) {
    const start = startOfWeek(w.completed_at)
    byWeek.set(start, [...(byWeek.get(start) ?? []), w])
  }
  return [...byWeek].map(([start, list]) => ({
    start,
    label: start === thisWeek ? 'This week' : start === lastWeek ? 'Last week' : `Week of ${formatDay(new Date(start))}`,
    workouts: list,
  }))
})

async function load() {
  error.value = ''
  loading.value = true
  try {
    const from = weekStart(1 - weeks.value)
    workouts.value = await store.listWorkouts(from, nextWeek(weekStart()))
    loadMacros(toIso(from), today())
  } catch {
    error.value = 'Could not load your workouts.'
  } finally {
    loading.value = false
  }
}

// The macros are an extra: without them the history just doesn't show them.
async function loadMacros(from: string, to: string) {
  try {
    const days = await nutrition.listNutritionDays(from, to)
    macros.value = new Map([...macros.value, ...days.filter((d) => d.macros).map((d) => [d.date, d.macros as DayMacros] as const)])
  } catch {
    // keep what is shown
  }
}

function loadMore() {
  weeks.value = Math.min(weeks.value + STEP, MAX_WEEKS)
  load()
}

// The workouts of the days the calendar shows (its whole weeks, so the days of the months around
// it are marked too).
const monthWorkouts = ref<WorkoutSummary[]>([])
const monthLoading = ref(false)
const monthError = ref('')
let monthRequest = 0

// Only the latest request is kept, so stepping quickly through the months can't show an older answer.
async function loadMonth() {
  const id = ++monthRequest
  const grid = monthGrid(month.value)
  const last = grid[grid.length - 1]
  monthError.value = ''
  monthLoading.value = true
  monthWorkouts.value = []
  try {
    const result = await store.listWorkouts(startOf(grid[0]), startOf(addDays(last, 1)))
    if (id !== monthRequest) return
    monthWorkouts.value = result
    loadMacros(grid[0], last < today() ? last : today())
  } catch {
    if (id === monthRequest) monthError.value = 'Could not load your workouts.'
  } finally {
    if (id === monthRequest) monthLoading.value = false
  }
}

// The training names are an extra too.
trainingsStore.loadTrainings().catch(() => {})

// Each view loads when it is shown: the list once, the calendar per month.
watch(
  [view, month],
  ([v]) => {
    if (v === 'calendar') loadMonth()
    else if (!workouts.value.length && !loading.value) load()
  },
  { immediate: true },
)
</script>
