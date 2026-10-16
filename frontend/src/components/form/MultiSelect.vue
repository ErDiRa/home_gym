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
        <span
          v-for="option in selected"
          :key="option.id"
          class="inline-flex items-center gap-0.5 rounded-md bg-indigo-50 py-0.5 pr-1 pl-2 text-xs font-medium text-indigo-700"
          :class="{ 'pr-2': disabled }"
        >
          {{ option.name }}
          <button
            v-if="!disabled"
            type="button"
            :aria-label="`Remove ${option.name}`"
            class="rounded p-0.5 hover:bg-indigo-100"
            @click.stop="remove(option.id)"
          >
            <XMarkIcon class="size-3" aria-hidden="true" />
          </button>
        </span>
        <ComboboxInput
          v-if="!disabled"
          ref="input"
          :display-value="() => query"
          :placeholder="selected.length ? 'Search…' : placeholder"
          class="min-w-24 flex-1 border-0 bg-transparent px-1 py-0.5 text-slate-900 outline-none placeholder:text-slate-400"
          @change="query = $event.target.value"
          @focus="placeOptions"
          @keydown.backspace="removeLastOnEmpty"
        />
        <span v-else-if="!selected.length" class="px-1 py-0.5 text-slate-400">{{ placeholder }}</span>
      </div>
      <ComboboxButton
        v-if="!disabled"
        class="absolute inset-y-0 right-0 flex items-center pr-2"
        @click="placeOptions"
      >
        <ChevronUpDownIcon class="size-4 text-slate-400" aria-hidden="true" />
      </ComboboxButton>

      <Transition
        leave-active-class="transition-opacity duration-100"
        leave-to-class="opacity-0"
        @after-leave="query = ''"
      >
        <ComboboxOptions
          :class="openUp ? 'bottom-full mb-1' : 'top-full mt-1'"
          class="absolute z-20 max-h-60 w-full overflow-auto rounded-lg border border-slate-200 bg-white py-1 text-sm shadow-lg focus:outline-none"
        >
          <li v-if="!filtered.length" class="px-3 py-2 text-slate-500">No matches.</li>
          <ComboboxOption
            v-for="option in filtered"
            :key="option.id"
            v-slot="{ active, selected: isSelected }"
            :value="option.id"
            as="template"
          >
            <li
              :class="[
                'flex cursor-pointer items-center justify-between gap-2 px-3 py-2',
                active ? 'bg-indigo-50 text-indigo-900' : 'text-slate-800',
              ]"
            >
              <span :class="{ 'font-medium': isSelected }" class="truncate">{{ option.name }}</span>
              <CheckIcon v-if="isSelected" class="size-4 shrink-0 text-indigo-600" aria-hidden="true" />
            </li>
          </ComboboxOption>
        </ComboboxOptions>
      </Transition>
    </div>
  </Combobox>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import {
  Combobox,
  ComboboxButton,
  ComboboxInput,
  ComboboxLabel,
  ComboboxOption,
  ComboboxOptions,
} from '@headlessui/vue'
import { CheckIcon, ChevronUpDownIcon, XMarkIcon } from '@heroicons/vue/16/solid'
import type { Option } from '../../types'

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

// The list is at most max-h-60 (15rem) tall. If it doesn't fit below the field but there is
// more room above it, open upwards instead (e.g. fields near the bottom of the page).
const OPTIONS_HEIGHT = 240
const field = ref<HTMLElement | null>(null)
const input = ref<{ $el: HTMLInputElement | null } | null>(null)
const openUp = ref(false)

function placeOptions() {
  const rect = field.value?.getBoundingClientRect()
  if (!rect) return
  const below = window.innerHeight - rect.bottom
  openUp.value = below < OPTIONS_HEIGHT && rect.top > below
}
</script>
