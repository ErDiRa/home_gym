<template>
  <span
    :class="type === 'mobility' ? 'border-emerald-200 bg-emerald-50 text-emerald-700' : 'border-orange-200 bg-orange-50 text-orange-700'"
    class="rounded-full border px-2 py-0.5 text-xs font-semibold"
  >
    {{ exerciseTypeLabels[type] }}
  </span>
</template>

<script setup lang="ts">
import { exerciseTypeLabels, type ExerciseType } from '../../types'

defineProps<{ type: ExerciseType }>()
</script>
