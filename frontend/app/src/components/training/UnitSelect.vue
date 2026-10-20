<template>
  <Combobox :model-value="model" immediate nullable :disabled="creating" as="div" @update:model-value="pick">
    <div ref="field" class="relative">
      <ComboboxInput
        :display-value="(id) => units.find((u) => u.id === id)?.name ?? ''"
        :aria-label="label"
        placeholder="Unit"
        class="block h-8 w-full rounded-md border border-slate-300 bg-white py-1 pr-6 pl-2 text-sm text-slate-900 outline-none placeholder:text-slate-400 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-200"
        @change="query = $event.target.value"
        @focus="place"
      />
      <ComboboxButton class="absolute inset-y-0 right-0 flex items-center pr-1" @click="place">
        <ChevronUpDownIcon class="size-4 text-slate-400" aria-hidden="true" />
      </ComboboxButton>

      <SelectOptions
        v-slot="{ rowClass }"
        :options="filtered"
        :open-up="openUp"
        size="sm"
        panel-class="w-40"
        :empty="canCreate ? '' : 'Type to add a unit.'"
        @closed="query = ''"
      >
        <ComboboxOption v-if="canCreate" v-slot="{ active }" :value="CREATE" as="template">
          <li
            :class="[
              rowClass,
              'flex cursor-pointer items-center gap-1',
              active ? 'bg-indigo-50 text-indigo-900' : 'text-indigo-700',
            ]"
          >
            <PlusIcon class="size-4 shrink-0" aria-hidden="true" />
            <span class="truncate">Add "{{ query.trim() }}"</span>
          </li>
        </ComboboxOption>
      </SelectOptions>
    </div>
  </Combobox>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { Combobox, ComboboxButton, ComboboxInput, ComboboxOption } from '@headlessui/vue'
import { ChevronUpDownIcon, PlusIcon } from '@heroicons/vue/16/solid'
import { toast } from 'vue-sonner'
import { ApiError } from '../../api'
import SelectOptions from '../form/SelectOptions.vue'
import { useDropPlacement } from '../form/useDropPlacement'
import { useLookupsStore, type AmountPart } from '../../stores/lookups'

// A compact searchable select of rep or load units holding the unit's id. Typing a name that
// doesn't exist offers to add it as a new unit.
const props = defineProps<{ part: AmountPart; label: string }>()
const model = defineModel<string | null>({ required: true })
const lookups = useLookupsStore()

// Value of the "add" option; never a unit id.
const CREATE = '__create__'

const units = computed(() => lookups.unitsOf(props.part))
const query = ref('')
const creating = ref(false)
const filtered = computed(() => {
  const q = query.value.trim().toLowerCase()
  return q ? units.value.filter((u) => u.name.toLowerCase().includes(q)) : units.value
})
// Offer "Add" only for a name that doesn't exist yet (case-insensitive, like the server's index).
const canCreate = computed(() => {
  const q = query.value.trim().toLowerCase()
  return q !== '' && !units.value.some((u) => u.name.toLowerCase() === q)
})

async function pick(value: string | null) {
  if (value !== CREATE) {
    model.value = value
    return
  }
  const name = query.value.trim()
  creating.value = true
  try {
    const unit = await lookups.addUnit(props.part, name)
    model.value = unit.id
    toast.success(`Unit "${unit.name}" added.`)
  } catch (e) {
    toast.error(e instanceof ApiError && [400, 409].includes(e.status) ? e.message : 'Could not add the unit.')
  } finally {
    creating.value = false
  }
}

// The list opens above the field when it doesn't fit below.
const { field, openUp, place } = useDropPlacement()
</script>
