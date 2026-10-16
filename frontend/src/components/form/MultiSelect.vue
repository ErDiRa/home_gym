<template>
  <Listbox v-model="model" multiple :disabled="disabled" as="div">
    <ListboxLabel class="mb-1 block text-sm font-medium text-slate-800">
      {{ label }}<span v-if="required" class="ml-0.5 text-red-600" aria-hidden="true">*</span>
    </ListboxLabel>

    <div class="relative">
      <ListboxButton
        ref="button"
        class="flex min-h-10 w-full items-center justify-between gap-2 rounded-lg border border-slate-300 bg-white px-3 py-2 text-left text-sm text-slate-900 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-200 focus:outline-none disabled:bg-slate-50 disabled:text-slate-500"
        @click="placeOptions"
      >
        <span class="truncate" :class="{ 'text-slate-400': !selectedNames }">
          {{ selectedNames || placeholder }}
        </span>
        <ChevronUpDownIcon class="size-4 shrink-0 text-slate-400" aria-hidden="true" />
      </ListboxButton>

      <Transition leave-active-class="transition-opacity duration-100" leave-to-class="opacity-0">
        <ListboxOptions
          :class="openUp ? 'bottom-full mb-1' : 'top-full mt-1'"
          class="absolute z-20 max-h-60 w-full overflow-auto rounded-lg border border-slate-200 bg-white py-1 text-sm shadow-lg focus:outline-none"
        >
          <ListboxOption
            v-for="option in options"
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
          </ListboxOption>
        </ListboxOptions>
      </Transition>
    </div>
  </Listbox>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { Listbox, ListboxButton, ListboxLabel, ListboxOption, ListboxOptions } from '@headlessui/vue'
import { CheckIcon, ChevronUpDownIcon } from '@heroicons/vue/16/solid'
import type { Option } from '../../types'

const props = withDefaults(
  defineProps<{ label: string; options: Option[]; placeholder?: string; required?: boolean; disabled?: boolean }>(),
  { placeholder: 'Select…' },
)
// Ids of the selected options.
const model = defineModel<string[]>({ required: true })

// The list is at most max-h-60 (15rem) tall. If it doesn't fit below the button but there is
// more room above it, open upwards instead (e.g. fields near the bottom of the page).
const OPTIONS_HEIGHT = 240
const button = ref<{ el: HTMLElement | null } | null>(null)
const openUp = ref(false)

function placeOptions() {
  const rect = button.value?.el?.getBoundingClientRect()
  if (!rect) return
  const below = window.innerHeight - rect.bottom
  openUp.value = below < OPTIONS_HEIGHT && rect.top > below
}

const selectedNames = computed(() =>
  props.options
    .filter((o) => model.value.includes(o.id))
    .map((o) => o.name)
    .join(', '),
)
</script>
