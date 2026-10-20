<template>
  <Dialog :open="open" class="relative z-50" @close="close">
    <div class="fixed inset-0 bg-slate-900/40" aria-hidden="true" />
    <div class="fixed inset-0 flex items-start justify-center overflow-y-auto p-4 sm:pt-[8vh]">
      <DialogPanel class="w-full max-w-2xl rounded-2xl bg-white p-5 shadow-2xl sm:rounded-xl">
        <DialogTitle class="text-base font-semibold text-slate-900">Import a training</DialogTitle>

        <template v-if="!parsed">
          <DialogDescription class="mt-1 text-sm text-slate-600">
            Paste the JSON of a training week or choose its file. It opens in the form as a draft, so you can check
            it before saving.
          </DialogDescription>
          <TextInput
            v-model="text"
            multiline
            rows="12"
            spellcheck="false"
            placeholder='{ "format": "training-import/1", … }'
            class="mt-4 font-mono text-xs!"
          />
          <p v-if="error" class="mt-2 text-sm text-red-600">{{ error }}</p>
          <div class="mt-5 flex flex-wrap items-center gap-2">
            <input ref="file" type="file" accept=".json,application/json" class="hidden" @change="readFile" />
            <AppButton variant="secondary" class="mr-auto" @click="fileInput?.click()">
              <ArrowUpTrayIcon class="size-4" aria-hidden="true" />
              Choose file
            </AppButton>
            <AppButton variant="secondary" @click="close">Cancel</AppButton>
            <AppButton :disabled="!text.trim()" @click="read()">Continue</AppButton>
          </div>
        </template>

        <template v-else>
          <DialogDescription class="mt-1 text-sm text-slate-600">
            "{{ parsed.name }}" uses names you don't have yet. Pick the matching exercise or create a new one.
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
                <SegmentedControl
                  v-model="row.type"
                  :options="typeOptions"
                  label="Type"
                  size="xs"
                />
                <div class="grid gap-2 sm:grid-cols-2">
                  <MultiSelect v-model="row.jointIds" label="Joints" required :options="lookups.joints" />
                  <MultiSelect v-model="row.muscleGroupIds" label="Muscle groups" required :options="lookups.muscleGroups" />
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
            <AppButton variant="secondary" :disabled="importing" @click="parsed = null">Back</AppButton>
            <AppButton :disabled="importing || !!rowProblem" @click="finish">
              {{ importing ? 'Importing…' : 'Import' }}
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
import { useExercisesStore } from '../../stores/exercises'
import { useLookupsStore, type AmountPart } from '../../stores/lookups'
import { exerciseTypeLabels, type Exercise, type ExerciseType } from '../../types'
import {
  importErrorMessage,
  nameKey,
  parseImport,
  toImportedTraining,
  unresolvedNames,
  type ImportedTraining,
  type TrainingImport,
} from './trainingImport'

// Reads an import (pasted or from a file), lets the user map or create the exercises it names
// that don't exist yet, creates those and the missing units, and hands the rows to the form.
const open = defineModel<boolean>('open', { required: true })
const emit = defineEmits<{ imported: [training: ImportedTraining] }>()

const exercisesStore = useExercisesStore()
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

const text = ref('')
const error = ref('')
const parsed = ref<TrainingImport | null>(null)
const rows = ref<MappingRow[]>([])
const units = ref<{ part: AmountPart; name: string }[]>([])
const importing = ref(false)
const fileInput = useTemplateRef<HTMLInputElement>('file')

function close() {
  if (importing.value) return
  open.value = false
  parsed.value = null
  error.value = ''
}

async function readFile(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = '' // so choosing the same file again reads it again
  if (!file) return
  text.value = await file.text()
  await read()
}

// Parses the text; without unknown names it is imported right away, else the mapping shows.
async function read() {
  error.value = ''
  try {
    if (!lookups.joints.length) await lookups.loadExerciseLookups()
    const imp = parseImport(
      text.value,
      lookups.blockKinds.map((k) => k.code),
    )
    const missing = unresolvedNames(imp, exercisesStore.exercises)
    if (!missing.exercises.length && !missing.units.length) {
      emitRows(imp)
      return
    }
    rows.value = missing.exercises.map((name) => mappingRow(imp, name))
    units.value = missing.units
    parsed.value = imp
  } catch (e) {
    error.value = e instanceof ApiError ? 'Could not load the joints and muscle groups.' : importErrorMessage(e)
  }
}

// A row for an unknown exercise, prefilled from the import's suggestion for it, if any.
function mappingRow(imp: TrainingImport, name: string): MappingRow {
  const suggestion = imp.new_exercises?.find((n) => nameKey(n.name) === nameKey(name))
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
    if (r.choice === 'new' && (!r.jointIds.length || !r.muscleGroupIds.length))
      return `${r.name} needs at least one joint and muscle group.`
  }
  return ''
})

// Creates the missing units and exercises, then hands over the rows. If something fails half
// way, what was created is kept and matched by name from then on.
async function finish() {
  if (!parsed.value || rowProblem.value) return
  importing.value = true
  const imp = parsed.value
  const byName = new Map<string, Exercise>(exercisesStore.exercises.map((e) => [nameKey(e.name), e]))
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
      byName.set(nameKey(r.name), created)
    }
    await exercisesStore.loadExercises()
    importing.value = false
    emitRows(imp, byName)
  } catch (e) {
    importing.value = false
    toast.error(e instanceof ApiError && e.status === 400 ? e.message : 'Could not create the exercises and units.')
    await exercisesStore.loadExercises().catch(() => {})
    // Leave out what was created before the failure, so the next try doesn't create it twice.
    const missing = unresolvedNames(imp, exercisesStore.exercises)
    rows.value = rows.value.filter((r) => missing.exercises.some((n) => nameKey(n) === nameKey(r.name)))
    units.value = missing.units
  }
}

function emitRows(imp: TrainingImport, byName = new Map(exercisesStore.exercises.map((e) => [nameKey(e.name), e]))) {
  emit('imported', toImportedTraining(imp, byName))
  text.value = ''
  close()
}
</script>
