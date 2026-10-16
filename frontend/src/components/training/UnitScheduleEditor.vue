<template>
  <div class="flex flex-wrap items-center gap-x-4 gap-y-2">
    <div class="flex flex-wrap items-center gap-1" role="group" :aria-label="`Weekdays for ${label}`">
      <button
        v-for="(day, i) in weekdayLabels"
        :key="day"
        type="button"
        :aria-pressed="weekdays.includes(i + 1)"
        :class="
          weekdays.includes(i + 1)
            ? 'border-indigo-600 bg-indigo-600 text-white'
            : 'border-slate-300 bg-white text-slate-600 hover:bg-slate-100'
        "
        class="h-8 w-11 rounded-md border text-xs font-semibold"
        @click="toggleWeekday(i + 1)"
      >
        {{ day }}
      </button>
    </div>
    <div class="flex flex-wrap items-center gap-1.5">
      <span
        v-for="date in dates"
        :key="date"
        class="inline-flex items-center gap-0.5 rounded-md bg-indigo-50 py-0.5 pr-1 pl-2 text-xs font-medium text-indigo-700"
      >
        {{ formatDate(date) }}
        <button
          type="button"
          class="rounded p-0.5 hover:bg-indigo-100"
          :aria-label="`Remove ${formatDate(date)}`"
          @click="dates = dates.filter((d) => d !== date)"
        >
          <XMarkIcon class="size-3" aria-hidden="true" />
        </button>
      </span>
      <input
        type="date"
        title="Add a specific date"
        :aria-label="`Add a date for ${label}`"
        class="h-8 rounded-md border border-slate-300 bg-white px-2 text-xs text-slate-700 outline-none focus:border-indigo-500 focus:ring-2 focus:ring-indigo-200"
        @change="addDate"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { XMarkIcon } from '@heroicons/vue/24/outline'

// When a unit is done: weekday toggles (ISO, 1 = Monday) plus a list of specific dates.
defineProps<{ label: string }>()
const weekdays = defineModel<number[]>('weekdays', { required: true })
const dates = defineModel<string[]>('dates', { required: true })

const weekdayLabels = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun']

function toggleWeekday(day: number) {
  weekdays.value = weekdays.value.includes(day)
    ? weekdays.value.filter((d) => d !== day)
    : [...weekdays.value, day].sort((a, b) => a - b)
}

function addDate(event: Event) {
  const input = event.target as HTMLInputElement
  if (input.value && !dates.value.includes(input.value)) dates.value = [...dates.value, input.value].sort()
  input.value = ''
}

// Dates are YYYY-MM-DD; parse them as local dates so they don't shift by a day.
function formatDate(date: string) {
  const [y, m, d] = date.split('-').map(Number)
  return new Date(y, m - 1, d).toLocaleDateString(undefined, { weekday: 'short', day: 'numeric', month: 'short', year: 'numeric' })
}
</script>
