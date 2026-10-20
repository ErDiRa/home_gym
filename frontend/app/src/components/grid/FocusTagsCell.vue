<template>
  <div class="flex h-full items-center gap-1">
    <AppBadge v-for="focus in focuses" :key="focus.id" color="indigo" :title="focus.name">{{ focus.alias }}</AppBadge>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { ICellRendererParams } from 'ag-grid-community'
import AppBadge from '../ui/AppBadge.vue'
import type { Focus } from '../../types'

// AG Grid cell renderer: shows focuses as alias tags, full name on hover; params.value is the cell's value.
const props = defineProps<{ params: ICellRendererParams }>()
const focuses = computed<Focus[]>(() => props.params.value ?? [])
</script>
