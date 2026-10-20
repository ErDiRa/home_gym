<template>
  <section>
    <p v-if="loading && !week" class="mt-6 text-center text-sm text-slate-500">Loading your week…</p>

    <div v-else-if="error" class="mt-6 rounded-2xl border border-red-200 bg-red-50 p-6 text-center">
      <p class="font-medium text-red-800">{{ error }}</p>
      <AppButton variant="secondary" size="touch" class="mt-3" @click="load">Try again</AppButton>
    </div>

    <template v-else-if="week">
      <!-- A workout in progress of another training than the one shown can be resumed from here. -->
      <RouterLink
        v-if="week.current && week.current.training_id !== week.training?.id"
        :to="{ name: 'workout', params: { id: week.current.id } }"
        class="mb-4 flex min-h-11 items-center gap-2 rounded-xl border border-indigo-200 bg-indigo-50 px-4 py-2 text-sm text-indigo-900 active:bg-indigo-100"
      >
        <PlayCircleIcon class="size-5 shrink-0" aria-hidden="true" />
        <span class="min-w-0 flex-1 truncate">In progress: {{ week.current.session_name }}</span>
        <ChevronRightIcon class="size-4 shrink-0" aria-hidden="true" />
      </RouterLink>

      <template v-if="week.training">
        <p class="text-sm text-slate-500">
          <template v-if="week.week_number">Week {{ week.week_number }} of {{ week.training.weeks }} · </template>
          {{ formatPeriod(week.training.start_date, week.training.end_date) }}
        </p>
        <!-- Several active trainings (e.g. strength, warm-ups, mobility): the heading switches. -->
        <SelectField
          v-if="week.trainings.length > 1"
          :model-value="week.training.id"
          :options="week.trainings"
          :searchable="false"
          size="touch"
          aria-label="Training"
          field-class="text-xl! font-bold tracking-tight"
          class="mt-1"
          @update:model-value="(id) => id && pick(id)"
        />
        <h1 v-else class="text-2xl font-bold tracking-tight text-slate-900">{{ week.training.name }}</h1>
        <p
          v-if="!week.week_number || !week.planned.length"
          class="mt-6 rounded-2xl border border-dashed border-slate-300 bg-white p-4 text-center text-sm text-slate-500"
        >
          {{
            !week.week_number
              ? week.training.start_date > today()
                ? 'This training starts later. Nothing is planned this week.'
                : 'This training has ended. Plan the next one in the dashboard.'
              : 'No sessions planned this week.'
          }}
        </p>
      </template>
      <div v-else class="rounded-2xl border border-dashed border-slate-300 bg-white p-6 text-center">
        <p class="font-medium text-slate-800">No active training</p>
        <p class="mt-1 text-sm text-slate-500">Plan one in the dashboard and mark it active to track it here.</p>
        <AppButton :to="{ name: 'trainings' }" variant="secondary" size="touch" class="mt-4">Open trainings</AppButton>
      </div>

      <!-- This week's sessions: the done ones checked, then what's left. -->
      <section v-if="sessions.length" class="mt-6" aria-labelledby="week-heading">
        <h2 id="week-heading" class="font-semibold text-slate-900">This week</h2>
        <ul class="mt-2 space-y-3">
          <li v-for="s in sessions" :key="s.key">
            <RouterLink
              v-if="s.workoutId"
              :to="{ name: 'workout', params: { id: s.workoutId } }"
              class="block rounded-2xl active:opacity-80"
            >
              <SessionCard :name="s.name" :lines="s.lines" :done="s.done" />
            </RouterLink>
            <button
              v-else
              type="button"
              class="block w-full rounded-2xl text-left active:opacity-80 disabled:opacity-60"
              :disabled="starting"
              @click="start(s.slotId!)"
            >
              <SessionCard :name="s.name" :lines="s.lines" :done="s.done" />
            </button>
          </li>
        </ul>
      </section>
    </template>
  </section>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { toast } from 'vue-sonner'
