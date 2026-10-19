<template>
  <section>
    <h1 class="text-2xl font-bold tracking-tight text-slate-900">History</h1>

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
          <RouterLink
            :to="{ name: 'workout', params: { id: w.id } }"
            class="flex min-h-14 items-center gap-3 rounded-xl border border-slate-200 bg-white px-4 py-3 active:bg-slate-50"
          >
            <span class="min-w-0 flex-1">
              <span class="block truncate font-medium text-slate-900">{{ w.session_name }}</span>
              <span class="block text-sm text-slate-500">
                {{ formatDay(w.completed_at) }} · {{ w.set_count }} {{ w.set_count === 1 ? 'set' : 'sets' }}
              </span>
              <span v-if="w.notes" class="mt-0.5 block truncate text-sm text-slate-600">{{ w.notes }}</span>
            </span>
            <ChevronRightIcon class="size-5 shrink-0 text-slate-400" aria-hidden="true" />
          </RouterLink>
        </li>
      </ul>
    </section>

    <AppButton v-if="weeks < MAX_WEEKS" variant="secondary" size="touch" class="mt-6 w-full" :disabled="loading" @click="loadMore">
      Show earlier weeks
    </AppButton>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ChevronRightIcon } from '@heroicons/vue/24/outline'
import AppButton from '../../components/button/AppButton.vue'
import { formatDay, nextWeek, weekStart } from '../../dates'
import { useWorkoutsStore } from '../../stores/workouts'
import type { WorkoutSummary } from '../../types'

const store = useWorkoutsStore()
const STEP = 8 // weeks loaded at a time
const MAX_WEEKS = 52 // the server lists at most a year at once

const weeks = ref(STEP)
const workouts = ref<WorkoutSummary[]>([])
const loading = ref(false)
const error = ref('')

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
    workouts.value = await store.listWorkouts(weekStart(1 - weeks.value), nextWeek(weekStart()))
  } catch {
    error.value = 'Could not load your workouts.'
  } finally {
    loading.value = false
  }
}

function loadMore() {
  weeks.value = Math.min(weeks.value + STEP, MAX_WEEKS)
  load()
}

onMounted(load)
</script>
