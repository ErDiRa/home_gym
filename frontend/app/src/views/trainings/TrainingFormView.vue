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
      <label class="block text-sm font-medium text-slate-800">
        Name<span class="text-red-600" aria-hidden="true"> *</span>
        <input v-model="name" type="text" required maxlength="100" placeholder="e.g. Phase 1" :class="inputClass" />
      </label>
      <fieldset>
        <legend class="text-sm font-medium text-slate-800">Period<span class="text-red-600" aria-hidden="true"> *</span></legend>
        <div class="mt-1 flex flex-wrap items-end gap-3">
          <label class="text-xs font-medium text-slate-600">
            Start (a Monday)
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
              :aria-pressed="weekCount === w"
              :class="
                weekCount === w
                  ? 'border-indigo-600 bg-indigo-600 text-white'
                  : 'border-slate-300 bg-white text-slate-600 hover:bg-slate-100'
              "
              class="h-[38px] rounded-md border px-3 text-xs font-semibold"
              @click="setWeekCount(w)"
            >
              {{ w }} weeks
            </button>
          </div>
          <label class="text-xs font-medium text-slate-600">
            Weeks
            <input
              :value="weekCount"
              type="number"
              min="1"
              :max="MAX_WEEKS"
              required
              :class="[inputClass, 'w-24!']"
              @change="setWeekCount(Number(($event.target as HTMLInputElement).value))"
            />
          </label>
        </div>
        <p v-if="endDate" class="mt-1 text-xs text-slate-500">
          Ends on {{ formatDate(endDate, { weekday: 'short', day: 'numeric', month: 'short', year: 'numeric' }) }}. Sessions are
          done on any days of their week.
        </p>
      </fieldset>
      <label class="flex items-start gap-2 text-sm text-slate-800">
        <input v-model="active" type="checkbox" class="mt-0.5 size-4 rounded border-slate-300 accent-indigo-600" />
        <span>
          <span class="font-medium">Active</span>
          <span class="block text-xs text-slate-500">Shown in the tracker. Several trainings can be active, e.g. strength, warm-ups and mobility side by side.</span>
        </span>
      </label>
      <label class="block text-sm font-medium text-slate-800">
        Description
        <textarea v-model="description" rows="2" :class="inputClass" />
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
          <select
            v-if="library.sessions.length"
            aria-label="Add a session from the library"
            class="rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-700 hover:bg-slate-50"
            :disabled="copying"
            @change="addFromLibrary"
          >
            <option value="">+ From library…</option>
            <option v-for="s in library.sessions" :key="s.id" :value="s.id">{{ s.name }}</option>
          </select>
        </div>
      </section>

      <section class="space-y-2" aria-labelledby="weeks-heading">
        <div>
          <h2 id="weeks-heading" class="text-sm font-medium text-slate-800">Weeks</h2>
          <p class="text-xs text-slate-500">Plan every week with up to {{ MAX_SLOTS }} of the sessions; the same one may appear more than once.</p>
        </div>
        <WeekPlanner v-model="weeks" :sessions="sessions" :start-date="startDate" />
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
import SessionEditor from '../../components/training/SessionEditor.vue'
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
import { addDays, formatDate, isMonday, nextMonday } from '../../dates'
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
// Set when editing an existing training (route /trainings/:id), absent when creating one.
const trainingId = route.params.id as string | undefined

const inputClass =
  'mt-1 block w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm font-normal outline-none focus:border-indigo-500 focus:ring-2 focus:ring-indigo-200'

const MAX_WEEKS = 52
const durationWeeks = [4, 6, 8, 12]

const name = ref('')
const description = ref('')
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
function setWeekCount(n: number) {
  if (!Number.isInteger(n) || n < 1 || n > MAX_WEEKS) return
  const list = weeks.value
  if (n < list.length && list.slice(n).some((w) => w.slots.length) && !confirm(`Drop the plans of weeks ${n + 1} to ${list.length}?`))
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
    ])
    if (training) {
      active.value = training.active
      startDate.value = training.start_date
      name.value = training.name
      description.value = training.description
      ;({ sessions: sessions.value, weeks: weeks.value } = toRows(training))
    }
  } catch (e) {
    toast.error(e instanceof ApiError && e.status === 404 ? 'Training not found.' : 'Could not load the training.')
  }
})

// Copies a template of the library into the training, as a new session of its own.
async function addFromLibrary(event: Event) {
  const select = event.target as HTMLSelectElement
  const id = select.value
  select.value = '' // back to the prompt, ready for the next one
  if (!id) return
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
function removeSession(index: number) {
  const session = sessions.value[index]
  const planned = weeks.value.some((w) => w.slots.some((s) => s.sessionKey === session.key))
  const hasExercises = session.blocks.some((b) => b.exercises.length)
  if ((planned || hasExercises) && !confirm(`Remove "${session.name}"${planned ? ' and its place in the weeks' : ''}?`)) return
  sessions.value.splice(index, 1)
  for (const w of weeks.value) w.slots = w.slots.filter((s) => s.sessionKey !== session.key)
}

async function submit() {
  if (problem.value) return
  const input = toInput(
    {
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
