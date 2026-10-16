<script setup lang="ts">
import type { Accessory } from '../types'

defineProps<{
  item: Accessory
  favorite?: boolean
}>()
defineEmits<{ toggle: [] }>()
</script>

<template>
  <div class="rounded-xl bg-white p-5 shadow transition hover:shadow-md">
    <div class="mb-4 flex h-28 items-center justify-center rounded-lg bg-indigo-50 text-4xl">
      🎒
    </div>
    <p class="text-xs uppercase tracking-wide text-slate-400">{{ item.category }}</p>
    <h2 class="text-lg font-semibold">{{ item.name }}</h2>
    <div class="mt-3 flex items-center justify-between">
      <span class="font-bold text-indigo-600">${{ item.price }}</span>
      <button
        class="rounded-full px-3 py-1 text-sm transition"
        :class="favorite ? 'bg-rose-100 text-rose-600' : 'bg-slate-100 text-slate-600 hover:bg-slate-200'"
        @click="$emit('toggle')"
      >
        {{ favorite ? '♥ Saved' : '♡ Save' }}
      </button>
    </div>
  </div>
</template>
