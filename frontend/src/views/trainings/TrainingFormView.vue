<template>
  <div class="max-w-4xl">
    <RouterLink
      :to="fromClient ? { name: 'client-edit', params: { id: fromClient } } : { name: 'trainings' }"
      class="mb-4 inline-flex items-center gap-x-1.5 text-sm font-medium text-slate-600 hover:text-indigo-600"
    >
      <ArrowLeftIcon class="size-4" aria-hidden="true" />
      {{ fromClient ? 'Back to client' : 'Back to trainings' }}
    </RouterLink>

    <div class="mb-1 flex flex-wrap items-center justify-between gap-4">
      <h1 class="text-xl font-bold text-slate-900">{{ trainingId ? 'Edit training' : 'New training' }}</h1>
      <div v-if="trainingId" class="flex gap-2">
        <AppButton variant="secondary" size="md" :disabled="saving" @click="duplicateOpen = !duplicateOpen">
          <DocumentDuplicateIcon class="size-4" aria-hidden="true" />
          Duplicate
        </AppButton>
        <AppButton variant="secondary" size="md" :disabled="saving" @click="remove">
          <TrashIcon class="size-4" aria-hidden="true" />
          Delete
        </AppButton>
      </div>
    </div>
    <div v-if="duplicateOpen" class="mt-3 flex flex-wrap items-end gap-2 rounded-lg border border-slate-200 bg-white p-3">
      <div class="min-w-56 flex-1">
        <SelectField v-model="duplicateClientId" :options="clientOptions" label="Copy this training for" placeholder="Select client" />
      </div>
      <AppButton size="md" :disabled="saving || !duplicateClientId" @click="duplicate">Duplicate</AppButton>
      <AppButton variant="secondary" size="md" :disabled="saving" @click="duplicateOpen = false">Cancel</AppButton>
      <p class="w-full text-xs text-slate-500">
        The copy is saved as-is, inactive. Unsaved changes on this page are not included.
      </p>
    </div>
    <p class="mt-1 mb-6 text-sm text-slate-500"><span class="text-red-600">*</span> Required</p>

    <form class="space-y-4" @submit.prevent="submit">
      <div class="grid gap-4 sm:grid-cols-2">
        <SelectField v-model="clientId" :options="clientOptions" label="Client" placeholder="Select client" required />
        <label class="block text-sm font-medium text-slate-800">
          Name<span class="text-red-600" aria-hidden="true"> *</span>
          <input v-model="name" type="text" required maxlength="100" placeholder="e.g. Phase 1" :class="inputClass" />
        </label>
      </div>
      <label class="flex items-start gap-2 text-sm text-slate-800">
        <input v-model="active" type="checkbox" class="mt-0.5 size-4 rounded border-slate-300 accent-indigo-600" />
        <span>
          <span class="font-medium">Active</span>
          <span class="block text-xs text-slate-500">The client's current training. Activating it deactivates their other ones.</span>
        </span>
      </label>
      <label class="block text-sm font-medium text-slate-800">
        Description
        <textarea v-model="description" rows="2" :class="inputClass" />
      </label>

      <section class="space-y-4">
        <h2 class="text-sm font-medium text-slate-800">Units</h2>
        <p
          v-if="!units.length"
          class="rounded-lg border border-dashed border-slate-300 px-4 py-6 text-center text-sm text-slate-500"
        >
          No units yet. A unit is one session, e.g. "Day A", done on weekdays and/or dates.
        </p>
        <div v-for="(unit, u) in units" :key="unit.key" class="rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
          <div class="mb-2 flex items-center gap-2">
            <input
              v-model="unit.name"
              type="text"
              required
              maxlength="100"
              placeholder="Unit name, e.g. Day A"
              :aria-label="`Unit ${u + 1} name`"
              :class="[inputClass, 'mt-0! flex-1 text-base! font-semibold']"
            />
            <MoveRemoveButtons
              :index="u"
              :count="units.length"
              :label="unit.name || `unit ${u + 1}`"
              @move="move(units, u, $event)"
              @remove="removeUnit(u)"
            />
          </div>
          <input
            v-model="unit.notes"
            type="text"
            placeholder="Notes for this unit"
            :aria-label="`Unit ${u + 1} notes`"
            :class="[inputClass, 'mt-0! mb-3']"
          />
          <div class="mb-3">
            <p class="mb-1 text-xs font-medium text-slate-600">When</p>
            <UnitScheduleEditor
              v-model:weekdays="unit.weekdays"
              v-model:dates="unit.dates"
              :label="unit.name || `unit ${u + 1}`"
            />
          </div>

          <div class="space-y-3">
            <TrainingBlockEditor
              v-for="(block, b) in unit.blocks"
              :key="block.key"
              v-model="unit.blocks[b]"
              :index="b"
              :count="unit.blocks.length"
              :exercises="exercises"
              @move="move(unit.blocks, b, $event)"
              @remove="unit.blocks.splice(b, 1)"
            />
          </div>
          <AppButton variant="secondary" size="sm" class="mt-3" @click="unit.blocks.push(newBlock())">
            <PlusIcon class="size-4" aria-hidden="true" />
            Add block
          </AppButton>
        </div>
        <AppButton variant="secondary" size="md" @click="units.push(newUnit(`Day ${String.fromCharCode(65 + units.length)}`))">
          <PlusIcon class="size-4" aria-hidden="true" />
          Add unit
        </AppButton>
      </section>

      <div class="flex items-center justify-end gap-3 border-t border-slate-200 pt-4">
        <p v-if="problem" class="mr-auto text-sm text-amber-700">{{ problem }}</p>
        <AppButton variant="secondary" :disabled="saving" @click="goBack">Cancel</AppButton>
        <AppButton type="submit" :disabled="saving || !!problem">
          {{ saving ? 'Saving…' : trainingId ? 'Save' : 'Create' }}
        </AppButton>
      </div>
    </form>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { toast } from 'vue-sonner'
