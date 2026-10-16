<template>
  <component
    :is="to ? RouterLink : 'button'"
    :to="to"
    :type="to ? undefined : type"
    :disabled="to ? undefined : disabled"
    :class="[
      'font-semibold shadow-xs focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-indigo-600 disabled:cursor-not-allowed disabled:opacity-50',
      sizes[size],
      variants[variant],
    ]"
  >
    <slot />
  </component>
</template>

<script setup lang="ts">
import { RouterLink, type RouteLocationRaw } from 'vue-router'

export type ButtonVariant = 'primary' | 'secondary'
export type ButtonSize = 'xs' | 'sm' | 'md' | 'lg' | 'xl'

withDefaults(
  defineProps<{
    variant?: ButtonVariant
    size?: ButtonSize
    type?: 'button' | 'submit' | 'reset'
    disabled?: boolean
    /** Renders a router link styled as a button instead of a <button>. */
    to?: RouteLocationRaw
  }>(),
  { variant: 'primary', size: 'lg', type: 'button', disabled: false, to: undefined },
)

const variants: Record<ButtonVariant, string> = {
  primary: 'bg-indigo-600 text-white enabled:hover:bg-indigo-500',
  secondary:
    'bg-white text-gray-900 inset-ring inset-ring-gray-300 enabled:hover:bg-gray-50',
}

const sizes: Record<ButtonSize, string> = {
  xs: 'rounded-sm px-2 py-1 text-xs',
  sm: 'rounded-sm px-2 py-1 text-sm',
  md: 'rounded-md px-2.5 py-1.5 text-sm',
  lg: 'rounded-md px-3 py-2 text-sm',
  xl: 'rounded-md px-3.5 py-2.5 text-sm',
}
</script>
