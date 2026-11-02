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
        v-for="(block, bi) in blocks"
        :key="block.key"
        class="mt-4 rounded-2xl border border-slate-200 bg-white px-4 pt-3 shadow-sm"
        :aria-label="`Block ${block.letter}`"
      >
        <!-- Blocks start collapsed for an overview; tapping the header opens one. -->
        <button
          type="button"
          class="flex min-h-11 w-full items-center gap-2 text-left"
          :class="expanded.has(block.key) ? 'border-b border-slate-100 pb-2' : 'pb-3'"
          :aria-expanded="expanded.has(block.key)"
          @click="toggleExpanded(block.key)"
        >
          <span class="flex size-7 shrink-0 items-center justify-center rounded-md bg-indigo-600 text-sm font-bold text-white">
            {{ block.letter }}
          </span>
          <span class="min-w-0 flex-1">
            <span class="block truncate text-sm font-semibold text-slate-800">{{ block.title }}</span>
            <!-- A WOD's long notes are shown inside, not in the header. -->
            <span v-if="block.hint || (block.notes && !block.wod)" class="block text-xs text-slate-500">
              {{ [block.hint, block.wod ? '' : block.notes].filter(Boolean).join(' · ') }}
            </span>
          </span>
          <span class="shrink-0 text-xs text-slate-500">{{ blockProgress(block) }}</span>
          <ChevronDownIcon
            class="size-5 shrink-0 text-slate-400 transition-transform"
            :class="{ '-rotate-90': !expanded.has(block.key) }"
            aria-hidden="true"
          />
        </button>
        <div v-show="expanded.has(block.key)">
          <ComplexSets v-if="block.complex" v-model="blocks[bi]" :workout-id="workout.id" @toggle="toggleRound" />
          <WodMovements v-else-if="block.wod" :lines="block.wod.lines" :notes="block.notes" :mark="listMark(block.wod)" />
          <div v-else class="divide-y divide-slate-100">
            <ExerciseSets
              v-for="(card, i) in block.exercises"
              :key="card.key"
              v-model="block.exercises[i]"
              :done-on="doneOn"
              :workout-id="workout.id"
              @toggle="toggle(card, $event)"
            />
          </div>
          <EmomReps
            v-if="block.wod?.scoreType === 'emom'"
            :wod="block.wod"
            :cards="block.exercises"
            :save="(grid) => saveEmomReps(block, grid)"
          />
          <WodResult v-if="block.wod" v-model="block.wod" @toggle="toggleScore(block.wod)" />
        </div>
      </section>

      <section v-if="extras.length || extraResults.length" class="mt-4 rounded-2xl border border-slate-200 bg-white px-4 pt-3 shadow-sm">
        <button
          type="button"
          class="flex min-h-11 w-full items-center gap-2 text-left text-sm font-semibold text-slate-800"
          :class="expanded.has('extras') ? 'border-b border-slate-100 pb-2' : 'pb-3'"
          :aria-expanded="expanded.has('extras')"
          @click="toggleExpanded('extras')"
        >
          <span class="flex-1">{{ extras.length ? 'Other logged sets' : 'Other logged scores' }}</span>
          <ChevronDownIcon
            class="size-5 shrink-0 text-slate-400 transition-transform"
            :class="{ '-rotate-90': !expanded.has('extras') }"
            aria-hidden="true"
          />
        </button>
        <div v-show="expanded.has('extras')" class="divide-y divide-slate-100">
          <ExerciseSets
            v-for="(card, i) in extras"
            :key="card.key"
            v-model="extras[i]"
            :done-on="doneOn"
            :workout-id="workout.id"
            @toggle="toggle(card, $event)"
          />
          <p v-for="r in extraResults" :key="r.id" class="py-3 text-sm text-slate-700">
            <span class="font-semibold text-indigo-600">{{ blockLetter(r.block_position - 1) }}</span>
            {{ scoreFormat(r.score_type)?.label }}: <span class="font-semibold">{{ scoreText(r) }}</span>
            <span v-if="r.notes" class="text-slate-500"> · {{ r.notes }}</span>
          </p>
        </div>
      </section>

      <WorkoutMacros :day="workout.completed_at ? toIso(new Date(workout.completed_at)) : today()" />

      <WhoopCard v-if="workout.completed_at && whoop.status?.connected" :workout-id="workout.id" :done-at="workout.completed_at" />

      <!-- Finishing: notes on how it went and, optionally, a rating per exercise. -->
      <section v-if="finishing" ref="finishSection" class="mt-6 rounded-2xl border border-slate-200 bg-white p-4 shadow-sm">
        <h2 class="font-semibold text-slate-900">{{ workout.completed_at ? 'Notes and ratings' : 'Finish workout' }}</h2>
        <label class="mt-3 block text-sm font-medium text-slate-800">
          Done on
          <TextInput v-model="doneOn" type="date" size="touch" :max="today()" required class="mt-1" />
        </label>
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
import { ArrowLeftIcon, ChevronDownIcon, TrashIcon } from '@heroicons/vue/24/outline'
import { ApiError } from '../../api'
import AppButton from '../../components/button/AppButton.vue'
import StarRating from '../../components/exercise/StarRating.vue'
import { newMaxOffers } from '../../components/exercise/maxes'
import TextInput from '../../components/form/TextInput.vue'
import WorkoutMacros from '../../components/nutrition/WorkoutMacros.vue'
import ComplexSets from '../../components/workout/ComplexSets.vue'
import EmomReps from '../../components/workout/EmomReps.vue'
import ExerciseSets from '../../components/workout/ExerciseSets.vue'
import WhoopCard from '../../components/workout/WhoopCard.vue'
import WodMovements from '../../components/workout/WodMovements.vue'
import WodResult from '../../components/workout/WodResult.vue'
import { blockLetter } from '../../components/training/trainingForm'
import { scoreFormat, scoreText } from '../../components/training/wod'
import { confirmDialog } from '../../confirm'
import {
  attachNotes,
  buildCards,
  changed,
  emomLinesIn,
  fillEmomScore,
  num,
  progressOf,
  roundState,
  scoreChanged,
  listMark,
  scoreDraft,
  scoreValues,
  type BlockCard,
  type ExerciseCard,
  type Round,
  type SetRow,
  type WodCard,
} from '../../components/workout/workoutPlan'
import { formatDay, formatTime, toIso, today } from '../../dates'
import { useExercisesStore } from '../../stores/exercises'
import { useLookupsStore } from '../../stores/lookups'
import { useTrainingsStore } from '../../stores/trainings'
import { useWhoopStore } from '../../stores/whoop'
import { useWorkoutsStore } from '../../stores/workouts'
import type { TrainingDetail, Workout, WorkoutResult } from '../../types'

