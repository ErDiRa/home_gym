<template>
  <section class="pb-24">
    <AppButton :to="{ name: workout?.completed_at ? 'history' : 'track' }" variant="ghost" size="touch" class="-ml-3 text-slate-600!">
      <ArrowLeftIcon class="size-4" aria-hidden="true" />
      {{ workout?.completed_at ? 'History' : 'This week' }}
    </AppButton>

    <p v-if="loading" class="mt-6 text-center text-sm text-slate-500">Loading the workout…</p>
    <div v-else-if="error" class="mt-6 rounded-2xl border border-red-200 bg-red-50 p-6 text-center">
      <p class="font-medium text-red-800">{{ error }}</p>
    </div>

    <template v-else-if="workout">
      <p class="mt-2 text-sm text-slate-500">
        <template v-if="workout.completed_at">Done {{ formatDay(workout.completed_at) }}</template>
        <template v-else>Started {{ formatTime(workout.started_at) }}</template>
        <template v-if="training"> · {{ training.name }}</template>
      </p>
      <h1 class="text-2xl font-bold tracking-tight text-slate-900">{{ workout.session_name }}</h1>
      <p v-if="session?.notes" class="mt-1 text-sm text-slate-600">{{ session.notes }}</p>
      <p v-if="!session" class="mt-2 rounded-xl bg-amber-50 p-3 text-sm text-amber-800">
        This session is no longer in its training, so only the logged sets are shown.
      </p>

      <section
        v-for="block in blocks"
        :key="block.key"
        class="mt-4 rounded-2xl border border-slate-200 bg-white px-4 pt-3 shadow-sm"
        :aria-label="`Block ${block.letter}`"
      >
        <header class="flex items-center gap-2 border-b border-slate-100 pb-2">
          <span class="flex size-7 shrink-0 items-center justify-center rounded-md bg-indigo-600 text-sm font-bold text-white">
            {{ block.letter }}
          </span>
          <div class="min-w-0">
            <p class="truncate text-sm font-semibold text-slate-800">{{ block.title }}</p>
            <p v-if="block.hint || block.notes" class="text-xs text-slate-500">
              {{ [block.hint, block.notes].filter(Boolean).join(' · ') }}
            </p>
          </div>
        </header>
        <div class="divide-y divide-slate-100">
          <ExerciseSets
            v-for="(card, i) in block.exercises"
            :key="card.key"
            v-model="block.exercises[i]"
            @toggle="toggle(card, $event)"
          />
        </div>
        <WodResult v-if="block.wod" v-model="block.wod" @toggle="toggleScore(block.wod)" />
      </section>

      <section v-if="extras.length || extraResults.length" class="mt-4 rounded-2xl border border-slate-200 bg-white px-4 pt-3 shadow-sm">
        <p class="border-b border-slate-100 pb-2 text-sm font-semibold text-slate-800">
          {{ extras.length ? 'Other logged sets' : 'Other logged scores' }}
        </p>
        <div class="divide-y divide-slate-100">
          <ExerciseSets v-for="(card, i) in extras" :key="card.key" v-model="extras[i]" @toggle="toggle(card, $event)" />
          <p v-for="r in extraResults" :key="r.id" class="py-3 text-sm text-slate-700">
            <span class="font-semibold text-indigo-600">{{ blockLetter(r.block_position - 1) }}</span>
            {{ scoreFormat(r.score_type)?.label }}: <span class="font-semibold">{{ scoreText(r) }}</span> · {{ r.rx ? 'Rx' : 'Scaled' }}
            <span v-if="r.notes" class="text-slate-500"> · {{ r.notes }}</span>
          </p>
        </div>
      </section>

      <!-- Finishing: notes on how it went and, optionally, a rating per exercise. -->
      <section v-if="finishing" ref="finishSection" class="mt-6 rounded-2xl border border-slate-200 bg-white p-4 shadow-sm">
        <h2 class="font-semibold text-slate-900">{{ workout.completed_at ? 'Notes and ratings' : 'Finish workout' }}</h2>
        <label class="mt-3 block text-sm font-medium text-slate-800">
          How did it go?
          <TextInput v-model="notes" multiline size="touch" rows="3" maxlength="2000" class="mt-1" />
        </label>

        <p class="mt-4 text-sm font-medium text-slate-800">Rate the exercises <span class="font-normal text-slate-500">(optional)</span></p>
        <ul class="mt-1 divide-y divide-slate-100">
          <li v-for="e in rateable" :key="e.id" class="py-2">
            <p class="text-sm text-slate-700">{{ e.name }}</p>
            <StarRating
              :model-value="ratingOf(e.id)"
              :label="`Rating for ${e.name}`"
              class="mt-1"
              @update:model-value="ratingDrafts.set(e.id, $event)"
            />
          </li>
        </ul>

        <AppButton size="xl" class="mt-4 w-full" :disabled="saving" @click="finish">
          {{ workout.completed_at ? 'Save' : 'Save and finish' }}
        </AppButton>
      </section>

      <AppButton variant="secondary" size="touch" class="mt-6 w-full text-red-700!" :disabled="saving" @click="remove">
        <TrashIcon class="size-5" aria-hidden="true" />
        {{ workout.completed_at ? 'Delete workout' : 'Discard workout' }}
      </AppButton>
    </template>

    <!-- The bar to finish from, at the bottom of the screen while scrolling through the sets. -->
    <div
      v-if="workout && !finishing"
      class="fixed inset-x-0 bottom-0 z-20 border-t border-slate-200 bg-white/95 pb-[env(safe-area-inset-bottom)] backdrop-blur"
    >
      <div class="mx-auto flex max-w-md items-center gap-3 px-4 py-3">
        <p class="flex-1 text-sm text-slate-600">
          <template v-if="plannedCount || !wods.length">
            <span class="font-semibold text-slate-900">{{ loggedCount }}</span> of {{ plannedCount }} sets
          </template>
          <template v-if="plannedCount && wods.length"> · </template>
          <template v-if="wods.length">
            <span class="font-semibold text-slate-900">{{ scoredCount }}</span> of {{ wods.length }}
            {{ wods.length === 1 ? 'score' : 'scores' }}
          </template>
        </p>
        <AppButton size="xl" @click="openFinish">{{ workout.completed_at ? 'Notes' : 'Finish' }}</AppButton>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { toast } from 'vue-sonner'
