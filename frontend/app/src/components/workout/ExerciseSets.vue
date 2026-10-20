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
      <AppButton
        v-if="videoUrl"
        :href="videoUrl"
        variant="ghost"
        size="touch"
        icon
        class="-mt-2 -mr-2"
        :aria-label="`Video for ${card.name}`"
      >
        <VideoCameraIcon class="size-6" aria-hidden="true" />
      </AppButton>
    </div>

    <!-- One row per set: reps and load as done, and a button that logs, saves or undoes it. -->
    <div class="mt-2 grid grid-cols-[2rem_1fr_1fr_2.75rem] items-center gap-x-2 gap-y-1.5">
      <span class="text-xs font-medium text-slate-500">Set</span>
      <span class="text-xs font-medium text-slate-500">{{ card.repsUnitName || 'reps' }}{{ card.perSide ? ' / side' : '' }}</span>
      <SelectField
        v-model="card.loadUnit"
        :options="lookups.loadUnits"
        :searchable="false"
        size="sm"
        option-size="touch"
        :aria-label="`${card.label} load unit`"
        field-class="h-7! border-transparent! bg-transparent! pl-0! text-xs! font-medium text-slate-500!"
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
          v-model.number="row.load"
          type="number"
          inputmode="decimal"
          min="0"
          step="any"
          placeholder="–"
          :aria-label="`${card.label} set ${row.number} load`"
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
    <AppButton variant="text" size="touch" class="mt-1 -ml-3" @click="addSet">
      <PlusIcon class="size-4" aria-hidden="true" />
      Add set
    </AppButton>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { ArrowPathIcon, CheckIcon, PlusIcon, VideoCameraIcon } from '@heroicons/vue/24/outline'
import AppButton from '../button/AppButton.vue'
import SelectField from '../form/SelectField.vue'
import TextInput from '../form/TextInput.vue'
import { useExercisesStore } from '../../stores/exercises'
import { useLookupsStore } from '../../stores/lookups'
import { changed, type ExerciseCard, type SetRow } from './workoutPlan'

// An exercise of the workout with its sets. Logging is up to the parent, which owns the card.
const card = defineModel<ExerciseCard>({ required: true })
const emit = defineEmits<{ toggle: [row: SetRow] }>()
const lookups = useLookupsStore()
const exercises = useExercisesStore()

const videoUrl = computed(() => exercises.exercises.find((e) => e.id === card.value.exerciseId)?.video_urls[0])

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
