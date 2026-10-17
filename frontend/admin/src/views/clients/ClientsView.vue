<template>
  <div class="flex h-[calc(100vh-8rem)] flex-col lg:h-[calc(100vh-4rem)]">
    <div class="mb-6 flex items-center justify-between gap-4">
      <h1 class="text-xl font-bold text-slate-900">Clients</h1>
      <div class="flex items-center gap-4">
        <span class="text-sm text-slate-500">{{ clients.length }} clients</span>
        <AppButton :to="{ name: 'client-new' }">New client</AppButton>
      </div>
    </div>
    <input
      v-model="query"
      type="search"
      placeholder="Search clients..."
      class="mb-4 w-full rounded-lg border border-slate-300 bg-white px-4 py-2 outline-none focus:border-indigo-500 focus:ring-2 focus:ring-indigo-200"
    />

    <p v-if="error" class="text-center text-red-600">{{ error }}</p>
    <AgGridVue
      v-else
      class="min-h-0 flex-1"
      :row-data="clients"
      :column-defs="columnDefs"
      :default-col-def="defaultColDef"
      :quick-filter-text="query"
      row-class="cursor-pointer"
      @row-clicked="openClient"
      :loading="loading"
      overlay-no-rows-template="No clients yet."
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
import { useClientsStore } from '../../stores/clients'
import type { Client } from '../../types'

// AG Grid only includes the features registered here: all free (community) ones.
ModuleRegistry.registerModules([AllCommunityModule])

const router = useRouter()
const store = useClientsStore()
const { clients, loading } = storeToRefs(store)
const query = ref('')
const error = ref('')

const columnDefs: ColDef<Client>[] = [
  { field: 'last_name', headerName: 'Last name', flex: 1, minWidth: 130 },
  { field: 'first_name', headerName: 'First name', flex: 1, minWidth: 130 },
  { field: 'email', headerName: 'Email', flex: 2, minWidth: 200 },
  { field: 'active_training', headerName: 'Active training', flex: 2, minWidth: 160 },
  { field: 'training_count', headerName: 'Trainings', flex: 1, minWidth: 110 },
  {
    field: 'created_at',
    headerName: 'Created',
    flex: 1,
    minWidth: 120,
    valueFormatter: (p) => (p.value ? new Date(p.value as string).toLocaleDateString() : ''),
  },
]
const defaultColDef: ColDef = { sortable: true, filter: true, resizable: true }

function openClient(event: RowClickedEvent<Client>) {
  if (event.data) router.push({ name: 'client-edit', params: { id: event.data.id } })
}

onMounted(async () => {
  try {
    await store.loadClients()
  } catch {
    error.value = 'Could not load clients.'
    toast.error(error.value)
  }
})
</script>