import { ArrowLeftIcon, TrashIcon } from '@heroicons/vue/24/outline'
import { ApiError } from '../../api'
import AppButton from '../../components/button/AppButton.vue'
import StarRating from '../../components/exercise/StarRating.vue'
import TextInput from '../../components/form/TextInput.vue'
import ExerciseSets from '../../components/workout/ExerciseSets.vue'
import WodResult from '../../components/workout/WodResult.vue'
import { blockLetter } from '../../components/training/trainingForm'
import { scoreFormat, scoreText } from '../../components/training/wod'
import { confirmDialog } from '../../confirm'
import {
  buildCards,
  changed,
  num,
  scoreChanged,
  scoreDraft,
  scoreValues,
  type BlockCard,
  type ExerciseCard,
  type SetRow,
  type WodCard,
} from '../../components/workout/workoutPlan'
import { formatDay, formatTime } from '../../dates'
import { useExercisesStore } from '../../stores/exercises'
import { useLookupsStore } from '../../stores/lookups'
import { useTrainingsStore } from '../../stores/trainings'
import { useWorkoutsStore } from '../../stores/workouts'
import type { TrainingDetail, Workout, WorkoutResult } from '../../types'

const route = useRoute()
const router = useRouter()
const workouts = useWorkoutsStore()
const trainings = useTrainingsStore()
const exercises = useExercisesStore()
const lookups = useLookupsStore()
const id = route.params.id as string

const workout = ref<Workout | null>(null)
const training = ref<TrainingDetail | null>(null)
const blocks = ref<BlockCard[]>([])
const extras = ref<ExerciseCard[]>([])
const extraResults = ref<WorkoutResult[]>([])
const loading = ref(true)
const error = ref('')
const saving = ref(false)
const finishing = ref(false)
const finishSection = ref<HTMLElement | null>(null)
const notes = ref('')
// Ratings picked while finishing, by exercise id; saved with the workout.
const ratingDrafts = reactive(new Map<string, number>())

const session = computed(() => training.value?.sessions.find((s) => s.id === workout.value?.session_id) ?? null)
const allCards = computed(() => [...blocks.value.flatMap((b) => b.exercises), ...extras.value])
const plannedCount = computed(() => allCards.value.reduce((n, c) => n + c.rows.length, 0))
const loggedCount = computed(() => allCards.value.reduce((n, c) => n + c.rows.filter((r) => r.logged).length, 0))
const wods = computed(() => blocks.value.flatMap((b) => (b.wod ? [b.wod] : [])))
const scoredCount = computed(() => wods.value.filter((w) => w.result).length)
// Each exercise once, in workout order.
const rateable = computed(() => [...new Map(allCards.value.map((c) => [c.exerciseId, { id: c.exerciseId, name: c.name }])).values()])
const ratingOf = (exerciseId: string) => ratingDrafts.get(exerciseId) ?? exercises.ratings.get(exerciseId)?.rating ?? 0


onMounted(async () => {
  try {
    const [loaded] = await Promise.all([
      workouts.getWorkout(id),
      lookups.loadTrainingLookups(),
      exercises.loadExercises(),
      exercises.loadRatings(),
    ])
    workout.value = loaded
    notes.value = loaded.notes
    // The training may have been deleted since; then only the logged sets are shown.
    if (loaded.training_id) training.value = await trainings.getTraining(loaded.training_id).catch(() => null)
    const cards = buildCards(session.value, loaded.sets, loaded.results, lookups.kindOf, lookups.defaultUnit('load'))
    blocks.value = cards.blocks
    extras.value = cards.extras
    extraResults.value = cards.extraResults
    // A finished workout opens with its notes, for reading and correcting.
    finishing.value = !!loaded.completed_at
  } catch (e) {
    error.value = e instanceof ApiError && e.status === 404 ? 'Workout not found.' : 'Could not load the workout.'
  } finally {
    loading.value = false
  }
})

