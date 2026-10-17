<template>
  <div class="rounded-lg border border-slate-200 bg-slate-50 p-3">
    <div class="mb-2 flex flex-wrap items-center gap-2">
      <span
        class="flex size-7 shrink-0 items-center justify-center rounded-md bg-indigo-600 text-sm font-bold text-white"
        :title="`Block ${letter}`"
      >
        {{ letter }}
      </span>
      <div role="radiogroup" :aria-label="`Block ${letter} kind`" class="flex rounded-md border border-slate-300 bg-white p-0.5">
        <label
          v-for="k in blockKinds"
          :key="k.code"
          :title="k.description"
          :class="[
            'cursor-pointer rounded px-2.5 py-1 text-xs font-medium has-focus-visible:ring-2 has-focus-visible:ring-indigo-200',
            block.kind === k.code ? 'bg-indigo-600 text-white' : 'text-slate-700 hover:bg-slate-100',
          ]"
        >
          <input v-model="block.kind" type="radio" :name="`block-kind-${block.key}`" :value="k.code" class="sr-only" />
          {{ k.name }}
        </label>
      </div>
      <input
        v-model="block.name"
        type="text"
        maxlength="100"
        placeholder="Name (optional), e.g. Warm-up"
        :aria-label="`Block ${letter} name`"
        :class="[inputClass, 'min-w-40 flex-1 font-medium']"
      />
      <MoveRemoveButtons :index="index" :count="count" :label="`block ${letter}`" @move="emit('move', $event)" @remove="emit('remove')" />
    </div>

    <div v-if="kind?.shared_sets || kind?.shared_reps || kind?.shared_load" class="mb-3 flex flex-wrap items-end gap-x-4 gap-y-2">
      <label v-if="kind?.shared_sets" class="text-xs font-medium text-slate-600">
        Sets<span class="text-red-600" aria-hidden="true"> *</span>
        <input
          v-model.number="block.sets"
          type="number"
          min="1"
          max="100"
          required
          :aria-label="`Block ${letter} sets`"
          :class="[inputClass, 'mt-1 block h-8 w-16']"
        />
      </label>
      <AmountInput v-if="kind?.shared_reps" v-model="block" part="reps" :label="`Block ${letter}`" hint="same for all" />
      <AmountInput v-if="kind?.shared_load" v-model="block" part="load" :label="`Block ${letter}`" hint="same for all" />
    </div>
    <input
      v-model="block.notes"
      type="text"
      placeholder="Notes, e.g. 90s rest between rounds"
      :aria-label="`Block ${letter} notes`"
      :class="[inputClass, 'mb-3 w-full']"
    />

    <ol v-if="block.exercises.length" class="mb-3 space-y-2">
      <li v-for="(item, i) in block.exercises" :key="item.key" class="rounded-md border border-slate-200 bg-white p-2">
        <div class="mb-2 flex items-center gap-2">
          <span class="w-7 shrink-0 text-sm font-semibold text-indigo-600">{{ exerciseLabel(block.exercises.length, index, i) }}</span>
          <span class="min-w-0 flex-1 truncate text-sm font-medium text-slate-900">{{ item.name }}</span>
          <RatingBadge v-if="ratings?.get(item.exercise_id)" :rating="ratings.get(item.exercise_id)!" />
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
            <input
              v-model.number="item.sets"
              type="number"
              min="1"
              max="100"
              required
              :aria-label="`${exerciseLabel(block.exercises.length, index, i)} sets`"
              :class="[inputClass, 'mt-1 block h-8 w-16']"
            />
          </label>
          <AmountInput
            v-if="!kind?.shared_reps"
            v-model="block.exercises[i]"
            part="reps"
            :label="exerciseLabel(block.exercises.length, index, i)"
          />
          <AmountInput
            v-if="!kind?.shared_load"
            v-model="block.exercises[i]"
            part="load"
            :label="exerciseLabel(block.exercises.length, index, i)"
          />
          <label class="text-xs font-medium text-slate-600" title="Eccentric, pause, concentric, pause in seconds; X = explosive">
            Tempo
            <input
              v-model="item.tempo"
              type="text"
              maxlength="20"
              placeholder="31X1"
              :aria-label="`${exerciseLabel(block.exercises.length, index, i)} tempo`"
              :class="[inputClass, 'mt-1 block h-8 w-20']"
            />
          </label>
          <label class="min-w-40 flex-1 text-xs font-medium text-slate-600">
            Notes
            <input v-model="item.notes" type="text" placeholder="e.g. slow eccentric" :class="[inputClass, 'mt-1 block h-8 w-full']" />
          </label>
        </div>
      </li>
    </ol>
    <p v-if="problem" class="mb-3 text-xs text-amber-700">{{ problem }}</p>

    <AppButton v-if="!full" variant="secondary" size="sm" @click="pickerOpen = true">
      <PlusIcon class="size-4" aria-hidden="true" />
      Add exercises…
    </AppButton>
    <ExercisePickerDialog
      :open="pickerOpen"
      :title="`Add exercises to block ${letter}`"
      :exercises="exercises"
      :in-block="block.exercises.map((e) => e.exercise_id)"
      :limit="remaining"
      :ratings="ratings"
      :rated-by="ratedBy"
      @close="pickerOpen = false"
      @add="addExercises"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { PlusIcon } from '@heroicons/vue/24/outline'
import AppButton from '../button/AppButton.vue'
import ExerciseTypeBadge from '../exercise/ExerciseTypeBadge.vue'
import RatingBadge from '../exercise/RatingBadge.vue'
import AmountInput from './AmountInput.vue'
import ExercisePickerDialog from './ExercisePickerDialog.vue'
import MoveRemoveButtons from './MoveRemoveButtons.vue'
import { blockKinds, kindOf } from './lookups'
import { blockLetter, blockProblem, emptyAmount, exerciseLabel, move, nextKey, type BlockRow } from './trainingForm'
import type { Exercise, ExerciseRating } from '../../types'

const props = defineProps<{
  index: number
  count: number
  exercises: Exercise[]
  ratings: Map<string, ExerciseRating> | null // the client's ratings by exercise id; null = no client
  ratedBy: string // the client's name
}>()
const emit = defineEmits<{ move: [by: -1 | 1]; remove: [] }>()
// Edited in place; the parent owns the training's rows.
const block = defineModel<BlockRow>({ required: true })

const inputClass =
  'rounded-md border border-slate-300 bg-white px-2 py-1.5 text-sm font-normal text-slate-900 outline-none focus:border-indigo-500 focus:ring-2 focus:ring-indigo-200'

const letter = computed(() => blockLetter(props.index))
const kind = computed(() => kindOf(block.value.kind))
// Switching kinds never drops exercises; a mismatch shows up here instead.
const problem = computed(() => blockProblem(block.value))
const full = computed(() => kind.value?.max_exercises != null && block.value.exercises.length >= kind.value.max_exercises)
// How many more exercises the kind takes; null = no limit.
const remaining = computed(() =>
  kind.value?.max_exercises == null ? null : Math.max(kind.value.max_exercises - block.value.exercises.length, 0),
)

const pickerOpen = ref(false)

function addExercises(exercises: Exercise[]) {
  exercises.forEach(addExercise)
  pickerOpen.value = false
}

function addExercise(exercise: Exercise) {
  block.value.exercises.push({
    key: nextKey(),
    exercise_id: exercise.id,
    name: exercise.name,
    type: exercise.type,
    sets: block.value.sets,
    tempo: '',
    notes: '',
    ...emptyAmount(),
  })
}
</script>
