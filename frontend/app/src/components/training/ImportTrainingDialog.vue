<template>
  <Dialog :open="open" class="relative z-50" @close="close">
    <div class="fixed inset-0 bg-slate-900/40" aria-hidden="true" />
    <div class="fixed inset-0 flex items-start justify-center overflow-y-auto p-4 sm:pt-[8vh]">
      <DialogPanel class="w-full max-w-2xl rounded-2xl bg-white p-5 shadow-2xl sm:rounded-xl">
        <DialogTitle class="text-base font-semibold text-slate-900">
          {{ file?.kind === 'cycle' ? 'Import a cycle' : 'Import' }}
        </DialogTitle>

        <template v-if="step === 'read'">
          <DialogDescription class="mt-1 text-sm text-slate-600">
            Paste the JSON of a training week or a whole cycle, or choose its file. A training opens in the form as a
            draft; a cycle is shown before it is imported.
          </DialogDescription>
          <TextInput
            v-model="text"
            multiline
            rows="12"
            spellcheck="false"
            placeholder='{ "format": "training-import/1", … } or { "format": "cycle-import/1", … }'
            class="mt-4 font-mono text-xs!"
          />
          <p v-if="error" class="mt-2 text-sm text-red-600">{{ error }}</p>
          <div class="mt-5 flex flex-wrap items-center gap-2">
            <input ref="file" type="file" accept=".json,application/json" class="hidden" @change="readFile" />
            <!-- On phones the file button takes its own line, Cancel and Continue the next one. -->
            <AppButton variant="secondary" class="max-sm:w-full sm:mr-auto" @click="fileInput?.click()">
              <ArrowUpTrayIcon class="size-4" aria-hidden="true" />
              Choose file
            </AppButton>
            <AppButton variant="secondary" class="max-sm:ml-auto" @click="close">Cancel</AppButton>
            <AppButton :disabled="!text.trim()" @click="read()">Continue</AppButton>
          </div>
        </template>

        <template v-else-if="step === 'map'">
          <DialogDescription class="mt-1 text-sm text-slate-600">
            "{{ file?.imp.name }}" uses names you don't have yet. Pick the matching exercise or create a new one.
          </DialogDescription>
          <ul class="mt-4 space-y-3">
            <li v-for="row in rows" :key="row.name" class="rounded-lg border border-slate-200 p-3">
              <div class="flex flex-wrap items-center justify-between gap-2">
                <span class="font-medium text-slate-900">{{ row.name }}</span>
                <SegmentedControl
                  v-model="row.choice"
                  :options="[
                    { value: 'existing', label: 'Use existing' },
                    { value: 'new', label: 'Create new' },
                  ]"
                  :label="`What to do with ${row.name}`"
                  size="xs"
                />
              </div>
              <SelectField
                v-if="row.choice === 'existing'"
                v-model="row.exerciseId"
                :options="exercisesStore.exercises"
                :aria-label="`Exercise for ${row.name}`"
                placeholder="Select an exercise…"
                class="mt-2"
              />
              <div v-else class="mt-2 space-y-2">
                <SegmentedControl v-model="row.type" :options="typeOptions" label="Type" size="xs" />
                <div class="grid gap-2 sm:grid-cols-2">
                  <MultiSelect v-model="row.jointIds" label="Joints" :options="lookups.joints" />
                  <MultiSelect v-model="row.muscleGroupIds" label="Muscle groups" :options="lookups.muscleGroups" />
                </div>
              </div>
            </li>
          </ul>
          <p v-if="units.length" class="mt-3 text-sm text-slate-600">
            New units, created on import:
            {{ units.map((u) => `${u.name} (${u.part})`).join(', ') }}
          </p>

          <div class="mt-5 flex flex-wrap items-center justify-end gap-2">
            <p v-if="rowProblem" class="mr-auto text-sm text-amber-700">{{ rowProblem }}</p>
            <AppButton variant="secondary" :disabled="importing" @click="step = 'read'">Back</AppButton>
            <AppButton :disabled="importing || !!rowProblem" @click="finish">
              {{ importing ? 'Creating…' : file?.kind === 'cycle' ? 'Continue' : 'Import' }}
            </AppButton>
          </div>
        </template>

        <template v-else-if="step === 'preview' && file?.kind === 'cycle'">
          <DialogDescription class="mt-1 text-sm text-slate-600">
            The cycle and all its trainings are created at once. You can change everything afterwards.
          </DialogDescription>
          <div class="mt-4 space-y-4">
            <div>
              <p class="font-medium text-slate-900">{{ file.imp.name }}</p>
              <p class="text-sm text-slate-600">
                {{ formatPeriod(file.imp.start_date, cycleEnd) }} · {{ file.imp.weeks }}
                {{ file.imp.weeks === 1 ? 'week' : 'weeks' }}
              </p>
            </div>
            <div v-if="file.imp.goals?.length">
              <h3 class="text-xs font-semibold tracking-wide text-slate-500 uppercase">Goals</h3>
              <ol class="mt-1 list-decimal space-y-0.5 pl-5 text-sm text-slate-700">
                <li v-for="(g, i) in file.imp.goals" :key="i">
                  {{ g.title }}<span v-if="g.target" class="text-slate-500"> · {{ g.target }}</span>
                </li>
              </ol>
            </div>
            <div>
              <h3 class="text-xs font-semibold tracking-wide text-slate-500 uppercase">
                Trainings ({{ previewTrainings.length }})
              </h3>
              <ul class="mt-1 divide-y divide-slate-100 rounded-lg border border-slate-200 text-sm">
                <li v-for="t in previewTrainings" :key="t.key" class="flex flex-wrap items-baseline gap-x-3 px-3 py-1.5">
                  <span class="w-16 shrink-0 text-xs text-slate-500">Week {{ t.week }}</span>
                  <span class="min-w-0 flex-1 font-medium text-slate-800">{{ t.name }}</span>
                  <span class="text-xs text-slate-500">{{ t.period }} · {{ t.sessions }} sessions</span>
                </li>
              </ul>
              <p v-if="!previewTrainings.length" class="mt-1 text-sm text-slate-500">
                No trainings yet: add them to the cycle later.
              </p>
            </div>
          </div>
          <div class="mt-5 flex flex-wrap items-center justify-end gap-2">
            <p v-if="previewProblem" class="mr-auto text-sm text-amber-700">{{ previewProblem }}</p>
            <AppButton variant="secondary" :disabled="importing" @click="step = 'read'">Back</AppButton>
            <AppButton :disabled="importing || !request" @click="importCycle">
              {{ importing ? 'Importing…' : 'Import cycle' }}
            </AppButton>
          </div>
        </template>
      </DialogPanel>
    </div>
  </Dialog>
