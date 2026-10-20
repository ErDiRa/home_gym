<template>
  <!-- Classes style the wrapper; other attributes (aria-label, title…) go to the button. -->
  <div ref="field" :class="$attrs.class" class="relative inline-block">
    <Menu>
      <MenuButton
        :as="AppButton"
        v-bind="{ ...$attrs, class: undefined }"
        :variant="variant"
        :size="size"
        :disabled="disabled"
        @click="place"
      >
        <slot />
        <ChevronDownIcon class="-mr-0.5 size-4 opacity-70" aria-hidden="true" />
      </MenuButton>

      <Transition
        enter-active-class="transition duration-100 ease-out"
        enter-from-class="scale-95 opacity-0"
        leave-active-class="transition duration-75 ease-in"
        leave-to-class="scale-95 opacity-0"
      >
        <MenuItems
          :class="openUp ? 'bottom-full mb-1 origin-bottom-left' : 'top-full mt-1 origin-top-left'"
          class="absolute left-0 z-20 max-h-60 min-w-48 overflow-auto rounded-lg border border-slate-200 bg-white py-1 text-sm shadow-lg focus:outline-none"
        >
          <!-- Menu rows, styled like the options of the selects. -->
          <MenuItem v-for="item in items" :key="item.id" v-slot="{ active }">
            <button
              type="button"
              :class="[active ? 'bg-indigo-50 text-indigo-900' : 'text-slate-800', 'block w-full truncate px-3 py-2 text-left']"
              @click="emit('select', item.id)"
            >
              {{ item.name }}
            </button>
          </MenuItem>
          <p v-if="!items.length" class="px-3 py-2 text-slate-500">{{ empty }}</p>
        </MenuItems>
      </Transition>
    </Menu>
  </div>
</template>

<script setup lang="ts">
import { Menu, MenuButton, MenuItem, MenuItems } from '@headlessui/vue'
import { ChevronDownIcon } from '@heroicons/vue/16/solid'
import AppButton, { type ButtonSize, type ButtonVariant } from '../button/AppButton.vue'
import { useDropPlacement } from '../form/useDropPlacement'
import type { Option } from '../../types'

// A button that opens a list to pick one item from, e.g. "Add session ▾"; emits the item's id.
defineOptions({ inheritAttrs: false })
withDefaults(
  defineProps<{ items: Option[]; variant?: ButtonVariant; size?: ButtonSize; disabled?: boolean; empty?: string }>(),
  { variant: 'secondary', size: 'md', disabled: false, empty: 'Nothing to add.' },
)
const emit = defineEmits<{ select: [id: string] }>()

// The list opens above the button when it doesn't fit below.
const { field, openUp, place } = useDropPlacement()
</script>
