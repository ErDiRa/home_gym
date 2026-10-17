<template>
  <Combobox :model-value="model" immediate nullable :disabled="creating" as="div" @update:model-value="pick">
    <div ref="field" class="relative">
      <ComboboxInput
        :display-value="(id) => units.find((u) => u.id === id)?.name ?? ''"
        :aria-label="label"
        placeholder="Unit"
        class="block h-8 w-full rounded-md border border-slate-300 bg-white py-1 pr-6 pl-2 text-sm text-slate-900 outline-none placeholder:text-slate-400 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-200"
        @change="query = $event.target.value"
        @focus="placeOptions"
      />
      <ComboboxButton class="absolute inset-y-0 right-0 flex items-center pr-1" @click="placeOptions">
        <ChevronUpDownIcon class="size-4 text-slate-400" aria-hidden="true" />
      </ComboboxButton>

      <!-- The query is reset once the list has closed, so it doesn't flash back to all units. -->
      <Transition leave-active-class="transition-opacity duration-100" leave-to-class="opacity-0" @after-leave="query = ''">
        <ComboboxOptions
          :class="openUp ? 'bottom-full mb-1' : 'top-full mt-1'"
          class="absolute z-20 max-h-60 w-40 overflow-auto rounded-lg border border-slate-200 bg-white py-1 text-sm shadow-lg focus:outline-none"
        >
          <ComboboxOption v-for="u in filtered" :key="u.id" v-slot="{ active, selected }" :value="u.id" as="template">
            <li
              :class="[
                'flex cursor-pointer items-center justify-between gap-2 px-3 py-1.5',
                active ? 'bg-indigo-50 text-indigo-900' : 'text-slate-800',
              ]"
            >
              <span :class="{ 'font-medium': selected }" class="truncate">{{ u.name }}</span>
              <CheckIcon v-if="selected" class="size-4 shrink-0 text-indigo-600" aria-hidden="true" />
            </li>
          </ComboboxOption>
          <ComboboxOption v-if="canCreate" v-slot="{ active }" :value="CREATE" as="template">
            <li
              :class="[
                'flex cursor-pointer items-center gap-1 px-3 py-1.5',
                active ? 'bg-indigo-50 text-indigo-900' : 'text-indigo-700',
              ]"
            >
              <PlusIcon class="size-4 shrink-0" aria-hidden="true" />
              <span class="truncate">Add "{{ query.trim() }}"</span>
            </li>
          </ComboboxOption>
          <li v-if="!filtered.length && !canCreate" class="px-3 py-1.5 text-slate-500">Type to add a unit.</li>
        </ComboboxOptions>
      </Transition>
    </div>
  </Combobox>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { Combobox, ComboboxButton, ComboboxInput, ComboboxOption, ComboboxOptions } from '@headlessui/vue'
import { CheckIcon, ChevronUpDownIcon, PlusIcon } from '@heroicons/vue/16/solid'
import { toast } from 'vue-sonner'
import { ApiError } from '../../api'
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

// Open upwards when the list (max-h-60, 15rem) doesn't fit below the field.
const OPTIONS_HEIGHT = 240
const field = ref<HTMLElement | null>(null)
const openUp = ref(false)

function placeOptions() {
  const rect = field.value?.getBoundingClientRect()
  if (!rect) return
  const below = window.innerHeight - rect.bottom
  openUp.value = below < OPTIONS_HEIGHT && rect.top > below
}
</script>