</template>

<script setup lang="ts">
import { computed, ref, useTemplateRef } from 'vue'
import { Dialog, DialogDescription, DialogPanel, DialogTitle } from '@headlessui/vue'
import { ArrowUpTrayIcon } from '@heroicons/vue/24/outline'
import { toast } from 'vue-sonner'
import { ApiError } from '../../api'
import AppButton from '../button/AppButton.vue'
import MultiSelect from '../form/MultiSelect.vue'
import SegmentedControl from '../form/SegmentedControl.vue'
import SelectField from '../form/SelectField.vue'
import TextInput from '../form/TextInput.vue'
import { addDays, formatPeriod } from '../../dates'
import { useCyclesStore } from '../../stores/cycles'
import { useExercisesStore } from '../../stores/exercises'
import { useLookupsStore, type AmountPart } from '../../stores/lookups'
import { exerciseTypeLabels, type CycleImportInput, type Exercise, type ExerciseType, type ImportedCycle } from '../../types'
import { mergedNewExercises, parseAnyImport, toCycleImportInput, trainingStart, type AnyImport } from '../cycle/cycleImport'
import {
  importErrorMessage,
  nameKey,
  toImportedTraining,
  unresolvedNames,
  type ImportedTraining,
  type ImportNewExercise,
  type WithSessions,
} from './trainingImport'

