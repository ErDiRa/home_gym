<template>
  <div class="border-t border-slate-100 py-3">
    <div class="flex items-baseline justify-between gap-2">
      <p class="text-sm font-semibold text-slate-800">Score</p>
      <p v-if="wod.result" class="text-sm font-semibold text-emerald-700">
        {{ scoreText(wod.result, wod.intervals) }} · {{ wod.result.rx ? 'Rx' : 'Scaled' }}
      </p>
    </div>

    <div class="mt-2 flex flex-wrap items-end gap-3">
      <template v-if="wod.scoreType === 'for_time'">
        <label v-if="!wod.draft.capped" class="text-xs font-medium text-slate-500">
          Time
          <DurationInput v-model="wod.draft.time_seconds" label="Time" size="touch" class="mt-1" />
        </label>
        <label v-else class="text-xs font-medium text-slate-500">
          Reps done
          <TextInput v-model.number="wod.draft.reps" v-bind="countInput" aria-label="Reps done" class="mt-1 w-24!" />
        </label>
        <AppCheckbox v-model="wod.draft.capped" class="pb-3">Time capped</AppCheckbox>
      </template>
      <template v-else-if="wod.scoreType === 'amrap'">
        <label class="text-xs font-medium text-slate-500">
          Rounds
          <TextInput v-model.number="wod.draft.rounds" v-bind="countInput" aria-label="Rounds" class="mt-1 w-20!" />
        </label>
        <span class="pb-3 font-semibold text-slate-400" aria-hidden="true">+</span>
        <label class="text-xs font-medium text-slate-500">
          Reps
          <TextInput v-model.number="wod.draft.reps" v-bind="countInput" placeholder="0" aria-label="Extra reps" class="mt-1 w-20!" />
        </label>
      </template>
      <label v-else-if="wod.scoreType === 'max_reps'" class="text-xs font-medium text-slate-500">
        Reps
        <TextInput v-model.number="wod.draft.reps" v-bind="countInput" aria-label="Reps" class="mt-1 w-24!" />
      </label>
      <label v-else-if="wod.scoreType === 'max_calories'" class="text-xs font-medium text-slate-500">
        Calories
        <TextInput v-model.number="wod.draft.calories" v-bind="countInput" aria-label="Calories" class="mt-1 w-24!" />
      </label>
      <label v-else class="text-xs font-medium text-slate-500">
        Rounds completed
        <span class="mt-1 flex items-center gap-2">
          <TextInput v-model.number="wod.draft.rounds" v-bind="countInput" :max="wod.intervals ?? undefined" aria-label="Rounds completed" class="w-20!" />
          <span v-if="wod.intervals" class="text-base text-slate-500">of {{ wod.intervals }}</span>
        </span>
      </label>
    </div>

    <SegmentedControl v-model="rx" :options="rxOptions" label="Done as prescribed or scaled" class="mt-3" />
    <TextInput v-model="wod.draft.notes" size="touch" maxlength="1000" placeholder="Notes, e.g. what was scaled" aria-label="Score notes" class="mt-3" />

    <AppButton
      size="touch"
      :variant="!wod.result ? 'secondary' : scoreChanged(wod) ? 'warning' : 'success'"
      :disabled="wod.saving"
      :title="!wod.result ? undefined : scoreChanged(wod) ? undefined : 'Undo the score'"
      class="mt-3 w-full"
      @click="emit('toggle')"
    >
      <ArrowPathIcon v-if="wod.result && scoreChanged(wod)" class="size-5" aria-hidden="true" />
      <CheckIcon v-else class="size-5" aria-hidden="true" />
      {{ !wod.result ? 'Log score' : scoreChanged(wod) ? 'Save score' : 'Score logged · undo' }}
    </AppButton>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { ArrowPathIcon, CheckIcon } from '@heroicons/vue/24/outline'
import AppButton from '../button/AppButton.vue'
import AppCheckbox from '../form/AppCheckbox.vue'
import DurationInput from '../form/DurationInput.vue'
import SegmentedControl from '../form/SegmentedControl.vue'
import TextInput from '../form/TextInput.vue'
import { scoreText } from '../training/wod'
import { scoreChanged, type WodCard } from './workoutPlan'

// The score of a WOD block: inputs for its format, Rx or scaled, and a note. The button logs it,
// saves an edit (amber) or undoes it (green), like a set; saving is up to the parent.
const wod = defineModel<WodCard>({ required: true })
const emit = defineEmits<{ toggle: [] }>()

const countInput = { type: 'number', inputmode: 'numeric', min: 0, step: 1, size: 'touch' } as const
const rxOptions = [
  { value: 'rx', label: 'Rx', title: 'Done as prescribed' },
  { value: 'scaled', label: 'Scaled', title: 'Load, reps or movements scaled' },
]
const rx = computed({
  get: () => (wod.value.draft.rx ? 'rx' : 'scaled'),
  set: (v: string) => (wod.value.draft.rx = v === 'rx'),
})
</script>
