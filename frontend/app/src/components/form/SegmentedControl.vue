<template>
  <RadioGroup v-model="model" class="inline-flex rounded-md border border-slate-300 bg-white p-0.5">
    <RadioGroupLabel class="sr-only">{{ label }}</RadioGroupLabel>
    <RadioGroupOption
      v-for="option in options"
      :key="String(option.value)"
      v-slot="{ checked, active }"
      :value="option.value"
      as="template"
    >
      <div
        :title="option.title"
        :class="[
          sizeClasses[size],
          'flex cursor-pointer items-center rounded font-medium whitespace-nowrap focus:outline-none',
          checked ? 'bg-indigo-600 text-white' : 'text-slate-700 hover:bg-slate-100',
          { 'ring-2 ring-indigo-200': active },
        ]"
      >
        {{ option.label }}
      </div>
    </RadioGroupOption>
  </RadioGroup>
</template>

<script setup lang="ts" generic="T extends string | number">
import { RadioGroup, RadioGroupLabel, RadioGroupOption } from '@headlessui/vue'

export interface Segment<V> {
  value: V
  label: string
  /** Tooltip, e.g. what the choice means. */
  title?: string
}

// One choice out of a few, shown side by side (e.g. strength / mobility). The model may hold a
// value none of the options has; then none is checked.
withDefaults(defineProps<{ options: Segment<T>[]; label: string; size?: 'xs' | 'sm' }>(), { size: 'sm' })
const model = defineModel<T | ''>({ required: true })

// sm is as tall as a md TextInput (40px with the border and padding).
const sizeClasses = {
  xs: 'h-6 px-2.5 text-xs',
  sm: 'h-[34px] px-4 text-sm',
}
</script>
