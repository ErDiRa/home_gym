<template>
  <!-- A <button>, or a RouterLink styled the same when `to` is set (type and disabled only apply to a button). -->
  <component
    :is="to ? RouterLink : 'button'"
    :to="to"
    :type="to ? undefined : type"
    :disabled="to ? undefined : disabled"
    :class="[
      'inline-flex items-center justify-center gap-1.5 border font-medium whitespace-nowrap transition-colors',
      'focus-visible:ring-2 focus-visible:ring-indigo-500 focus-visible:ring-offset-2 focus-visible:outline-none',
      'disabled:cursor-not-allowed disabled:opacity-50',
      sizeClasses[size],
      variantClasses[variant],
    ]"
  >
    <slot />
  </component>
</template>

<script setup lang="ts">
import { RouterLink, type RouteLocationRaw } from 'vue-router'

export type ButtonVariant = 'primary' | 'secondary'
// touch and xl meet the 44px minimum tap target of the mobile tracker.
export type ButtonSize = 'xs' | 'sm' | 'md' | 'lg' | 'xl' | 'touch'

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

const variantClasses: Record<ButtonVariant, string> = {
  primary: 'border-indigo-600 bg-indigo-600 text-white hover:border-indigo-700 hover:bg-indigo-700',
  secondary: 'border-slate-300 bg-white text-slate-700 hover:border-slate-400 hover:bg-slate-100',
}

const sizeClasses: Record<ButtonSize, string> = {
  xs: 'h-7 rounded px-2 text-xs',
  sm: 'h-8 rounded px-2.5 text-sm',
  md: 'h-9 rounded-lg px-3 text-sm',
  lg: 'h-10 rounded-lg px-4 text-sm',
  xl: 'h-12 rounded-xl px-5 text-base',
  touch: 'h-11 min-w-11 rounded-xl px-3 text-sm',
}
</script>
