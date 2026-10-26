<template>
  <div class="flex h-list-page flex-col lg:h-[calc(100dvh-4rem)]">
    <div class="mb-6 flex flex-wrap items-center justify-between gap-x-4 gap-y-2">
      <h1 class="text-xl font-bold text-slate-900">Exercises</h1>
      <div class="flex items-center gap-x-4 gap-y-2">
        <span class="hidden text-sm text-slate-500 sm:inline">{{ exercises.length }} exercises</span>
        <AppButton :to="{ name: 'exercise-new' }">New exercise</AppButton>
      </div>
    </div>
    <div class="mb-4 flex flex-wrap gap-3">
      <TextInput v-model="query" type="search" placeholder="Search exercises..." class="min-w-60 flex-1" />
      <SelectField
        v-model="ratingFilter"
        :options="ratingOptions"
        :searchable="false"
        aria-label="Filter by your rating"
        class="w-full sm:w-52"
      />
    </div>

    <p v-if="error" class="text-center text-red-600">{{ error }}</p>
    <AgGridVue
      v-else
      class="min-h-0 flex-1"
      :row-data="shown"
      :get-row-id="(p: GetRowIdParams<Exercise>) => p.data.id"
      @grid-ready="(e: GridReadyEvent) => (gridApi = e.api)"
      :column-defs="columnDefs"
      :row-height="phoneRowHeight(phone)"
      :default-col-def="defaultColDef"
      row-class="cursor-pointer"
      @row-clicked="openExercise"
      :loading="loading"
      overlay-no-rows-template="No exercises found."
    />
  </div>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { computed, onMounted, ref, shallowRef } from 'vue'
import { AgGridVue } from 'ag-grid-vue3'
import {
  AllCommunityModule,
  ModuleRegistry,
  type ColDef,
  type GridApi,
  type GridReadyEvent,
  type GetRowIdParams,
  type RowClickedEvent,
} from 'ag-grid-community'
import { useRouter } from 'vue-router'
import { toast } from 'vue-sonner'
import AppButton from '../../components/button/AppButton.vue'
import { phoneColumns, phoneRowHeight } from '../../components/grid/phoneColumns'
import SelectField from '../../components/form/SelectField.vue'
import TextInput from '../../components/form/TextInput.vue'
import ExerciseTypeCell from '../../components/grid/ExerciseTypeCell.vue'
import { searchExercises } from '../../components/exercise/exerciseSearch'
import FocusTagsCell from '../../components/grid/FocusTagsCell.vue'
import RatingCell from '../../components/grid/RatingCell.vue'
import VideoLinkCell from '../../components/grid/VideoLinkCell.vue'
import { useExercisesStore } from '../../stores/exercises'
import { useLookupsStore } from '../../stores/lookups'
import { formatLoad, inKg } from '../../components/exercise/maxes'
import { exerciseTypeLabels, type Exercise, type ExerciseType, type Focus } from '../../types'
import { usePhone } from '../../usePhone'

// AG Grid only includes the features registered here: all free (community) ones.
ModuleRegistry.registerModules([AllCommunityModule])

const router = useRouter()
const store = useExercisesStore()
const { exercises, loading, ratings } = storeToRefs(store)
const lookups = useLookupsStore()
const query = ref('')
const error = ref('')
const gridApi = shallowRef<GridApi | null>(null)

// The user's own rating of an exercise, 0 when not rated.
const ratingOf = (id: string) => ratings.value.get(id)?.rating ?? 0

// Filters the list by the user's rating: how helpful they found an exercise.
const ratingOptions = [
  { id: 'all', name: 'All ratings' },
  { id: 'none', name: 'Not rated' },
  { id: '5', name: '5 stars' },
  { id: '4', name: '4 stars or more' },
  { id: '3', name: '3 stars or more' },
  { id: 'low', name: '2 stars or fewer' },
]
const ratingFilter = ref<string | null>('all')
const matches: Record<string, (r: number) => boolean> = {
  all: () => true,
  none: (r) => r === 0,
  '5': (r) => r === 5,
  '4': (r) => r >= 4,
  '3': (r) => r >= 3,
  low: (r) => r > 0 && r <= 2,
}
// Searched first, so the search index is kept while typing; the grid's sorting still applies.
const shown = computed(() => {
  const match = matches[ratingFilter.value ?? 'all'] ?? matches.all
  return searchExercises(exercises.value, query.value).filter((e) => match(ratingOf(e.id)))
})

