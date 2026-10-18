<template>
  <div class="max-w-4xl">
    <RouterLink
      :to="{ name: 'trainings' }"
      class="mb-4 inline-flex items-center gap-x-1.5 text-sm font-medium text-slate-600 hover:text-indigo-600"
    >
      <ArrowLeftIcon class="size-4" aria-hidden="true" />
      Back to trainings
    </RouterLink>

    <div class="mb-1 flex flex-wrap items-center justify-between gap-4">
      <h1 class="text-xl font-bold text-slate-900">{{ trainingId ? 'Edit training' : 'New training' }}</h1>
      <div v-if="trainingId" class="flex gap-2">
        <AppButton
          variant="secondary"
          size="md"
          :disabled="saving"
          title="Copies the saved training as the next phase: inactive, starting the day after this one ends"
          @click="duplicate"
        >
          <DocumentDuplicateIcon class="size-4" aria-hidden="true" />
          Duplicate
        </AppButton>
        <AppButton variant="secondary" size="md" :disabled="saving" @click="remove">
          <TrashIcon class="size-4" aria-hidden="true" />
          Delete
        </AppButton>
      </div>
    </div>
    <p class="mt-1 mb-6 text-sm text-slate-500"><span class="text-red-600">*</span> Required</p>

    <form class="space-y-4" @submit.prevent="submit">
      <div class="grid gap-4 sm:grid-cols-[1fr_auto]">
        <label class="block text-sm font-medium text-slate-800">
          Name<span class="text-red-600" aria-hidden="true"> *</span>
          <input v-model="name" type="text" required maxlength="100" placeholder="e.g. Phase 1" :class="inputClass" />
        </label>
        <label class="block text-sm font-medium text-slate-800">
          Sessions per week<span class="text-red-600" aria-hidden="true"> *</span>
          <input
            v-model.number="sessionsPerWeek"
            type="number"
            min="1"
            max="14"
            required
            :class="[inputClass, 'sm:w-36']"
          />
          <span class="mt-1 block text-xs font-normal text-slate-500">Done on any days; units in turn.</span>
        </label>
      </div>
      <fieldset>
        <legend class="text-sm font-medium text-slate-800">Period<span class="text-red-600" aria-hidden="true"> *</span></legend>
        <div class="mt-1 flex flex-wrap items-end gap-3">
          <label class="text-xs font-medium text-slate-600">
            Start
            <input
              :value="startDate"
              type="date"
              required
              :class="[inputClass, 'w-44!']"
              @change="setStart(($event.target as HTMLInputElement).value)"
            />
          </label>
          <div class="flex gap-1" role="group" aria-label="Duration">
            <button
              v-for="w in durationWeeks"
              :key="w"
              type="button"
              :disabled="!startDate"
              :aria-pressed="weeks === w"
              :class="
                weeks === w
                  ? 'border-indigo-600 bg-indigo-600 text-white'
                  : 'border-slate-300 bg-white text-slate-600 hover:bg-slate-100 disabled:opacity-50'
              "
              class="h-[38px] rounded-md border px-3 text-xs font-semibold"
              @click="setWeeks(w)"
            >
              {{ w }} weeks
            </button>
          </div>
          <label class="text-xs font-medium text-slate-600">
            End
            <input v-model="endDate" type="date" required :min="startDate || undefined" :class="[inputClass, 'w-44!']" />
          </label>
        </div>
        <p v-if="periodLength" class="mt-1 text-xs text-slate-500">{{ periodLength }}</p>
      </fieldset>
      <label class="flex items-start gap-2 text-sm text-slate-800">
        <input v-model="active" type="checkbox" class="mt-0.5 size-4 rounded border-slate-300 accent-indigo-600" />
        <span>
          <span class="font-medium">Active</span>
          <span class="block text-xs text-slate-500">The training the tracker shows. Activating it deactivates your other ones.</span>
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
          No units yet. A unit is one session, e.g. "Day A"; the tracker suggests them in turn.
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
          <div class="space-y-3">
            <TrainingBlockEditor
              v-for="(block, b) in unit.blocks"
              :key="block.key"
              v-model="unit.blocks[b]"
              :index="b"
              :count="unit.blocks.length"
              @move="move(unit.blocks, b, $event)"
              @remove="unit.blocks.splice(b, 1)"
            />
          </div>
          <AppButton variant="secondary" size="sm" class="mt-3" @click="unit.blocks.push(newBlock())">
            <PlusIcon class="size-4" aria-hidden="true" />
            Add block
          </AppButton>
        </div>
        <!-- New units are named after the next letter: Day A, Day B... -->
        <AppButton variant="secondary" size="md" @click="units.push(newUnit(`Day ${String.fromCharCode(65 + units.length)}`))">
          <PlusIcon class="size-4" aria-hidden="true" />
          Add unit
        </AppButton>
      </section>

      <!-- Sticks to the bottom of the screen so saving doesn't need scrolling down. -->
      <div class="sticky bottom-0 z-10 flex items-center justify-end gap-3 border-t border-slate-200 bg-slate-50/95 py-4 backdrop-blur">
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
import { ApiError } from '../../api'
import AppButton from '../../components/button/AppButton.vue'
import MoveRemoveButtons from '../../components/training/MoveRemoveButtons.vue'
import TrainingBlockEditor from '../../components/training/TrainingBlockEditor.vue'
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
import { addDays, daysBetween, nextMonday } from '../../dates'
import { useExercisesStore } from '../../stores/exercises'
import { useLookupsStore } from '../../stores/lookups'
import { useTrainingsStore } from '../../stores/trainings'