import { ChevronRightIcon, PlayCircleIcon } from '@heroicons/vue/24/outline'
import { ApiError } from '../../api'
import AppButton from '../../components/button/AppButton.vue'
import SelectField from '../../components/form/SelectField.vue'
import SessionCard from '../../components/workout/SessionCard.vue'
import { sessionSummary, type SummaryLine } from '../../components/workout/workoutPlan'
import { formatPeriod, today } from '../../dates'
import { useTrainingsStore } from '../../stores/trainings'
import { useWorkoutsStore } from '../../stores/workouts'
import type { TrainingDetail } from '../../types'

interface Session {
  key: string
  name: string
  lines: SummaryLine[]
  done: boolean
  workoutId?: string // done or already started: opens the workout
  slotId?: string // otherwise: starts the planned session
}

const route = useRoute()
const router = useRouter()
const store = useWorkoutsStore()
const trainings = useTrainingsStore()
const { week, loading } = storeToRefs(store)
const training = ref<TrainingDetail | null>(null)
const error = ref('')
const starting = ref(false)

const linesOf = (sessionId: string | null) => {
  const session = training.value?.sessions.find((s) => s.id === sessionId)
  return session ? sessionSummary(session) : []
}

// The training shown, chosen in the URL (?training=id) so a reload or going back keeps it;
// without one the server picks the first active training.
const chosen = computed(() => (typeof route.query.training === 'string' ? route.query.training : undefined))

// The week's planned sessions of the training shown, in order, each done when a finished workout
// was started from its slot (so of a session planned twice, only the one done is checked).
// Workouts of this training that match no slot (e.g. its plan changed) follow as done cards.
const sessions = computed((): Session[] => {
  const w = week.value
  if (!w) return []
  const doneBySlot = new Map(w.done.filter((d) => d.week_session_id).map((d) => [d.week_session_id, d]))
  const list: Session[] = w.planned.map((slot) => {
    const done = doneBySlot.get(slot.id)
    const current = w.current?.week_session_id === slot.id ? w.current : undefined
    return {
      key: slot.id,
      name: slot.name,
      lines: linesOf(slot.session_id),
      done: !!done,
      workoutId: done?.id ?? current?.id,
      slotId: slot.id,
    }
  })
  const planned = new Set(w.planned.map((slot) => slot.id))
  const ofTraining = (trainingId: string | null) => !!w.training && trainingId === w.training.id
  for (const d of w.done) {
    if (!ofTraining(d.training_id) || (d.week_session_id && planned.has(d.week_session_id))) continue
    list.push({ key: d.id, name: d.session_name, lines: linesOf(d.session_id), done: true, workoutId: d.id })
  }
  // One in progress of another training gets the banner above instead.
  if (w.current && ofTraining(w.current.training_id) && !planned.has(w.current.week_session_id ?? '')) {
    const c = w.current
    list.push({ key: c.id, name: c.session_name, lines: linesOf(c.session_id), done: false, workoutId: c.id })
  }
  return list
})

async function load() {
  error.value = ''
  try {
    await store.loadWeek(chosen.value)
    const id = week.value?.training?.id
    // Without the plan the cards still show the sessions' names.
    training.value = id ? await trainings.getTraining(id).catch(() => null) : null
  } catch {
    error.value = 'Could not load your week.'
  }
}

async function start(slotId: string) {
  starting.value = true
  try {
    const workout = await store.startWorkout(slotId)
    await router.push({ name: 'workout', params: { id: workout.id } })
  } catch (e) {
    // Another workout was started meanwhile (e.g. on another device): show it.
    if (e instanceof ApiError && e.status === 409) {
      toast.error('Another workout is already in progress.')
      await load()
    } else {
      toast.error('Could not start the workout.')
    }
  } finally {
    starting.value = false
  }
}

// Switching keeps the choice in the URL; the watch below loads it.
function pick(trainingId: string) {
  router.replace({ query: { ...route.query, training: trainingId } })
}

watch(chosen, load, { immediate: true })
</script>
