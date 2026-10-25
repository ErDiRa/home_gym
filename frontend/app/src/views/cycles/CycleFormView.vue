<template>
  <div class="max-w-4xl">
    <RouterLink
      :to="{ name: 'cycles' }"
      class="mb-4 inline-flex items-center gap-x-1.5 text-sm font-medium text-slate-600 hover:text-indigo-600"
    >
      <ArrowLeftIcon class="size-4" aria-hidden="true" />
      Back to cycles
    </RouterLink>

    <div class="mb-1 flex flex-wrap items-center justify-between gap-4">
      <h1 class="text-xl font-bold text-slate-900">{{ cycleId ? 'Edit cycle' : 'New cycle' }}</h1>
      <AppButton v-if="cycleId" variant="secondary" size="md" :disabled="saving" @click="remove">
        <TrashIcon class="size-4" aria-hidden="true" />
        Delete
      </AppButton>
    </div>
    <p class="mt-1 mb-6 text-sm text-slate-500"><span class="text-red-600">*</span> Required</p>

    <form class="space-y-4" @submit.prevent="submit">
      <label class="block text-sm font-medium text-slate-800">
        Name<span class="text-red-600" aria-hidden="true"> *</span>
        <TextInput v-model="name" type="text" required maxlength="100" placeholder="e.g. Mobility" class="mt-1" />
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
            :model-value="weeks ?? ''"
            :options="durationWeeks.map((w) => ({ value: w, label: `${w} weeks` }))"
            label="Duration"
            @update:model-value="weeks = Number($event)"
          />
          <label class="text-xs font-medium text-slate-600">
            Weeks
            <TextInput v-model.number="weeks" type="number" min="1" :max="MAX_CYCLE_WEEKS" required class="mt-1 w-24!" />
          </label>
        </div>
        <p v-if="endDate" class="mt-1 text-xs text-slate-500">
          Ends on {{ formatDate(endDate, { weekday: true }) }}. Moving the cycle doesn't move its trainings.
        </p>
      </fieldset>
      <label class="block text-sm font-medium text-slate-800">
        Description
        <TextInput v-model="description" multiline rows="2" class="mt-1" />
      </label>

      <section class="space-y-2" aria-labelledby="goals-heading">
        <div>
          <h2 id="goals-heading" class="text-sm font-medium text-slate-800">Goals</h2>
          <p class="text-xs text-slate-500">What the cycle should achieve, with a target to check at the end if you like.</p>
        </div>
        <GoalListEditor v-model="goals" />
      </section>

      <!-- Sticks to the bottom of the screen so saving doesn't need scrolling down. -->
      <div class="sticky bottom-0 z-10 flex items-center justify-end gap-3 border-t border-slate-200 bg-slate-50/95 py-4 backdrop-blur">
        <p v-if="problem" class="mr-auto text-sm text-amber-700">{{ problem }}</p>
        <AppButton variant="secondary" :disabled="saving" :to="{ name: 'cycles' }">Cancel</AppButton>
        <AppButton type="submit" :disabled="saving || !!problem">
          {{ saving ? 'Saving…' : cycleId ? 'Save' : 'Create' }}
        </AppButton>
      </div>
    </form>

    <section v-if="cycle" class="mt-8 space-y-3" aria-labelledby="trainings-heading">
      <div class="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h2 id="trainings-heading" class="text-sm font-medium text-slate-800">Trainings</h2>
          <p class="text-xs text-slate-500">
            The trainings of the cycle by week. Several can run in the same week, e.g. a hips and an elbow training.
          </p>
        </div>
        <div class="flex flex-wrap gap-2">
          <AppButton variant="secondary" size="md" :to="{ name: 'training-new', query: { cycle: cycle.id } }">
            <PlusIcon class="size-4" aria-hidden="true" />
            New training
          </AppButton>
          <AppButton variant="secondary" size="md" :to="{ name: 'training-new', query: { import: '1', cycle: cycle.id } }">
            <ArrowUpTrayIcon class="size-4" aria-hidden="true" />
            Import week
          </AppButton>
          <ActionMenu :items="looseTrainings" :disabled="moving" empty="Every training is in a cycle." @select="addTraining">
            Add existing
          </ActionMenu>
        </div>
      </div>

      <CycleTimeline :start-date="cycle.start_date" :weeks="cycle.weeks" :trainings="cycle.trainings" />

      <ul v-if="cycle.trainings.length" class="divide-y divide-slate-100 rounded-lg border border-slate-200 bg-white">
        <li v-for="t in cycle.trainings" :key="t.id" class="flex flex-wrap items-center gap-x-3 gap-y-1 px-3 py-2">
          <RouterLink
            :to="{ name: 'training-edit', params: { id: t.id }, query: { cycle: cycle.id } }"
            class="text-sm font-medium text-slate-800 hover:text-indigo-600"
          >
            {{ t.name }}
          </RouterLink>
          <span class="text-xs text-slate-500">{{ formatPeriod(t.start_date, t.end_date) }}</span>
          <ActiveBadge v-if="t.active" />
          <AppBadge v-if="!withinCycle(t, cycle)" color="yellow">Outside the cycle's weeks</AppBadge>
          <AppButton variant="text" size="sm" class="ml-auto" :disabled="moving" @click="removeTraining(t.id)">
            Remove from cycle
          </AppButton>
        </li>
      </ul>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { toast } from 'vue-sonner'