// Reads an import (pasted or from a file): a single training, handed to the form as rows, or a
// whole cycle, previewed and then imported at once. Either way it first lets the user map or
// create the exercises it names that don't exist yet, and creates those and the missing units.
const open = defineModel<boolean>('open', { required: true })
const emit = defineEmits<{ imported: [training: ImportedTraining]; cycleImported: [cycle: ImportedCycle] }>()

const exercisesStore = useExercisesStore()
const cyclesStore = useCyclesStore()
const lookups = useLookupsStore()
const typeOptions = (Object.keys(exerciseTypeLabels) as ExerciseType[]).map((t) => ({
  value: t,
  label: exerciseTypeLabels[t],
}))

interface MappingRow {
  name: string
  choice: 'existing' | 'new'
  exerciseId: string | null
  type: ExerciseType
  jointIds: string[]
  muscleGroupIds: string[]
  videoUrls: string[] // from the suggestion, kept for a new exercise
}

const step = ref<'read' | 'map' | 'preview'>('read')
const text = ref('')
const error = ref('')
const file = ref<AnyImport | null>(null)
const rows = ref<MappingRow[]>([])
const units = ref<{ part: AmountPart; name: string }[]>([])
const request = ref<CycleImportInput | null>(null) // the cycle import, ready to send
const previewProblem = ref('')
const importing = ref(false)
const fileInput = useTemplateRef<HTMLInputElement>('file')

// The trainings of the file and its suggestions for new exercises.
const trainingsOf = (f: AnyImport): WithSessions[] => (f.kind === 'training' ? [f.imp] : (f.imp.trainings ?? []))
const suggestionsOf = (f: AnyImport): ImportNewExercise[] =>
  f.kind === 'training' ? (f.imp.new_exercises ?? []) : mergedNewExercises(f.imp)
const knownExercises = () => new Map<string, Exercise>(exercisesStore.exercises.map((e) => [nameKey(e.name), e]))

function close() {
  if (importing.value) return
  open.value = false
  step.value = 'read'
  file.value = null
  error.value = ''
}

async function readFile(event: Event) {
  const input = event.target as HTMLInputElement
  const picked = input.files?.[0]
  input.value = '' // so choosing the same file again reads it again
  if (!picked) return
  text.value = await picked.text()
  await read()
}

// Parses the text; without unknown names it goes on right away, else the mapping shows.
async function read() {
  error.value = ''
  try {
    if (!lookups.joints.length) await lookups.loadExerciseLookups()
    const parsed = parseAnyImport(
      text.value,
      lookups.blockKinds.map((k) => k.code),
    )
    file.value = parsed
    const missing = unresolvedNames(trainingsOf(parsed), exercisesStore.exercises)
    if (!missing.exercises.length && !missing.units.length) {
      proceed(parsed, knownExercises())
      return
    }
    const suggestions = suggestionsOf(parsed)
    rows.value = missing.exercises.map((name) => mappingRow(suggestions, name))
    units.value = missing.units
    step.value = 'map'
  } catch (e) {
    error.value = e instanceof ApiError ? 'Could not load the joints and muscle groups.' : importErrorMessage(e)
  }
}

// A row for an unknown exercise, prefilled from the import's suggestion for it, if any.
function mappingRow(suggestions: ImportNewExercise[], name: string): MappingRow {
  const suggestion = suggestions.find((n) => nameKey(n.name) === nameKey(name))
  const idsOf = (names: string[] | undefined, options: { id: string; name: string }[]) =>
    options.filter((o) => names?.some((n) => nameKey(n) === nameKey(o.name))).map((o) => o.id)
  return {
    name,
    choice: 'new',
    exerciseId: null,
    type: suggestion?.type && suggestion.type in exerciseTypeLabels ? suggestion.type : 'strength',
    jointIds: idsOf(suggestion?.joints, lookups.joints),
    muscleGroupIds: idsOf(suggestion?.muscle_groups, lookups.muscleGroups),
    videoUrls: (suggestion?.video_urls ?? []).filter((u) => typeof u === 'string').slice(0, 10),
  }
}

