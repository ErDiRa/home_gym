<template>
  <a
    v-if="url"
    :href="url"
    target="_blank"
    @click.stop
    rel="noopener noreferrer"
    class="font-semibold text-indigo-600 hover:underline"
  >
    Watch ▶
  </a>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { ICellRendererParams } from 'ag-grid-community'

const props = defineProps<{ params: ICellRendererParams }>()
// Only http(s) links are rendered, never e.g. javascript: URLs.
const url = computed(() => (/^https?:\/\//i.test(props.params.value ?? '') ? props.params.value : ''))
</script>
