<template>
  <Dialog :open="open" class="relative z-50" @close="close">
    <div class="fixed inset-0 bg-slate-900/40" aria-hidden="true" />
    <div class="fixed inset-0 flex items-start justify-center overflow-y-auto p-4 sm:pt-[8vh]">
      <DialogPanel class="w-full max-w-2xl rounded-2xl bg-white p-5 shadow-2xl sm:rounded-xl">
        <DialogTitle class="text-base font-semibold text-slate-900">Import exercises</DialogTitle>

        <template v-if="step === 'read'">
          <DialogDescription class="mt-1 text-sm text-slate-600">
            Paste the JSON of your exercises, or choose its file. You see what will be created before anything is saved.
          </DialogDescription>
          <TextInput
            v-model="text"
            multiline
            rows="12"
            spellcheck="false"
            placeholder='{ "format": "exercise-import/1", "exercises": [ { "name": "…" } ] }'
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
            <AppButton :disabled="!text.trim()" @click="read">Continue</AppButton>
          </div>
        </template>

        <!-- What will be created, and what is skipped or left out. -->
        <template v-else>
          <DialogDescription class="mt-1 text-sm text-slate-600">
            {{ toCreate.length }} new {{ toCreate.length === 1 ? 'exercise' : 'exercises' }}<template v-if="skipped">
              · {{ skipped }} already in your library, skipped</template
            >. You can change everything afterwards.
          </DialogDescription>
          <ul class="mt-4 divide-y divide-slate-100 rounded-lg border border-slate-200 text-sm">
            <li v-for="p in plan" :key="p.name" class="px-3 py-2" :class="{ 'text-slate-400': !p.input }">
              <div class="flex flex-wrap items-center gap-2">
                <span class="min-w-0 flex-1 font-medium" :class="p.input ? 'text-slate-900' : 'text-slate-500'">{{ p.name }}</span>
                <ExerciseTypeBadge :type="p.type" />
                <AppBadge v-if="!p.input">Already in your library</AppBadge>
              </div>
              <p v-if="p.input && details(p)" class="mt-0.5 text-xs text-slate-500">{{ details(p) }}</p>
              <p v-if="p.input && p.unknown.length" class="mt-0.5 text-xs text-amber-700">Unknown, left out: {{ p.unknown.join(', ') }}</p>
            </li>
          </ul>
          <div class="mt-5 flex flex-wrap items-center justify-end gap-2">
            <AppButton variant="secondary" :disabled="importing" @click="step = 'read'">Back</AppButton>
            <AppButton :disabled="importing || !toCreate.length" @click="importAll">
              {{ importing ? 'Importing…' : `Import ${toCreate.length} ${toCreate.length === 1 ? 'exercise' : 'exercises'}` }}
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
import { useExercisesStore } from '../../stores/exercises'
import { useLookupsStore } from '../../stores/lookups'
import AppButton from '../button/AppButton.vue'
import TextInput from '../form/TextInput.vue'
import AppBadge from '../ui/AppBadge.vue'
import { importErrorMessage } from '../training/trainingImport'
import ExerciseTypeBadge from './ExerciseTypeBadge.vue'
import { parseExerciseImport, planExerciseImport, type ExerciseImport, type PlannedExercise } from './exerciseImport'

// Reads an exercise import (pasted or from a file), shows what will be created (the exercises the
// user doesn't have yet, their joints, muscle groups and focuses; names nothing matches are left
// out) and creates them one by one.
const open = defineModel<boolean>('open', { required: true })
const emit = defineEmits<{ imported: [count: number] }>()

const store = useExercisesStore()
const lookups = useLookupsStore()

const step = ref<'read' | 'preview'>('read')
const text = ref('')
const error = ref('')
const parsed = ref<ExerciseImport | null>(null)
const plan = ref<PlannedExercise[]>([])
const importing = ref(false)
const fileInput = useTemplateRef<HTMLInputElement>('file')

const toCreate = computed(() => plan.value.filter((p) => p.input))
const skipped = computed(() => plan.value.length - toCreate.value.length)

// E.g. "Hip · Core · HIP".
const details = (p: PlannedExercise) => [...p.joints, ...p.muscleGroups, ...p.focuses].join(' · ')

function close() {
  if (importing.value) return
  open.value = false
  step.value = 'read'
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

async function read() {
  error.value = ''
  try {
    if (!lookups.joints.length) await lookups.loadExerciseLookups()
    if (!store.exercises.length) await store.loadExercises()
    parsed.value = parseExerciseImport(text.value)
    replan()
    step.value = 'preview'
  } catch (e) {
    error.value = e instanceof ApiError ? 'Could not load your exercises and the lists.' : importErrorMessage(e)
  }
}

function replan() {
  if (parsed.value) plan.value = planExerciseImport(parsed.value, store.exercises, lookups)
}

// Creates them one after the other. If one fails, the ones before stay and show as already in the
// library, so trying again doesn't create them twice.
async function importAll() {
  importing.value = true
  let created = 0
  try {
    for (const p of toCreate.value) {
      await store.createExercise(p.input!)
      created++
    }
  } catch (e) {
    toast.error(e instanceof ApiError && e.status === 400 ? e.message : 'Could not create all exercises.')
  } finally {
    importing.value = false
    await store.loadExercises().catch(() => {})
  }
  replan()
  if (created) emit('imported', created)
  if (!toCreate.value.length) {
    text.value = ''
    close()
  }
}
</script>
