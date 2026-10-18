<template>
  <div class="flex shrink-0 items-center">
    <button type="button" :disabled="index === 0" :class="iconButton" :aria-label="`Move ${label} up`" @click="emit('move', -1)">
      <ChevronUpIcon class="size-4" aria-hidden="true" />
    </button>
    <button
      type="button"
      :disabled="index === count - 1"
      :class="iconButton"
      :aria-label="`Move ${label} down`"
      @click="emit('move', 1)"
    >
      <ChevronDownIcon class="size-4" aria-hidden="true" />
    </button>
    <button type="button" :class="iconButton" :aria-label="`Remove ${label}`" @click="emit('remove')">
      <XMarkIcon class="size-4" aria-hidden="true" />
    </button>
  </div>
</template>

<script setup lang="ts">
import { ChevronDownIcon, ChevronUpIcon, XMarkIcon } from '@heroicons/vue/24/outline'

// Up / down / remove controls for an item in an ordered list.
defineProps<{ index: number; count: number; label: string }>()
const emit = defineEmits<{ move: [by: -1 | 1]; remove: [] }>()

const iconButton =
  'rounded p-1 text-slate-500 hover:bg-slate-100 hover:text-slate-800 disabled:opacity-30 disabled:hover:bg-transparent'
</script>
