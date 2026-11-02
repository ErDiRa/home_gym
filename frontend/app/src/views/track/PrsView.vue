<template>
  <section>
    <h1 class="text-2xl font-bold tracking-tight text-slate-900">PRs</h1>
    <p class="mt-1 text-sm text-slate-500">Your 1RM and rep maxes per exercise. Log new ones in a workout or in the dashboard.</p>

    <p v-if="loading" class="mt-6 text-center text-sm text-slate-500">Loading your PRs…</p>
    <div v-else-if="error" class="mt-6 rounded-2xl border border-red-200 bg-red-50 p-6 text-center">
      <p class="font-medium text-red-800">{{ error }}</p>
      <AppButton variant="secondary" size="touch" class="mt-3" @click="load">Try again</AppButton>
    </div>
    <p
      v-else-if="!rows.length"
      class="mt-6 rounded-2xl border border-dashed border-slate-300 bg-white p-6 text-center text-sm text-slate-500"
    >
      No PRs yet.
    </p>

    <template v-else>
      <TextInput v-model="query" type="search" placeholder="Search exercises..." size="touch" class="mt-4" />
      <p v-if="!shown.length" class="mt-6 text-center text-sm text-slate-500">No exercise matches “{{ query }}”.</p>

      <!-- One card per exercise with a max: the 1RM and the rep maxes; tapping shows every entry. -->
      <ul class="mt-4 space-y-2">
        <li v-for="r in shown" :key="r.id" class="overflow-hidden rounded-xl border border-slate-200 bg-white">
          <button
            type="button"
            class="flex min-h-14 w-full items-center gap-3 px-4 py-3 text-left active:bg-slate-50"
            :aria-expanded="expanded.has(r.id)"
            @click="toggle(r.id)"
          >
            <span class="min-w-0 flex-1">
              <span class="block truncate font-medium text-slate-900">{{ r.name }}</span>
              <span class="block text-sm text-slate-500">
                {{ formatDate(r.since) }}
                <template v-if="r.best"> · best {{ formatLoad(r.best.load_value, r.best.load_unit_name) }}</template>
              </span>
              <span v-if="r.repMaxes" class="block text-sm text-slate-500">{{ r.repMaxes }}</span>
            </span>
            <span class="shrink-0 text-lg font-semibold text-slate-900">
              <span v-if="r.current.estimated" class="text-slate-500" title="Estimated from a rep max">≈ </span>{{ formatLoad(r.current.load_value, r.current.load_unit_name) }}
            </span>
            <ChevronDownIcon
              :class="['size-5 shrink-0 text-slate-400 transition-transform', { 'rotate-180': expanded.has(r.id) }]"
              aria-hidden="true"
            />
          </button>
          <MaxHistory v-if="expanded.has(r.id)" :exercise-id="r.id" touch readonly class="border-t border-slate-100 px-4 py-1" />
        </li>
      </ul>
    </template>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ChevronDownIcon } from '@heroicons/vue/24/outline'
import AppButton from '../../components/button/AppButton.vue'
import MaxHistory from '../../components/exercise/MaxHistory.vue'
import { searchExercises } from '../../components/exercise/exerciseSearch'
import { effectiveMax, formatLoad, prIds, repMaxText, repsOf, type EffectiveMax } from '../../components/exercise/maxes'
import TextInput from '../../components/form/TextInput.vue'
import { formatDate } from '../../dates'
import { useExercisesStore } from '../../stores/exercises'
import { useLookupsStore } from '../../stores/lookups'
import type { ExerciseMax } from '../../types'

// The tracker's PR list, read only: 1RMs and rep maxes are logged from a workout's sets or in the dashboard.

const store = useExercisesStore()
const lookups = useLookupsStore()
const query = ref('')
const loading = ref(true)
const error = ref('')
const expanded = ref(new Set<string>())

interface MaxRow {
  id: string
  name: string
  current: EffectiveMax
  since: string
  best: ExerciseMax | null // the best 1RM, when not the current one
  repMaxes: string
}

const unitOf = (id: string) => lookups.loadUnits.find((u) => u.id === id)

// One row per exercise with a max, like the dashboard's PR list: the 1RM (or an estimate from a
// rep max), the rep maxes below.
const rows = computed<MaxRow[]>(() =>
  store.exercises.flatMap((e) => {
    const history = store.maxes.get(e.id) ?? []
    const current = effectiveMax(history, unitOf)
    if (!current) return []
    const ones = history.filter((m) => repsOf(m) === 1)
    const prs = prIds(ones, unitOf)
    // The last PR is the best 1RM; without the units yet, the newest one stands in.
    const best = ones.find((m) => prs.has(m.id)) ?? ones[0] ?? null
    return [{ id: e.id, name: e.name, current, since: history[0].achieved_on, best: best && !current.estimated && best.load_value !== current.load_value ? best : null, repMaxes: repMaxText(history, unitOf) }]
  }),
)
// The fuzzy search of the exercise list, over the exercises with a 1RM.
const shown = computed(() => {
  const byId = new Map(rows.value.map((r) => [r.id, r]))
  return searchExercises(
    store.exercises.filter((e) => byId.has(e.id)),
    query.value,
  ).map((e) => byId.get(e.id)!)
})

function toggle(id: string) {
  const next = new Set(expanded.value)
  if (!next.delete(id)) next.add(id)
  expanded.value = next
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    await Promise.all([store.loadExercises(), store.loadMaxes(), lookups.loadTrainingLookups()])
  } catch {
    error.value = 'Could not load your PRs.'
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>
