<template>
  <div class="flex items-center gap-1" role="group" :aria-label="label">
    <TextInput
      v-model.number="minutes"
      type="number"
      inputmode="numeric"
      min="0"
      step="1"
      placeholder="min"
      :aria-label="`${label}, minutes`"
      :size="size"
      :class="size === 'touch' ? 'w-20!' : 'w-14!'"
    />
    <span class="font-semibold text-slate-400" aria-hidden="true">:</span>
    <TextInput
      v-model.number="seconds"
      type="number"
      inputmode="numeric"
      min="0"
      max="59"
      step="1"
      placeholder="sec"
      :aria-label="`${label}, seconds`"
      :size="size"
      :class="size === 'touch' ? 'w-20!' : 'w-14!'"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import TextInput, { type InputSize } from './TextInput.vue'

// A duration as minutes and seconds (minutes may exceed 59, e.g. 90:00), bound to seconds; null
// while both are empty. touch is the tracker's size.
withDefaults(defineProps<{ label: string; size?: InputSize }>(), { size: 'sm' })
const model = defineModel<number | null>({ required: true })

const combine = (m: number | null, s: number | null) =>
  m === null && s === null ? null : Math.max(0, Math.trunc(m ?? 0)) * 60 + Math.max(0, Math.trunc(s ?? 0))

const minutes = computed({
  get: () => (model.value === null ? null : Math.floor(model.value / 60)),
  set: (m: number | null) => (model.value = combine(m, seconds.value)),
})
const seconds = computed({
  get: () => (model.value === null ? null : model.value % 60),
  set: (s: number | null) => (model.value = combine(minutes.value, s)),
})
</script>
