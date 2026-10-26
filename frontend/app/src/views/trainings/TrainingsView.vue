<template>
  <div class="flex h-[calc(100vh-8rem)] flex-col lg:h-[calc(100vh-4rem)]">
    <div class="mb-6 flex items-center justify-between gap-4">
      <h1 class="text-xl font-bold text-slate-900">Trainings</h1>
      <div class="flex items-center gap-4">
        <span class="text-sm text-slate-500">{{ trainings.length }} trainings</span>
        <AppButton variant="secondary" :to="{ name: 'training-new', query: { import: '1' } }">Import</AppButton>
        <AppButton :to="{ name: 'training-new' }">New training</AppButton>
      </div>
    </div>
    <div class="mb-4 flex flex-wrap gap-3">
      <TextInput v-model="query" type="search" placeholder="Search trainings..." class="min-w-60 flex-1" />
      <SelectField
        v-model="cycleFilter"
        :options="cycleOptions"
        :searchable="false"
        aria-label="Filter by cycle"
        class="w-56"
      />
    </div>

    <p v-if="error" class="text-center text-red-600">{{ error }}</p>
    <AgGridVue
      v-else
      class="min-h-0 flex-1"
      :row-data="shown"
      :column-defs="columnDefs"
      :default-col-def="defaultColDef"
      row-class="cursor-pointer"
      @row-clicked="openTraining"
      :loading="loading"
      overlay-no-rows-template="No trainings yet."
    />
  </div>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { computed, onMounted, ref } from 'vue'
import { AgGridVue } from 'ag-grid-vue3'
import { AllCommunityModule, ModuleRegistry, type ColDef, type RowClickedEvent } from 'ag-grid-community'
import { useRouter } from 'vue-router'
import { toast } from 'vue-sonner'
import AppButton from '../../components/button/AppButton.vue'
import SelectField from '../../components/form/SelectField.vue'
import TextInput from '../../components/form/TextInput.vue'
import ActiveCell from '../../components/grid/ActiveCell.vue'
import { searchTrainings } from '../../components/training/trainingSearch'
import { formatDay, formatPeriod } from '../../dates'
import { useTrainingsStore } from '../../stores/trainings'
import type { Training } from '../../types'

// AG Grid only includes the features registered here: all free (community) ones.
ModuleRegistry.registerModules([AllCommunityModule])

const router = useRouter()
const store = useTrainingsStore()
const { trainings, loading } = storeToRefs(store)
const query = ref('')
const error = ref('')

// '' = all trainings, 'none' = the ones in no cycle, else a cycle's id.
const cycleFilter = ref<string | null>('')
const cycleOptions = computed(() => {
  const cycles = new Map<string, string>()
  for (const t of trainings.value) if (t.cycle_id) cycles.set(t.cycle_id, t.cycle_name ?? '')
  const byName = [...cycles].sort((a, b) => a[1].localeCompare(b[1])).map(([id, name]) => ({ id, name }))
  return [{ id: '', name: 'All cycles' }, { id: 'none', name: 'No cycle' }, ...byName]
})
// Searched first, so the search index is kept while typing; the grid's sorting still applies.
const shown = computed(() => {
  const f = cycleFilter.value
  const found = searchTrainings(trainings.value, query.value)
  if (!f) return found
  return found.filter((t) => (f === 'none' ? !t.cycle_id : t.cycle_id === f))
})

const columnDefs: ColDef<Training>[] = [
  { field: 'name', headerName: 'Name', flex: 2, minWidth: 160 },
  {
    field: 'active',
    headerName: 'Status',
    flex: 1,
    minWidth: 100,
    cellRenderer: ActiveCell,
    // Filter by the label rather than true/false.
    valueFormatter: (p) => (p.value ? 'Active' : 'Inactive'),
  },
  {
    colId: 'period',
    headerName: 'Period',
    flex: 2,
    minWidth: 190,
    // Sorts and filters by the start date (YYYY-MM-DD sorts as text); shows both dates.
    valueGetter: (p) => p.data?.start_date ?? '',
    valueFormatter: (p) => (p.data ? formatPeriod(p.data.start_date, p.data.end_date) : ''),
  },
  { field: 'weeks', headerName: 'Weeks', flex: 1, minWidth: 90 },
  { field: 'cycle_name', headerName: 'Cycle', flex: 2, minWidth: 140 },
  { field: 'description', headerName: 'Description', flex: 3, minWidth: 200 },
  { field: 'session_count', headerName: 'Sessions', flex: 1, minWidth: 110 },
  { field: 'planned_count', headerName: 'Planned', flex: 1, minWidth: 100, headerTooltip: 'Planned sessions over all weeks' },
  {
    field: 'updated_at',
    headerName: 'Updated',
    flex: 1,
    minWidth: 120,
    valueFormatter: (p) => (p.value ? formatDay(p.value as string, { weekday: false }) : ''),
  },
]
const defaultColDef: ColDef = { sortable: true, filter: true, resizable: true }

function openTraining(event: RowClickedEvent<Training>) {
  if (event.data) router.push({ name: 'training-edit', params: { id: event.data.id } })
}

onMounted(async () => {
  try {
    await store.loadTrainings()
  } catch {
    error.value = 'Could not load trainings.'
    toast.error(error.value)
  }
})
</script>
