<template>
  <div>
    <p class="flex items-baseline justify-between gap-2 text-sm">
      <span class="text-slate-700">
        <span class="font-semibold text-slate-900">{{ macros ? formatAmount(macros.kcal) : '–' }}</span>
        <template v-if="target"> / {{ formatAmount(target.kcal) }}</template> kcal
      </span>
      <span v-if="target" class="text-xs text-slate-500" :title="sourceTitle">{{ target.source === 'whoop' ? 'WHOOP' : 'estimated' }}</span>
    </p>
    <dl class="mt-1.5 grid grid-cols-3 gap-3">
      <div v-for="row in rows" :key="row.key">
        <dt class="sr-only">{{ row.label }}</dt>
        <dd class="text-xs text-slate-600">
          <span class="font-medium text-slate-900">{{ row.short }}</span>
          {{ row.eaten === null ? '–' : formatAmount(row.eaten) }}<template v-if="row.target !== null"> / {{ formatAmount(row.target) }}</template> g
        </dd>
        <dd class="mt-1 h-1.5 overflow-hidden rounded-full bg-slate-100" aria-hidden="true">
          <div class="h-full rounded-full" :class="toneClasses[shareTone(row.share)]" :style="{ width: `${Math.min(row.share ?? 0, 1) * 100}%` }" />
        </dd>
      </div>
    </dl>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { DayMacros, MacroTarget } from '../../types'
import { formatAmount, macroRows, shareTone } from './macros'

// A day's macros against its target: the energy, then protein, fat and carbs as bars (green on
// track within ±10 %, amber below, rose above).
const props = defineProps<{ macros: DayMacros | null; target: MacroTarget | null }>()

const rows = computed(() => macroRows(props.macros, props.target))
const sourceTitle = computed(() =>
  props.target?.source === 'whoop'
    ? 'The energy WHOOP measured that day'
    : 'Estimated from your body and the day’s workouts, to keep your weight',
)

const toneClasses = {
  none: 'bg-slate-300',
  low: 'bg-amber-400',
  ok: 'bg-emerald-500',
  high: 'bg-rose-400',
}
</script>
