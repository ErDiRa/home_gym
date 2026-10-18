<template>
  <fieldset class="min-w-0">
    <legend class="mb-1 text-xs font-medium text-slate-600">
      {{ part === 'reps' ? 'Reps' : 'Load' }}<span v-if="part === 'reps'" class="text-red-600" aria-hidden="true"> *</span>
      <span v-if="hint" class="font-normal text-slate-400"> · {{ hint }}</span>
    </legend>
    <div class="flex flex-wrap items-center gap-1">
      <input
        :value="value ?? ''"
        type="number"
        min="0"
        step="any"
        :placeholder="part === 'reps' ? '8' : '—'"
        :aria-label="`${label} ${part}`"
        :class="numberClass"
        @input="value = parse($event)"
      />
      <span class="text-sm text-slate-400" aria-hidden="true">–</span>
      <input
        :value="max ?? ''"
        type="number"
        min="0"
        step="any"
        placeholder="max"
        :aria-label="`${label} ${part}, upper end of a range (optional)`"
        :class="numberClass"
        @input="max = parse($event)"
      />
      <div class="w-24">
        <UnitSelect v-model="unit" :part="part" :label="`${label} ${part} unit`" />
      </div>
      <label v-if="part === 'reps'" class="ml-1 flex items-center gap-1 text-xs text-slate-600">
        <input v-model="amount.per_side" type="checkbox" class="size-3.5 rounded border-slate-300 accent-indigo-600" />
        /side
      </label>
    </div>
  </fieldset>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import UnitSelect from './UnitSelect.vue'
import { useLookupsStore, type AmountPart } from '../../stores/lookups'
import type { Amount } from '../../types'

// Edits the reps or the load of an amount in place: a value, an optional upper end (a range
// such as 8-10) and a unit; reps can also be per side.
const props = defineProps<{ part: AmountPart; label: string; hint?: string }>()
const amount = defineModel<Amount>({ required: true })
const lookups = useLookupsStore()

const numberClass =
  'h-8 w-16 rounded-md border border-slate-300 bg-white px-2 text-sm text-slate-900 outline-none placeholder:text-slate-400 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-200'

// An empty or unparsable input means "not set" (null), never 0.
const parse = (e: Event) => {
  const raw = (e.target as HTMLInputElement).value
  return raw === '' || Number.isNaN(Number(raw)) ? null : Number(raw)
}

// value, max and unit map onto the reps_* or load_* fields, depending on part.
const value = computed({
  get: () => (props.part === 'reps' ? amount.value.reps_value : amount.value.load_value),
  set: (v) => {
    // Typing a value picks the default unit, so the common case needs no unit click.
    if (v !== null && unit.value === null) unit.value = lookups.defaultUnit(props.part)
    if (props.part === 'reps') amount.value.reps_value = v
    else amount.value.load_value = v
  },
})

const max = computed({
  get: () => (props.part === 'reps' ? amount.value.reps_max : amount.value.load_max),
  set: (v) => {
    if (props.part === 'reps') amount.value.reps_max = v
    else amount.value.load_max = v
  },
})

const unit = computed({
  get: () => (props.part === 'reps' ? amount.value.reps_unit : amount.value.load_unit),
  set: (id) => {
    if (props.part === 'reps') amount.value.reps_unit = id
    else {
      amount.value.load_unit = id
      if (id) lookups.lastLoadUnit = id // becomes the default for the next load
    }
  },
})
</script>
