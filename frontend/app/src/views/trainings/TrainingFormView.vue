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
      <AppButton v-else variant="secondary" size="md" :disabled="saving" @click="importOpen = true">
        <ArrowUpTrayIcon class="size-4" aria-hidden="true" />
        Import JSON
      </AppButton>
    </div>
    <ImportTrainingDialog
      v-if="!trainingId"
      v-model:open="importOpen"
      @imported="applyImport"
      @cycle-imported="openImportedCycle"
    />
    <p class="mt-1 mb-6 text-sm text-slate-500"><span class="text-red-600">*</span> Required</p>

    <form class="space-y-4" @submit.prevent="submit">
      <label class="block text-sm font-medium text-slate-800">
        Name<span class="text-red-600" aria-hidden="true"> *</span>
        <TextInput v-model="name" type="text" required maxlength="100" placeholder="e.g. Phase 1" class="mt-1" />
      </label>
      <fieldset>
        <legend class="text-sm font-medium text-slate-800">Period<span class="text-red-600" aria-hidden="true"> *</span></legend>
        <div class="mt-1 flex flex-wrap items-end gap-3">
          <label class="text-xs font-medium text-slate-600">
            Start (a Monday)
            <TextInput
              :model-value="startDate"
              type="date"
              required
              class="mt-1 w-44!"
              @change="setStart(($event.target as HTMLInputElement).value)"
            />
          </label>
          <SegmentedControl
            :model-value="weekCount"
            :options="durationWeeks.map((w) => ({ value: w, label: `${w} weeks` }))"
            label="Duration"
            @update:model-value="setWeekCount(Number($event))"
          />
          <label class="text-xs font-medium text-slate-600">
            Weeks
            <TextInput
              :model-value="weekCount"
              type="number"
              min="1"
              :max="MAX_WEEKS"
              required
              class="mt-1 w-24!"
              @change="setWeekCount(Number(($event.target as HTMLInputElement).value))"
            />
          </label>
        </div>
        <p v-if="endDate" class="mt-1 text-xs text-slate-500">
          Ends on {{ formatDate(endDate, { weekday: true }) }}. Sessions are
          done on any days of their week.
        </p>
      </fieldset>
      <div>
        <SelectField
          v-model="cycleField"
          :options="cycleOptions"
          label="Cycle"
          placeholder="No cycle"
          class="max-w-md"
        />
        <p v-if="cycleHint" class="mt-1 text-xs text-slate-500">{{ cycleHint }}</p>
        <p v-if="outsideCycle" class="mt-1 text-xs text-amber-700">{{ outsideCycle }}</p>
      </div>
      <AppCheckbox
        v-model="active"
        description="Shown in the tracker. Several trainings can be active, e.g. strength, warm-ups and mobility side by side."
      >
        Active
      </AppCheckbox>
      <label class="block text-sm font-medium text-slate-800">
        Description
        <TextInput v-model="description" multiline rows="2" class="mt-1" />
      </label>

      <section class="space-y-4" aria-labelledby="sessions-heading">
        <div>
          <h2 id="sessions-heading" class="text-sm font-medium text-slate-800">Sessions</h2>
          <p class="text-xs text-slate-500">
            The training's own sessions. Added from the library they are copies: changing one here or there doesn't change the other.
          </p>
        </div>
        <p
          v-if="!sessions.length"
          class="rounded-lg border border-dashed border-slate-300 px-4 py-6 text-center text-sm text-slate-500"
        >
          No sessions yet. Add a new one or one from your library.
        </p>
        <SessionEditor
          v-for="(session, i) in sessions"
          :key="session.key"
          v-model="sessions[i]"
          :index="i"
          :count="sessions.length"
          :label="`Session ${i + 1}`"
          collapsible
          :start-collapsed="!!session.id"
          @move="move(sessions, i, $event)"
          @remove="removeSession(i)"
        />
        <div class="flex flex-wrap items-center gap-2">
          <AppButton variant="secondary" size="md" @click="sessions.push(newSession(`Session ${sessions.length + 1}`))">
            <PlusIcon class="size-4" aria-hidden="true" />
            New session
          </AppButton>
          <ActionMenu v-if="library.sessions.length" :items="library.sessions" :disabled="copying" @select="addFromLibrary">
            <BookOpenIcon class="size-4" aria-hidden="true" />
            From library
          </ActionMenu>
        </div>
      </section>

      <section class="space-y-2" aria-labelledby="weeks-heading">
        <div>
          <h2 id="weeks-heading" class="text-sm font-medium text-slate-800">Weeks</h2>
          <p class="text-xs text-slate-500">Plan every week with up to {{ MAX_SLOTS }} of the sessions; the same one may appear more than once.</p>
        </div>
        <WeekPlanner v-model="weeks" :sessions="sessions" :start-date="startDate" />
      </section>

      <!-- Always in reach, so saving doesn't need scrolling down. -->
      <FormActions :problem="problem">
        <AppButton variant="secondary" :disabled="saving" @click="goBack">Cancel</AppButton>
        <AppButton type="submit" :disabled="saving || !!problem">
          {{ saving ? 'Saving…' : trainingId ? 'Save' : 'Create' }}
        </AppButton>
      </FormActions>
    </form>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { toast } from 'vue-sonner'
