<template>
  <textarea
    v-if="multiline"
    ref="el"
    :value="model ?? ''"
    :class="[baseClass, toneClasses[tone], textSize, 'px-3 py-2']"
    @input="model = ($event.target as HTMLTextAreaElement).value as T"
  />
  <input
    v-else
    ref="el"
    :value="model ?? ''"
    :class="[baseClass, toneClasses[tone], sizeClasses[size]]"
    @input="model = ($event.target as HTMLInputElement).value as T"
  />
</template>

<script setup lang="ts" generic="T extends string | number | null | undefined">
import { computed, useTemplateRef } from 'vue'

export type InputSize = 'sm' | 'md' | 'touch'
// What a template ref to a TextInput gets.
export interface TextInputExposed {
  el: HTMLInputElement | HTMLTextAreaElement | null
  focus(): void
}

// A text, number, date… input (attributes such as type, placeholder or required pass through), or
// a textarea with `multiline`. Use v-model, v-model.number (empty = null) or :model-value with your
// own @change. touch is the tracker's 44px size with text-base, so iOS doesn't zoom in.
const props = withDefaults(
  defineProps<{ size?: InputSize; multiline?: boolean; tone?: 'default' | 'success' }>(),
  { size: 'md', multiline: false, tone: 'default' },
)
const [model, modifiers] = defineModel<T, 'number' | 'trim'>({
  set(value) {
    if (!modifiers.number || typeof value !== 'string') return value
    // An empty or unparsable input means "not set" (null), never 0.
    return (value.trim() === '' || Number.isNaN(Number(value)) ? null : Number(value)) as T
  },
})

const el = useTemplateRef<HTMLInputElement | HTMLTextAreaElement>('el')
// el: the input itself, e.g. for a dialog's initial focus.
defineExpose({ el, focus: () => el.value?.focus() } as unknown as TextInputExposed)

const baseClass =
  'block w-full min-w-0 border font-normal text-slate-900 outline-none placeholder:text-slate-400 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-200 disabled:bg-slate-50 disabled:text-slate-500'
const toneClasses = {
  default: 'border-slate-300 bg-white',
  success: 'border-emerald-300 bg-emerald-50',
}
const sizeClasses: Record<InputSize, string> = {
  sm: 'h-8 rounded-md px-2 text-sm',
  md: 'h-10 rounded-md px-3 text-sm',
  touch: 'h-11 rounded-xl px-3 text-base',
}
const textSize = computed(() => (props.size === 'touch' ? 'rounded-xl text-base' : 'rounded-md text-sm'))
</script>
