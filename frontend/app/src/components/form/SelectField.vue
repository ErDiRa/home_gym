<template>
  <!-- Searchable: a combobox to type into. Otherwise a listbox, which opens no keyboard on phones. -->
  <Combobox v-if="searchable" v-model="model" immediate nullable :disabled="disabled" as="div">
    <ComboboxLabel v-if="label" :class="labelClass">
      {{ label }}<span v-if="required" class="ml-0.5 text-red-600" aria-hidden="true">*</span>
    </ComboboxLabel>
    <div ref="field" class="relative">
      <ComboboxInput
        :display-value="(id) => nameOf(id as string | null)"
        :placeholder="placeholder"
        :aria-label="!label ? ariaLabel : undefined"
        :class="[fieldClasses, 'outline-none placeholder:text-slate-400']"
        @change="query = $event.target.value"
        @focus="place"
      />
      <ComboboxButton :class="chevronClasses" @click="place">
        <ChevronUpDownIcon :class="chevronSize" aria-hidden="true" />
      </ComboboxButton>
      <SelectOptions :options="filtered" :open-up="openUp" :size="optionSize ?? size" @closed="query = ''" />
    </div>
  </Combobox>

  <Listbox v-else v-model="model" :disabled="disabled" as="div">
    <ListboxLabel v-if="label" :class="labelClass">
      {{ label }}<span v-if="required" class="ml-0.5 text-red-600" aria-hidden="true">*</span>
    </ListboxLabel>
    <div ref="field" class="relative">
      <ListboxButton :aria-label="!label ? ariaLabel : undefined" :class="[fieldClasses, 'text-left']" @click="place">
        <span :class="{ 'text-slate-400': model == null }" class="block truncate">{{ nameOf(model) || placeholder }}</span>
        <span :class="chevronClasses">
          <ChevronUpDownIcon :class="chevronSize" aria-hidden="true" />
        </span>
      </ListboxButton>
      <SelectOptions :options="options" :open-up="openUp" :size="optionSize ?? size" panel-class="min-w-32" listbox />
    </div>
  </Listbox>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import {
  Combobox,
  ComboboxButton,
  ComboboxInput,
  ComboboxLabel,
  Listbox,
  ListboxButton,
  ListboxLabel,
} from '@headlessui/vue'
import { ChevronUpDownIcon } from '@heroicons/vue/16/solid'
import SelectOptions from './SelectOptions.vue'
import { useDropPlacement } from './useDropPlacement'
import type { Option } from '../../types'

export type FieldSize = 'sm' | 'md' | 'touch'

// A select holding one option's id: searchable (a combobox, the default) or a plain listbox for
// short lists, e.g. in the tracker. touch is the tracker's 44px size with text-base.
const props = withDefaults(
  defineProps<{
    options: Option[]
    label?: string
    /** Accessible name when there is no visible label. */
    ariaLabel?: string
    placeholder?: string
    required?: boolean
    disabled?: boolean
    searchable?: boolean
    size?: FieldSize
    /** Size of the option rows, if not the field's (e.g. a small field with touch rows). */
    optionSize?: FieldSize
    /** Extra classes for the field, e.g. a large heading font. */
    fieldClass?: string
  }>(),
  { label: '', ariaLabel: undefined, placeholder: 'Select…', searchable: true, size: 'md', optionSize: undefined, fieldClass: '' },
)
const model = defineModel<string | null>({ required: true })

const query = ref('')
const filtered = computed(() => {
  const q = query.value.trim().toLowerCase()
  return q ? props.options.filter((o) => o.name.toLowerCase().includes(q)) : props.options
})
const nameOf = (id: string | null) => props.options.find((o) => o.id === id)?.name ?? ''

const { field, openUp, place } = useDropPlacement()

const labelClass = 'mb-1 block text-sm font-medium text-slate-800'
const sizeClasses: Record<FieldSize, string> = {
  sm: 'h-8 rounded-md pr-6 pl-2 text-sm',
  md: 'h-10 rounded-lg pr-8 pl-3 text-sm',
  touch: 'h-11 rounded-xl pr-10 pl-3 text-base',
}
const fieldClasses = computed(() => [
  'relative block w-full border border-slate-300 bg-white text-slate-900',
  'focus:border-indigo-500 focus:ring-2 focus:ring-indigo-200 focus:outline-none disabled:bg-slate-50 disabled:text-slate-500',
  sizeClasses[props.size],
  props.fieldClass,
])
const chevronClasses = computed(() => [
  'pointer-events-auto absolute inset-y-0 right-0 flex items-center',
  props.size === 'sm' ? 'pr-1' : props.size === 'md' ? 'pr-2' : 'pr-3',
])
const chevronSize = computed(() => (props.size === 'touch' ? 'size-5 text-slate-500' : 'size-4 text-slate-400'))
</script>