import { ArrowLeftIcon, DocumentDuplicateIcon, PlusIcon, TrashIcon } from '@heroicons/vue/24/outline'
import {
  ApiError,
  createTraining,
  deleteTraining,
  duplicateTraining,
  getTraining,
  listClients,
  listExercises,
  updateTraining,
} from '../../api'
import AppButton from '../../components/button/AppButton.vue'
import SelectField from '../../components/form/SelectField.vue'
import MoveRemoveButtons from '../../components/training/MoveRemoveButtons.vue'
import TrainingBlockEditor from '../../components/training/TrainingBlockEditor.vue'
import UnitScheduleEditor from '../../components/training/UnitScheduleEditor.vue'
import { loadTrainingLookups } from '../../components/training/lookups'
import {
  blockLetter,
  blockProblem,
  move,
  newBlock,
  newUnit,
  toInput,
  toRows,
  type UnitRow,
} from '../../components/training/trainingForm'
import type { Client, Exercise } from '../../types'

const router = useRouter()
const route = useRoute()
// Set when editing an existing training (route /trainings/:id), absent when creating one.
const trainingId = route.params.id as string | undefined
// Where to go after saving: back to the client when we came from there (?client=…).
const fromClient = typeof route.query.client === 'string' ? route.query.client : null

const inputClass =
  'mt-1 block w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm font-normal outline-none focus:border-indigo-500 focus:ring-2 focus:ring-indigo-200'

const clientId = ref<string | null>(fromClient)
const name = ref('')
const description = ref('')
// New trainings are usually the client's next phase, so they start active.
const active = ref(!trainingId)
const clients = ref<Client[]>([])
const clientOptions = computed(() =>
  clients.value.map((c) => ({ id: c.id, name: `${c.last_name}, ${c.first_name} (${c.email})` })),
)
const duplicateOpen = ref(false)
const duplicateClientId = ref<string | null>(null)
const units = ref<UnitRow[]>(trainingId ? [] : [newUnit('Day A')])
const exercises = ref<Exercise[]>([])
const saving = ref(false)

// Why the training can't be saved yet, if anything; mirrors the server's checks.
const problem = computed(() => {
  if (!clientId.value) return 'Select a client.'
  if (!name.value.trim()) return 'Give the training a name.'
  if (units.value.some((u) => !u.name.trim())) return 'Every unit needs a name.'
  for (const u of units.value) {
    for (const [b, block] of u.blocks.entries()) {
      const p = blockProblem(block)
      if (p) return `${u.name}, block ${blockLetter(b)}: ${p}`
    }
  }
  return ''
})

onMounted(async () => {
  try {
    const [loadedExercises, loadedClients, training] = await Promise.all([
      listExercises(),
      listClients(),
      trainingId ? getTraining(trainingId) : undefined,
      loadTrainingLookups(),
    ])
    exercises.value = loadedExercises
    clients.value = loadedClients
    if (training) {
      clientId.value = training.client_id
      active.value = training.active
      name.value = training.name
      description.value = training.description
      units.value = toRows(training)
    }
  } catch (e) {
    toast.error(e instanceof ApiError && e.status === 404 ? 'Training not found.' : 'Could not load the training.')
  }
})

function removeUnit(index: number) {
  const unit = units.value[index]
  if (unit.blocks.some((b) => b.exercises.length) && !confirm(`Remove "${unit.name}" with all its blocks?`)) return
  units.value.splice(index, 1)
}

async function submit() {
  if (problem.value || !clientId.value) return
  const input = toInput(
    { client_id: clientId.value, name: name.value, description: description.value, active: active.value },
    units.value,
  )
  saving.value = true
  try {
    await (trainingId ? updateTraining(trainingId, input) : createTraining(input))
    toast.success(trainingId ? 'Training saved.' : 'Training created.')
    await goBack()
  } catch (e) {
    toast.error(e instanceof ApiError && e.status === 400 ? e.message : 'Could not save the training.')
  } finally {
    saving.value = false
  }
}

async function remove() {
  if (!trainingId || !confirm(`Delete the training "${name.value}"?`)) return
  saving.value = true
  try {
    await deleteTraining(trainingId)
    toast.success('Training deleted.')
    await goBack()
  } catch {
    toast.error('Could not delete the training.')
  } finally {
    saving.value = false
  }
}

function goBack() {
  return fromClient ? router.push({ name: 'client-edit', params: { id: fromClient } }) : router.push({ name: 'trainings' })
}

async function duplicate() {
  if (!trainingId || !duplicateClientId.value) return
  saving.value = true
  try {
    const copy = await duplicateTraining(trainingId, duplicateClientId.value)
    toast.success('Training duplicated. You are now editing the copy.')
    duplicateOpen.value = false
    duplicateClientId.value = null
    // The layout keys pages by path, so this mounts a fresh form for the copy.
    await router.push({ name: 'training-edit', params: { id: copy.id } })
  } catch (e) {
    toast.error(e instanceof ApiError && e.status === 400 ? e.message : 'Could not duplicate the training.')
  } finally {
    saving.value = false
  }
}
</script>
