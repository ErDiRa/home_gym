<template>
  <div class="py-3">
    <div class="flex items-start gap-2">
      <span class="w-8 shrink-0 pt-0.5 text-sm font-semibold text-indigo-600">{{ card.label }}</span>
      <div class="min-w-0 flex-1">
        <p class="font-semibold text-slate-900">{{ card.name }}</p>
        <p v-if="card.planned" class="text-sm text-slate-600">
          {{ card.planned }}<span v-if="approxLoad" class="font-medium text-slate-800"> ≈ {{ approxLoad }}</span>
        </p>
        <!-- The 1RM loads in % 1RM are calculated from; set or changed in a sheet. -->
        <button
          v-if="!card.bodyweight && (card.percentPlan || max)"
          type="button"
          class="-my-1 min-h-8 text-left text-sm font-medium text-indigo-600 hover:text-indigo-800"
          @click="maxOpen = true"
        >
          {{ max ? (max.estimated ? `≈ 1RM ${formatLoad(max.load_value, max.load_unit_name)} (from your ${repsLabel(max.reps)})` : `1RM ${formatLoad(max.load_value, max.load_unit_name)}`) : 'Set your 1RM to calculate the load' }}
        </button>
        <p v-if="card.tempo || card.notes" class="text-sm text-slate-500">
          <span v-if="card.tempo">Tempo {{ card.tempo }}</span>
          <span v-if="card.tempo && card.notes"> · </span>
          {{ card.notes }}
        </p>
      </div>
      <ExerciseActions :exercise-id="card.exerciseId" :name="card.name" />
      <ExerciseMaxDialog :open="maxOpen" :exercise-id="card.exerciseId" :name="card.name" touch @close="maxOpen = false" />
    </div>

    <!-- One row per set: reps and load as done, and a button that logs, saves or undoes it. A WOD's
         exercises have none (the block has one score), unless sets were logged there before. On
         bodyweight only the reps, until weight is added ("BW + kg"). An endurance exercise also takes
         each set's average as the machine showed it (watts, cal/h or a pace). -->
    <div
      v-if="!card.scored || card.rows.length"
      class="mt-3 grid items-center gap-x-3 gap-y-2"
      :style="{ gridTemplateColumns: columns }"
    >
      <span class="text-center text-xs font-medium text-slate-500">Set</span>
      <span class="pl-3 text-xs font-medium text-slate-500">{{ card.repsUnitName || 'reps' }}{{ card.perSide ? ' / side' : '' }}</span>
      <div v-if="card.showLoad" class="flex min-w-0 items-center gap-1">
        <span v-if="card.bodyweight" class="shrink-0 pl-3 text-xs font-medium text-slate-500">BW +</span>
        <SelectField
          class="min-w-0 flex-1"
          :model-value="card.loadUnit"
          :field-class="`h-7! border-transparent! bg-transparent! ${card.bodyweight ? 'pl-0!' : 'pl-3!'} text-xs! font-medium text-slate-500!`"
          :options="loadUnits"
          :searchable="false"
          size="sm"
          option-size="touch"
          :aria-label="`${card.label} load unit`"
          @update:model-value="switchUnit"
        />
      </div>
      <SelectField
        v-if="card.endurance"
        :model-value="card.intensityMetric"
        :options="intensityMetrics"
        :searchable="false"
        size="sm"
        option-size="touch"
        :aria-label="`${card.label} average in`"
        field-class="h-7! border-transparent! bg-transparent! pl-3! text-xs! font-medium text-slate-500!"
        @update:model-value="switchMetric"
      />
      <span />

      <template v-for="row in card.rows" :key="row.number">
        <span class="text-center text-sm font-semibold text-slate-500">{{ row.number }}</span>
        <TextInput
          v-model.number="row.reps"
          type="number"
          inputmode="decimal"
          min="0"
          step="any"
          :aria-label="`${card.label} set ${row.number} ${card.repsUnitName || 'reps'}`"
          size="touch"
          :tone="row.logged && !changed(card, row) ? 'success' : 'default'"
        />
        <TextInput
          v-if="card.showLoad"
          v-model.number="row.load"
          type="number"
          inputmode="decimal"
          min="0"
          step="any"
          :placeholder="lastLoad === null ? '–' : `${lastLoad}`"
          :title="lastLoad === null ? undefined : `Last time: ${lastLoad} ${unitOf(card.loadUnit)?.name ?? ''}`"
          :aria-label="`${card.label} set ${row.number} load`"
          size="touch"
          :tone="row.logged && !changed(card, row) ? 'success' : 'default'"
        />
        <IntensityInput
          v-if="card.endurance"
          v-model="row.intensity"
          :metric="card.intensityMetric"
          :aria-label="`${card.label} set ${row.number} average ${intensityName(card.intensityMetric)}`"
          size="touch"
          :tone="row.logged && !changed(card, row) ? 'success' : 'default'"
        />
        <AppButton
          size="touch"
          icon
          :variant="!row.logged ? 'secondary' : changed(card, row) ? 'warning' : 'success'"
          :disabled="row.saving"
          :aria-label="buttonLabel(row)"
          :title="buttonLabel(row)"
          :class="{ 'text-slate-400!': !row.logged }"
          @click="emit('toggle', row)"
        >
          <ArrowPathIcon v-if="row.logged && changed(card, row)" class="size-5" aria-hidden="true" />
          <CheckIcon v-else class="size-6" aria-hidden="true" />
        </AppButton>
      </template>
    </div>
    <!-- Logged sets above the 1RM or a rep max (3RM, 5RM…): saved as a new one only when confirmed. -->
    <div v-for="o in card.newMaxes" :key="o.reps" class="mt-2 flex flex-wrap items-center gap-2 rounded-xl bg-green-50 p-3" role="status">
      <TrophyIcon class="size-5 shrink-0 text-green-600" aria-hidden="true" />
      <p class="min-w-0 flex-1 text-sm text-green-900">
        New {{ repsLabel(o.reps) }}: <span class="font-semibold">{{ offerText(o) }}</span>?
        <span v-if="o.reps > 1 && estimateText(o)" class="block text-green-800">≈ {{ estimateText(o) }} 1RM</span>
      </p>
      <div class="flex gap-2">
        <AppButton variant="ghost" size="touch" @click="dismiss(o)">Not now</AppButton>
        <AppButton variant="success" size="touch" :disabled="savingMax" @click="saveNewMax(o)">Save as {{ repsLabel(o.reps) }}</AppButton>
      </div>
    </div>
    <ExerciseNote ref="noteRef" v-model="card" :workout-id="workoutId" />
    <div v-if="!card.scored" class="mt-2 -ml-3 flex flex-wrap">
      <AppButton variant="text" size="touch" @click="addSet">
        <PlusIcon class="size-4" aria-hidden="true" />
        Add set
      </AppButton>
      <AppButton v-if="!card.showLoad" variant="text" size="touch" @click="card.showLoad = true">
        <PlusIcon class="size-4" aria-hidden="true" />
        Add weight
      </AppButton>
      <AppButton v-if="!card.note && !noteRef?.editing" variant="text" size="touch" @click="noteRef?.edit()">
        <ChatBubbleBottomCenterTextIcon class="size-4" aria-hidden="true" />
        Add note
      </AppButton>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, useTemplateRef, watch } from 'vue'
