<template>
  <div class="border-t border-slate-100 py-3">
    <div class="flex items-baseline justify-between gap-2">
      <p class="text-sm font-semibold text-slate-800">Score</p>
      <p v-if="wod.result" class="text-sm font-semibold text-emerald-700">
        {{ scoreText(wod.result, wod.intervals) }}
      </p>
    </div>

    <div class="mt-2 flex flex-wrap items-end gap-3">
      <template v-if="wod.scoreType === 'for_time'">
        <label v-if="!wod.draft.capped" class="text-xs font-medium text-slate-500">
          Time
          <DurationInput v-model="wod.draft.time_seconds" label="Time" size="touch" class="mt-1" />
        </label>
        <label v-else-if="!progress" class="text-xs font-medium text-slate-500">
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
      <!-- Scored by the parts logged per movement (WodParts): their total, not typed in. -->
      <p v-else-if="wod.partsTotal !== null && wod.scoreType !== 'emom'" class="text-base text-slate-600">
        <span class="font-semibold text-slate-900">{{ wod.partsTotal }}</span>
        {{ wod.scoreType === 'max_calories' ? 'cal from the intervals' : 'reps from the movements' }}
      </p>
      <label v-else-if="wod.scoreType === 'max_reps'" class="text-xs font-medium text-slate-500">
        Reps
        <TextInput v-model.number="wod.draft.reps" v-bind="countInput" aria-label="Reps" class="mt-1 w-24!" />
      </label>
      <label v-else-if="wod.scoreType === 'max_calories'" class="text-xs font-medium text-slate-500">
        Calories
        <TextInput v-model.number="wod.draft.calories" v-bind="countInput" aria-label="Calories" class="mt-1 w-24!" />
      </label>
      <!-- An EMOM with reps per movement is scored by them; else by the rounds completed. -->
      <p v-else-if="wod.partsTotal !== null" class="text-base text-slate-600">
        <span class="font-semibold text-slate-900">{{ wod.draft.reps }}</span> reps in {{ wod.draft.rounds ?? 0 }}
        <template v-if="wod.intervals">of {{ wod.intervals }}</template> intervals
      </p>
      <label v-else class="text-xs font-medium text-slate-500">
        Rounds completed
        <span class="mt-1 flex items-center gap-2">
          <TextInput v-model.number="wod.draft.rounds" v-bind="countInput" :max="wod.intervals ?? undefined" aria-label="Rounds completed" class="w-20!" />
          <span v-if="wod.intervals" class="text-base text-slate-500">of {{ wod.intervals }}</span>
        </span>
      </label>
    </div>

    <!-- Capped: how far it got; the total reps and the finished exercises follow from the plan. -->
    <div v-if="progress && wod.draft.capped" class="mt-3 space-y-2">
      <div class="flex items-end gap-3">
        <label v-if="wod.rounds > 1" class="text-xs font-medium text-slate-500">
          Full rounds
          <TextInput v-model.number="wod.draft.progress.rounds" v-bind="countInput" :max="wod.rounds - 1" aria-label="Full rounds done" class="mt-1 w-20!" />
        </label>
        <div class="min-w-0 flex-1">
          <SelectField
            v-model="step"
            :options="stepOptions"
            :searchable="false"
            label="Next exercise"
            size="touch"
            option-size="touch"
            aria-label="The exercise it got to"
          />
        </div>
        <label class="text-xs font-medium text-slate-500">
          Reps done
          <TextInput v-model.number="wod.draft.progress.reps" v-bind="countInput" :max="repsMax" aria-label="Reps done of that exercise" class="mt-1 w-20!" />
        </label>
      </div>
      <p class="text-sm text-slate-600">
        <span class="font-semibold text-slate-900">{{ total }} reps</span> · {{ finished }} of {{ wod.rounds * wod.lines.length }} exercises finished
      </p>
    </div>

    <TextInput v-model="wod.draft.notes" size="touch" maxlength="1000" placeholder="Notes" aria-label="Score notes" class="mt-3" />

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
import { computed, watch } from 'vue'
import { ArrowPathIcon, CheckIcon } from '@heroicons/vue/24/outline'
import AppButton from '../button/AppButton.vue'
import AppCheckbox from '../form/AppCheckbox.vue'
import DurationInput from '../form/DurationInput.vue'
import TextInput from '../form/TextInput.vue'
import { scoreText } from '../training/wod'
import SelectField from '../form/SelectField.vue'
import { scoreChanged, takesProgress, progressTotal, type WodCard } from './workoutPlan'
import { clampProgress, finishedSteps } from './wodProgress'

// The score of a WOD block: inputs for its format and a note. The button logs it,
// saves an edit (amber) or undoes it (green), like a set; saving is up to the parent.
const wod = defineModel<WodCard>({ required: true })
const emit = defineEmits<{ toggle: [] }>()

// A capped "for time" takes how far it got instead of the reps: the inputs are clamped to the plan,
// and the reps (the score) follow from them.
const progress = computed(() => takesProgress(wod.value))
const stepOptions = computed(() => wod.value.lines.map((l, i) => ({ id: String(i), name: `${i + 1} · ${l.reps ? `${l.reps} ` : ''}${l.name}` })))
const step = computed({
  get: () => String(wod.value.draft.progress.step),
  set: (v: string | null) => (wod.value.draft.progress.step = Number(v ?? 0)),
})
const clamped = computed(() => clampProgress(wod.value.lines, wod.value.rounds, wod.value.draft.progress))
const repsMax = computed(() => Math.max((wod.value.lines[clamped.value.step]?.repsValue ?? 1) - 1, 0))
const total = computed(() => progressTotal(wod.value))
const finished = computed(() => finishedSteps(wod.value.lines, clamped.value))
watch(
  () => [progress.value && wod.value.draft.capped, total.value] as const,
  ([capped, reps]) => {
    if (capped) wod.value.draft.reps = reps
  },
)

const countInput = { type: 'number', inputmode: 'numeric', min: 0, step: 1, size: 'touch' } as const
</script>
