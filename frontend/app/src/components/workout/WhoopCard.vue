<template>
  <section class="mt-4 rounded-2xl border border-slate-200 bg-white p-4 shadow-sm" aria-label="WHOOP">
    <button type="button" class="flex min-h-11 w-full items-center gap-2 text-left" :aria-expanded="open" @click="open = !open">
      <h2 class="flex-1 font-semibold text-slate-900">WHOOP</h2>
      <span v-if="data" class="text-xs text-slate-500">{{ data.sport_name }} · {{ formatTime(data.start_at) }}–{{ formatTime(data.end_at) }}</span>
      <ChevronDownIcon class="size-5 shrink-0 text-slate-400 transition-transform" :class="{ '-rotate-90': !open }" aria-hidden="true" />
    </button>

    <template v-if="!open" />
    <p v-else-if="loading" class="mt-2 text-sm text-slate-500">Loading…</p>

    <!-- Linked: the activity's numbers and the recovery of that day. -->
    <template v-else-if="data && !picking">
      <p v-if="data.strain === null" class="mt-2 rounded-xl bg-amber-50 p-3 text-sm text-amber-800">
        WHOOP hasn't scored this activity yet. Re-fetch in a few minutes.
      </p>
      <dl v-else class="mt-3 grid grid-cols-4 gap-2 text-center">
        <div v-for="m in metrics" :key="m.label" class="rounded-xl bg-slate-50 py-2">
          <dt class="text-xs text-slate-500">{{ m.label }}</dt>
          <dd class="font-semibold text-slate-900">{{ m.value }}</dd>
        </div>
      </dl>
      <ul v-if="zones.length" class="mt-3 space-y-1 text-xs text-slate-600" aria-label="Heart-rate zones">
        <li v-for="z in zones" :key="z.zone" class="flex items-center gap-2">
          <span class="w-12">Zone {{ z.zone }}</span>
          <span class="h-2 rounded-full bg-indigo-500" :style="{ width: `${z.share * 60}%` }" />
          <span>{{ z.minutes }} min</span>
        </li>
      </ul>
      <p v-if="data.recovery_score !== null" class="mt-3 text-sm text-slate-700">
        Recovery <span class="font-semibold">{{ data.recovery_score }}%</span> · RHR {{ data.resting_hr }} · HRV
        {{ Math.round(data.hrv_ms ?? 0) }} ms
      </p>
      <div class="mt-3 flex gap-2">
        <AppButton variant="secondary" size="touch" class="flex-1" :disabled="busy" @click="link(data.whoop_workout_id)">Re-fetch</AppButton>
        <AppButton variant="secondary" size="touch" class="flex-1" :disabled="busy" @click="pick">Change</AppButton>
        <AppButton variant="ghost" size="touch" class="text-red-700!" :disabled="busy" @click="unlink">Unlink</AppButton>
      </div>
    </template>

    <!-- Picking: the WHOOP activities of the workout's day. -->
    <template v-else-if="picking">
      <label class="mt-2 block text-sm font-medium text-slate-800">
        Activities of
        <TextInput v-model="day" type="date" size="touch" :max="today()" class="mt-1" @change="loadCandidates" />
      </label>
      <p v-if="!candidates" class="mt-2 text-sm text-slate-500">Loading the day's WHOOP activities…</p>
      <p v-else-if="!candidates.length" class="mt-2 text-sm text-slate-500">No WHOOP activities that day.</p>
      <ul v-else class="mt-2 divide-y divide-slate-100">
        <li v-for="c in candidates" :key="c.id">
          <button
            type="button"
            class="flex min-h-14 w-full items-center justify-between gap-2 text-left disabled:opacity-50"
            :disabled="busy"
            @click="link(c.id)"
          >
            <span>
              <span class="block font-medium text-slate-900">{{ c.sport_name }}</span>
              <span class="text-sm text-slate-500">{{ formatTime(c.start) }}–{{ formatTime(c.end) }}</span>
            </span>
            <span class="text-sm text-slate-600">
              <template v-if="c.strain !== null">Strain {{ c.strain.toFixed(1) }} · {{ c.avg_hr }} bpm</template>
              <template v-else>Not scored yet</template>
            </span>
          </button>
        </li>
      </ul>
      <AppButton variant="ghost" size="touch" class="mt-1 -ml-3" @click="picking = false">Cancel</AppButton>
    </template>

    <AppButton v-else variant="secondary" size="touch" class="mt-3 w-full" @click="pick">Retrieve WHOOP data</AppButton>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { toast } from 'vue-sonner'