// Joints and muscle groups are arrays: show them comma-separated. AG Grid's column filters
// filter array (object) columns by this formatted text.
const listColumn: ColDef<Exercise> = {
  valueFormatter: (p) => (p.value as string[] | null)?.join(', ') ?? '',
}

const columns: ColDef<Exercise>[] = [
  { field: 'name', headerName: 'Name', flex: 2, minWidth: 160 },
  {
    field: 'type',
    headerName: 'Type',
    flex: 1,
    minWidth: 110,
    cellRenderer: ExerciseTypeCell,
    // Filter by the label, not the stored key.
    valueFormatter: (p) => exerciseTypeLabels[p.value as ExerciseType] ?? '',
  },
  { field: 'joints', headerName: 'Joints', flex: 1, minWidth: 130, ...listColumn },
  { field: 'muscle_groups', headerName: 'Muscle groups', flex: 1, minWidth: 160, ...listColumn },
  {
    field: 'focuses',
    headerName: 'Focus',
    flex: 1,
    minWidth: 160,
    cellRenderer: FocusTagsCell,
    // Sort and filter by name and alias, not by the objects.
    valueFormatter: (p) => (p.value as Focus[] | null)?.map((f) => f.name).join(', ') ?? '',
    filterValueGetter: (p) => p.data?.focuses?.map((f) => `${f.name} ${f.alias}`).join(' ') ?? '',
    // Sorted by the first focus, which the server returns alphabetically.
    comparator: (a: Focus[], b: Focus[]) => (a[0]?.name ?? '').localeCompare(b[0]?.name ?? ''),
  },
  {
    colId: 'rating',
    headerName: 'Your rating',
    flex: 1,
    minWidth: 180, // the five stars, also on a phone
    cellRenderer: RatingCell,
    // Sorts by the rating; the filter above the grid filters by it.
    valueGetter: (p) => (p.data ? ratingOf(p.data.id) : 0),
    filter: false,
  },
  {
    colId: 'max',
    headerName: '1RM',
    flex: 1,
    minWidth: 100,
    // Sorts by the current max in kg, so kg and lb compare; shows it as entered.
    valueGetter: (p) => {
      const max = p.data ? store.currentMax(p.data.id) : null
      return max ? (inKg(max.load_value, lookups.loadUnits.find((u) => u.id === max.load_unit)) ?? 0) : null
    },
    valueFormatter: (p) => {
      const max = p.data ? store.currentMax(p.data.id) : null
      return max ? formatLoad(max.load_value, max.load_unit_name) : ''
    },
    filter: false,
  },
  { field: 'description', headerName: 'Description', flex: 3, minWidth: 200 },
  {
    field: 'video_urls',
    headerName: 'Videos',
    flex: 1,
    minWidth: 110,
    sortable: false,
    filter: false,
    cellRenderer: VideoLinkCell,
  },
]
const defaultColDef: ColDef = { sortable: true, filter: true, resizable: true }
// A phone shows only the columns that tell the rows apart.
const phone = usePhone()
const columnDefs = computed(() => phoneColumns(columns, phone.value, ['name', 'rating']))

function openExercise(event: RowClickedEvent<Exercise>) {
  if (event.data) router.push({ name: 'exercise-edit', params: { id: event.data.id } })
}

async function load() {
  // Without the ratings the list still shows, just unrated.
  const ratingsLoaded = store.loadRatings().catch(() => toast.error('Could not load your ratings.'))
  // Likewise without the maxes; the units sort them across kg and lb.
  const maxesLoaded = Promise.all([store.loadMaxes(), lookups.loadTrainingLookups()]).catch(() =>
    toast.error('Could not load your 1RMs.'),
  )
  try {
    await store.loadExercises()
  } catch {
    error.value = 'Could not load exercises.'
    toast.error(error.value)
  }
  await Promise.all([ratingsLoaded, maxesLoaded])
  // The rating and 1RM columns' sort values are cached by the grid.
  gridApi.value?.refreshCells({ columns: ['rating', 'max'], force: true })
}

onMounted(load)
</script>
