<template>
  <fieldset class="min-w-0">
    <legend class="mb-1 flex items-center gap-3 text-xs font-medium text-slate-600">
      <span>
        {{ part === 'reps' ? 'Reps' : 'Load' }}<span v-if="part === 'reps' && !optional" class="text-red-600" aria-hidden="true"> *</span>
        <span v-if="hint" class="font-normal text-slate-400"> · {{ hint }}</span>
      </span>
      <!-- On bodyweight (e.g. pull-ups) the load is the weight added to it, if any. -->
      <AppCheckbox
        v-if="part === 'load'"
        v-model="bodyweight"
        size="sm"
        class="items-center font-normal"
        title="On bodyweight, plus the weight entered, if any"
      >
        BW<span class="sr-only"> ({{ label }} on bodyweight)</span>
      </AppCheckbox>
    </legend>
    <div class="flex flex-wrap items-center gap-1">
      <span v-if="part === 'load' && amount.load_bodyweight" class="text-sm text-slate-400" aria-hidden="true">BW +</span>
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
      <!-- Opens the loads logged of the exercise in its newest workouts. -->
      <AppButton
        v-if="suggestion"
        variant="text"
        size="xs"
        :aria-label="`${label} load: view the logged loads`"
        :title="`Last logged: ${suggestion.text} on ${suggestion.on}`"
        @click="historyOpen = true"
      >
        View logs
      </AppButton>
      <LoadHistoryDialog
        v-if="last"
        :open="historyOpen"
        :exercise-id="last.exercise_id"
        :name="exerciseName"
        @close="historyOpen = false"
        @pick="pick"
      />
      <AppCheckbox v-if="part === 'reps'" v-model="amount.per_side" size="sm" class="ml-1 items-center">/side</AppCheckbox>
    </div>
  </fieldset>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import AppButton from '../button/AppButton.vue'
import AppCheckbox from '../form/AppCheckbox.vue'
import TextInput from '../form/TextInput.vue'
import LoadHistoryDialog from './LoadHistoryDialog.vue'
import UnitSelect from './UnitSelect.vue'
import { formatDay } from '../../dates'
import { useExercisesStore } from '../../stores/exercises'
import { useLookupsStore, type AmountPart } from '../../stores/lookups'
import type { Amount, LastLoad, LoadHistorySet } from '../../types'

// Edits the reps or the load of an amount in place: a value, an optional upper end (a range
// such as 8-10) and a unit; reps can also be per side. Reps are required unless optional (the
// exercises of a WOD). A load can be on bodyweight, its value then the weight added to it; last
// is the load last logged of the exercise, opening the loads logged of it to pick one from.
const props = defineProps<{ part: AmountPart; label: string; hint?: string; optional?: boolean; last?: LastLoad | null }>()
const amount = defineModel<Amount>({ required: true })
const lookups = useLookupsStore()

// value, max and unit map onto the reps_* or load_* fields, depending on part.
const value = computed({
  get: () => (props.part === 'reps' ? amount.value.reps_value : amount.value.load_value),
  set: (v) => {
    // Typing a value picks the default unit, so the common case needs no unit click.
    // Weight added to bodyweight is in a weight unit.
    if (v !== null && unit.value === null) unit.value = lookups.defaultUnit(props.part, amount.value.load_bodyweight)
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

// Never on bodyweight, as the logged loads leave out weight added to bodyweight.
const suggestion = computed(() => {
  const last = props.last
  if (props.part !== 'load' || !last || amount.value.load_bodyweight) return null
  return { text: `${last.load_value} ${last.load_unit_name}`, on: formatDay(last.done_at) }
})

const historyOpen = ref(false)
const exercises = useExercisesStore()
const exerciseName = computed(() => exercises.exercises.find((e) => e.id === props.last?.exercise_id)?.name ?? '')

// Plans the load of a set picked from the history.
function pick(set: LoadHistorySet) {
  amount.value.load_value = set.load_value
  amount.value.load_max = null
  unit.value = set.load_unit
  historyOpen.value = false
}

// Bodyweight alone needs no unit; weight added to it is in a weight unit (kg or lb).
const bodyweight = computed({
  get: () => amount.value.load_bodyweight,
  set: (on: boolean) => {
    const a = amount.value
    a.load_bodyweight = on
    if (!on) return
    if (a.load_value === null) a.load_unit = null
    else if (!lookups.loadUnits.find((u) => u.id === a.load_unit)?.kg_per_unit) a.load_unit = lookups.defaultUnit('load', true)
  },
})
</script>