import { ChevronDownIcon } from '@heroicons/vue/24/outline'
import { ApiError } from '../../api'
import AppButton from '../button/AppButton.vue'
import TextInput from '../form/TextInput.vue'
import { formatTime, toIso, today } from '../../dates'
import { useWhoopStore } from '../../stores/whoop'
import type { WhoopCandidate, WhoopData } from '../../types'

// The WHOOP activity of a finished workout: picked from the activities of a day (the workout's
// by default, or another one), then shown with its strain, heart rate, energy, zones and the
// day's recovery.
const props = defineProps<{ workoutId: string; doneAt: string }>()

const whoop = useWhoopStore()
const data = ref<WhoopData | null>(null)
const candidates = ref<WhoopCandidate[] | null>(null)
const loading = ref(true)
const picking = ref(false)
const busy = ref(false)
const open = ref(true)
// The day whose activities are listed (YYYY-MM-DD).
const day = ref(toIso(new Date(props.doneAt)))

const metrics = computed(() => {
  const d = data.value
  if (!d) return []
  return [
    { label: 'Strain', value: d.strain?.toFixed(1) ?? '–' },
    { label: 'Avg HR', value: d.avg_hr ?? '–' },
    { label: 'Max HR', value: d.max_hr ?? '–' },
    { label: 'kcal', value: d.kilojoule === null ? '–' : Math.round(d.kilojoule / 4.184) },
  ]
})

const zones = computed(() => {
  const ms = data.value?.zone_ms ?? []
  const max = Math.max(...ms, 1)
  return ms.map((t, zone) => ({ zone, minutes: Math.round(t / 60_000), share: t / max })).filter((z) => z.zone > 0)
})

onMounted(async () => {
  try {
    data.value = await whoop.getWorkoutWhoop(props.workoutId)
  } catch {
    toast.error('Could not load the WHOOP data.')
  } finally {
    loading.value = false
  }
})

function failed(e: unknown, fallback: string) {
  toast.error(e instanceof ApiError && e.status < 500 ? e.message : fallback)
}

// Starts picking from the activities of the workout's day.
function pick() {
  picking.value = true
  day.value = toIso(new Date(props.doneAt))
  loadCandidates()
}

// Lists the WHOOP activities of the chosen local day.
async function loadCandidates() {
  if (!day.value) return
  candidates.value = null
  const [y, m, d] = day.value.split('-').map(Number)
  const from = new Date(y, m - 1, d)
  const to = new Date(y, m - 1, d + 1)
  try {
    candidates.value = await whoop.listWhoopCandidates(props.workoutId, from, to)
  } catch (e) {
    picking.value = false
    failed(e, 'Could not load the WHOOP activities.')
  }
}

async function link(whoopWorkoutId: string) {
  busy.value = true
  try {
    data.value = await whoop.linkWhoop(props.workoutId, whoopWorkoutId)
    picking.value = false
  } catch (e) {
    failed(e, 'Could not fetch the WHOOP data.')
  } finally {
    busy.value = false
  }
}

async function unlink() {
  busy.value = true
  try {
    await whoop.unlinkWhoop(props.workoutId)
    data.value = null
  } catch (e) {
    failed(e, 'Could not unlink the WHOOP data.')
  } finally {
    busy.value = false
  }
}
</script>
