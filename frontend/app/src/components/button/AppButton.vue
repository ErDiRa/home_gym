<template>
  <!-- A <button>, or a link styled the same: a RouterLink with `to`, an external <a> (new tab) with
       `href`. type and disabled only apply to a button. -->
  <component
    :is="to ? RouterLink : href ? 'a' : 'button'"
    :to="to"
    :href="href"
    :target="href ? '_blank' : undefined"
    :rel="href ? 'noopener noreferrer' : undefined"
    :type="to || href ? undefined : type"
    :disabled="to || href ? undefined : disabled"
    :aria-pressed="pressed"
    :class="[
      'inline-flex shrink-0 items-center justify-center gap-1.5 border font-medium whitespace-nowrap transition-colors',
      'focus-visible:ring-2 focus-visible:ring-indigo-500 focus-visible:ring-offset-2 focus-visible:outline-none',
      'disabled:cursor-not-allowed disabled:opacity-50',
      sizeClasses[size],
      icon ? iconSizeClasses[size] : paddingClasses[size],
      variantClasses[pressed ? 'primary' : variant],
    ]"
  >
    <slot />
  </component>
</template>

<script setup lang="ts">
import { RouterLink, type RouteLocationRaw } from 'vue-router'

// primary/secondary for actions, danger for destructive ones, ghost and text for quiet ones (icons,
// "Add set"), inverse on the dark top bars, success/warning for states (a logged or changed set).
export type ButtonVariant = 'primary' | 'secondary' | 'danger' | 'ghost' | 'text' | 'inverse' | 'success' | 'warning'
// touch and xl meet the 44px minimum tap target of the mobile tracker.
export type ButtonSize = 'xs' | 'sm' | 'md' | 'lg' | 'xl' | 'touch'

withDefaults(
  defineProps<{
    variant?: ButtonVariant
    size?: ButtonSize
    type?: 'button' | 'submit' | 'reset'
    disabled?: boolean
    /** A square button holding only an icon (give it an aria-label). */
    icon?: boolean
    /** Makes it a toggle: pressed shows the primary style and sets aria-pressed. */
    pressed?: boolean
    /** Renders a router link styled as a button instead of a <button>. */
    to?: RouteLocationRaw
    /** Renders an external link (opened in a new tab) styled as a button. */
    href?: string
  }>(),
  { variant: 'primary', size: 'lg', type: 'button', disabled: false, icon: false, pressed: undefined, to: undefined, href: undefined },
)

const variantClasses: Record<ButtonVariant, string> = {
  primary: 'border-indigo-600 bg-indigo-600 text-white hover:border-indigo-700 hover:bg-indigo-700',
  secondary: 'border-slate-300 bg-white text-slate-700 hover:border-slate-400 hover:bg-slate-100 active:bg-slate-100',
  danger: 'border-red-600 bg-red-600 text-white hover:border-red-700 hover:bg-red-700',
  ghost: 'border-transparent text-slate-500 hover:bg-slate-100 hover:text-slate-800 active:bg-slate-100',
  text: 'border-transparent text-indigo-600 hover:bg-indigo-50 hover:text-indigo-800 active:bg-indigo-50',
  inverse: 'border-transparent text-slate-300 hover:bg-white/10 hover:text-white active:bg-white/10',
  success: 'border-emerald-500 bg-emerald-500 text-white hover:bg-emerald-600 active:bg-emerald-600',
  warning: 'border-amber-500 bg-amber-500 text-white hover:bg-amber-600 active:bg-amber-600',
}

const sizeClasses: Record<ButtonSize, string> = {
  xs: 'h-7 rounded text-xs',
  sm: 'h-8 rounded text-sm',
  md: 'h-9 rounded-lg text-sm',
  lg: 'h-10 rounded-lg text-sm',
  xl: 'h-12 rounded-xl text-base',
  touch: 'h-11 min-w-11 rounded-xl text-sm',
}

const paddingClasses: Record<ButtonSize, string> = {
  xs: 'px-2',
  sm: 'px-2.5',
  md: 'px-3',
  lg: 'px-4',
  xl: 'px-5',
  touch: 'px-3',
}

const iconSizeClasses: Record<ButtonSize, string> = {
  xs: 'w-7',
  sm: 'w-8',
  md: 'w-9',
  lg: 'w-10',
  xl: 'w-12',
  touch: 'w-11',
}
</script>
