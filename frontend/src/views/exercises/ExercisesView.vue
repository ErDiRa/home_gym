<template>
  <div class="flex h-[calc(100vh-8rem)] flex-col lg:h-[calc(100vh-4rem)]">
    <div class="mb-6 flex items-center justify-between gap-4">
      <h1 class="text-xl font-bold text-slate-900">Exercises</h1>
      <div class="flex items-center gap-4">
        <span class="text-sm text-slate-500">{{ exercises.length }} exercises</span>
        <AppButton v-if="state.user?.is_admin" :to="{ name: 'exercise-new' }">New exercise</AppButton>
      </div>
    </div>
    <input
      v-model="query"
      type="search"
      placeholder="Search exercises..."
      class="mb-4 w-full rounded-lg border border-slate-300 bg-white px-4 py-2 outline-none focus:border-indigo-500 focus:ring-2 focus:ring-indigo-200"
    />

    <p v-if="error" class="text-center text-red-600">{{ error }}</p>
    <AgGridVue
      v-else
      class="min-h-0 flex-1"
      :row-data="exercises"
      :column-defs="columnDefs"
      :default-col-def="defaultColDef"
      :quick-filter-text="query"
      row-class="cursor-pointer"
      @row-clicked="openExercise"
      :loading="loading"
      overlay-no-rows-template="No exercises found."
    />
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { AgGridVue } from 'ag-grid-vue3'
import { AllCommunityModule, ModuleRegistry, type ColDef, type RowClickedEvent } from 'ag-grid-community'
import { useRouter } from 'vue-router'
import { toast } from 'vue-sonner'
import { listExercises } from '../../api'
import AppButton from '../../components/button/AppButton.vue'
import ExerciseTypeCell from '../../components/grid/ExerciseTypeCell.vue'
import FocusTagsCell from '../../components/grid/FocusTagsCell.vue'
import VideoLinkCell from '../../components/grid/VideoLinkCell.vue'
import { state } from '../../session'
import { exerciseTypeLabels, type Exercise, type ExerciseType, type Focus } from '../../types'

ModuleRegistry.registerModules([AllCommunityModule])

const router = useRouter()
const exercises = ref<Exercise[]>([])
const query = ref('')
const loading = ref(true)
const error = ref('')

// Joints and muscle groups are arrays: show them comma-separated and make them searchable.
const listColumn: ColDef<Exercise> = {
  valueFormatter: (p) => (p.value as string[] | null)?.join(', ') ?? '',
  getQuickFilterText: (p) => (p.value as string[] | null)?.join(' ') ?? '',
}

const columnDefs: ColDef<Exercise>[] = [
  { field: 'name', headerName: 'Name', flex: 2, minWidth: 160 },
  {
    field: 'type',
    headerName: 'Type',
    flex: 1,
    minWidth: 110,
    cellRenderer: ExerciseTypeCell,
    // Filter and search by the label, not the stored key.
    valueFormatter: (p) => exerciseTypeLabels[p.value as ExerciseType] ?? '',
    getQuickFilterText: (p) => exerciseTypeLabels[p.value as ExerciseType] ?? '',
  },
  { field: 'joints', headerName: 'Joints', flex: 1, minWidth: 130, ...listColumn },
  { field: 'muscle_groups', headerName: 'Muscle groups', flex: 1, minWidth: 160, ...listColumn },
  {
    field: 'focuses',
    headerName: 'Focus',
    flex: 1,
    minWidth: 160,
    cellRenderer: FocusTagsCell,
    // Sort, filter and search by name and alias, not by the objects.
    valueFormatter: (p) => (p.value as Focus[] | null)?.map((f) => f.name).join(', ') ?? '',
    getQuickFilterText: (p) => (p.value as Focus[] | null)?.map((f) => `${f.name} ${f.alias}`).join(' ') ?? '',
    comparator: (a: Focus[], b: Focus[]) => (a[0]?.name ?? '').localeCompare(b[0]?.name ?? ''),
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

function openExercise(event: RowClickedEvent<Exercise>) {
  if (event.data) router.push({ name: 'exercise-edit', params: { id: event.data.id } })
}

async function load() {
  try {
    exercises.value = await listExercises()
  } catch {
    error.value = 'Could not load exercises.'
    toast.error(error.value)
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>
