<template>
  <div class="rounded-lg border border-slate-200 bg-slate-50 p-3">
    <div class="flex flex-wrap items-center gap-2" :class="{ 'mb-2': !collapsed }">
      <AppButton
        variant="ghost"
        size="sm"
        icon
        :aria-expanded="!collapsed"
        :aria-controls="bodyId"
        :aria-label="`${collapsed ? 'Expand' : 'Collapse'} block ${letter}`"
        @click="collapsed = !collapsed"
      >
        <ChevronRightIcon class="size-5 transition-transform" :class="{ 'rotate-90': !collapsed }" aria-hidden="true" />
      </AppButton>
      <span
        class="flex size-7 shrink-0 items-center justify-center rounded-md bg-indigo-600 text-sm font-bold text-white"
        :title="`Block ${letter}`"
      >
        {{ letter }}
      </span>
      <SegmentedControl v-model="block.kind" :options="kindOptions" :label="`Block ${letter} kind`" size="xs" />
      <TextInput
        v-model="block.name"
        type="text"
        maxlength="100"
        placeholder="Name (optional), e.g. Warm-up"
        :aria-label="`Block ${letter} name`"
        size="sm"
        class="w-auto! min-w-40 flex-1 font-medium!"
      />
      <MoreMenu :items="menuItems" :label="`More actions of block ${letter}`" @select="onMenu" />
      <MoveRemoveButtons :index="index" :count="count" :label="`block ${letter}`" @move="emit('move', $event)" @remove="emit('remove')" />
    </div>
    <p v-if="collapsed" class="mt-1 ml-10 text-xs text-slate-500">
      {{ summary }}
      <span v-if="problem" class="ml-1 text-amber-700">· {{ problem }}</span>
    </p>

    <div v-show="!collapsed" :id="bodyId">
      <!-- A WOD: how it is scored, its time (a cap or the duration), the EMOM's interval and rounds. -->
      <div v-if="kind?.scored" class="mb-3 flex flex-wrap items-end gap-x-4 gap-y-2">
        <fieldset class="min-w-0">
          <legend class="mb-1 text-xs font-medium text-slate-600">Score<span class="text-red-600" aria-hidden="true"> *</span></legend>
          <SegmentedControl v-model="scoreType" :options="scoreOptions" :label="`Block ${letter} score`" size="xs" />
        </fieldset>
        <fieldset v-if="format" class="min-w-0">
          <legend class="mb-1 text-xs font-medium text-slate-600">
            {{ format.timeLabel }}<span v-if="format.timeRequired" class="text-red-600" aria-hidden="true"> *</span>
          </legend>
          <DurationInput v-model="block.time_seconds" :label="`Block ${letter} ${format.timeLabel.toLowerCase()}`" />
        </fieldset>
        <fieldset v-if="format?.hasInterval" class="min-w-0">
          <legend class="mb-1 text-xs font-medium text-slate-600">Every</legend>
          <DurationInput v-model="interval" :label="`Block ${letter} interval`" />
        </fieldset>
        <label v-if="format && !format.hasInterval" class="text-xs font-medium text-slate-600">
          Rounds
          <TextInput
            v-model.number="block.rounds"
            type="number"
            min="1"
            max="100"
            placeholder="—"
            :aria-label="`Block ${letter} rounds (optional)`"
            size="sm"
            class="mt-1 w-16!"
          />
        </label>
      </div>
      <div v-else-if="kind?.shared_sets || kind?.shared_reps || kind?.shared_load" class="mb-3 flex flex-wrap items-end gap-x-4 gap-y-2">
        <label v-if="kind?.shared_sets" class="text-xs font-medium text-slate-600">
          Sets<span class="text-red-600" aria-hidden="true"> *</span>
          <TextInput
            v-model.number="block.sets"
            type="number"
            min="1"
            max="100"
            required
            :aria-label="`Block ${letter} sets`"
            size="sm"
            class="mt-1 w-16!"
          />
        </label>
        <AmountInput v-if="kind?.shared_reps" v-model="block" part="reps" :label="`Block ${letter}`" hint="same for all" />
        <AmountInput
          v-if="kind?.shared_load"
          v-model="block"
          part="load"
          :label="`Block ${letter}`"
          hint="same for all"
          :last="blockLastLoad"
        />
      </div>
      <TextInput
        v-model="block.notes"
        type="text"
        placeholder="Notes, e.g. 90s rest between rounds"
        :aria-label="`Block ${letter} notes`"
        size="sm"
        class="mb-3"
      />

      <ol v-if="block.exercises.length" class="mb-3 space-y-2">
        <li v-for="(item, i) in block.exercises" :key="item.key" class="rounded-md border border-slate-200 bg-white p-2">
          <div class="mb-2 flex items-center gap-2">
            <span class="w-7 shrink-0 text-sm font-semibold text-indigo-600">{{ exerciseLabel(block.exercises.length, index, i) }}</span>
            <span class="min-w-0 flex-1 truncate text-sm font-medium text-slate-900">{{ item.name }}</span>
            <RatingBadge v-if="ratings.get(item.exercise_id)" :rating="ratings.get(item.exercise_id)!" />
            <ExerciseTypeBadge :type="item.type" />
            <MoveRemoveButtons
              :index="i"
              :count="block.exercises.length"
              :label="item.name"
              @move="move(block.exercises, i, $event)"
              @remove="block.exercises.splice(i, 1)"
            />
          </div>
          <div class="flex flex-wrap items-end gap-x-4 gap-y-2 pl-9">
            <label v-if="!kind?.shared_sets" class="text-xs font-medium text-slate-600">
              Sets<span class="text-red-600" aria-hidden="true"> *</span>
              <TextInput
                v-model.number="item.sets"
                type="number"
                min="1"
                max="100"
                required
                :aria-label="`${exerciseLabel(block.exercises.length, index, i)} sets`"
                size="sm"
                class="mt-1 w-16!"
              />
            </label>
            <AmountInput
              v-if="!kind?.shared_reps"
              v-model="block.exercises[i]"
              part="reps"
              :label="exerciseLabel(block.exercises.length, index, i)"
              :optional="kind?.scored"
            />
            <AmountInput
              v-if="!kind?.shared_load"
              v-model="block.exercises[i]"
              part="load"
              :label="exerciseLabel(block.exercises.length, index, i)"
              :last="lastLoads.get(item.exercise_id)"
            />
            <IntensityTargetInput
              v-if="item.type === 'endurance'"
              v-model="block.exercises[i]"
              :label="exerciseLabel(block.exercises.length, index, i)"
            />
            <label class="text-xs font-medium text-slate-600" title="Eccentric, pause, concentric, pause in seconds; X = explosive">
              Tempo
              <TextInput
                v-model="item.tempo"
                type="text"
                maxlength="20"
                placeholder="31X1"
                :aria-label="`${exerciseLabel(block.exercises.length, index, i)} tempo`"
                size="sm"
                class="mt-1 w-20!"
              />
            </label>
            <label class="min-w-40 flex-1 text-xs font-medium text-slate-600">
              Notes
              <TextInput v-model="item.notes" type="text" placeholder="e.g. slow eccentric" size="sm" class="mt-1" />
            </label>
          </div>
        </li>
      </ol>
      <p v-if="problem" class="mb-3 text-xs text-amber-700">{{ problem }}</p>

      <AppButton v-if="!full" variant="secondary" size="sm" @click="pickerOpen = true">
        <PlusIcon class="size-4" aria-hidden="true" />
        Add exercises…
      </AppButton>
    </div>
    <SaveBlockDialog :open="saveOpen" :block="block" @close="saveOpen = false" />
    <ExercisePickerDialog
      :open="pickerOpen"
      :title="`Add exercises to block ${letter}`"
      :in-block="block.exercises.map((e) => e.exercise_id)"
      :limit="remaining"
      @close="pickerOpen = false"
      @add="addExercises"
    />
  </div>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { computed, ref, useId, watch } from 'vue'
