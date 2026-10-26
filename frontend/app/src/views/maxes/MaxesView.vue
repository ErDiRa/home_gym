<template>
  <div class="flex h-[calc(100vh-8rem)] flex-col lg:h-[calc(100vh-4rem)]">
    <div class="mb-6 flex items-center justify-between gap-4">
      <h1 class="text-xl font-bold text-slate-900">PRs</h1>
      <div class="flex items-center gap-4">
        <span class="text-sm text-slate-500">{{ rows.length }} exercises</span>
        <AppButton @click="pickerOpen = true">
          <PlusIcon class="size-4" aria-hidden="true" />
          Log PR
        </AppButton>
      </div>
    </div>
    <p class="mb-4 text-sm text-slate-500">
      Your one-rep max (1RM) per exercise, in kg or lb. The newest entry counts: loads planned in % 1RM are
      calculated from it. Only you see it.
    </p>
    <TextInput v-model="query" type="search" placeholder="Search exercises..." class="mb-4" />

    <p v-if="error" class="text-center text-red-600">{{ error }}</p>
    <AgGridVue
      v-else
      class="min-h-0 flex-1"
      :row-data="shown"
      :get-row-id="(p: GetRowIdParams<MaxRow>) => p.data.id"
      :column-defs="columnDefs"
      :default-col-def="defaultColDef"
      row-class="cursor-pointer"
      @row-clicked="(e: RowClickedEvent<MaxRow>) => e.data && open(e.data.id, e.data.name, false)"
      :loading="loading"
      overlay-no-rows-template="No PRs yet. Log one with “Log PR”."
    />

    <!-- Logging a PR starts with finding the exercise. -->
    <ExercisePickerDialog
      :open="pickerOpen"
      title="Log a PR: pick the exercise"
      :in-block="[]"
      :limit="1"
      add-label="Log PR"
      @close="pickerOpen = false"
      @add="([e]) => e && open(e.id, e.name, true)"
    />
    <ExerciseMaxDialog
      :open="!!opened"
      :exercise-id="opened?.id ?? ''"
      :name="opened?.name ?? ''"
      @close="opened = null"
      @added="added"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { AgGridVue } from 'ag-grid-vue3'
import {
  AllCommunityModule,
  ModuleRegistry,
  type ColDef,
  type GetRowIdParams,
  type RowClickedEvent,
} from 'ag-grid-community'
import { PlusIcon } from '@heroicons/vue/24/outline'
import { toast } from 'vue-sonner'
import AppButton from '../../components/button/AppButton.vue'
import ExerciseMaxDialog from '../../components/exercise/ExerciseMaxDialog.vue'
import { searchExercises } from '../../components/exercise/exerciseSearch'
import { formatLoad, inKg, prIds } from '../../components/exercise/maxes'
import TextInput from '../../components/form/TextInput.vue'
import ExercisePickerDialog from '../../components/training/ExercisePickerDialog.vue'
import { formatDate } from '../../dates'
import { useExercisesStore } from '../../stores/exercises'
import { useLookupsStore } from '../../stores/lookups'
import type { ExerciseMax } from '../../types'

// AG Grid only includes the features registered here: all free (community) ones.
ModuleRegistry.registerModules([AllCommunityModule])

const store = useExercisesStore()
const lookups = useLookupsStore()
const query = ref('')
const loading = ref(true)
const error = ref('')
const pickerOpen = ref(false)
// The exercise whose 1RM history is open; logging closes it again once saved.
const opened = ref<{ id: string; name: string; logging: boolean } | null>(null)

// One row per exercise with a 1RM: the current one, the best and how many entries.
interface MaxRow {
  id: string
  name: string
  current: ExerciseMax
  currentKg: number
  best: ExerciseMax
  bestKg: number
  entries: number
}

const unitOf = (id: string) => lookups.loadUnits.find((u) => u.id === id)
const kgOf = (m: ExerciseMax) => inKg(m.load_value, unitOf(m.load_unit)) ?? 0

const rows = computed<MaxRow[]>(() =>
  store.exercises.flatMap((e) => {
    const history = store.maxes.get(e.id) ?? []
    if (!history.length) return []
    const prs = prIds(history, unitOf)
    // The last PR is the best entry; without the units yet, the current one stands in.
    const best = history.find((m) => prs.has(m.id)) ?? history[0]
    return [{ id: e.id, name: e.name, current: history[0], currentKg: kgOf(history[0]), best, bestKg: kgOf(best), entries: history.length }]
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

// kg and lb sort together by their weight in kg; shown as entered.
const columnDefs: ColDef<MaxRow>[] = [
  { field: 'name', headerName: 'Exercise', flex: 2, minWidth: 180 },
  {
    colId: 'current',
    headerName: 'Current 1RM',
    flex: 1,
    minWidth: 130,
    valueGetter: (p) => p.data?.currentKg ?? 0,
    valueFormatter: (p) => (p.data ? formatLoad(p.data.current.load_value, p.data.current.load_unit_name) : ''),
  },
  {
    colId: 'since',
    headerName: 'Lifted on',
    flex: 1,
    minWidth: 130,
    valueGetter: (p) => p.data?.current.achieved_on ?? '',
    valueFormatter: (p) => (p.value ? formatDate(p.value as string) : ''),
  },
  {
    colId: 'best',
    headerName: 'Best (PR)',
    flex: 1,
    minWidth: 160,
    valueGetter: (p) => p.data?.bestKg ?? 0,
    valueFormatter: (p) =>
      p.data ? `${formatLoad(p.data.best.load_value, p.data.best.load_unit_name)} · ${formatDate(p.data.best.achieved_on)}` : '',
  },
  { field: 'entries', headerName: 'Entries', width: 110 },
]
const defaultColDef: ColDef = { sortable: true, filter: false, resizable: true }

function open(id: string, name: string, logging: boolean) {
  pickerOpen.value = false
  opened.value = { id, name, logging }
}

// A PR logged through "Log PR" is done; one added to an opened row keeps its history open.
function added(m: ExerciseMax) {
  const name = opened.value?.name ?? ''
  toast.success(`Logged ${formatLoad(m.load_value, m.load_unit_name)} for ${name}.`)
  if (opened.value?.logging) opened.value = null
}

onMounted(async () => {
  try {
    await Promise.all([store.loadExercises(), store.loadRatings(), store.loadMaxes(), lookups.loadTrainingLookups()])
  } catch {
    error.value = 'Could not load your PRs.'
    toast.error(error.value)
  } finally {
    loading.value = false
  }
})
</script>
