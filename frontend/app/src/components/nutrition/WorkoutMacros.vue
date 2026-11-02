<template>
  <section class="mt-4 rounded-2xl border border-slate-200 bg-white p-4 shadow-sm" aria-label="Nutrition">
    <button type="button" class="flex min-h-11 w-full items-center gap-2 text-left" :aria-expanded="open" @click="open = !open">
      <h2 class="flex-1 font-semibold text-slate-900">Nutrition</h2>
      <span v-if="!open && summary" class="truncate text-xs text-slate-500">{{ summary }}</span>
      <ChevronDownIcon class="size-5 shrink-0 text-slate-400 transition-transform" :class="{ '-rotate-90': !open }" aria-hidden="true" />
    </button>

    <template v-if="!open" />
    <p v-else-if="!days" class="mt-2 text-sm text-slate-500">Loading…</p>
    <!-- The day before (what the workout was fueled with) and the workout's day; tapping one logs it. -->
    <ul v-else class="mt-1 divide-y divide-slate-100">
      <li v-for="(d, i) in days" :key="d.date">
        <button type="button" class="block w-full py-3 text-left active:bg-slate-50" @click="edit(d)">
          <span class="flex items-baseline justify-between text-sm">
            <span class="font-medium text-slate-800">{{ i === 0 ? 'Day before' : 'Workout day' }}</span>
            <span class="text-xs text-slate-500">{{ formatDate(d.date, { weekday: true, year: false }) }}</span>
          </span>
          <MacroBars v-if="d.macros || d.target" :macros="d.macros" :target="d.target" class="mt-1.5" />
          <span v-else class="mt-1 block text-sm text-slate-500">Nothing logged · tap to add</span>
        </button>
      </li>
    </ul>

    <MacroDaySheet
      :open="sheetOpen"
      :date="editing?.date ?? day"
      :macros="editing?.macros ?? null"
      :target="editing?.target ?? null"
      @close="sheetOpen = false"
      @saved="saved"
    />
  </section>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ChevronDownIcon } from '@heroicons/vue/24/outline'
import { addDays, formatDate } from '../../dates'
import { useNutritionStore } from '../../stores/nutrition'
import type { DayMacros, NutritionDay } from '../../types'
import MacroBars from './MacroBars.vue'
import MacroDaySheet from './MacroDaySheet.vue'
import { macrosLine } from './macros'

// The macros of a workout's day and the day before, against their targets, to see what the
// workout was fueled with; collapsed by default.
const props = defineProps<{ day: string }>()

const store = useNutritionStore()
const open = ref(false)
const days = ref<NutritionDay[] | null>(null)
const editing = ref<NutritionDay | null>(null)
const sheetOpen = ref(false)

// Collapsed: the macros logged the day before, if any.
const summary = computed(() => {
  const before = days.value?.[0]?.macros
  return before ? `Day before: ${macrosLine(before)}` : ''
})

async function load() {
  days.value = null
  try {
    days.value = await store.listNutritionDays(addDays(props.day, -1), props.day)
  } catch {
    days.value = []
  }
}

function edit(d: NutritionDay) {
  editing.value = d
  sheetOpen.value = true
}

function saved(macros: DayMacros | null) {
  const date = editing.value?.date
  days.value = days.value?.map((d) => (d.date === date ? { ...d, macros } : d)) ?? null
}

watch(() => props.day, load, { immediate: true })
</script>