const route = useRoute()
const router = useRouter()
const workouts = useWorkoutsStore()
const trainings = useTrainingsStore()
const exercises = useExercisesStore()
const lookups = useLookupsStore()
const whoop = useWhoopStore()
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
// The day the workout was done: today by default, or its date once finished; changeable for a
// workout logged on another day.
const doneOn = ref(today())
// The sections opened by the user (all start collapsed): block keys, and 'extras' for the other logged sets.
const expanded = reactive(new Set<string>())
// Ratings picked while finishing, by exercise id; saved with the workout.
const ratingDrafts = reactive(new Map<string, number>())

const session = computed(() => training.value?.sessions.find((s) => s.id === workout.value?.session_id) ?? null)
const allCards = computed(() => [...blocks.value.flatMap((b) => b.exercises), ...extras.value])
// Sets to log: a complex's rounds count as one set each.
const progress = computed(() =>
  [...blocks.value.map(progressOf), ...extras.value.map((c) => ({ logged: c.rows.filter((r) => r.logged).length, total: c.rows.length }))].reduce(
    (sum, p) => ({ logged: sum.logged + p.logged, total: sum.total + p.total }),
    { logged: 0, total: 0 },
  ),
)
const plannedCount = computed(() => progress.value.total)
const loggedCount = computed(() => progress.value.logged)
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
      exercises.loadMaxes(),
      // Without the last loads the sets just suggest none.
      workouts.loadLastLoads().catch(() => {}),
      // Without WHOOP (not configured or failing) the workout just shows no WHOOP card.
      whoop.loadStatus().catch(() => {}),
    ])
    workout.value = loaded
    notes.value = loaded.notes
    if (loaded.completed_at) doneOn.value = toIso(new Date(loaded.completed_at))
    // The training may have been deleted since; then only the logged sets are shown.
    if (loaded.training_id) training.value = await trainings.getTraining(loaded.training_id).catch(() => null)
    const cards = buildCards(session.value, loaded.sets, loaded.results, lookups.kindOf, lookups.defaultUnit('load'), {
      unitOf: loadUnitOf,
      maxOf: (exerciseId) => {
        const max = exercises.effectiveMax(exerciseId, loadUnitOf)
        return max ? { value: max.load_value, unit: loadUnitOf(max.load_unit) } : null
      },
      lastOf: (exerciseId) => {
        const last = workouts.lastLoads.get(exerciseId)
        return last ? { value: last.load_value, unit: loadUnitOf(last.load_unit) } : null
      },
      weightUnit: lookups.defaultUnit('load', true),
    })
    attachNotes([...cards.blocks.flatMap((b) => b.exercises), ...cards.extras], loaded.exercise_notes)
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

