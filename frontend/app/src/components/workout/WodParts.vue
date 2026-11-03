<template>
  <div class="border-t border-slate-100 py-3">
    <p class="text-sm font-semibold text-slate-800">Reps per movement</p>

    <!-- Editing: one round at a time, an input per movement done in it. -->
    <template v-if="editing">
      <nav class="mt-2 flex items-center gap-1" aria-label="Rounds">
        <AppButton variant="ghost" size="touch" icon aria-label="Previous round" :disabled="round === 1" @click="round--">
          <ChevronLeftIcon class="size-5" aria-hidden="true" />
        </AppButton>
        <p class="flex-1 text-center text-sm font-semibold text-slate-900" aria-live="polite">Round {{ round }} of {{ roundCount }}</p>
        <AppButton variant="ghost" size="touch" icon aria-label="Next round" :disabled="round === roundCount" @click="round++">
          <ChevronRightIcon class="size-5" aria-hidden="true" />
        </AppButton>
      </nav>
      <ul class="mt-2 space-y-2">
        <li v-for="i in emomLinesIn(wod, round)" :key="wod.lines[i].key" class="flex items-center gap-3">
          <label :for="inputId(i)" class="min-w-0 flex-1 text-base text-slate-900">
            <span v-if="wod.lines[i].reps" class="font-semibold">{{ wod.lines[i].reps }}</span> {{ wod.lines[i].name }}
          </label>
          <TextInput
            :id="inputId(i)"
            v-model.number="draft[round - 1][i]"
            type="number"
            inputmode="numeric"
            min="0"
            step="1"
            size="touch"
            :placeholder="wod.lines[i].repsUnit === 'reps' && wod.lines[i].repsValue !== null ? String(wod.lines[i].repsValue) : 'reps'"
            class="w-20!"
          />
        </li>
      </ul>
      <!-- The sum of all rounds, below the inputs it adds up. -->
      <p class="mt-2 flex items-center gap-3 border-t border-slate-100 pt-2 text-base">
        <span class="flex-1 font-semibold text-slate-800">Total</span>
        <span class="w-20 pl-3 font-semibold text-slate-900">{{ repsTotal(draft) }}</span>
      </p>
      <AppButton v-if="roundCount > 1" variant="text" size="touch" class="mt-2 -ml-3" @click="copyToOthers">
        <DocumentDuplicateIcon class="size-5" aria-hidden="true" />
        Copy to the other rounds
      </AppButton>
      <div class="mt-3 space-y-2">
        <AppButton size="touch" class="w-full" :disabled="saving" @click="saveReps">
          {{ saving ? 'Saving…' : 'Save' }}
        </AppButton>
        <AppButton variant="secondary" size="touch" class="w-full" :disabled="saving" @click="editing = false">Cancel</AppButton>
      </div>
    </template>

    <!-- Logged: each movement's reps by round and its sum, then the sum of all. -->
    <template v-else>
      <template v-if="hasLogged">
        <ul class="mt-2 space-y-2 text-sm" aria-label="Reps by movement">
          <li v-for="(line, i) in wod.lines" :key="line.key" class="flex gap-3">
            <span class="min-w-0 flex-1 truncate text-slate-700">{{ line.name }}</span>
            <span class="text-slate-500">{{ logged.map((r) => r[i] ?? '–').join(' · ') }}</span>
            <span class="w-10 text-right font-semibold text-slate-900">{{ lineTotal(i) }}</span>
          </li>
        </ul>
        <p class="mt-2 flex gap-3 border-t border-slate-100 pt-2 text-sm">
          <span class="flex-1 font-semibold text-slate-800">Total</span>
          <span class="w-10 text-right font-semibold text-slate-900">{{ repsTotal(logged) }}</span>
        </p>
      </template>
      <AppButton variant="secondary" size="touch" class="mt-3 w-full" @click="edit">
        <PencilSquareIcon class="size-5" aria-hidden="true" />
        {{ hasLogged ? 'Edit reps' : 'Enter reps per movement' }}
      </AppButton>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, useId } from 'vue'
import { ChevronLeftIcon, ChevronRightIcon, DocumentDuplicateIcon, PencilSquareIcon } from '@heroicons/vue/24/outline'
import { toast } from 'vue-sonner'
import AppButton from '../button/AppButton.vue'
import TextInput from '../form/TextInput.vue'
import { emomLinesIn, emomReps, emomRoundCount, repsTotal, type ExerciseCard, type WodCard } from './workoutPlan'

// The reps done of each movement of an EMOM, round by round (e.g. the max reps of a 20 s station),
// counted up into the score's total. Entered in an edit mode that pages through the rounds; save
// (the parent's: logging the sets) resolves to whether it worked, which ends the edit mode.
const props = defineProps<{ wod: WodCard; cards: ExerciseCard[]; save: (grid: (number | null)[][]) => Promise<boolean> }>()

const editing = ref(false)
const saving = ref(false)
const round = ref(1)
const draft = ref<(number | null)[][]>([])
const idPrefix = useId()
const inputId = (line: number) => `${idPrefix}-${line}`

const roundCount = computed(() => emomRoundCount(props.wod))
const logged = computed(() => emomReps(props.wod, props.cards))
const hasLogged = computed(() => logged.value.some((r) => r.some((v) => v !== null)))
const lineTotal = (line: number) => logged.value.reduce((sum, r) => sum + (r[line] ?? 0), 0)

// Starts from what was logged, at the first round with a gap.
function edit() {
  draft.value = logged.value.map((r) => [...r])
  const open = draft.value.findIndex((r, n) => emomLinesIn(props.wod, n + 1).some((i) => r[i] === null))
  round.value = open === -1 ? 1 : open + 1
  editing.value = true
}

// The values of the round shown go to every other round (the movements done in it).
function copyToOthers() {
  const source = draft.value[round.value - 1]
  draft.value.forEach((r, n) => {
    if (n === round.value - 1) return
    for (const i of emomLinesIn(props.wod, n + 1)) r[i] = source[i]
  })
  toast.success(`Copied round ${round.value} to the other rounds.`)
}

async function saveReps() {
  saving.value = true
  try {
    if (await props.save(draft.value)) editing.value = false
  } finally {
    saving.value = false
  }
}
</script>