import { ArrowPathIcon, ChatBubbleBottomCenterTextIcon, CheckIcon, PlusIcon, TrophyIcon } from '@heroicons/vue/24/outline'
import { toast } from 'vue-sonner'
import AppButton from '../button/AppButton.vue'
import SelectField from '../form/SelectField.vue'
import IntensityInput from '../form/IntensityInput.vue'
import TextInput from '../form/TextInput.vue'
import ExerciseMaxDialog from '../exercise/ExerciseMaxDialog.vue'
import ExerciseActions from './ExerciseActions.vue'
import ExerciseNote from './ExerciseNote.vue'
import { useExercisesStore } from '../../stores/exercises'
import { useLookupsStore } from '../../stores/lookups'
import { convertLoad, estimateOneRm, formatLoad, repsLabel, roundLoad, type MaxOffer } from '../exercise/maxes'
import { intensityMetrics, intensityName } from '../exercise/intensity'
import { changed, num, suggestedLoad, type ExerciseCard, type SetRow } from './workoutPlan'
import type { IntensityMetric, LoggedSet, WorkoutSet } from '../../types'

// An exercise of the workout with its sets. Logging is up to the parent, which owns the card;
// doneOn is the workout's day, the day a 1RM saved from it was lifted.
const card = defineModel<ExerciseCard>({ required: true })
const props = defineProps<{ doneOn: string; workoutId: string }>()
const noteRef = useTemplateRef<InstanceType<typeof ExerciseNote>>('noteRef')
const emit = defineEmits<{ toggle: [row: SetRow] }>()
const lookups = useLookupsStore()
const exercises = useExercisesStore()

