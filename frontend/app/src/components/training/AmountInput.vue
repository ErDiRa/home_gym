<template>
  <fieldset class="min-w-0">
    <legend class="mb-1 text-xs font-medium text-slate-600">
      {{ part === 'reps' ? 'Reps' : 'Load' }}<span v-if="part === 'reps' && !optional" class="text-red-600" aria-hidden="true"> *</span>
      <span v-if="hint" class="font-normal text-slate-400"> · {{ hint }}</span>
    </legend>
    <div class="flex flex-wrap items-center gap-1">
      <TextInput
        v-model.number="value"
        type="number"
        min="0"
        step="any"
        :placeholder="part === 'reps' && !optional ? '8' : '—'"
        :aria-label="`${label} ${part}`"
        size="sm"
        class="w-16!"
      />
      <span class="text-sm text-slate-400" aria-hidden="true">–</span>
      <TextInput
        v-model.number="max"
        type="number"
        min="0"
        step="any"
        placeholder="max"
        :aria-label="`${label} ${part}, upper end of a range (optional)`"
        size="sm"
        class="w-16!"
      />
      <div class="w-24">
        <UnitSelect v-model="unit" :part="part" :label="`${label} ${part} unit`" />
      </div>
      <AppCheckbox v-if="part === 'reps'" v-model="amount.per_side" size="sm" class="ml-1 items-center">/side</AppCheckbox>
    </div>
  </fieldset>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import AppCheckbox from '../form/AppCheckbox.vue'
import TextInput from '../form/TextInput.vue'
import UnitSelect from './UnitSelect.vue'
import { useLookupsStore, type AmountPart } from '../../stores/lookups'
import type { Amount } from '../../types'

// Edits the reps or the load of an amount in place: a value, an optional upper end (a range
// such as 8-10) and a unit; reps can also be per side. Reps are required unless optional (the
// exercises of a WOD).
const props = defineProps<{ part: AmountPart; label: string; hint?: string; optional?: boolean }>()
const amount = defineModel<Amount>({ required: true })
const lookups = useLookupsStore()

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
