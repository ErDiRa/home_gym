<template>
  <article
    class="relative overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm"
    :class="{ 'active:opacity-80': to || startable, 'opacity-60': startable && disabled }"
  >
    <header class="flex items-center gap-3 bg-slate-100 py-3 pr-2 pl-4">
      <div class="min-w-0 flex-1">
        <!-- The name's link or button stretches over the whole card; only the chevron sits above it. -->
        <h3 class="truncate text-lg font-semibold text-slate-900">
          <RouterLink v-if="to" :to="to" class="after:absolute after:inset-0 after:rounded-2xl">{{ name }}</RouterLink>
          <button
            v-else-if="startable"
            type="button"
            class="text-left after:absolute after:inset-0 after:rounded-2xl"
            :disabled="disabled"
            @click="emit('start')"
          >
            {{ name }}
          </button>
          <template v-else>{{ name }}</template>
        </h3>
        <p v-if="!expanded && lines.length" class="text-xs text-slate-500">
          {{ lines.length }} {{ lines.length === 1 ? 'block' : 'blocks' }}
        </p>
      </div>
      <CheckCircleIcon v-if="done" class="size-9 shrink-0 text-emerald-500" aria-label="Done" />
      <span v-else class="flex size-9 shrink-0 items-center justify-center rounded-full bg-white" aria-label="Not done">
        <CheckIcon class="size-5 text-slate-300" aria-hidden="true" />
      </span>
      <button
        v-if="lines.length"
        type="button"
        class="relative z-10 flex size-11 shrink-0 items-center justify-center rounded-full text-slate-500 active:bg-slate-200"
        :aria-expanded="expanded"
        :aria-label="`${expanded ? 'Collapse' : 'Expand'} ${name}`"
        @click="expanded = !expanded"
      >
        <ChevronDownIcon class="size-5 transition-transform" :class="{ '-rotate-90': !expanded }" aria-hidden="true" />
      </button>
    </header>

    <ul v-if="lines.length" v-show="expanded" class="space-y-2.5 px-4 py-3">
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
import { CheckIcon, ChevronDownIcon } from '@heroicons/vue/24/outline'
import { CheckCircleIcon } from '@heroicons/vue/24/solid'
import { ref } from 'vue'
import type { RouteLocationRaw } from 'vue-router'
import type { SummaryLine } from './workoutPlan'

// A planned session of the week: its name and its blocks at a glance, checked once done. The
// blocks start collapsed for an overview of the week. With `to` the card opens that route, with
// `startable` it emits start; otherwise it only shows the plan.
defineProps<{
  name: string
  lines: SummaryLine[]
  done: boolean
  to?: RouteLocationRaw
  startable?: boolean
  disabled?: boolean
}>()

const emit = defineEmits<{ start: [] }>()

const expanded = ref(false)
</script>
