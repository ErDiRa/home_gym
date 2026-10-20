<template>
  <!-- The query of a searchable field is reset once the list has closed, so it doesn't flash back to all options. -->
  <Transition leave-active-class="transition-opacity duration-100" leave-to-class="opacity-0" @after-leave="emit('closed')">
    <component
      :is="listbox ? ListboxOptions : ComboboxOptions"
      :class="[
        openUp ? 'bottom-full mb-1' : 'top-full mt-1',
        size === 'touch' ? 'rounded-xl text-base' : 'rounded-lg text-sm',
        'absolute z-20 max-h-60 min-w-full overflow-auto border border-slate-200 bg-white py-1 shadow-lg focus:outline-none',
        panelClass,
      ]"
    >
      <component
        :is="listbox ? ListboxOption : ComboboxOption"
        v-for="option in options"
        :key="option.id"
        v-slot="{ active, selected }"
        :value="option.id"
        as="template"
      >
        <li
          :class="[
            rowClass,
            'flex cursor-pointer items-center justify-between gap-2',
            active ? 'bg-indigo-50 text-indigo-900' : 'text-slate-800',
          ]"
        >
          <span :class="{ 'font-medium': selected }" class="truncate">{{ option.name }}</span>
          <CheckIcon v-if="selected" class="size-4 shrink-0 text-indigo-600" aria-hidden="true" />
        </li>
      </component>
      <!-- Extra rows after the options, e.g. "Add …". -->
      <slot :row-class="rowClass" />
      <li v-if="!options.length && empty" :class="[rowClass, 'text-slate-500']">{{ empty }}</li>
    </component>
  </Transition>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { ComboboxOption, ComboboxOptions, ListboxOption, ListboxOptions } from '@headlessui/vue'
import { CheckIcon } from '@heroicons/vue/16/solid'
import type { Option } from '../../types'

// The option list of SelectField, MultiSelect and UnitSelect: must sit inside a Headless UI
// Combobox, or a Listbox when `listbox` is set.
const props = withDefaults(
  defineProps<{
    options: Option[]
    openUp: boolean
    listbox?: boolean
    size?: 'sm' | 'md' | 'touch'
    /** Shown when there are no options; empty string for nothing. */
    empty?: string
    panelClass?: string
  }>(),
  { listbox: false, size: 'md', empty: 'No matches.', panelClass: '' },
)
const emit = defineEmits<{ closed: [] }>()

// Touch rows are at least 44px tall.
const rowClass = computed(() => ({ sm: 'px-3 py-1.5', md: 'px-3 py-2', touch: 'min-h-11 px-4 py-2.5' })[props.size])
</script>