// Logs a set, saves its changed values, or undoes it when it is logged and unchanged. offerMax
// offers a logged single above the 1RM as the new one.
async function toggle(card: ExerciseCard, row: SetRow, offerMax = true) {
  if (!workout.value || row.saving) return
  row.saving = true
  try {
    if (row.logged && !changed(card, row)) {
      await workouts.deleteSet(workout.value.id, row.logged.id)
      row.logged = null
      if (offerMax) offerNewMax(card)
      return
    }
    // Whatever was entered is saved, also only the reps or only the load; just not nothing.
    const reps = num(row.reps)
    const load = num(row.load)
    const intensity = card.endurance ? num(row.intensity) : null
    if (reps === null && load === null) {
      toast.error(`Enter the ${card.repsUnitName || 'reps'}${card.showLoad ? ' or the load' : ''} of set ${row.number} first.`)
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
      bodyweight: card.bodyweight,
      intensity_metric: intensity === null ? null : card.intensityMetric,
      intensity_value: intensity,
    })
    row.reps = reps
    row.load = load
    row.intensity = intensity
    if (offerMax) offerNewMax(card)
  } catch (e) {
    toast.error(e instanceof ApiError && e.status === 400 ? e.message : 'Could not save the set.')
  } finally {
    row.saving = false
  }
}

// Logs a round of a complex (its sets not logged or changed), or undoes it when logged and unchanged.
// A single of a complex isn't offered as a 1RM.
async function toggleRound(round: Round) {
  const state = roundState(round)
  const { card, row } = round.sets[0]
  if (state !== 'logged' && num(row.reps) === null && num(row.load) === null) {
    toast.error(`Enter the ${card.repsUnitName || 'reps'}${card.showLoad ? ' or the load' : ''} of set ${round.number} first.`)
    return
  }
  const sets = state === 'logged' ? round.sets : round.sets.filter((s) => !s.row.logged || changed(s.card, s.row))
  await Promise.all(sets.map((s) => toggle(s.card, s.row, false)))
}

const loadUnitOf = (id: string) => lookups.loadUnits.find((u) => u.id === id)

