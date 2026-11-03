<template>
  <section>
    <nav class="mt-4 flex items-center gap-1" aria-label="Months">
      <AppButton variant="ghost" size="touch" icon aria-label="Previous month" @click="month = addMonths(month, -1)">
        <ChevronLeftIcon class="size-5" aria-hidden="true" />
      </AppButton>
      <h2 id="history-month" class="min-w-0 flex-1 px-1 font-semibold text-slate-900">{{ formatMonth(month) }}</h2>
      <AppButton v-if="!isCurrent" variant="text" size="touch" @click="month = thisMonth">Current</AppButton>
      <AppButton
        variant="ghost"
        size="touch"
        icon
        aria-label="Next month"
        :disabled="isCurrent"
        @click="month = addMonths(month, 1)"
      >
        <ChevronRightIcon class="size-5" aria-hidden="true" />
      </AppButton>
    </nav>

    <!-- Monday to Sunday; a day with workouts has one dot per workout (at most three). -->
    <div class="mt-2 grid grid-cols-7 gap-1">
      <span v-for="d in WEEKDAYS" :key="d" class="pb-1 text-center text-xs font-medium text-slate-500" aria-hidden="true">
        {{ d }}
      </span>
      <button
        v-for="day in days"
        :key="day"
        type="button"
        :aria-label="dayLabel(day)"
        :aria-pressed="day === selected"
        :disabled="day > todayIso"
        :class="[
          'flex min-h-12 flex-col items-center justify-center gap-1 rounded-lg text-sm focus:outline-none focus-visible:ring-2 focus-visible:ring-indigo-300 disabled:opacity-40',
          day === selected
            ? 'bg-indigo-600 text-white'
            : [monthOf(day) === month ? 'text-slate-900' : 'text-slate-400', 'active:bg-slate-100'],
          day === todayIso && day !== selected ? 'ring-1 ring-indigo-400' : '',
          byDay.has(day) ? 'font-semibold' : '',
        ]"
        @click="selected = day"
      >
        {{ Number(day.slice(8)) }}
        <span class="flex h-1.5 gap-0.5" aria-hidden="true">
          <span
            v-for="i in Math.min(byDay.get(day)?.length ?? 0, 3)"
            :key="i"
            :class="['size-1.5 rounded-full', day === selected ? 'bg-white' : 'bg-indigo-500']"
          />
        </span>
      </button>
    </div>

    <section v-if="selected" class="mt-6" :aria-label="formatDate(selected, { weekday: true })">
      <div class="flex items-baseline justify-between">
        <h2 class="font-semibold text-slate-900">{{ formatDate(selected, { weekday: true }) }}</h2>
        <span v-if="selectedWorkouts.length" class="text-sm text-slate-500">
          {{ selectedWorkouts.length }} {{ selectedWorkouts.length === 1 ? 'session' : 'sessions' }}
        </span>
      </div>
      <p v-if="loading && !workouts.length" class="mt-2 text-sm text-slate-500">Loading…</p>
      <p
        v-else-if="!selectedWorkouts.length"
        class="mt-2 rounded-xl border border-dashed border-slate-300 bg-white px-4 py-3 text-sm text-slate-500"
      >
        Rest day
      </p>
      <ul v-else class="mt-2 space-y-2">
        <li v-for="w in selectedWorkouts" :key="w.id">
          <WorkoutRow :workout="w" :macros="macrosOf(w)" show-time />
        </li>
      </ul>
    </section>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { ChevronLeftIcon, ChevronRightIcon } from '@heroicons/vue/24/outline'
import AppButton from '../button/AppButton.vue'
import WorkoutRow from './WorkoutRow.vue'
import { addMonths, formatDate, formatMonth, monthGrid, monthOf, toIso, today } from '../../dates'
import type { WorkoutSummary } from '../../types'

// A month of finished workouts: the days trained marked in a grid, the picked day's workouts
// below it. The parent loads the month's workouts and keeps the month and day (in the URL).
const props = defineProps<{
  workouts: WorkoutSummary[]
  loading?: boolean
  /** The macros line of a workout's day. */
  macrosOf: (w: WorkoutSummary) => string
}>()
const month = defineModel<string>('month', { required: true })
const selected = defineModel<string | null>('selected', { required: true })

const WEEKDAYS = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun']
const todayIso = today()
const thisMonth = monthOf(todayIso)
const isCurrent = computed(() => month.value >= thisMonth)

const days = computed(() => monthGrid(month.value))

// The workouts per local day, oldest first within a day (the server sends them newest first).
const byDay = computed(() => {
  const map = new Map<string, WorkoutSummary[]>()
  for (const w of props.workouts) {
    const day = toIso(new Date(w.completed_at))
    map.set(day, [w, ...(map.get(day) ?? [])])
  }
  return map
})

const selectedWorkouts = computed(() => (selected.value ? (byDay.value.get(selected.value) ?? []) : []))

function dayLabel(day: string): string {
  const n = byDay.value.get(day)?.length ?? 0
  return `${formatDate(day, { weekday: true })}${n ? `, ${n} ${n === 1 ? 'workout' : 'workouts'}` : ''}`
}
</script>
