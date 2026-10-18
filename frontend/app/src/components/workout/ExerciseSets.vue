<template>
  <div class="py-3">
    <div class="flex items-start gap-2">
      <span class="w-8 shrink-0 pt-0.5 text-sm font-semibold text-indigo-600">{{ card.label }}</span>
      <div class="min-w-0 flex-1">
        <p class="font-semibold text-slate-900">{{ card.name }}</p>
        <p v-if="card.planned" class="text-sm text-slate-600">{{ card.planned }}</p>
        <p v-if="card.tempo || card.notes" class="text-sm text-slate-500">
          <span v-if="card.tempo">Tempo {{ card.tempo }}</span>
          <span v-if="card.tempo && card.notes"> · </span>
          {{ card.notes }}
        </p>
      </div>
      <a
        v-if="videoUrl"
        :href="videoUrl"
        target="_blank"
        rel="noopener noreferrer"
        class="-mt-2 -mr-2 flex size-11 shrink-0 items-center justify-center rounded-xl text-slate-500 active:bg-slate-100"
        :aria-label="`Video for ${card.name}`"
      >
        <VideoCameraIcon class="size-6" aria-hidden="true" />
      </a>
    </div>

    <!-- One row per set: reps and load as done, and a button that logs, saves or undoes it. -->
    <div class="mt-2 grid grid-cols-[2rem_1fr_1fr_2.75rem] items-center gap-x-2 gap-y-1.5">
      <span class="text-xs font-medium text-slate-500">Set</span>
      <span class="text-xs font-medium text-slate-500">{{ card.repsUnitName || 'reps' }}{{ card.perSide ? ' / side' : '' }}</span>
      <select
        v-model="card.loadUnit"
        class="h-7 w-full rounded-md border-0 bg-transparent p-0 text-xs font-medium text-slate-500"
        :aria-label="`${card.label} load unit`"
      >
        <option v-for="u in lookups.loadUnits" :key="u.id" :value="u.id">{{ u.name }}</option>
      </select>
      <span />

      <template v-for="row in card.rows" :key="row.number">
        <span class="text-center text-sm font-semibold text-slate-500">{{ row.number }}</span>
        <input
          v-model.number="row.reps"
          type="number"
          inputmode="decimal"
          min="0"
          step="any"
          :aria-label="`${card.label} set ${row.number} ${card.repsUnitName || 'reps'}`"
          :class="[inputClass, rowClass(row)]"
        />
        <input
          v-model.number="row.load"
          type="number"
          inputmode="decimal"
          min="0"
          step="any"
          placeholder="–"
          :aria-label="`${card.label} set ${row.number} load`"
          :class="[inputClass, rowClass(row)]"
        />
        <button
          type="button"
          :disabled="row.saving"
          :aria-label="buttonLabel(row)"
          :title="buttonLabel(row)"
          :class="[
            'flex size-11 items-center justify-center rounded-xl border transition-colors disabled:opacity-50',
            !row.logged
              ? 'border-slate-300 bg-white text-slate-400 active:bg-slate-100'
              : changed(card, row)
                ? 'border-amber-500 bg-amber-500 text-white active:bg-amber-600'
                : 'border-emerald-500 bg-emerald-500 text-white active:bg-emerald-600',
          ]"
          @click="emit('toggle', row)"
        >
          <ArrowPathIcon v-if="row.logged && changed(card, row)" class="size-5" aria-hidden="true" />
          <CheckIcon v-else class="size-6" aria-hidden="true" />
        </button>
      </template>
    </div>
    <button
      type="button"
      class="mt-1 flex min-h-11 items-center gap-1.5 rounded-xl px-2 text-sm font-medium text-indigo-600 active:bg-indigo-50"
      @click="addSet"
    >
      <PlusIcon class="size-4" aria-hidden="true" />
      Add set
    </button>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { ArrowPathIcon, CheckIcon, PlusIcon, VideoCameraIcon } from '@heroicons/vue/24/outline'
import { useExercisesStore } from '../../stores/exercises'
import { useLookupsStore } from '../../stores/lookups'
import { changed, type ExerciseCard, type SetRow } from './workoutPlan'

// An exercise of the workout with its sets. Logging is up to the parent, which owns the card.
const card = defineModel<ExerciseCard>({ required: true })
const emit = defineEmits<{ toggle: [row: SetRow] }>()
const lookups = useLookupsStore()
const exercises = useExercisesStore()

// text-base keeps iOS from zooming into the inputs.
const inputClass =
  'h-11 w-full min-w-0 rounded-xl border px-3 text-base text-slate-900 outline-none focus:border-indigo-500 focus:ring-2 focus:ring-indigo-200'

const videoUrl = computed(() => exercises.exercises.find((e) => e.id === card.value.exerciseId)?.video_urls[0])

const rowClass = (row: SetRow) => (row.logged && !changed(card.value, row) ? 'border-emerald-300 bg-emerald-50' : 'border-slate-300 bg-white')

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
    logged: null,
    saving: false,
  })
}
</script>