import { ArrowLeftIcon, ArrowUpTrayIcon, BookOpenIcon, DocumentDuplicateIcon, PlusIcon, TrashIcon } from '@heroicons/vue/24/outline'
import { ApiError } from '../../api'
import AppButton from '../../components/button/AppButton.vue'
import FormActions from '../../components/form/FormActions.vue'
import AppCheckbox from '../../components/form/AppCheckbox.vue'
import SegmentedControl from '../../components/form/SegmentedControl.vue'
import SelectField from '../../components/form/SelectField.vue'
import TextInput from '../../components/form/TextInput.vue'
import ActionMenu from '../../components/ui/ActionMenu.vue'
import ImportTrainingDialog from '../../components/training/ImportTrainingDialog.vue'
import SessionEditor from '../../components/training/SessionEditor.vue'
import type { ImportedTraining } from '../../components/training/trainingImport'
import type { ImportedCycle } from '../../types'
import WeekPlanner from '../../components/training/WeekPlanner.vue'
import {
  MAX_SLOTS,
  move,
  newSession,
  newWeek,
  nextKey,
  sessionProblem,
  toInput,
  toRows,
  toSessionRow,
  type SessionRow,
  type WeekRow,
} from '../../components/training/trainingForm'
import { confirmDialog } from '../../confirm'
import { addDays, formatDate, formatPeriod, isMonday, nextMonday } from '../../dates'
import { useCyclesStore } from '../../stores/cycles'
import { useExercisesStore } from '../../stores/exercises'
import { useLookupsStore } from '../../stores/lookups'
import { useSessionsStore } from '../../stores/sessions'
import { useTrainingsStore } from '../../stores/trainings'

const router = useRouter()
const route = useRoute()
const trainingsStore = useTrainingsStore()
const exercisesStore = useExercisesStore()
const lookups = useLookupsStore()
const library = useSessionsStore()
const cyclesStore = useCyclesStore()
// Set when editing an existing training (route /trainings/:id), absent when creating one.
const trainingId = route.params.id as string | undefined

const MAX_WEEKS = 52
const durationWeeks = [4, 6, 8, 12]

const name = ref('')
const description = ref('')
// The cycle the training belongs to; a new one starts in the cycle it was created from (?cycle=).
const cycleId = ref<string | null>(trainingId ? null : ((route.query.cycle as string | undefined) ?? null))
// Set by an import naming a cycle that doesn't exist.
const cycleHint = ref('')
// New trainings are usually the next phase, so they start active, next Monday, for 4 weeks,
// with one session planned once a week.
const active = ref(!trainingId)
const startDate = ref(trainingId ? '' : nextMonday())
const firstSession = newSession('Session 1')
const sessions = ref<SessionRow[]>(trainingId ? [] : [firstSession])
const weeks = ref<WeekRow[]>(
  trainingId ? [] : Array.from({ length: 4 }, () => newWeek([{ key: nextKey(), sessionKey: firstSession.key }])),
)
const weekCount = computed(() => weeks.value.length)
// The last Sunday of the weeks.
const endDate = computed(() => (startDate.value && weekCount.value ? addDays(startDate.value, weekCount.value * 7 - 1) : ''))
const saving = ref(false)
const copying = ref(false)

