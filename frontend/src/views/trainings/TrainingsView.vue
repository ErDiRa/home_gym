<template>
  <div class="flex h-[calc(100vh-8rem)] flex-col lg:h-[calc(100vh-4rem)]">
    <div class="mb-6 flex items-center justify-between gap-4">
      <h1 class="text-xl font-bold text-slate-900">Trainings</h1>
      <div class="flex items-center gap-4">
        <span class="text-sm text-slate-500">{{ trainings.length }} trainings</span>
        <AppButton :to="{ name: 'training-new' }">New training</AppButton>
      </div>
    </div>
    <input
      v-model="query"
      type="search"
      placeholder="Search trainings..."
      class="mb-4 w-full rounded-lg border border-slate-300 bg-white px-4 py-2 outline-none focus:border-indigo-500 focus:ring-2 focus:ring-indigo-200"
    />

    <p v-if="error" class="text-center text-red-600">{{ error }}</p>
    <AgGridVue
      v-else
      class="min-h-0 flex-1"
      :row-data="trainings"
      :column-defs="columnDefs"
      :default-col-def="defaultColDef"
      :quick-filter-text="query"
      row-class="cursor-pointer"
      @row-clicked="openTraining"
      :loading="loading"
      overlay-no-rows-template="No trainings yet."
    />
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { AgGridVue } from 'ag-grid-vue3'
import { AllCommunityModule, ModuleRegistry, type ColDef, type RowClickedEvent } from 'ag-grid-community'
import { useRouter } from 'vue-router'
import { toast } from 'vue-sonner'
import { listTrainings } from '../../api'
import AppButton from '../../components/button/AppButton.vue'
import ActiveCell from '../../components/grid/ActiveCell.vue'
import type { Training } from '../../types'

// AG Grid only includes the features registered here: all free (community) ones.
ModuleRegistry.registerModules([AllCommunityModule])

const router = useRouter()
const trainings = ref<Training[]>([])
const query = ref('')
const loading = ref(true)
const error = ref('')

const columnDefs: ColDef<Training>[] = [
  {
    colId: 'client',
    headerName: 'Client',
    flex: 2,
    minWidth: 160,
    // "Last, First" so the column sorts by last name.
    valueGetter: (p) => (p.data ? `${p.data.client_last_name}, ${p.data.client_first_name}` : ''),
  },
  { field: 'name', headerName: 'Name', flex: 2, minWidth: 160 },
  {
    field: 'active',
    headerName: 'Status',
    flex: 1,
    minWidth: 100,
    cellRenderer: ActiveCell,
    // Filter and search by the label rather than true/false.
    valueFormatter: (p) => (p.value ? 'Active' : 'Inactive'),
    getQuickFilterText: (p) => (p.value ? 'Active' : ''), // so searching "active" skips inactive ones
  },
  { field: 'description', headerName: 'Description', flex: 3, minWidth: 200 },
  { field: 'unit_count', headerName: 'Units', flex: 1, minWidth: 90 },
  { field: 'exercise_count', headerName: 'Exercises', flex: 1, minWidth: 110 },
  {
    field: 'updated_at',
    headerName: 'Updated',
    flex: 1,
    minWidth: 120,
    valueFormatter: (p) => (p.value ? new Date(p.value as string).toLocaleDateString() : ''),
  },
]
const defaultColDef: ColDef = { sortable: true, filter: true, resizable: true }

function openTraining(event: RowClickedEvent<Training>) {
  if (event.data) router.push({ name: 'training-edit', params: { id: event.data.id } })
}

onMounted(async () => {
  try {
    trainings.value = await listTrainings()
  } catch {
    error.value = 'Could not load trainings.'
    toast.error(error.value)
  } finally {
    loading.value = false
  }
})
</script>
