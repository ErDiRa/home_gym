<template>
  <Listbox v-model="model" as="div" multiple :disabled="disabled">
    <ListboxLabel class="block text-sm/6 font-medium text-slate-900">{{ label }}<span v-if="required" class="text-red-600" aria-hidden="true"> *</span></ListboxLabel>
    <div class="relative mt-1">
      <ListboxButton
        ref="button"
        @click="placeOptions"
        class="grid w-full cursor-default grid-cols-1 rounded-md bg-white py-1.5 pr-2 pl-3 text-left text-slate-900 outline-1 -outline-offset-1 outline-slate-300 disabled:bg-slate-50 disabled:text-slate-500 focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-indigo-600 sm:text-sm/6"
      >
        <span class="col-start-1 row-start-1 truncate pr-6" :class="{ 'text-slate-400': !selectedNames }">
          {{ selectedNames || placeholder }}
        </span>
        <ChevronUpDownIcon
          class="col-start-1 row-start-1 size-5 self-center justify-self-end text-slate-500 sm:size-4"
          aria-hidden="true"
        />
      </ListboxButton>

      <transition leave-active-class="transition ease-in duration-100" leave-to-class="opacity-0">
        <ListboxOptions
          :class="openUp ? 'bottom-full mb-1' : 'mt-1'"
          class="absolute z-10 max-h-60 w-full overflow-auto rounded-md bg-white py-1 text-base shadow-lg outline-1 outline-black/5 sm:text-sm"
        >
          <ListboxOption
            v-for="option in options"
            :key="option.id"
            v-slot="{ active, selected }"
            as="template"
            :value="option.id"
          >
            <li
              :class="[
                active ? 'bg-indigo-600 text-white outline-hidden' : 'text-slate-900',
                'relative cursor-default py-2 pr-4 pl-8 select-none',
              ]"
            >
              <span :class="[selected ? 'font-semibold' : 'font-normal', 'block truncate']">
                {{ option.name }}
              </span>
              <span
                v-if="selected"
                :class="[
                  active ? 'text-white' : 'text-indigo-600',
                  'absolute inset-y-0 left-0 flex items-center pl-1.5',
                ]"
              >
                <CheckIcon class="size-5" aria-hidden="true" />
              </span>
            </li>
          </ListboxOption>
        </ListboxOptions>
      </transition>
    </div>
  </Listbox>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { Listbox, ListboxButton, ListboxLabel, ListboxOption, ListboxOptions } from '@headlessui/vue'
import { ChevronUpDownIcon } from '@heroicons/vue/16/solid'
import { CheckIcon } from '@heroicons/vue/20/solid'
import type { Option } from '../../types'

const props = withDefaults(defineProps<{ label: string; options: Option[]; placeholder?: string; required?: boolean; disabled?: boolean }>(), {
  placeholder: 'Select…',
})
// The options are at most max-h-60 (15rem) tall. If they don't fit below the button but there
// is more room above it, open upwards instead (e.g. fields near the bottom of the page).
const OPTIONS_HEIGHT = 240
const button = ref<{ el: HTMLElement | null } | null>(null)
const openUp = ref(false)

function placeOptions() {
  const rect = button.value?.el?.getBoundingClientRect()
  if (!rect) return
  const below = window.innerHeight - rect.bottom
  openUp.value = below < OPTIONS_HEIGHT && rect.top > below
}

// Ids of the selected options.
const model = defineModel<string[]>({ required: true })

const selectedNames = computed(() =>
  props.options
    .filter((o) => model.value.includes(o.id))
    .map((o) => o.name)
    .join(', '),
)
</script>