// The select holds '' for "No cycle", as its options need an id.
const cycleOptions = computed(() => [{ id: '', name: 'No cycle' }, ...cyclesStore.cycles.map((c) => ({ id: c.id, name: c.name }))])
const cycleField = computed({
  get: () => cycleId.value ?? '',
  set: (id: string | null) => {
    cycleId.value = id || null
    cycleHint.value = ''
  },
})
// A training may run outside its cycle's weeks (the cycle can still be moved), so this only warns.
const outsideCycle = computed(() => {
  const cycle = cyclesStore.cycles.find((c) => c.id === cycleId.value)
  if (!cycle || !startDate.value || !endDate.value) return ''
  if (startDate.value >= cycle.start_date && endDate.value <= cycle.end_date) return ''
  return `Runs outside the cycle's weeks (${formatPeriod(cycle.start_date, cycle.end_date)}).`
})
const importOpen = ref(false)

// Any date picks the Monday of its week, as trainings run over whole weeks.
function setStart(value: string) {
  if (!value) {
    startDate.value = ''
    return
  }
  const weekday = (new Date(`${value}T12:00`).getDay() + 6) % 7 // 0 = Monday
  startDate.value = addDays(value, -weekday)
}

// More weeks start with the plan of the last one; fewer drop the last weeks' plans.
async function setWeekCount(n: number) {
  if (!Number.isInteger(n) || n < 1 || n > MAX_WEEKS) return
  const list = weeks.value
  if (
    n < list.length &&
    list.slice(n).some((w) => w.slots.length) &&
    !(await confirmDialog({
      title: `Shorten to ${n} ${n === 1 ? 'week' : 'weeks'}?`,
      message:
        n + 1 === list.length ? `The plan of week ${list.length} is dropped.` : `The plans of weeks ${n + 1} to ${list.length} are dropped.`,
      confirmLabel: 'Shorten',
      danger: true,
    }))
  )
    return
  while (list.length > n) list.pop()
  while (list.length < n) {
    const last = list[list.length - 1]
    list.push(newWeek((last?.slots ?? []).map((s) => ({ key: nextKey(), sessionKey: s.sessionKey }))))
  }
}

// Why the training can't be saved yet, if anything; mirrors the server's checks.
const problem = computed(() => {
  if (!name.value.trim()) return 'Give the training a name.'
  if (!startDate.value) return 'Set the start date.'
  if (!isMonday(startDate.value)) return 'The training must start on a Monday.'
  if (!weekCount.value) return 'Set the number of weeks.'
  for (const s of sessions.value) {
    const p = sessionProblem(s)
    if (p) return p
  }
  const tooFull = weeks.value.findIndex((w) => w.slots.length > MAX_SLOTS)
  if (tooFull >= 0) return `Week ${tooFull + 1} has more than ${MAX_SLOTS} sessions.`
  return ''
})

onMounted(async () => {
  try {
    const [training] = await Promise.all([
      trainingId ? trainingsStore.getTraining(trainingId) : undefined,
      exercisesStore.loadExercises(),
      exercisesStore.loadRatings(),
      lookups.loadTrainingLookups(),
      library.loadSessions(),
      cyclesStore.loadCycles(),
    ])
    if (training) {
      cycleId.value = training.cycle_id
      active.value = training.active
      startDate.value = training.start_date
      name.value = training.name
      description.value = training.description
      ;({ sessions: sessions.value, weeks: weeks.value } = toRows(training))
    }
  } catch (e) {
    toast.error(e instanceof ApiError && e.status === 404 ? 'Training not found.' : 'Could not load the training.')
    return
  }
  // The trainings list links here with ?import=1 to start with the import.
  if (!trainingId && route.query.import) importOpen.value = true
})