// Logs a set, saves its changed values, or undoes it when it is logged and unchanged.
async function toggle(card: ExerciseCard, row: SetRow) {
  if (!workout.value || row.saving) return
  row.saving = true
  try {
    if (row.logged && !changed(card, row)) {
      await workouts.deleteSet(workout.value.id, row.logged.id)
      row.logged = null
      return
    }
    // Whatever was entered is saved, also only the reps or only the load; just not nothing.
    const reps = num(row.reps)
    const load = num(row.load)
    if (reps === null && load === null) {
      toast.error(`Enter the ${card.repsUnitName || 'reps'} or the load of set ${row.number} first.`)
      return
    }
    row.logged = await workouts.logSet(workout.value.id, {
      exercise_id: card.exerciseId,
      block_position: card.blockPosition,
      exercise_position: card.position,
      set_number: row.number,
      reps_value: reps,
      reps_unit: card.repsUnit,
      per_side: card.perSide,
      load_value: load,
      load_unit: load === null ? null : card.loadUnit,
    })
    row.reps = reps
    row.load = load
  } catch (e) {
    toast.error(e instanceof ApiError && e.status === 400 ? e.message : 'Could not save the set.')
  } finally {
    row.saving = false
  }
}

// Logs the score of a WOD block, saves its changed values, or undoes it when logged and unchanged.
async function toggleScore(wod: WodCard) {
  if (!workout.value || wod.saving) return
  wod.saving = true
  try {
    if (wod.result && !scoreChanged(wod)) {
      await workouts.deleteResult(workout.value.id, wod.result.id)
      wod.result = null
      return
    }
    wod.result = await workouts.logResult(workout.value.id, {
      block_position: wod.blockPosition,
      score_type: wod.scoreType,
      ...scoreValues(wod.scoreType, wod.draft),
      rx: wod.draft.rx,
      notes: wod.draft.notes,
    })
    wod.draft = scoreDraft(wod.result)
  } catch (e) {
    toast.error(e instanceof ApiError && e.status === 400 ? e.message : 'Could not save the score.')
  } finally {
    wod.saving = false
  }
}

async function openFinish() {
  finishing.value = true
  await nextTick()
  finishSection.value?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

async function finish() {
  if (!workout.value) return
  // Logging every set and score isn't required, but finishing with some left open asks first.
  const open = plannedCount.value - loggedCount.value
  const openScores = wods.value.length - scoredCount.value
  if (!workout.value.completed_at && (open > 0 || openScores > 0)) {
    const nothing = !loggedCount.value && !scoredCount.value
    const parts = [
      open ? `${open} of ${plannedCount.value} sets` : '',
      openScores ? `${openScores} of ${wods.value.length} ${wods.value.length === 1 ? 'score' : 'scores'}` : '',
    ].filter(Boolean)
    const ok = await confirmDialog({
      title: nothing ? 'Nothing logged' : `${parts.join(' and ')} not logged`,
      message: nothing
        ? 'Only your notes and ratings are saved; sets and scores can still be logged later from the history. Finish it anyway?'
        : 'You can still log them later from the history. Finish the workout anyway?',
      confirmLabel: 'Finish anyway',
      cancelLabel: 'Keep logging',
    })
    if (!ok) return
  }
  saving.value = true
  try {
    const done = !workout.value.completed_at
    workout.value = await workouts.finishWorkout(workout.value.id, notes.value)
    // Only ratings that differ from the saved ones are sent.
    const ratings = [...ratingDrafts].filter(([exerciseId, r]) => exercises.ratings.get(exerciseId)?.rating !== r)
    await Promise.all(ratings.map(([exerciseId, r]) => exercises.rateExercise(exerciseId, r, exercises.ratings.get(exerciseId)?.notes ?? '')))
    ratingDrafts.clear()
    toast.success(done ? 'Workout done. Well trained!' : 'Workout saved.')
    if (done) await router.push({ name: 'track' })
  } catch (e) {
    toast.error(e instanceof ApiError && e.status === 400 ? e.message : 'Could not save the workout.')
  } finally {
    saving.value = false
  }
}

async function remove() {
  if (!workout.value) return
  const finished = !!workout.value.completed_at
  const ok = await confirmDialog({
    title: finished ? 'Delete this workout?' : 'Discard this workout?',
    message: finished ? 'The workout and its sets are deleted.' : 'The workout and the sets logged so far are deleted.',
    confirmLabel: finished ? 'Delete' : 'Discard',
    danger: true,
  })
  if (!ok) return
  saving.value = true
  try {
    await workouts.deleteWorkout(workout.value.id)
    toast.success(finished ? 'Workout deleted.' : 'Workout discarded.')
    await router.push({ name: finished ? 'history' : 'track' })
  } catch {
    toast.error('Could not delete the workout.')
  } finally {
    saving.value = false
  }
}
</script>