const router = useRouter()
const route = useRoute()
const trainingsStore = useTrainingsStore()
const exercisesStore = useExercisesStore()
const lookups = useLookupsStore()
// Set when editing an existing training (route /trainings/:id), absent when creating one.
const trainingId = route.params.id as string | undefined

const inputClass =
  'mt-1 block w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm font-normal outline-none focus:border-indigo-500 focus:ring-2 focus:ring-indigo-200'

const name = ref('')
const description = ref('')
const sessionsPerWeek = ref(3)
// New trainings are usually the next phase, so they start active, next Monday, for 4 weeks.
const active = ref(!trainingId)
const durationWeeks = [4, 6, 8, 12]
const startDate = ref(trainingId ? '' : nextMonday())
const endDate = ref(trainingId ? '' : addDays(startDate.value, 4 * 7 - 1))
// Days in the period, both ends included; 0 while it's incomplete or reversed.
const periodDays = computed(() =>
  startDate.value && endDate.value ? Math.max(daysBetween(startDate.value, endDate.value) + 1, 0) : 0,
)
// The duration button matching the period, if any.
const weeks = computed(() => (periodDays.value % 7 === 0 ? periodDays.value / 7 : null))
const periodLength = computed(() => {
  const days = periodDays.value
  if (!days) return ''
  return days % 7 === 0 ? `${days / 7} ${days === 7 ? 'week' : 'weeks'}` : `${days} ${days === 1 ? 'day' : 'days'}`
})
const units = ref<UnitRow[]>(trainingId ? [] : [newUnit('Day A')])
const saving = ref(false)

// A new start moves the end along, keeping the period's length.
function setStart(value: string) {
  if (value && startDate.value && endDate.value) endDate.value = addDays(endDate.value, daysBetween(startDate.value, value))
  startDate.value = value
}

function setWeeks(w: number) {
  endDate.value = addDays(startDate.value, w * 7 - 1)
}

// Why the training can't be saved yet, if anything; mirrors the server's checks.
const problem = computed(() => {
  if (!name.value.trim()) return 'Give the training a name.'
  const spw = sessionsPerWeek.value
  if (!Number.isInteger(spw) || spw < 1 || spw > 14) return 'Sessions per week must be 1-14.'
  if (!startDate.value || !endDate.value) return 'Set the start and end date.'
  if (endDate.value < startDate.value) return "The end date can't be before the start date." // YYYY-MM-DD compares as text
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
    const [training] = await Promise.all([
      trainingId ? trainingsStore.getTraining(trainingId) : undefined,
      exercisesStore.loadExercises(),
      exercisesStore.loadRatings(),
      lookups.loadTrainingLookups(),
    ])
    if (training) {
      active.value = training.active
      sessionsPerWeek.value = training.sessions_per_week
      startDate.value = training.start_date
      endDate.value = training.end_date
      name.value = training.name
      description.value = training.description
      units.value = toRows(training)
    }
  } catch (e) {
    toast.error(e instanceof ApiError && e.status === 404 ? 'Training not found.' : 'Could not load the training.')
  }
})

// Asks first only when the unit has exercises that would be lost.
function removeUnit(index: number) {
  const unit = units.value[index]
  if (unit.blocks.some((b) => b.exercises.length) && !confirm(`Remove "${unit.name}" with all its blocks?`)) return
  units.value.splice(index, 1)
}

async function submit() {
  if (problem.value) return
  const input = toInput(
    {
      name: name.value,
      description: description.value,
      active: active.value,
      start_date: startDate.value,
      end_date: endDate.value,
      sessions_per_week: sessionsPerWeek.value,
    },
    units.value,
  )
  saving.value = true
  try {
    if (trainingId) {
      // Stay on the page so further changes can follow.
      await trainingsStore.updateTraining(trainingId, input)
      toast.success('Training saved.')
    } else {
      await trainingsStore.createTraining(input)
      toast.success('Training created.')
      await goBack()
    }
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
    await trainingsStore.deleteTraining(trainingId)
    toast.success('Training deleted.')
    await goBack()
  } catch {
    toast.error('Could not delete the training.')
  } finally {
    saving.value = false
  }
}

function goBack() {
  return router.push({ name: 'trainings' })
}

// Copies the saved training (unsaved changes on the page are not included).
async function duplicate() {
  if (!trainingId) return
  saving.value = true
  try {
    const copy = await trainingsStore.duplicateTraining(trainingId)
    toast.success('Training duplicated. You are now editing the copy.')
    // The layout keys pages by path, so this mounts a fresh form for the copy.
    await router.push({ name: 'training-edit', params: { id: copy.id } })
  } catch (e) {
    toast.error(e instanceof ApiError && e.status === 400 ? e.message : 'Could not duplicate the training.')
  } finally {
    saving.value = false
  }
}
</script>
