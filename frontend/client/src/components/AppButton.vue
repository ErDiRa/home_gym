<template>
  <button
    :type="type"
    :disabled="disabled"
    :class="[
      'inline-flex items-center justify-center gap-2 border font-medium whitespace-nowrap transition-colors select-none',
      'focus-visible:ring-2 focus-visible:ring-indigo-500 focus-visible:ring-offset-2 focus-visible:outline-none',
      'disabled:cursor-not-allowed disabled:opacity-50',
      sizeClasses[size],
      variantClasses[variant],
    ]"
  >
    <slot />
  </button>
</template>

<script setup lang="ts">
export type ButtonVariant = 'primary' | 'secondary' | 'ghost'
// Sized for touch: md is the 44px minimum tap target, lg the main call to action.
export type ButtonSize = 'md' | 'lg'

withDefaults(
  defineProps<{
    variant?: ButtonVariant
    size?: ButtonSize
    type?: 'button' | 'submit' | 'reset'
    disabled?: boolean
  }>(),
  { variant: 'primary', size: 'lg', type: 'button', disabled: false },
)

const variantClasses: Record<ButtonVariant, string> = {
  primary: 'border-indigo-600 bg-indigo-600 text-white active:bg-indigo-700',
  secondary: 'border-slate-300 bg-white text-slate-700 active:bg-slate-100',
  ghost: 'border-transparent bg-transparent text-slate-600 active:bg-slate-100',
}

const sizeClasses: Record<ButtonSize, string> = {
  md: 'h-11 min-w-11 rounded-xl px-3 text-sm',
  lg: 'h-12 rounded-xl px-5 text-base',
}
</script>
