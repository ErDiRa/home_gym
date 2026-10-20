<template>
  <Combobox v-model="model" multiple immediate :disabled="disabled" as="div">
    <ComboboxLabel class="mb-1 block text-sm font-medium text-slate-800">
      {{ label }}<span v-if="required" class="ml-0.5 text-red-600" aria-hidden="true">*</span>
    </ComboboxLabel>

    <div class="relative">
      <div
        ref="field"
        :class="disabled ? 'bg-slate-50' : 'cursor-text bg-white'"
        class="flex min-h-10 w-full flex-wrap items-center gap-1 rounded-lg border border-slate-300 py-1.5 pr-8 pl-2 text-sm focus-within:border-indigo-500 focus-within:ring-2 focus-within:ring-indigo-200"
        @click="input?.$el?.focus()"
      >
        <AppBadge
          v-for="option in selected"
          :key="option.id"
          color="indigo"
          :removable="!disabled"
          :remove-label="`Remove ${option.name}`"
          @remove="remove(option.id)"
        >
          {{ option.name }}
        </AppBadge>
        <ComboboxInput
          v-if="!disabled"
          ref="input"
          :display-value="() => query"
          :placeholder="selected.length ? 'Search…' : placeholder"
          class="min-w-24 flex-1 border-0 bg-transparent px-1 py-0.5 text-slate-900 outline-none placeholder:text-slate-400"
          @change="query = $event.target.value"
          @focus="place"
          @keydown.backspace="removeLastOnEmpty"
        />
        <span v-else-if="!selected.length" class="px-1 py-0.5 text-slate-400">{{ placeholder }}</span>
      </div>
      <ComboboxButton v-if="!disabled" class="absolute inset-y-0 right-0 flex items-center pr-2" @click="place">
        <ChevronUpDownIcon class="size-4 text-slate-400" aria-hidden="true" />
      </ComboboxButton>

      <SelectOptions :options="filtered" :open-up="openUp" @closed="query = ''" />
    </div>
  </Combobox>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { Combobox, ComboboxButton, ComboboxInput, ComboboxLabel } from '@headlessui/vue'
import { ChevronUpDownIcon } from '@heroicons/vue/16/solid'
import AppBadge from '../ui/AppBadge.vue'
import SelectOptions from './SelectOptions.vue'
import { useDropPlacement } from './useDropPlacement'
import type { Option } from '../../types'

// A searchable multi-select that shows the picks as removable badges inside the field.
const props = withDefaults(
  defineProps<{ label: string; options: Option[]; placeholder?: string; required?: boolean; disabled?: boolean }>(),
  { placeholder: 'Select…' },
)
// Ids of the selected options.
const model = defineModel<string[]>({ required: true })

// Text typed into the field; filters the options by name, case-insensitively.
const query = ref('')
const filtered = computed(() => {
  const q = query.value.trim().toLowerCase()
  return q ? props.options.filter((o) => o.name.toLowerCase().includes(q)) : props.options
})
// Start a fresh search after each pick, so several options can be found one after another.
watch(model, () => (query.value = ''))

const selected = computed(() => props.options.filter((o) => model.value.includes(o.id)))

function remove(id: string) {
  model.value = model.value.filter((v) => v !== id)
}

// Backspace in an empty search field removes the last selected option.
function removeLastOnEmpty() {
  if (query.value === '' && model.value.length) model.value = model.value.slice(0, -1)
}

// The list opens above the field when it doesn't fit below.
const { field, openUp, place } = useDropPlacement()
// ComboboxInput is a component; its $el is the <input>, focused when the field is clicked.
const input = ref<{ $el: HTMLInputElement | null } | null>(null)
</script>
