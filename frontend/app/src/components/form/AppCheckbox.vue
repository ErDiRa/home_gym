<template>
  <label :class="[size === 'sm' ? 'gap-1 text-xs text-slate-600' : 'gap-2 text-sm text-slate-800', 'flex cursor-pointer items-start']">
    <input
      v-model="model"
      type="checkbox"
      :class="[size === 'sm' ? 'size-3.5' : 'mt-0.5 size-4', 'shrink-0 rounded border-slate-300 accent-indigo-600']"
    />
    <span>
      <span :class="{ 'font-medium': description }"><slot /></span>
      <span v-if="description" class="block text-xs text-slate-500">{{ description }}</span>
    </span>
  </label>
</template>

<script setup lang="ts">
// A checkbox with its label (the slot) and an optional description below it.
withDefaults(defineProps<{ description?: string; size?: 'sm' | 'md' }>(), { description: '', size: 'md' })
const model = defineModel<boolean>({ required: true })
</script>
