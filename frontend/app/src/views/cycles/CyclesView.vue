<template>
  <div class="flex h-[calc(100vh-8rem)] flex-col lg:h-[calc(100vh-4rem)]">
    <div class="mb-6 flex items-center justify-between gap-4">
      <h1 class="text-xl font-bold text-slate-900">Cycles</h1>
      <div class="flex items-center gap-4">
        <span class="text-sm text-slate-500">{{ cycles.length }} cycles</span>
        <AppButton variant="secondary" :to="{ name: 'training-new', query: { import: '1' } }">Import</AppButton>
        <AppButton :to="{ name: 'cycle-new' }">New cycle</AppButton>
      </div>
    </div>
    <p class="mb-4 text-sm text-slate-500">
      A cycle is a planned period of weeks with goals, e.g. 8 weeks of mobility for better hips. It groups trainings.
    </p>
    <TextInput v-model="query" type="search" placeholder="Search cycles..." class="mb-4" />

    <p v-if="error" class="text-center text-red-600">{{ error }}</p>
    <AgGridVue
      v-else
      class="min-h-0 flex-1"
      :row-data="cycles"
      :column-defs="columnDefs"
      :default-col-def="defaultColDef"
      :quick-filter-text="query"
      row-class="cursor-pointer"
      @row-clicked="openCycle"
      :loading="loading"
      overlay-no-rows-template="No cycles yet."
    />
  </div>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { onMounted, ref } from 'vue'
import { AgGridVue } from 'ag-grid-vue3'
import { AllCommunityModule, ModuleRegistry, type ColDef, type RowClickedEvent } from 'ag-grid-community'
import { useRouter } from 'vue-router'
import { toast } from 'vue-sonner'
import AppButton from '../../components/button/AppButton.vue'
import { cycleStatus } from '../../components/cycle/cycleStatus'
import TextInput from '../../components/form/TextInput.vue'
import { formatDay, formatPeriod } from '../../dates'
import { useCyclesStore } from '../../stores/cycles'
import type { CycleSummary } from '../../types'

// AG Grid only includes the features registered here: all free (community) ones.
ModuleRegistry.registerModules([AllCommunityModule])

const router = useRouter()
const store = useCyclesStore()
const { cycles, loading } = storeToRefs(store)
const query = ref('')
const error = ref('')

const columnDefs: ColDef<CycleSummary>[] = [
  { field: 'name', headerName: 'Name', flex: 2, minWidth: 160 },
  {
    colId: 'status',
    headerName: 'Now',
    flex: 1,
    minWidth: 140,
    sortable: false,
    valueGetter: (p) => (p.data ? cycleStatus(p.data) : ''),
  },
  {
    colId: 'period',
    headerName: 'Period',
    flex: 2,
    minWidth: 190,
    // Sorts and filters by the start date (YYYY-MM-DD sorts as text); shows both dates.
    valueGetter: (p) => p.data?.start_date ?? '',
    valueFormatter: (p) => (p.data ? formatPeriod(p.data.start_date, p.data.end_date) : ''),
    getQuickFilterText: (p) => (p.data ? formatPeriod(p.data.start_date, p.data.end_date) : ''),
  },
  { field: 'weeks', headerName: 'Weeks', flex: 1, minWidth: 90 },
  { field: 'goal_count', headerName: 'Goals', flex: 1, minWidth: 90 },
  { field: 'training_count', headerName: 'Trainings', flex: 1, minWidth: 110 },
  {
    field: 'updated_at',
    headerName: 'Updated',
    flex: 1,
    minWidth: 120,
    valueFormatter: (p) => (p.value ? formatDay(p.value as string, { weekday: false }) : ''),
  },
]
const defaultColDef: ColDef = { sortable: true, filter: true, resizable: true }

function openCycle(event: RowClickedEvent<CycleSummary>) {
  if (event.data) router.push({ name: 'cycle-edit', params: { id: event.data.id } })
}

onMounted(async () => {
  try {
    await store.loadCycles()
  } catch {
    error.value = 'Could not load cycles.'
    toast.error(error.value)
  }
})
</script>
