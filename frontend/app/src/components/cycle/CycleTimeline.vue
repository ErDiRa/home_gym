<template>
  <div class="overflow-hidden rounded-lg border border-slate-200 bg-white">
    <ol class="divide-y divide-slate-100">
      <li
        v-for="week in rows"
        :key="week.number"
        class="flex items-start gap-3 px-3 py-2"
        :class="{ 'bg-indigo-50/60': week.current }"
      >
        <div class="w-28 shrink-0 text-xs">
          <span class="font-semibold text-slate-700">Week {{ week.number }}</span>
          <span class="block text-slate-500">{{ formatDate(week.start, { year: false }) }}</span>
        </div>
        <div class="flex min-h-7 flex-1 flex-wrap items-center gap-1.5">
          <RouterLink
            v-for="t in week.trainings"
            :key="t.id"
            :to="{ name: 'training-edit', params: { id: t.id } }"
            class="rounded focus-visible:ring-2 focus-visible:ring-indigo-300 focus-visible:outline-none"
          >
            <AppBadge :color="t.active ? 'indigo' : 'gray'" size="md">{{ t.name }}</AppBadge>
          </RouterLink>
          <span v-if="!week.trainings.length" class="text-xs text-slate-400">No training</span>
        </div>
      </li>
    </ol>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import AppBadge from '../ui/AppBadge.vue'
import { addDays, formatDate, today } from '../../dates'
import type { CycleTraining } from '../../types'

// The cycle's weeks as rows, each with the trainings running in it (several may share a week);
// the current week is highlighted. Trainings outside the weeks don't show here.
const props = defineProps<{ startDate: string; weeks: number; trainings: CycleTraining[] }>()

const rows = computed(() =>
  Array.from({ length: props.weeks }, (_, i) => {
    const start = addDays(props.startDate, i * 7)
    const end = addDays(start, 6)
    return {
      number: i + 1,
      start,
      current: today() >= start && today() <= end,
      trainings: props.trainings.filter((t) => t.start_date <= end && t.end_date >= start),
    }
  }),
)
</script>
