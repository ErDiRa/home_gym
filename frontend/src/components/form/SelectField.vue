<template>
  <Combobox v-model="model" immediate nullable :disabled="disabled" as="div">
    <ComboboxLabel v-if="label" class="mb-1 block text-sm font-medium text-slate-800">
      {{ label }}<span v-if="required" class="ml-0.5 text-red-600" aria-hidden="true">*</span>
    </ComboboxLabel>

    <div ref="field" class="relative">
      <ComboboxInput
        :display-value="(id) => options.find((o) => o.id === id)?.name ?? ''"
        :placeholder="placeholder"
        class="block h-10 w-full rounded-lg border border-slate-300 bg-white py-2 pr-8 pl-3 text-sm text-slate-900 outline-none placeholder:text-slate-400 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-200 disabled:bg-slate-50 disabled:text-slate-500"
        @change="query = $event.target.value"
        @focus="placeOptions"
      />
      <ComboboxButton class="absolute inset-y-0 right-0 flex items-center pr-2" @click="placeOptions">
        <ChevronUpDownIcon class="size-4 text-slate-400" aria-hidden="true" />
      </ComboboxButton>

      <Transition leave-active-class="transition-opacity duration-100" leave-to-class="opacity-0" @after-leave="query = ''">
        <ComboboxOptions
          :class="openUp ? 'bottom-full mb-1' : 'top-full mt-1'"
          class="absolute z-20 max-h-60 w-full overflow-auto rounded-lg border border-slate-200 bg-white py-1 text-sm shadow-lg focus:outline-none"
        >
          <li v-if="!filtered.length" class="px-3 py-2 text-slate-500">No matches.</li>
          <ComboboxOption
            v-for="option in filtered"
            :key="option.id"
            v-slot="{ active, selected }"
            :value="option.id"
            as="template"
          >
            <li
              :class="[
                'flex cursor-pointer items-center justify-between gap-2 px-3 py-2',
                active ? 'bg-indigo-50 text-indigo-900' : 'text-slate-800',
              ]"
            >
              <span :class="{ 'font-medium': selected }" class="truncate">{{ option.name }}</span>
              <CheckIcon v-if="selected" class="size-4 shrink-0 text-indigo-600" aria-hidden="true" />
            </li>
          </ComboboxOption>
        </ComboboxOptions>
      </Transition>
    </div>
  </Combobox>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import {
  Combobox,
  ComboboxButton,
  ComboboxInput,
  ComboboxLabel,
  ComboboxOption,
  ComboboxOptions,
} from '@headlessui/vue'
import { CheckIcon, ChevronUpDownIcon } from '@heroicons/vue/16/solid'
import type { Option } from '../../types'

// A searchable select holding one option's id.
const props = withDefaults(
  defineProps<{ options: Option[]; label?: string; placeholder?: string; required?: boolean; disabled?: boolean }>(),
  { label: '', placeholder: 'Select…' },
)
const model = defineModel<string | null>({ required: true })

const query = ref('')
const filtered = computed(() => {
  const q = query.value.trim().toLowerCase()
  return q ? props.options.filter((o) => o.name.toLowerCase().includes(q)) : props.options
})

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
