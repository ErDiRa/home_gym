<template>
  <div class="flex h-full items-center">
    <ActiveBadge v-if="params.value" />
  </div>
</template>

<script setup lang="ts">
import type { ICellRendererParams } from 'ag-grid-community'
import ActiveBadge from '../training/ActiveBadge.vue'

defineProps<{ params: ICellRendererParams }>()
</script>
