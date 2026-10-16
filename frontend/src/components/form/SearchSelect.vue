<template>
  <Combobox :model-value="null" immediate :disabled="disabled" as="div" @update:model-value="pick">
    <ComboboxLabel v-if="label" class="mb-1 block text-sm font-medium text-slate-800">{{ label }}</ComboboxLabel>

    <div ref="field" class="relative">
      <ComboboxInput
        :display-value="() => query"
        :placeholder="placeholder"
        class="block h-10 w-full rounded-lg border border-slate-300 bg-white py-2 pr-8 pl-3 text-sm text-slate-900 outline-none placeholder:text-slate-400 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-200 disabled:bg-slate-50"
        @change="query = $event.target.value"
        @focus="placeOptions"
      />
      <ComboboxButton class="absolute inset-y-0 right-0 flex items-center pr-2" @click="placeOptions">
        <MagnifyingGlassIcon class="size-4 text-slate-400" aria-hidden="true" />
      </ComboboxButton>

      <Transition leave-active-class="transition-opacity duration-100" leave-to-class="opacity-0" @after-leave="query = ''">
        <ComboboxOptions
          :class="openUp ? 'bottom-full mb-1' : 'top-full mt-1'"
          class="absolute z-20 max-h-60 w-full overflow-auto rounded-lg border border-slate-200 bg-white py-1 text-sm shadow-lg focus:outline-none"
        >
          <li v-if="!filtered.length" class="px-3 py-2 text-slate-500">No matches.</li>
          <ComboboxOption v-for="option in filtered" :key="option.id" v-slot="{ active }" :value="option" as="template">
            <li
              :class="[
                'flex cursor-pointer items-center justify-between gap-2 px-3 py-2',
                active ? 'bg-indigo-50 text-indigo-900' : 'text-slate-800',
              ]"
            >
              <span class="truncate">{{ option.name }}</span>
              <slot name="option-extra" :option="option" />
            </li>
          </ComboboxOption>
        </ComboboxOptions>
      </Transition>
    </div>
  </Combobox>
</template>

<script setup lang="ts" generic="T extends Option">
import { computed, ref } from 'vue'
import {
  Combobox,
  ComboboxButton,
  ComboboxInput,
  ComboboxLabel,
  ComboboxOption,
  ComboboxOptions,
} from '@headlessui/vue'
import { MagnifyingGlassIcon } from '@heroicons/vue/16/solid'
import type { Option } from '../../types'

// A search field that emits the picked option and then clears itself, for adding items to a
// list (e.g. exercises to a training) rather than holding a value.
const props = withDefaults(
  defineProps<{ options: T[]; label?: string; placeholder?: string; disabled?: boolean }>(),
  { label: '', placeholder: 'Search…' },
)
const emit = defineEmits<{ select: [option: T] }>()

const query = ref('')
const filtered = computed(() => {
  const q = query.value.trim().toLowerCase()
  return q ? props.options.filter((o) => o.name.toLowerCase().includes(q)) : props.options
})

function pick(option: T | null) {
  if (option) emit('select', option)
  query.value = ''
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