import { ChevronRightIcon, PlusIcon } from '@heroicons/vue/24/outline'
import AppButton from '../button/AppButton.vue'
import DurationInput from '../form/DurationInput.vue'
import SegmentedControl from '../form/SegmentedControl.vue'
import TextInput from '../form/TextInput.vue'
import ExerciseTypeBadge from '../exercise/ExerciseTypeBadge.vue'
import RatingBadge from '../exercise/RatingBadge.vue'
import AmountInput from './AmountInput.vue'
import IntensityTargetInput from './IntensityTargetInput.vue'
import ExercisePickerDialog from './ExercisePickerDialog.vue'
import SaveBlockDialog from './SaveBlockDialog.vue'
import MoreMenu from '../ui/MoreMenu.vue'
import MoveRemoveButtons from '../ui/MoveRemoveButtons.vue'
import { blockLetter, blockProblem, exerciseLabel, exerciseRow, move, type BlockRow } from './trainingForm'
import { DEFAULT_INTERVAL, scoreFormat, scoreFormats, wodSummary } from './wod'
import { useExercisesStore } from '../../stores/exercises'
import { useLookupsStore } from '../../stores/lookups'
import { useWorkoutsStore } from '../../stores/workouts'
import type { Exercise, ScoreType } from '../../types'