import { ArrowLeftIcon, ArrowUpTrayIcon, PlusIcon, TrashIcon } from '@heroicons/vue/24/outline'
import { ApiError } from '../../api'
import AppButton from '../../components/button/AppButton.vue'
import CycleTimeline from '../../components/cycle/CycleTimeline.vue'
import GoalListEditor from '../../components/cycle/GoalListEditor.vue'
import { goalsProblem, MAX_CYCLE_WEEKS, toCycleInput, toGoalRows, type GoalRow } from '../../components/cycle/cycleForm'
import { withinCycle } from '../../components/cycle/cycleStatus'
import SegmentedControl from '../../components/form/SegmentedControl.vue'
import TextInput from '../../components/form/TextInput.vue'
import ActiveBadge from '../../components/training/ActiveBadge.vue'
import { inputOf } from '../../components/training/trainingForm'
import ActionMenu from '../../components/ui/ActionMenu.vue'
import AppBadge from '../../components/ui/AppBadge.vue'
import { confirmWithOption } from '../../confirm'
import { addDays, formatDate, formatPeriod, isMonday, mondayOf, nextMonday } from '../../dates'
import { useCyclesStore } from '../../stores/cycles'
import { useLookupsStore } from '../../stores/lookups'
import { useTrainingsStore } from '../../stores/trainings'
import type { Cycle } from '../../types'

const router = useRouter()
const route = useRoute()
const store = useCyclesStore()
const trainingsStore = useTrainingsStore()
const lookups = useLookupsStore()
// Set when editing an existing cycle (route /cycles/:id), absent when creating one.
const cycleId = route.params.id as string | undefined

const durationWeeks = [4, 6, 8, 10, 12]

const name = ref('')
const description = ref('')
const startDate = ref(cycleId ? '' : nextMonday())
const weeks = ref<number | null>(cycleId ? null : 8)
const goals = ref<GoalRow[]>([])
// The saved cycle with its trainings; null until loaded (and for a new cycle).
const cycle = ref<Cycle | null>(null)
const saving = ref(false)
const moving = ref(false) // a training being added to or removed from the cycle

// The last Sunday of the weeks.
const endDate = computed(() =>
  startDate.value && weeks.value && weeks.value > 0 ? addDays(startDate.value, weeks.value * 7 - 1) : '',
)

// Any date picks the Monday of its week, as cycles run over whole weeks.
function setStart(value: string) {
  startDate.value = value ? mondayOf(value) : ''
}