const rowProblem = computed(() => {
  for (const r of rows.value) {
    if (r.choice === 'existing' && !r.exerciseId) return `Pick an exercise for ${r.name}.`
  }
  return ''
})

// Creates the missing units and exercises, then goes on. If something fails half way, what was
// created is kept and matched by name from then on.
async function finish() {
  if (!file.value || rowProblem.value) return
  importing.value = true
  const parsed = file.value
  const byName = knownExercises()
  // The ids of the exercises created here, by name; the list reloaded after has them in full.
  const createdIds = new Map<string, string>()
  try {
    for (const u of units.value) await lookups.addUnit(u.part, u.name)
    for (const r of rows.value) {
      if (r.choice === 'existing') {
        byName.set(nameKey(r.name), exercisesStore.exercises.find((e) => e.id === r.exerciseId)!)
        continue
      }
      const created = await exercisesStore.createExercise({
        name: r.name,
        type: r.type,
        description: '',
        video_urls: r.videoUrls,
        joint_ids: r.jointIds,
        muscle_group_ids: r.muscleGroupIds,
        focus_ids: [],
      })
      createdIds.set(nameKey(r.name), created.id)
    }
    await exercisesStore.loadExercises()
    for (const [key, id] of createdIds) byName.set(key, exercisesStore.exercises.find((e) => e.id === id)!)
    importing.value = false
    proceed(parsed, byName)
  } catch (e) {
    importing.value = false
    toast.error(e instanceof ApiError && e.status === 400 ? e.message : 'Could not create the exercises and units.')
    await exercisesStore.loadExercises().catch(() => {})
    // Leave out what was created before the failure, so the next try doesn't create it twice.
    const missing = unresolvedNames(trainingsOf(parsed), exercisesStore.exercises)
    rows.value = rows.value.filter((r) => missing.exercises.some((n) => nameKey(n) === nameKey(r.name)))
    units.value = missing.units
  }
}

// Once every name is known: a training goes to the form, a cycle to its preview.
function proceed(parsed: AnyImport, byName: Map<string, Exercise>) {
  if (parsed.kind === 'training') {
    emit('imported', toImportedTraining(parsed.imp, byName))
    text.value = ''
    close()
    return
  }
  const built = toCycleImportInput(parsed.imp, byName)
  request.value = built.input
  previewProblem.value = built.problem
  step.value = 'preview'
}

// The last Sunday of the cycle and its trainings with their dates, for the preview.
const cycleEnd = computed(() =>
  file.value?.kind === 'cycle' ? addDays(file.value.imp.start_date, file.value.imp.weeks * 7 - 1) : '',
)
const previewTrainings = computed(() => {
  if (file.value?.kind !== 'cycle') return []
  const c = file.value.imp
  return (c.trainings ?? []).map((t, i) => {
    const start = trainingStart(c.start_date, t.week)
    return {
      key: i,
      week: t.week,
      name: t.name,
      period: formatPeriod(start, addDays(start, (t.weeks ?? 1) * 7 - 1)),
      sessions: t.sessions.length,
    }
  })
})

async function importCycle() {
  if (!request.value) return
  importing.value = true
  try {
    const cycle = await cyclesStore.importCycle(request.value)
    importing.value = false
    text.value = ''
    close()
    emit('cycleImported', cycle)
  } catch (e) {
    importing.value = false
    // A broken training names its week, e.g. 'Week 7 "…": Session 2, block B: …'.
    toast.error(e instanceof ApiError && e.status === 400 ? e.message : 'Could not import the cycle.')
  }
}
</script>
