<template>
  <div class="flex h-full items-center gap-1">
    <span
      v-for="focus in focuses"
      :key="focus.id"
      :title="focus.name"
      class="rounded-full bg-indigo-50 px-2 py-0.5 text-xs font-semibold text-indigo-700 border border-indigo-200"
    >
      {{ focus.alias }}
    </span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { ICellRendererParams } from 'ag-grid-community'
import type { Focus } from '../../types'

const props = defineProps<{ params: ICellRendererParams }>()
const focuses = computed<Focus[]>(() => props.params.value ?? [])
</script>
