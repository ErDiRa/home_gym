<template>
  <div class="flex h-list-page flex-col lg:h-[calc(100dvh-4rem)]">
    <div class="mb-2 flex flex-wrap items-center justify-between gap-x-4 gap-y-2">
      <h1 class="text-xl font-bold text-slate-900">Saved blocks</h1>
      <div class="flex items-center gap-x-4 gap-y-2">
        <span class="hidden text-sm text-slate-500 sm:inline">{{ templates.length }} blocks</span>
        <AppButton :to="{ name: 'block-new' }">New block</AppButton>
      </div>
    </div>
    <p class="mb-6 text-sm text-slate-500">
      Combinations of exercises you use often, e.g. a complex. "Add saved block" in a session copies one; its sets, reps and load are
      planned there. Save a block from any session with its bookmark button.
    </p>
    <TextInput v-model="query" type="search" placeholder="Search blocks or exercises..." class="mb-4" />

    <p v-if="error" class="text-center text-red-600">{{ error }}</p>
    <AgGridVue
      v-else
      class="min-h-0 flex-1"
      :row-data="shown"
      :column-defs="columnDefs"
      :row-height="phoneRowHeight(phone)"
      :default-col-def="defaultColDef"
      row-class="cursor-pointer"
      @row-clicked="openTemplate"
      :loading="loading"
      overlay-no-rows-template="No saved blocks yet."
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
import { useBlockTemplatesStore } from '../../stores/blockTemplates'
import { useLookupsStore } from '../../stores/lookups'
import type { BlockTemplate } from '../../types'
import { usePhone } from '../../usePhone'

// AG Grid only includes the features registered here: all free (community) ones.
ModuleRegistry.registerModules([AllCommunityModule])

const router = useRouter()
const store = useBlockTemplatesStore()
const lookups = useLookupsStore()
const { templates, loading } = storeToRefs(store)
const query = ref('')
const error = ref('')

const exerciseNames = (t: BlockTemplate) => t.exercises.map((e) => e.name).join(', ')
const searchTemplates = fuzzySearch<BlockTemplate>([
  { name: 'name', weight: 2, get: (t) => t.name },
  { name: 'exercises', weight: 1, get: (t) => t.exercises.map((e) => e.name) },
])
const shown = computed(() => searchTemplates(templates.value, query.value))

const columns: ColDef<BlockTemplate>[] = [
  { field: 'name', headerName: 'Name', flex: 2, minWidth: 180 },
  { field: 'kind', headerName: 'Kind', flex: 1, minWidth: 110, valueFormatter: (p) => lookups.kindOf(p.value as string)?.name ?? '' },
  { colId: 'exercises', headerName: 'Exercises', flex: 3, minWidth: 220, valueGetter: (p) => (p.data ? exerciseNames(p.data) : '') },
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
const columnDefs = computed(() => phoneColumns(columns, phone.value, ['name', 'exercises']))

function openTemplate(event: RowClickedEvent<BlockTemplate>) {
  if (event.data) router.push({ name: 'block-edit', params: { id: event.data.id } })
}

onMounted(async () => {
  try {
    await Promise.all([store.loadTemplates(), lookups.loadTrainingLookups()])
  } catch {
    error.value = 'Could not load the saved blocks.'
    toast.error(error.value)
  }
})
</script>
