<template>
  <section class="mt-4 rounded-2xl border border-slate-200 bg-white p-4 shadow-sm" aria-labelledby="body-heading">
    <div class="flex items-baseline justify-between gap-2">
      <h2 id="body-heading" class="font-semibold text-slate-900">Body</h2>
      <!-- Changes save on their own; this confirms it. -->
      <p class="flex items-center gap-1 text-xs text-slate-500" aria-live="polite">
        <template v-if="status === 'saving'">Saving…</template>
        <template v-else-if="status === 'saved'"><CheckIcon class="size-4 text-emerald-600" aria-hidden="true" /> Saved</template>
      </p>
    </div>
    <p class="mt-1 text-sm text-slate-600">
      For the daily nutrition targets: an estimate to keep your weight, from your body, your weight and body fat, your
      workouts and, when connected, the energy WHOOP measured. Not medical advice.
    </p>

    <p v-if="!store.profileLoaded" class="mt-3 text-sm text-slate-500">Loading…</p>
    <form v-else class="mt-3 space-y-3" @submit.prevent="saveProfile">
      <p v-if="!store.profile" class="text-sm text-slate-500">Saved once height, birth date and sex are entered.</p>
      <div class="grid grid-cols-2 gap-3">
        <label class="block text-sm font-medium text-slate-800">
          Height (cm)
          <TextInput
            v-model.number="draft.height_cm"
            type="number"
            inputmode="decimal"
            min="100"
            max="250"
            step="any"
            required
            size="touch"
            class="mt-1"
            @change="saveProfile"
          />
        </label>
        <label class="block text-sm font-medium text-slate-800">
          Born on
          <TextInput v-model="draft.birth_date" type="date" required size="touch" :max="today()" class="mt-1" @change="saveProfile" />
        </label>
      </div>
      <div class="text-sm font-medium text-slate-800">
        <p>Sex</p>
        <SegmentedControl v-model="draft.sex" :options="sexOptions" label="Sex" class="mt-1" @update:model-value="saveProfile" />
      </div>
      <SelectField
        v-model="activity"
        :options="activityOptions"
        :searchable="false"
        label="Your day outside training"
        size="touch"
        @update:model-value="saveProfile"
      />
    </form>

    <!-- The weight history: the newest weight (and body fat) is the one the targets use from its day
         on; with body fat, the resting energy comes from the lean mass. -->
    <h3 class="mt-5 text-sm font-semibold text-slate-900">Weight and body fat</h3>
    <form class="mt-2 space-y-2" @submit.prevent="logWeight">
      <div class="grid grid-cols-2 gap-2">
        <label class="block text-sm font-medium text-slate-800">
          Weight (kg)
          <TextInput v-model.number="weightKg" type="number" inputmode="decimal" min="20" max="400" step="any" required size="touch" class="mt-1" />
        </label>
        <label class="block text-sm font-medium text-slate-800">
          Body fat % <span class="font-normal text-slate-500">(optional)</span>
          <TextInput v-model.number="bodyFat" type="number" inputmode="decimal" min="3" max="60" step="any" size="touch" class="mt-1" />
        </label>
      </div>
      <div class="flex items-end gap-2">
        <label class="block flex-1 text-sm font-medium text-slate-800">
          On
          <TextInput v-model="weighedOn" type="date" required size="touch" :max="today()" class="mt-1" />
        </label>
        <AppButton type="submit" size="touch" :disabled="!weightKg || logging">Log</AppButton>
      </div>
    </form>
    <ul v-if="store.weights.length" class="mt-2 divide-y divide-slate-100" aria-label="Weight history">
      <li v-for="w in shownWeights" :key="w.id" class="flex min-h-11 items-center gap-2 text-sm">
        <span class="flex-1 text-slate-600">{{ formatDate(w.measured_on) }}</span>
        <span class="font-medium text-slate-900">
          {{ w.weight_kg }} kg<template v-if="w.body_fat_percent !== null"> · {{ w.body_fat_percent }} %</template>
        </span>
        <AppButton variant="ghost" size="touch" icon :aria-label="`Delete the weight of ${formatDate(w.measured_on)}`" @click="removeWeight(w)">
          <TrashIcon class="size-5 text-slate-400" aria-hidden="true" />
        </AppButton>
      </li>
    </ul>
    <AppButton v-if="store.weights.length > shownWeights.length" variant="text" size="touch" class="w-full" @click="showAll = true">
      Show all {{ store.weights.length }}
    </AppButton>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { CheckIcon, TrashIcon } from '@heroicons/vue/24/outline'
