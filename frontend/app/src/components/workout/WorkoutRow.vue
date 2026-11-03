<template>
  <RouterLink
    :to="{ name: 'workout', params: { id: workout.id } }"
    class="flex min-h-14 items-center gap-3 rounded-xl border border-slate-200 bg-white px-4 py-3 active:bg-slate-50"
  >
    <span class="min-w-0 flex-1">
      <span class="block truncate font-medium text-slate-900">{{ workout.session_name }}</span>
      <span class="block text-sm text-slate-500">
        {{ showTime ? formatTime(workout.completed_at) : formatDay(workout.completed_at) }} · {{ workout.set_count }}
        {{ workout.set_count === 1 ? 'set' : 'sets' }}
      </span>
      <span v-if="workout.notes" class="mt-0.5 block truncate text-sm text-slate-600">{{ workout.notes }}</span>
      <span v-if="macros" class="mt-0.5 block truncate text-xs text-slate-500">{{ macros }}</span>
    </span>
    <ChevronRightIcon class="size-5 shrink-0 text-slate-400" aria-hidden="true" />
  </RouterLink>
</template>

<script setup lang="ts">
import { ChevronRightIcon } from '@heroicons/vue/24/outline'
import { formatDay, formatTime } from '../../dates'
import type { WorkoutSummary } from '../../types'

// A finished workout in the history, opening it. showTime: the time it was finished instead of
// its day, for a list of one day's workouts.
defineProps<{ workout: WorkoutSummary; macros?: string; showTime?: boolean }>()
</script>
