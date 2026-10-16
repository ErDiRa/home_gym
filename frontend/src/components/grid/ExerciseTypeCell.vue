<template>
  <div class="flex h-full items-center">
    <span
      v-if="type"
      :class="type === 'mobility' ? 'border-emerald-200 bg-emerald-50 text-emerald-700' : 'border-orange-200 bg-orange-50 text-orange-700'"
      class="rounded-full border px-2 py-0.5 text-xs font-semibold"
    >
      {{ exerciseTypeLabels[type] }}
    </span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { ICellRendererParams } from 'ag-grid-community'
import { exerciseTypeLabels, type ExerciseType } from '../../types'

const props = defineProps<{ params: ICellRendererParams }>()
const type = computed<ExerciseType | undefined>(() => props.params.value)
</script>
