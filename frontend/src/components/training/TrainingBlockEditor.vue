<template>
  <div class="rounded-lg border border-slate-200 bg-slate-50 p-3">
    <div class="mb-2 flex items-center gap-2">
      <span class="w-14 shrink-0 text-xs font-semibold tracking-wide text-slate-400 uppercase">Block {{ index + 1 }}</span>
      <input
        v-model="block.name"
        type="text"
        maxlength="100"
        placeholder="Name (optional), e.g. Warm-up, Superset"
        :aria-label="`Block ${index + 1} name`"
        :class="[inputClass, 'flex-1 font-medium']"
      />
      <MoveRemoveButtons :index="index" :count="count" :label="`block ${index + 1}`" @move="emit('move', $event)" @remove="emit('remove')" />
    </div>
    <input
      v-model="block.notes"
      type="text"
      placeholder="Notes, e.g. 3 rounds, 90s rest"
      :aria-label="`Block ${index + 1} notes`"
      :class="[inputClass, 'mb-3 w-full']"
    />

    <ol v-if="block.exercises.length" class="mb-3 space-y-2">
      <li v-for="(item, i) in block.exercises" :key="item.key" class="rounded-md border border-slate-200 bg-white p-2">
        <div class="mb-2 flex items-center gap-2">
          <span class="w-5 shrink-0 text-sm font-semibold text-slate-400">{{ i + 1 }}.</span>
          <span class="min-w-0 flex-1 truncate text-sm font-medium text-slate-900">{{ item.name }}</span>
          <ExerciseTypeBadge :type="item.type" />
          <MoveRemoveButtons
            :index="i"
            :count="block.exercises.length"
            :label="item.name"
            @move="move(block.exercises, i, $event)"
            @remove="block.exercises.splice(i, 1)"
          />
        </div>
        <div class="grid grid-cols-[5rem_7rem_1fr] gap-2 pl-7 max-sm:grid-cols-2">
          <label class="text-xs font-medium text-slate-600">
            Sets
            <input v-model.number="item.sets" type="number" min="1" max="100" required :class="[inputClass, 'mt-1 w-full']" />
          </label>
          <label class="text-xs font-medium text-slate-600">
            Reps
            <input
              v-model="item.reps"
              type="text"
              maxlength="30"
              required
              placeholder="8-10, 30s"
              :class="[inputClass, 'mt-1 w-full']"
            />
          </label>
          <label class="text-xs font-medium text-slate-600 max-sm:col-span-2">
            Notes
            <input v-model="item.notes" type="text" placeholder="e.g. slow eccentric" :class="[inputClass, 'mt-1 w-full']" />
          </label>
        </div>
      </li>
    </ol>
    <p v-else class="mb-3 text-xs text-amber-700">Add at least one exercise to this block.</p>

    <SearchSelect :options="exercises" placeholder="Add exercise…" @select="addExercise">
      <template #option-extra="{ option }">
        <span class="text-xs text-slate-400">{{ exerciseTypeLabels[option.type] }}</span>
      </template>
    </SearchSelect>
  </div>
</template>

<script setup lang="ts">
import ExerciseTypeBadge from '../exercise/ExerciseTypeBadge.vue'
import SearchSelect from '../form/SearchSelect.vue'
import MoveRemoveButtons from './MoveRemoveButtons.vue'
import { move, nextKey, type BlockRow } from './trainingForm'
import { exerciseTypeLabels, type Exercise } from '../../types'

defineProps<{ index: number; count: number; exercises: Exercise[] }>()
const emit = defineEmits<{ move: [by: -1 | 1]; remove: [] }>()
// Edited in place; the parent owns the training's rows.
const block = defineModel<BlockRow>({ required: true })

const inputClass =
  'rounded-md border border-slate-300 bg-white px-2 py-1.5 text-sm font-normal text-slate-900 outline-none focus:border-indigo-500 focus:ring-2 focus:ring-indigo-200'

function addExercise(exercise: Exercise) {
  block.value.exercises.push({
    key: nextKey(),
    exercise_id: exercise.id,
    name: exercise.name,
    type: exercise.type,
    sets: 3,
    reps: '',
    notes: '',
  })
}
</script>
