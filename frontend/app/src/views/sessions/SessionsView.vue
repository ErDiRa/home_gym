<template>
  <div class="flex h-list-page flex-col lg:h-[calc(100dvh-4rem)]">
    <div class="mb-2 flex flex-wrap items-center justify-between gap-x-4 gap-y-2">
      <h1 class="text-xl font-bold text-slate-900">Sessions</h1>
      <div class="flex items-center gap-x-4 gap-y-2">
        <span class="hidden text-sm text-slate-500 sm:inline">{{ sessions.length }} sessions</span>
        <AppButton :to="{ name: 'session-new' }">New session</AppButton>
      </div>
    </div>
    <p class="mb-6 text-sm text-slate-500">
      Your library of session templates. Adding one to a training copies it, so the training keeps its version when you edit the
      template.
    </p>
    <TextInput
      v-model="query"
      type="search"
      placeholder="Search sessions..."
      class="mb-4"
    />

    <p v-if="error" class="text-center text-red-600">{{ error }}</p>
    <AgGridVue
      v-else
      class="min-h-0 flex-1"
      :row-data="shown"
      :column-defs="columnDefs"
      :row-height="phoneRowHeight(phone)"
      :default-col-def="defaultColDef"
      row-class="cursor-pointer"
      @row-clicked="openSession"
      :loading="loading"
      overlay-no-rows-template="No sessions yet."
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
import { phoneColumns, phoneRowHeight } from '../../components/grid/phoneColumns'
import TextInput from '../../components/form/TextInput.vue'
import { formatDay } from '../../dates'
import { fuzzySearch } from '../../fuzzySearch'
import { useSessionsStore } from '../../stores/sessions'
import type { LibrarySession } from '../../types'
import { usePhone } from '../../usePhone'

// AG Grid only includes the features registered here: all free (community) ones.
ModuleRegistry.registerModules([AllCommunityModule])

const router = useRouter()
const store = useSessionsStore()
const { sessions, loading } = storeToRefs(store)
const query = ref('')
const error = ref('')

const searchSessions = fuzzySearch<LibrarySession>([{ name: 'name', weight: 1, get: (s) => s.name }])
const shown = computed(() => searchSessions(sessions.value, query.value))

const columns: ColDef<LibrarySession>[] = [
  { field: 'name', headerName: 'Name', flex: 2, minWidth: 180 },
  { field: 'notes', headerName: 'Notes', flex: 3, minWidth: 200 },
  { field: 'block_count', headerName: 'Blocks', flex: 1, minWidth: 90 },
  { field: 'exercise_count', headerName: 'Exercises', flex: 1, minWidth: 110 },
  {
    field: 'updated_at',
    headerName: 'Updated',
    flex: 1,
    minWidth: 120,
    valueFormatter: (p) => (p.value ? formatDay(p.value as string, { weekday: false }) : ''),
  },
]
const defaultColDef: ColDef = { sortable: true, filter: true, resizable: true }
// A phone shows only the columns that tell the rows apart.
const phone = usePhone()
const columnDefs = computed(() => phoneColumns(columns, phone.value, ['name', 'exercise_count']))

function openSession(event: RowClickedEvent<LibrarySession>) {
  if (event.data) router.push({ name: 'session-edit', params: { id: event.data.id } })
}

onMounted(async () => {
  try {
    await store.loadSessions()
  } catch {
    error.value = 'Could not load sessions.'
    toast.error(error.value)
  }
})
</script>