// The logged sets that beat the user's 1RM or a rep max (3RM, 5RM…) are offered as new ones: per
// rep count the heaviest, whichever was logged first. Correcting or undoing a set updates the offers.
function offerNewMax(card: ExerciseCard) {
  // Weight added to bodyweight isn't a max; nor are the reps of another unit than reps.
  if (card.bodyweight || card.repsUnitName !== 'reps') {
    card.newMaxes = []
    return
  }
  const logged = card.rows.flatMap((r) => (r.logged && !changed(card, r) ? [r.logged] : []))
  card.newMaxes = newMaxOffers(logged, loadUnitOf, exercises.maxes.get(card.exerciseId) ?? [])
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
      notes: wod.draft.notes,
    })
    wod.draft = scoreDraft(wod.result, wod.lines, wod.rounds)
  } catch (e) {
    toast.error(e instanceof ApiError && e.status === 400 ? e.message : 'Could not save the score.')
  } finally {
    wod.saving = false
  }
}

// Logs the reps of an EMOM per round and movement as sets of its exercises (the set number is the
// round; in reps, whatever the movement was planned in, e.g. the reps of a 20 s station), removes
// the ones cleared, and scores the EMOM by them (fillEmomScore): saved right away when the score
// was logged. Resolves to whether everything was saved.
async function saveEmomReps(block: BlockCard, grid: (number | null)[][]): Promise<boolean> {
  const wod = block.wod
  if (!workout.value || !wod) return false
  const id = workout.value.id
  const repsUnit = lookups.repUnits.find((u) => u.name === 'reps')?.id
  try {
    await Promise.all(
      grid.flatMap((values, r) =>
        emomLinesIn(wod, r + 1).map(async (i) => {
          const card = block.exercises.find((c) => c.key === wod.lines[i].key)
          if (!card) return
          const value = num(values[i])
          const row = card.rows.find((x) => x.number === r + 1)
          if (value === row?.logged?.reps_value || (value === null && !row?.logged)) return
          if (value === null) {
            await workouts.deleteSet(id, row!.logged!.id)
            card.rows = card.rows.filter((x) => x !== row)
            return
          }
          const logged = await workouts.logSet(id, {
            exercise_id: card.exerciseId,
            block_position: card.blockPosition,
            exercise_position: card.position,
            set_number: r + 1,
            reps_value: value,
            reps_unit: repsUnit ?? card.repsUnit,
            per_side: card.perSide,
            load_value: null,
            load_unit: null,
            bodyweight: false,
            intensity_metric: null,
            intensity_value: null,
          })
          if (row) Object.assign(row, { reps: value, logged })
          else card.rows = [...card.rows, { number: r + 1, reps: value, load: null, intensity: null, logged, saving: false }].sort((a, b) => a.number - b.number)
        }),
      ),
    )
  } catch (e) {
    toast.error(e instanceof ApiError && e.status === 400 ? e.message : 'Could not save the reps.')
    return false
  }
  fillEmomScore(wod, block.exercises)
  if (wod.result && scoreChanged(wod)) await toggleScore(wod)
  return true
}

function toggleExpanded(key: string) {
  if (!expanded.delete(key)) expanded.add(key)
}

// How much of a block is logged, e.g. "4/9 sets" or "Score logged", shown in its header.
function blockProgress(block: BlockCard): string {
  const { logged, total } = progressOf(block)
  if (block.wod && !total) return block.wod.result ? 'Score logged' : 'No score'
  return `${logged}/${total} sets`
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
    workout.value = await workouts.finishWorkout(workout.value.id, notes.value, completedAt(workout.value))
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

// The completion time to send when the day was changed: that day at the time it was finished
// (or now). Undefined keeps the workout's date.
function completedAt(w: Workout): string | undefined {
  const at = w.completed_at ? new Date(w.completed_at) : new Date()
  if (!doneOn.value || doneOn.value === toIso(at)) return undefined
  const [y, m, d] = doneOn.value.split('-').map(Number)
  at.setFullYear(y, m - 1, d)
  return at.toISOString()
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