// A block with its exercises; it folds to its header and a summary line.
const props = defineProps<{
  index: number
  count: number
}>()
const emit = defineEmits<{ move: [by: -1 | 1]; remove: [] }>()
// Edited in place; the parent owns the training's rows.
const block = defineModel<BlockRow>({ required: true })
const lookups = useLookupsStore()
// The user's ratings by exercise id.
const { ratings } = storeToRefs(useExercisesStore())
// The load last logged of each exercise, suggested for the planned loads.
const { lastLoads } = storeToRefs(useWorkoutsStore())

const kindOptions = computed(() => lookups.blockKinds.map((k) => ({ value: k.code, label: k.name, title: k.description })))

const letter = computed(() => blockLetter(props.index))
const kind = computed(() => lookups.kindOf(block.value.kind))
// Switching kinds never drops exercises; a mismatch shows up here instead.
const problem = computed(() => blockProblem(block.value))

const collapsed = ref(false)
const bodyId = useId()
// Shown while folded, e.g. "AMRAP 12:00 · Thruster, Pull-up" or "4 sets · Back Squat".
const summary = computed(() => {
  const b = block.value
  const head = kind.value?.scored ? wodSummary(b, b.rounds) : kind.value?.shared_sets && b.sets ? `${b.sets} sets` : ''
  const names = b.exercises.length ? b.exercises.map((e) => e.name).join(', ') : 'No exercises'
  return head ? `${head} · ${names}` : names
})
// A shared load (a complex) is one weight for all its movements: the first one's last load.
const blockLastLoad = computed(() => block.value.exercises.map((e) => lastLoads.value.get(e.exercise_id)).find(Boolean) ?? null)
const full = computed(() => kind.value?.max_exercises != null && block.value.exercises.length >= kind.value.max_exercises)
// How many more exercises the kind takes; null = no limit.
const remaining = computed(() =>
  kind.value?.max_exercises == null ? null : Math.max(kind.value.max_exercises - block.value.exercises.length, 0),
)

const scoreOptions = scoreFormats.map((f) => ({ value: f.code, label: f.label, title: f.description }))
const format = computed(() => scoreFormat(block.value.score_type))
// The control's model is '' while no format is picked.
const scoreType = computed({
  get: () => block.value.score_type ?? '',
  set: (v: ScoreType | '') => (block.value.score_type = v || null),
})
// An EMOM goes every minute unless set otherwise.
const interval = computed({
  get: () => block.value.interval_seconds ?? DEFAULT_INTERVAL,
  set: (v: number | null) => (block.value.interval_seconds = v),
})
// A block turned into a WOD starts as "for time", the most common format.
watch(
  kind,
  (k) => {
    if (k?.scored && !block.value.score_type) block.value.score_type = 'for_time'
  },
  { immediate: true },
)

const pickerOpen = ref(false)
const saveOpen = ref(false)

// The block's further actions behind its "…" button.
const menuItems = computed(() => [
  {
    id: 'save',
    name: 'Save block',
    disabled: !block.value.exercises.length,
    title: block.value.exercises.length ? 'Save its exercises to reuse them in other sessions' : 'Add exercises first',
  },
])

function onMenu(id: string) {
  if (id === 'save') saveOpen.value = true
}

function addExercises(exercises: Exercise[]) {
  exercises.forEach(addExercise)
  pickerOpen.value = false
}

function addExercise(exercise: Exercise) {
  block.value.exercises.push(exerciseRow(exercise.id, exercise.name, exercise.type, block.value.sets))
}
</script>
