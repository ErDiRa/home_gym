<template>
  <div ref="field" class="relative inline-block shrink-0">
    <Menu>
      <MenuButton :as="AppButton" variant="ghost" :size="size" icon :aria-label="label" :title="label" @click="place">
        <EllipsisHorizontalIcon class="size-5" aria-hidden="true" />
      </MenuButton>

      <Transition
        enter-active-class="transition duration-100 ease-out"
        enter-from-class="scale-95 opacity-0"
        leave-active-class="transition duration-75 ease-in"
        leave-to-class="scale-95 opacity-0"
      >
        <MenuItems
          :class="openUp ? 'bottom-full mb-1 origin-bottom-right' : 'top-full mt-1 origin-top-right'"
          class="absolute right-0 z-20 w-max overflow-auto rounded-lg border border-slate-200 bg-white py-1 text-sm shadow-lg focus:outline-none"
        >
          <!-- Menu rows, styled like the ones of ActionMenu. -->
          <MenuItem v-for="item in items" :key="item.id" v-slot="{ active, disabled }" :disabled="item.disabled">
            <button
              type="button"
              :disabled="disabled"
              :title="item.title"
              :class="[
                disabled ? 'cursor-not-allowed text-slate-400' : active ? 'bg-indigo-50 text-indigo-900' : 'text-slate-800',
                'block w-full px-3 py-2 text-left whitespace-nowrap',
              ]"
              @click="emit('select', item.id)"
            >
              {{ item.name }}
            </button>
          </MenuItem>
        </MenuItems>
      </Transition>
    </Menu>
  </div>
</template>

<script setup lang="ts">
import { Menu, MenuButton, MenuItem, MenuItems } from '@headlessui/vue'
import { EllipsisHorizontalIcon } from '@heroicons/vue/24/outline'
import AppButton, { type ButtonSize } from '../button/AppButton.vue'
import { useDropPlacement } from '../form/useDropPlacement'

export type MoreMenuItem = { id: string; name: string; disabled?: boolean; title?: string }

// A "…" button opening a list of further actions of an item, e.g. "Save block"; emits the
// action's id. label names it for screen readers, e.g. "More actions of block A".
withDefaults(defineProps<{ items: MoreMenuItem[]; label: string; size?: ButtonSize }>(), { size: 'sm' })
const emit = defineEmits<{ select: [id: string] }>()

// The list opens above the button when it doesn't fit below.
const { field, openUp, place } = useDropPlacement()
</script>
