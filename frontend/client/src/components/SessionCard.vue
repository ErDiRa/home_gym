<template>
  <li
    class="overflow-hidden rounded-2xl border bg-white shadow-sm"
    :class="today ? 'border-indigo-500 ring-1 ring-indigo-500' : 'border-slate-200'"
    :aria-current="today ? 'date' : undefined"
  >
    <header class="flex items-center justify-between gap-2 border-b border-slate-100 bg-slate-50 px-4 py-2.5">
      <time :datetime="session.date" class="text-sm font-semibold text-slate-800">
        {{ formatDate(session.date, { weekday: 'long', day: 'numeric', month: 'long' }) }}
      </time>
      <span v-if="today" class="rounded-full bg-indigo-600 px-2 py-0.5 text-xs font-semibold text-white">Today</span>
    </header>
    <ul class="divide-y divide-slate-100 px-4">
      <li v-for="unit in session.units" :key="unit.id" class="py-3">
        <p class="font-semibold text-slate-900">{{ unit.name }}</p>
        <p v-if="unit.notes" class="mt-0.5 text-sm text-slate-500">{{ unit.notes }}</p>
      </li>
    </ul>
  </li>
</template>

<script setup lang="ts">
import { formatDate } from '../dates'
import type { Session } from '../schedule'

// One training day: its date, and the units done on it.
defineProps<{ session: Session; today?: boolean }>()
</script>
