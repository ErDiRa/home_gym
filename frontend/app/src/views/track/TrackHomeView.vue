<template>
  <section>
    <p v-if="loading && !week" class="mt-6 text-center text-sm text-slate-500">Loading your week…</p>

    <div v-else-if="error" class="mt-6 rounded-2xl border border-red-200 bg-red-50 p-6 text-center">
      <p class="font-medium text-red-800">{{ error }}</p>
      <AppButton variant="secondary" size="touch" class="mt-3" @click="load">Try again</AppButton>
    </div>

    <template v-else-if="week">
      <!-- A workout in progress comes first: it can be resumed even without an active training. -->
      <RouterLink
        v-if="week.current"
        :to="{ name: 'workout', params: { id: week.current.id } }"
        class="mb-6 flex items-center gap-3 rounded-2xl border border-indigo-500 bg-indigo-600 p-4 text-white shadow-sm active:bg-indigo-700"
      >
        <PlayCircleIcon class="size-10 shrink-0" aria-hidden="true" />
        <span class="min-w-0 flex-1">
          <span class="block text-sm text-indigo-100">In progress since {{ formatTime(week.current.started_at) }}</span>
          <span class="block truncate text-lg font-semibold">Resume {{ week.current.unit_name }}</span>
        </span>
      </RouterLink>

      <div v-if="!week.training" class="rounded-2xl border border-dashed border-slate-300 bg-white p-6 text-center">
        <p class="font-medium text-slate-800">No active training</p>
        <p class="mt-1 text-sm text-slate-500">Plan one in the dashboard and mark it active to track it here.</p>
        <AppButton :to="{ name: 'trainings' }" variant="secondary" size="touch" class="mt-4">Open trainings</AppButton>
      </div>

      <template v-else>
        <p class="text-sm text-slate-500">{{ formatPeriod(week.training.start_date, week.training.end_date) }}</p>
        <h1 class="text-2xl font-bold tracking-tight text-slate-900">{{ week.training.name }}</h1>

        <!-- Progress: one segment per planned session. -->
        <div class="mt-5">
          <div class="flex items-baseline justify-between">
            <h2 class="font-semibold text-slate-900">This week</h2>
            <p class="text-sm text-slate-600">
              <span class="font-semibold text-slate-900">{{ week.done.length }}</span> of {{ target }}
              {{ target === 1 ? 'session' : 'sessions' }}
            </p>
          </div>
          <div
            class="mt-2 flex gap-1.5"
            role="progressbar"
            :aria-valuenow="week.done.length"
            aria-valuemin="0"
            :aria-valuemax="target"
            :aria-label="`${week.done.length} of ${target} sessions done this week`"
          >
            <span
              v-for="i in target"
              :key="i"
              class="h-2 flex-1 rounded-full"
              :class="i <= week.done.length ? 'bg-emerald-500' : 'bg-slate-200'"
            />
          </div>
          <p v-if="week.done.length >= target" class="mt-2 text-sm font-medium text-emerald-700">
            Weekly goal reached. Extra sessions still count.
          </p>
        </div>

        <!-- What to do next, unless a workout is already running. -->
        <div v-if="!week.current && week.units.length" class="mt-6 rounded-2xl border border-slate-200 bg-white p-4 shadow-sm">
          <p class="text-sm text-slate-500">Up next</p>
          <p class="text-xl font-semibold text-slate-900">{{ nextUnit?.name }}</p>
          <AppButton size="xl" class="mt-3 w-full" :disabled="starting" @click="start(nextUnit!.id)">
            <PlayIcon class="size-5" aria-hidden="true" />
            Start {{ nextUnit?.name }}
          </AppButton>
          <template v-if="otherUnits.length">
            <p class="mt-4 text-sm text-slate-500">Or do another unit:</p>
            <div class="mt-2 flex flex-wrap gap-2">
              <AppButton
                v-for="unit in otherUnits"
                :key="unit.id"
                variant="secondary"
                size="touch"
                :disabled="starting"
                @click="start(unit.id)"
              >
                {{ unit.name }}
              </AppButton>
            </div>
          </template>
        </div>
        <p
          v-else-if="!week.units.length"
          class="mt-6 rounded-2xl border border-dashed border-slate-300 bg-white p-4 text-center text-sm text-slate-500"
        >
          This training has no units yet. Add some in the dashboard.
        </p>
      </template>

      <!-- Done this week: every finished workout, from any training. -->
      <section v-if="week.done.length || week.training" class="mt-6" aria-labelledby="done-heading">
        <h2 id="done-heading" class="font-semibold text-slate-900">Done</h2>
        <p v-if="!week.done.length" class="mt-2 text-sm text-slate-500">Nothing yet this week.</p>
        <ul v-else class="mt-2 space-y-2">
          <li v-for="w in [...week.done].reverse()" :key="w.id">
            <RouterLink
              :to="{ name: 'workout', params: { id: w.id } }"
              class="flex min-h-14 items-center gap-3 rounded-xl border border-slate-200 bg-white px-4 py-3 active:bg-slate-50"
            >
              <CheckCircleIcon class="size-6 shrink-0 text-emerald-500" aria-hidden="true" />
              <span class="min-w-0 flex-1">
                <span class="block truncate font-medium text-slate-900">{{ w.unit_name }}</span>
                <span class="block text-sm text-slate-500">
                  {{ formatDay(w.completed_at) }} · {{ w.set_count }} {{ w.set_count === 1 ? 'set' : 'sets' }}
                </span>
              </span>
              <ChevronRightIcon class="size-5 shrink-0 text-slate-400" aria-hidden="true" />
            </RouterLink>
          </li>
        </ul>
      </section>
    </template>
  </section>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { toast } from 'vue-sonner'
import { CheckCircleIcon, ChevronRightIcon, PlayCircleIcon, PlayIcon } from '@heroicons/vue/24/outline'
import { ApiError } from '../../api'
import AppButton from '../../components/button/AppButton.vue'
import { formatDay, formatPeriod } from '../../dates'
import { useWorkoutsStore } from '../../stores/workouts'

const router = useRouter()
const store = useWorkoutsStore()
const { week, loading } = storeToRefs(store)
const error = ref('')
const starting = ref(false)

const target = computed(() => week.value?.training?.sessions_per_week ?? 0)
// Units are done in turn; the server suggests the one after the last done.
const nextUnit = computed(() => week.value?.units.find((u) => u.id === week.value?.next_unit_id) ?? week.value?.units[0])
const otherUnits = computed(() => week.value?.units.filter((u) => u.id !== nextUnit.value?.id) ?? [])

const formatTime = (at: string) => new Date(at).toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' })

async function load() {
  error.value = ''
  try {
    await store.loadWeek()
  } catch {
    error.value = 'Could not load your week.'
  }
}

async function start(unitId: string) {
  starting.value = true
  try {
    const workout = await store.startWorkout(unitId)
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

onMounted(load)
</script>