// The 1RM (else an estimate from a rep max), and the load planned in % 1RM as a weight in the
// inputs' unit, e.g. "84–90 kg".
const unitOf = (id: string | null) => lookups.loadUnits.find((u) => u.id === id)
const max = computed(() => exercises.effectiveMax(card.value.exerciseId, unitOf))
const maxOpen = ref(false)
const maxLoad = computed(() => (max.value ? { value: max.value.load_value, unit: unitOf(max.value.load_unit) } : null))
const approxLoad = computed(() => {
  const plan = card.value.percentPlan
  const to = unitOf(card.value.loadUnit)
  const percent = lookups.loadUnits.find((u) => u.percent_of_max)
  if (!plan || !maxLoad.value || !to?.kg_per_unit) return ''
  const weight = (v: number) => roundLoad(convertLoad(v, percent, to, maxLoad.value) ?? 0)
  return `${plan.max === null ? weight(plan.value) : `${weight(plan.value)}–${weight(plan.max)}`} ${to.name}`
})

// Set, reps, load (unless on bodyweight without weight), the average of an endurance exercise, ✓.
const columns = computed(() => ['2rem', '1fr', card.value.showLoad && '1fr', card.value.endurance && '1fr', '2.75rem'].filter(Boolean).join(' '))

// Another metric isn't converted (machines differ): the values not logged yet are cleared, logged
// ones show as edited until saved in it.
function switchMetric(to: string | null) {
  if (!to) return
  card.value.intensityMetric = to as IntensityMetric
  for (const row of card.value.rows) if (!row.logged) row.intensity = null
}

// Weight added to bodyweight is in kg or lb.
const loadUnits = computed(() => (card.value.bodyweight ? lookups.loadUnits.filter((u) => u.kg_per_unit) : lookups.loadUnits))

// The load logged last time, as the placeholder of the load inputs when the plan has none.
const lastLoad = computed(() => suggestedLoad(card.value, (id) => unitOf(id), maxLoad.value))

// Another load unit converts the values in the inputs where it can (kg ⇄ lb, % 1RM ⇄ a weight
// with the 1RM); logged sets then show as edited until saved in the new unit.
function switchUnit(to: string | null) {
  const from = unitOf(card.value.loadUnit)
  for (const row of card.value.rows) {
    const load = num(row.load)
    if (load === null) continue
    const converted = convertLoad(load, from, unitOf(to), maxLoad.value)
    if (converted !== null) row.load = roundLoad(converted)
  }
  card.value.loadUnit = to
}

// A 1RM set while the inputs are still in % 1RM turns them into weights in its unit.
watch(max, (now, before) => {
  if (now && !before && unitOf(card.value.loadUnit)?.percent_of_max) switchUnit(now.load_unit)
})

// An offered max shows its load as logged, e.g. "105 kg", and, for a rep max, the 1RM it estimates.
const offerText = (o: MaxOffer<WorkoutSet | LoggedSet>) =>
  o.set.load_value != null ? formatLoad(o.set.load_value, unitOf(o.set.load_unit)?.name ?? '') : ''
const estimateText = (o: MaxOffer<WorkoutSet | LoggedSet>) =>
  o.set.load_value != null ? formatLoad(roundLoad(estimateOneRm(o.set.load_value, o.reps)), unitOf(o.set.load_unit)?.name ?? '') : ''
const savingMax = ref(false)

const dismiss = (o: MaxOffer<WorkoutSet | LoggedSet>) => (card.value.newMaxes = card.value.newMaxes.filter((x) => x !== o))

async function saveNewMax(o: MaxOffer<WorkoutSet | LoggedSet>) {
  const set = o.set
  if (set.load_value === null || !set.load_unit) return
  savingMax.value = true
  try {
    await exercises.addMax(card.value.exerciseId, {
      reps: o.reps,
      load_value: set.load_value,
      load_unit: set.load_unit,
      achieved_on: props.doneOn,
      workout_set_id: set.id,
      notes: '',
    })
    dismiss(o)
    toast.success(`New ${repsLabel(o.reps)} for ${card.value.name}: ${offerText(o)}.`)
  } catch {
    toast.error(`Could not save the ${repsLabel(o.reps)}.`)
  } finally {
    savingMax.value = false
  }
}

function buttonLabel(row: SetRow) {
  if (!row.logged) return `Log set ${row.number}`
  return changed(card.value, row) ? `Save set ${row.number}` : `Undo set ${row.number}`
}

// A further set starts with the values of the last one.
function addSet() {
  const last = card.value.rows[card.value.rows.length - 1]
  card.value.rows.push({
    number: (last?.number ?? 0) + 1,
    reps: last?.reps ?? null,
    load: last?.load ?? null,
    intensity: last?.intensity ?? null,
    logged: null,
    saving: false,
  })
}
</script>
