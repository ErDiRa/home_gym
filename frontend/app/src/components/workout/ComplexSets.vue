<template>
  <div class="py-3">
    <!-- The movements in order, each with its rating and videos. -->
    <ul class="space-y-1">
      <li v-for="card in block.exercises" :key="card.key" class="flex items-start gap-2">
        <span class="w-8 shrink-0 pt-0.5 text-sm font-semibold text-indigo-600">{{ card.label }}</span>
        <div class="min-w-0 flex-1">
          <p class="font-semibold text-slate-900">{{ card.name }}</p>
          <p v-if="card.tempo || card.notes" class="text-sm text-slate-500">
            <span v-if="card.tempo">Tempo {{ card.tempo }}</span>
            <span v-if="card.tempo && card.notes"> · </span>
            {{ card.notes }}
          </p>
        </div>
        <ExerciseActions :exercise-id="card.exerciseId" :name="card.name" aligned />
      </li>
    </ul>
    <p v-if="first.planned" class="mt-1 text-sm text-slate-600">{{ first.planned }}</p>

    <!-- One row per round: reps and load for every movement, and a button that logs, saves or
         undoes the whole round. -->
    <div class="mt-2 grid grid-cols-[2rem_1fr_1fr_2.75rem] items-center gap-x-2 gap-y-1.5">
      <span class="text-xs font-medium text-slate-500">Set</span>
      <span class="text-xs font-medium text-slate-500">{{ first.repsUnitName || 'reps' }}{{ first.perSide ? ' / side' : '' }}</span>
      <SelectField
        :model-value="first.loadUnit"
        :options="lookups.loadUnits"
        :searchable="false"
        size="sm"
        option-size="touch"
        :aria-label="`Block ${block.letter} load unit`"
        field-class="h-7! border-transparent! bg-transparent! pl-0! text-xs! font-medium text-slate-500!"
        @update:model-value="switchUnit"
      />
      <span />

      <template v-for="round in rounds" :key="round.number">
        <span class="text-center text-sm font-semibold text-slate-500">{{ round.number }}</span>
        <TextInput
          :model-value="round.sets[0].row.reps"
          type="number"
          inputmode="decimal"
          min="0"
          step="any"
          :aria-label="`Block ${block.letter} set ${round.number} ${first.repsUnitName || 'reps'}`"
          size="touch"
          :tone="roundState(round) === 'logged' ? 'success' : 'default'"
          @update:model-value="set(round, 'reps', $event)"
        />
        <TextInput
          :model-value="round.sets[0].row.load"
          type="number"
          inputmode="decimal"
          min="0"
          step="any"
          placeholder="–"
          :aria-label="`Block ${block.letter} set ${round.number} load`"
          size="touch"
          :tone="roundState(round) === 'logged' ? 'success' : 'default'"
          @update:model-value="set(round, 'load', $event)"
        />
        <AppButton
          size="touch"
          icon
          :variant="variants[roundState(round)]"
          :disabled="round.sets.some((s) => s.row.saving)"
          :aria-label="buttonLabel(round)"
          :title="buttonLabel(round)"
          :class="{ 'text-slate-400!': roundState(round) === 'open' }"
          @click="emit('toggle', round)"
        >
          <ArrowPathIcon v-if="roundState(round) === 'edited'" class="size-5" aria-hidden="true" />
          <CheckIcon v-else class="size-6" aria-hidden="true" />
        </AppButton>
      </template>
    </div>
    <AppButton variant="text" size="touch" class="mt-1 -ml-3" @click="addRound">
      <PlusIcon class="size-4" aria-hidden="true" />
      Add set
    </AppButton>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { ArrowPathIcon, CheckIcon, PlusIcon } from '@heroicons/vue/24/outline'
import AppButton from '../button/AppButton.vue'
import SelectField from '../form/SelectField.vue'
import TextInput from '../form/TextInput.vue'
import ExerciseActions from './ExerciseActions.vue'
import { useExercisesStore } from '../../stores/exercises'
import { useLookupsStore } from '../../stores/lookups'
import { convertLoad, roundLoad } from '../exercise/maxes'
import { num, roundsOf, roundState, type BlockCard, type Round } from './workoutPlan'

// A complex: its movements done one after the other with the same weight, logged as one row per
// round. Every movement keeps its own sets; a round's inputs write to all of them. Logging is up
// to the parent, which owns the block.
const block = defineModel<BlockCard>({ required: true })
const emit = defineEmits<{ toggle: [round: Round] }>()
const lookups = useLookupsStore()
const exercises = useExercisesStore()

const first = computed(() => block.value.exercises[0])
const rounds = computed(() => roundsOf(block.value))

// v-model.number's rule by hand: an empty or unparsable input is none.
function set(round: Round, field: 'reps' | 'load', value: unknown) {
  const v = typeof value === 'number' ? value : value === '' || value === null || Number.isNaN(Number(value)) ? null : Number(value)
  for (const s of round.sets) s.row[field] = v
}

// Another load unit converts the loads where it can (kg ⇄ lb, % 1RM ⇄ a weight with the first
// movement's 1RM); logged rounds then show as edited until saved in the new unit.
function switchUnit(to: string | null) {
  const unitOf = (id: string | null) => lookups.loadUnits.find((u) => u.id === id)
  const max = exercises.currentMax(first.value.exerciseId)
  const maxLoad = max ? { value: max.load_value, unit: unitOf(max.load_unit) } : null
  const from = unitOf(first.value.loadUnit)
  for (const round of rounds.value) {
    const load = num(round.sets[0].row.load)
    if (load === null) continue
    const converted = convertLoad(load, from, unitOf(to), maxLoad)
    if (converted !== null) set(round, 'load', roundLoad(converted))
  }
  for (const card of block.value.exercises) card.loadUnit = to
}

const variants = { open: 'secondary', edited: 'warning', logged: 'success' } as const

function buttonLabel(round: Round) {
  const state = roundState(round)
  return state === 'open' ? `Log set ${round.number}` : state === 'edited' ? `Save set ${round.number}` : `Undo set ${round.number}`
}

// A further round starts with the values of the last one.
function addRound() {
  const last = rounds.value[rounds.value.length - 1]?.sets[0]?.row
  const number = (last?.number ?? 0) + 1
  for (const card of block.value.exercises) {
    card.rows.push({ number, reps: last?.reps ?? null, load: last?.load ?? null, logged: null, saving: false })
  }
}
</script>
