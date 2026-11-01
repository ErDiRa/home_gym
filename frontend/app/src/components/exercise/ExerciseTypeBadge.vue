<template>
  <AppBadge :color="colors[type]">{{ exerciseTypeLabels[type] }}</AppBadge>
</template>

<script setup lang="ts">
import AppBadge, { type BadgeColor } from '../ui/AppBadge.vue'
import { exerciseTypeLabels, type ExerciseType } from '../../types'

defineProps<{ type: ExerciseType }>()

const colors: Record<ExerciseType, BadgeColor> = { mobility: 'green', strength: 'yellow', endurance: 'blue' }
</script>
