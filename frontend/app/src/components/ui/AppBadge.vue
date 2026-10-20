<template>
  <span :class="[colors.badge, 'inline-flex max-w-full shrink-0 items-center gap-1 rounded-md border px-2 py-0.5 font-medium', size === 'md' ? 'text-sm' : 'text-xs']">
    <slot />
    <!-- Small controls inside the badge (e.g. move a planned session); the remove button comes last. -->
    <span v-if="actions?.length || removable" class="-mr-1 inline-flex items-center">
      <button
        v-for="action in actions"
        :key="action.label"
        type="button"
        :disabled="action.disabled"
        :aria-label="action.label"
        :title="action.label"
        :class="[colors.button, buttonClass]"
        @click.stop="action.onClick"
      >
        <component :is="action.icon" class="size-3.5" aria-hidden="true" />
      </button>
      <button
        v-if="removable"
        type="button"
        :aria-label="removeLabel"
        :title="removeLabel"
        :class="[colors.button, buttonClass]"
        @click.stop="emit('remove')"
      >
        <XMarkIcon class="size-3.5" aria-hidden="true" />
      </button>
    </span>
  </span>
</template>

<script setup lang="ts">
import { computed, type Component } from 'vue'
import { XMarkIcon } from '@heroicons/vue/16/solid'

export type BadgeColor = 'gray' | 'red' | 'yellow' | 'green' | 'blue' | 'indigo' | 'purple' | 'pink'
export interface BadgeAction {
  icon: Component
  /** Accessible name and tooltip. */
  label: string
  disabled?: boolean
  onClick: () => void
}

// A tag or label: soft background, a border in the same color, optionally removable (×).
const props = withDefaults(
  defineProps<{ color?: BadgeColor; size?: 'sm' | 'md'; removable?: boolean; removeLabel?: string; actions?: BadgeAction[] }>(),
  { color: 'gray', size: 'sm', removable: false, removeLabel: 'Remove', actions: undefined },
)
const emit = defineEmits<{ remove: [] }>()

const buttonClass = 'rounded-sm p-px disabled:opacity-30 disabled:hover:bg-transparent'

const palette: Record<BadgeColor, { badge: string; button: string }> = {
  gray: {
    badge: 'border-slate-200 bg-slate-50 text-slate-600',
    button: 'text-slate-400 hover:bg-slate-200/70 hover:text-slate-700',
  },
  red: { badge: 'border-red-200 bg-red-50 text-red-700', button: 'text-red-400 hover:bg-red-100 hover:text-red-700' },
  yellow: {
    badge: 'border-amber-200 bg-amber-50 text-amber-800',
    button: 'text-amber-500 hover:bg-amber-100 hover:text-amber-800',
  },
  green: {
    badge: 'border-emerald-200 bg-emerald-50 text-emerald-700',
    button: 'text-emerald-500 hover:bg-emerald-100 hover:text-emerald-700',
  },
  blue: { badge: 'border-blue-200 bg-blue-50 text-blue-700', button: 'text-blue-400 hover:bg-blue-100 hover:text-blue-700' },
  indigo: {
    badge: 'border-indigo-200 bg-indigo-50 text-indigo-700',
    button: 'text-indigo-400 hover:bg-indigo-100 hover:text-indigo-700',
  },
  purple: {
    badge: 'border-purple-200 bg-purple-50 text-purple-700',
    button: 'text-purple-400 hover:bg-purple-100 hover:text-purple-700',
  },
  pink: { badge: 'border-pink-200 bg-pink-50 text-pink-700', button: 'text-pink-400 hover:bg-pink-100 hover:text-pink-700' },
}
const colors = computed(() => palette[props.color])
</script>
