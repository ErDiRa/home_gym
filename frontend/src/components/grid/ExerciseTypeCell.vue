<template>
  <div class="flex h-full items-center">
    <ExerciseTypeBadge v-if="type" :type="type" />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { ICellRendererParams } from 'ag-grid-community'
import ExerciseTypeBadge from '../exercise/ExerciseTypeBadge.vue'
import type { ExerciseType } from '../../types'

// AG Grid cell renderer: shows the exercise type as a badge; params.value is the cell's value.
const props = defineProps<{ params: ICellRendererParams }>()
const type = computed<ExerciseType | undefined>(() => props.params.value)
</script>
