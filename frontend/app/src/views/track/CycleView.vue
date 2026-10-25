<template>
  <section class="pb-24">
    <AppButton :to="{ name: 'track', query: backQuery }" variant="ghost" size="touch" class="-ml-3 text-slate-600!">
      <ArrowLeftIcon class="size-4" aria-hidden="true" />
      This week
    </AppButton>

    <p v-if="loading" class="mt-6 text-center text-sm text-slate-500">Loading the cycle…</p>
    <div v-else-if="error" class="mt-6 rounded-2xl border border-red-200 bg-red-50 p-6 text-center">
      <p class="font-medium text-red-800">{{ error }}</p>
    </div>

    <template v-else-if="cycle">
      <p class="mt-2 text-sm text-slate-500">Cycle · {{ formatPeriod(cycle.start_date, cycle.end_date) }}</p>
      <h1 class="text-2xl font-bold tracking-tight text-slate-900">{{ cycle.name }}</h1>
      <p v-if="cycle.description" class="mt-1 text-sm text-slate-600">{{ cycle.description }}</p>

      <div class="mt-4 flex items-center gap-3">
        <div
          class="h-2 flex-1 overflow-hidden rounded-full bg-slate-200"
          role="progressbar"
          :aria-valuenow="week ?? 0"
          aria-valuemin="0"
          :aria-valuemax="cycle.weeks"
          :aria-label="status"
        >
          <div class="h-full rounded-full bg-indigo-500" :style="{ width: `${progress}%` }" />
        </div>
        <span class="shrink-0 text-sm font-medium text-slate-700">{{ status }}</span>
      </div>

      <section class="mt-6" aria-labelledby="goals-heading">
        <h2 id="goals-heading" class="font-semibold text-slate-900">Goals</h2>
        <p
          v-if="!cycle.goals.length"
          class="mt-2 rounded-2xl border border-dashed border-slate-300 bg-white p-4 text-center text-sm text-slate-500"
        >
          No goals yet. Add them to the cycle in the dashboard.
        </p>
        <ul v-else class="mt-2 space-y-3">
          <li v-for="g in cycle.goals" :key="g.id" class="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm">
            <div class="flex items-start gap-3">
              <p class="min-w-0 flex-1 text-base font-medium text-slate-900">{{ g.title }}</p>
              <AppBadge :color="statusColors[g.status]">{{ goalStatusLabels[g.status] }}</AppBadge>
            </div>
            <p v-if="g.target" class="mt-1 text-sm text-slate-600">
              <span class="font-medium text-slate-700">Target:</span> {{ g.target }}
            </p>
            <p v-if="g.notes" class="mt-1 text-sm whitespace-pre-line text-slate-500">{{ g.notes }}</p>
          </li>
        </ul>
      </section>

      <section class="mt-6" aria-labelledby="trainings-heading">
        <h2 id="trainings-heading" class="font-semibold text-slate-900">Trainings</h2>
        <p v-if="!cycle.trainings.length" class="mt-2 text-sm text-slate-500">No trainings in this cycle yet.</p>
        <ul v-else class="mt-2 divide-y divide-slate-100 overflow-hidden rounded-2xl border border-slate-200 bg-white">
          <li
            v-for="t in cycle.trainings"
            :key="t.id"
            class="flex items-center justify-between gap-3 px-4 py-3"
            :class="{ 'bg-indigo-50': t.id === currentTraining }"
            :aria-current="t.id === currentTraining ? 'true' : undefined"
          >
            <div class="min-w-0">
              <p class="truncate text-sm" :class="t.id === currentTraining ? 'font-semibold text-indigo-800' : 'text-slate-800'">
                {{ t.name }}
              </p>
              <p class="text-xs text-slate-500">
                {{ formatDate(t.start_date, { year: false }) }} – {{ formatDate(t.end_date, { year: false }) }}
              </p>
            </div>
            <AppBadge v-if="t.start_date <= monday && t.end_date >= monday" color="indigo">This week</AppBadge>
          </li>
        </ul>
      </section>
    </template>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { ArrowLeftIcon } from '@heroicons/vue/24/outline'
import { ApiError } from '../../api'
import AppButton from '../../components/button/AppButton.vue'
import { cycleStatus } from '../../components/cycle/cycleStatus'
import AppBadge, { type BadgeColor } from '../../components/ui/AppBadge.vue'
import { formatDate, formatPeriod, toIso, weekOf, weekStart } from '../../dates'
import { useCyclesStore } from '../../stores/cycles'
import { goalStatusLabels, type Cycle, type GoalStatus } from '../../types'

// A cycle as the tracker shows it, opened from the training it came from (?training=): its
// period and progress, its goals with their status, and its trainings, that one highlighted.
// Read only; the cycle is edited in the dashboard.
const route = useRoute()
const store = useCyclesStore()
const cycle = ref<Cycle | null>(null)
const loading = ref(true)
const error = ref('')

const currentTraining = typeof route.query.training === 'string' ? route.query.training : null
// Back to the week with the same training chosen.
const backQuery = currentTraining ? { training: currentTraining } : {}
const monday = toIso(weekStart())

const statusColors: Record<GoalStatus, BadgeColor> = { open: 'gray', achieved: 'green', missed: 'red' }
const week = computed(() => (cycle.value ? weekOf(cycle.value.start_date, cycle.value.weeks, monday) : null))
const status = computed(() => (cycle.value ? cycleStatus(cycle.value) : ''))
// The share of the cycle reached: none before it starts, all once it has ended.
const progress = computed(() => {
  const c = cycle.value
  if (!c) return 0
  if (week.value !== null) return (week.value / c.weeks) * 100
  return monday > c.end_date ? 100 : 0
})

onMounted(async () => {
  try {
    cycle.value = await store.getCycle(route.params.id as string)
  } catch (e) {
    error.value = e instanceof ApiError && e.status === 404 ? 'Cycle not found.' : 'Could not load the cycle.'
  } finally {
    loading.value = false
  }
})
</script>
