<template>
  <article class="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm">
    <header class="flex items-center gap-3 bg-slate-100 px-4 py-3">
      <h3 class="min-w-0 flex-1 truncate text-lg font-semibold text-slate-900">{{ name }}</h3>
      <CheckCircleIcon v-if="done" class="size-9 shrink-0 text-emerald-500" aria-label="Done" />
      <span v-else class="flex size-9 shrink-0 items-center justify-center rounded-full bg-white" aria-label="Not done">
        <CheckIcon class="size-5 text-slate-300" aria-hidden="true" />
      </span>
    </header>

    <ul v-if="lines.length" class="space-y-2.5 px-4 py-3">
      <li v-for="line in lines" :key="line.key" class="flex items-start gap-3">
        <span
          class="flex h-7 min-w-8 shrink-0 items-center justify-center rounded-md bg-slate-100 px-1 text-xs font-semibold text-slate-600"
        >
          {{ line.label }}
        </span>
        <span class="min-w-0 flex-1 pt-0.5">
          <span class="block truncate text-sm font-medium text-slate-800">{{ line.name }}</span>
          <span v-if="line.detail" class="line-clamp-2 text-xs text-slate-500">{{ line.detail }}</span>
        </span>
      </li>
    </ul>
  </article>
</template>

<script setup lang="ts">
import { CheckIcon } from '@heroicons/vue/24/outline'
import { CheckCircleIcon } from '@heroicons/vue/24/solid'
import type { SummaryLine } from './workoutPlan'

// A planned session of the week: its name and its blocks at a glance, checked once done.
defineProps<{
  name: string
  lines: SummaryLine[]
  done: boolean
}>()
</script>
