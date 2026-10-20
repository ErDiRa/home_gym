<template>
  <AppBadge :color="type === 'mobility' ? 'green' : 'yellow'">{{ exerciseTypeLabels[type] }}</AppBadge>
</template>

<script setup lang="ts">
import AppBadge from '../ui/AppBadge.vue'
import { exerciseTypeLabels, type ExerciseType } from '../../types'

defineProps<{ type: ExerciseType }>()
</script>
