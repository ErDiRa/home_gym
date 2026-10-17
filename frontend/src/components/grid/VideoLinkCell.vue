<template>
  <span v-if="urls.length" class="inline-flex gap-2">
    <a
      v-for="(url, i) in urls"
      :key="i"
      :href="url"
      target="_blank"
      @click.stop
      rel="noopener noreferrer"
      :title="url"
      class="font-semibold text-indigo-600 hover:underline"
    >
      {{ i === 0 ? 'Watch ▶' : i + 1 }}
    </a>
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { ICellRendererParams } from 'ag-grid-community'

const props = defineProps<{ params: ICellRendererParams }>()
// Only http(s) links are rendered, never e.g. javascript: URLs.
const urls = computed(() => ((props.params.value ?? []) as string[]).filter((u) => /^https?:\/\//i.test(u)))
</script>