import { toast } from 'vue-sonner'
import { ApiError } from '../../api'
import { confirmDialog } from '../../confirm'
import { formatDate, today } from '../../dates'
import { useNutritionStore } from '../../stores/nutrition'
import type { BodyWeight, Option, Sex } from '../../types'
import AppButton from '../button/AppButton.vue'
import SegmentedControl from '../form/SegmentedControl.vue'
import SelectField from '../form/SelectField.vue'
import TextInput from '../form/TextInput.vue'

const store = useNutritionStore()

const sexOptions: { value: Sex; label: string }[] = [
  { value: 'male', label: 'Male' },
  { value: 'female', label: 'Female' },
]

// The activity factor scales the resting energy for the day outside training; workouts are added
// on top.
const ACTIVITY: Option[] = [
  { id: '1.2', name: 'Mostly sitting' },
  { id: '1.3', name: 'Desk job, some walking' },
  { id: '1.45', name: 'On my feet part of the day' },
  { id: '1.6', name: 'On my feet all day' },
  { id: '1.8', name: 'Physical work' },
]

const draft = reactive<{ height_cm: number | null; birth_date: string; sex: Sex | '' }>({
  height_cm: null,
  birth_date: '',
  sex: '',
})
const activity = ref<string | null>('1.3')
// A factor saved that isn't one of the choices stays choosable.
const activityOptions = computed(() =>
  !activity.value || ACTIVITY.some((o) => o.id === activity.value)
    ? ACTIVITY
    : [...ACTIVITY, { id: activity.value, name: `Factor ${activity.value}` }],
)
// Whether the last change is being saved or was saved.
const status = ref<'' | 'saving' | 'saved'>('')

const weightKg = ref<number | null>(null)
const bodyFat = ref<number | null>(null)
const weighedOn = ref(today())
const logging = ref(false)
const showAll = ref(false)
const shownWeights = computed(() => (showAll.value ? store.weights : store.weights.slice(0, 5)))

onMounted(async () => {
  try {
    await Promise.all([store.loadProfile(), store.loadWeights()])
  } catch {
    toast.error('Could not load your body.')
    return
  }
  const p = store.profile
  if (p) {
    Object.assign(draft, { height_cm: p.height_cm, birth_date: p.birth_date, sex: p.sex })
    activity.value = String(p.activity_factor)
  }
})

// Saves the profile after a change, once it is complete and valid (a height being typed or no
// birth date yet waits for the next change) and differs from the one saved.
async function saveProfile() {
  const height = draft.height_cm
  if (!draft.sex || !draft.birth_date || height === null || height < 100 || height > 250) return
  const input = {
    height_cm: height,
    birth_date: draft.birth_date,
    sex: draft.sex,
    activity_factor: activity.value ? Number(activity.value) : undefined,
  }
  const p = store.profile
  if (p && p.height_cm === input.height_cm && p.birth_date === input.birth_date && p.sex === input.sex && p.activity_factor === input.activity_factor) {
    return
  }
  status.value = 'saving'
  try {
    await store.saveProfile(input)
    status.value = 'saved'
  } catch (e) {
    status.value = ''
    toast.error(e instanceof ApiError && e.status === 400 ? e.message : 'Could not save your body.')
  }
}

async function logWeight() {
  if (!weightKg.value) return
  logging.value = true
  try {
    await store.logWeight({ measured_on: weighedOn.value, weight_kg: weightKg.value, body_fat_percent: bodyFat.value })
    weightKg.value = null
    bodyFat.value = null
  } catch (e) {
    toast.error(e instanceof ApiError && e.status === 400 ? e.message : 'Could not log the weight.')
  } finally {
    logging.value = false
  }
}

async function removeWeight(w: BodyWeight) {
  const ok = await confirmDialog({
    title: 'Delete this weight?',
    message: `${w.weight_kg} kg${w.body_fat_percent !== null ? ` (${w.body_fat_percent} % body fat)` : ''} on ${formatDate(w.measured_on)}.`,
    confirmLabel: 'Delete',
    danger: true,
  })
  if (!ok) return
  try {
    await store.removeWeight(w.id)
  } catch {
    toast.error('Could not delete the weight.')
  }
}
</script>