// Replaces the draft with an imported training (one week); asks first if the draft was edited.
async function applyImport(imported: ImportedTraining) {
  const edited = name.value.trim() || description.value.trim() || sessions.value.some((s) => s.blocks.some((b) => b.exercises.length))
  if (
    edited &&
    !(await confirmDialog({
      title: 'Replace the draft?',
      message: 'The imported training replaces what you entered so far.',
      confirmLabel: 'Replace',
    }))
  )
    return
  name.value = imported.name
  description.value = imported.description
  startDate.value = imported.startDate
  sessions.value = imported.sessions
  weeks.value = imported.weeks
  // A cycle named in the file is preselected; without one, the cycle it was opened from stays.
  if (imported.cycle) {
    const cycle = cyclesStore.cycles.find((c) => c.name.trim().toLowerCase() === imported.cycle!.toLowerCase())
    cycleId.value = cycle?.id ?? null
    cycleHint.value = cycle ? '' : `No cycle called "${imported.cycle}". Pick one or leave it empty.`
  }
  toast.success('Imported. Check the training, then create it.')
}

// A whole cycle was imported (nothing to do in this form): show it.
async function openImportedCycle(cycle: ImportedCycle) {
  const n = cycle.trainings.length
  toast.success(`Imported ${cycle.name} with ${n} ${n === 1 ? 'training' : 'trainings'}.`)
  await router.push({ name: 'cycle-edit', params: { id: cycle.id } })
}

// Copies a template of the library into the training, as a new session of its own.
async function addFromLibrary(id: string) {
  copying.value = true
  try {
    sessions.value.push(toSessionRow(await library.getSession(id), false))
  } catch {
    toast.error('Could not load the session.')
  } finally {
    copying.value = false
  }
}

// Removing a session also takes it out of the weeks; asks first when that loses something.
async function removeSession(index: number) {
  const session = sessions.value[index]
  const planned = weeks.value.some((w) => w.slots.some((s) => s.sessionKey === session.key))
  const hasExercises = session.blocks.some((b) => b.exercises.length)
  if (
    (planned || hasExercises) &&
    !(await confirmDialog({
      title: `Remove "${session.name || 'Unnamed session'}"?`,
      message: planned ? 'It is also taken out of the weeks it is planned in.' : 'Its blocks and exercises are removed with it.',
      confirmLabel: 'Remove',
      danger: true,
    }))
  )
    return
  // Found again by key, in case the list changed while the dialog was open.
  sessions.value = sessions.value.filter((s) => s.key !== session.key)
  for (const w of weeks.value) w.slots = w.slots.filter((s) => s.sessionKey !== session.key)
}

async function submit() {
  if (problem.value) return
  const input = toInput(
    {
      cycle_id: cycleId.value,
      name: name.value,
      description: description.value,
      active: active.value,
      start_date: startDate.value,
      weeks: weekCount.value,
    },
    sessions.value,
    weeks.value,
  )
  saving.value = true
  try {
    if (trainingId) {
      // Stay on the page so further changes can follow; reload to pick up the new ids.
      await trainingsStore.updateTraining(trainingId, input)
      const fresh = toRows(await trainingsStore.getTraining(trainingId))
      // The sessions come back in the same order: keeping their keys keeps the editors (and
      // which ones are folded) as they were.
      const keyOf = new Map(fresh.sessions.map((s, i) => [s.key, sessions.value[i]?.key ?? s.key]))
      for (const s of fresh.sessions) s.key = keyOf.get(s.key)!
      for (const w of fresh.weeks) for (const slot of w.slots) slot.sessionKey = keyOf.get(slot.sessionKey)!
      ;({ sessions: sessions.value, weeks: weeks.value } = fresh)
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
  if (!trainingId) return
  const ok = await confirmDialog({
    title: `Delete the training "${name.value}"?`,
    message: 'Its sessions and week plan are deleted. Workouts already done stay in the history.',
    confirmLabel: 'Delete',
    danger: true,
  })
  if (!ok) return
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

// Back to where the form was opened from: its cycle's page (?cycle=) or the trainings.
function goBack() {
  const fromCycle = route.query.cycle as string | undefined
  return router.push(fromCycle ? { name: 'cycle-edit', params: { id: fromCycle } } : { name: 'trainings' })
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