// Why the cycle can't be saved yet, if anything; mirrors the server's checks.
const problem = computed(() => {
  if (!name.value.trim()) return 'Give the cycle a name.'
  if (!startDate.value) return 'Set the start date.'
  if (!isMonday(startDate.value)) return 'The cycle must start on a Monday.'
  const w = weeks.value
  if (!w || !Number.isInteger(w) || w < 1 || w > MAX_CYCLE_WEEKS) return `Set 1-${MAX_CYCLE_WEEKS} weeks.`
  return goalsProblem(goals.value)
})

// The trainings in no cycle yet, to add one of them.
const looseTrainings = computed(() => trainingsStore.trainings.filter((t) => !t.cycle_id).map((t) => ({ id: t.id, name: t.name })))

function show(c: Cycle) {
  cycle.value = c
  name.value = c.name
  description.value = c.description
  startDate.value = c.start_date
  weeks.value = c.weeks
  goals.value = toGoalRows(c)
}

onMounted(async () => {
  if (!cycleId) return
  try {
    const [c] = await Promise.all([store.getCycle(cycleId), trainingsStore.loadTrainings()])
    show(c)
  } catch (e) {
    toast.error(e instanceof ApiError && e.status === 404 ? 'Cycle not found.' : 'Could not load the cycle.')
  }
})

async function submit() {
  if (problem.value) return
  const input = toCycleInput(
    { name: name.value, description: description.value, start_date: startDate.value, weeks: weeks.value! },
    goals.value,
  )
  saving.value = true
  try {
    if (cycleId) {
      show(await store.updateCycle(cycleId, input))
      toast.success('Cycle saved.')
    } else {
      // Stay on the new cycle's page, to add its trainings.
      const created = await store.createCycle(input)
      toast.success('Cycle created. Add its trainings below.')
      await router.replace({ name: 'cycle-edit', params: { id: created.id } })
    }
  } catch (e) {
    toast.error(e instanceof ApiError && e.status === 400 ? e.message : 'Could not save the cycle.')
  } finally {
    saving.value = false
  }
}

// Moves a training into this cycle (cycleId) or out of any (null): it is saved unchanged otherwise.
async function setTrainingCycle(trainingId: string, to: string | null, done: string) {
  moving.value = true
  try {
    const [training] = await Promise.all([trainingsStore.getTraining(trainingId), lookups.loadTrainingLookups()])
    await trainingsStore.updateTraining(trainingId, { ...inputOf(training), cycle_id: to })
    const [c] = await Promise.all([store.getCycle(cycleId!), trainingsStore.loadTrainings()])
    cycle.value = c
    toast.success(done)
  } catch (e) {
    toast.error(e instanceof ApiError && e.status === 400 ? e.message : 'Could not change the training.')
  } finally {
    moving.value = false
  }
}

const addTraining = (id: string) => setTrainingCycle(id, cycleId!, 'Training added to the cycle.')
const removeTraining = (id: string) => setTrainingCycle(id, null, 'Training removed from the cycle. It still exists.')

async function remove() {
  if (!cycleId || !cycle.value) return
  const count = cycle.value.trainings.length
  const answer = await confirmWithOption({
    title: `Delete the cycle "${name.value}"?`,
    message: count
      ? `Its goals are deleted. Its ${count === 1 ? 'training stays' : `${count} trainings stay`} without a cycle, unless you delete ${count === 1 ? 'it' : 'them'} too.`
      : 'Its goals are deleted. It has no trainings.',
    confirmLabel: 'Delete',
    danger: true,
    option: count ? `Delete ${count === 1 ? 'its training' : `its ${count} trainings`} too` : undefined,
  })
  if (!answer.ok) return
  saving.value = true
  try {
    await store.deleteCycle(cycleId, answer.option)
    toast.success(answer.option ? 'Cycle and its trainings deleted.' : 'Cycle deleted.')
    await router.push({ name: 'cycles' })
  } catch {
    toast.error('Could not delete the cycle.')
  } finally {
    saving.value = false
  }
}
</script>
