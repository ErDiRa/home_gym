<template>
  <div>
    <p v-if="!readonly" class="text-sm text-slate-600">
      <template v-if="current">
        Current 1RM <span class="font-semibold text-slate-900">{{ formatLoad(current.load_value, current.load_unit_name) }}</span>
        · {{ formatDate(current.achieved_on) }}
      </template>
      <template v-else-if="effective">
        No 1RM yet, estimated
        <span class="font-semibold text-slate-900">≈ {{ formatLoad(effective.load_value, effective.load_unit_name) }}</span>
        from your {{ repsLabel(effective.reps) }}.
      </template>
      <template v-else>No 1RM yet. Loads planned in % 1RM are calculated from it.</template>
    </p>
    <p v-if="repMaxLine" class="mt-1 text-sm text-slate-600">Rep maxes: <span class="font-medium text-slate-800">{{ repMaxLine }}</span></p>

    <!-- A new entry: it becomes the current max, the older ones stay as the history. -->
    <form v-if="!readonly" class="mt-3 grid grid-cols-[4.5rem_1fr_6rem] gap-2" @submit.prevent="add">
      <TextInput
        v-model.number="reps"
        type="number"
        inputmode="numeric"
        min="1"
        max="30"
        step="1"
        required
        placeholder="Reps"
        aria-label="Reps: 1 for a 1RM, 3 or 5 for a rep max"
        :size="fieldSize"
      />
      <TextInput
        v-model.number="value"
        type="number"
        inputmode="decimal"
        min="0"
        step="any"
        required
        placeholder="Load"
        aria-label="Load"
        :size="fieldSize"
      />
      <SelectField v-model="unit" :options="weightUnits" :searchable="false" aria-label="Load unit" :size="fieldSize" />
      <TextInput v-model="day" type="date" :max="today()" required aria-label="Lifted on" :size="fieldSize" class="col-span-3" />
      <TextInput v-model="notes" maxlength="1000" placeholder="Note (optional)" aria-label="Note" :size="fieldSize" class="col-span-3" />
      <AppButton type="submit" :size="touch ? 'xl' : 'md'" class="col-span-3" :disabled="saving || num(value) === null || num(reps) === null || !unit">
        {{ saving ? 'Saving…' : `Add ${repsLabel(num(reps) ?? 1)}` }}
      </AppButton>
    </form>

    <ul v-if="history.length" :class="['divide-y divide-slate-100', { 'mt-3': !readonly }]" aria-label="Max history">
      <li v-for="m in history" :key="m.id" class="flex items-center gap-2 py-1.5">
        <div class="min-w-0 flex-1">
          <p class="text-sm text-slate-800">
            <span class="font-semibold">{{ repsLabel(repsOf(m)) }} {{ formatLoad(m.load_value, m.load_unit_name) }}</span>
            <span v-if="repsOf(m) > 1" class="text-slate-500"> ≈ {{ formatLoad(oneRmOf(m), m.load_unit_name) }} 1RM</span>
            <span class="text-slate-500"> · {{ formatDate(m.achieved_on) }}</span>
            <AppBadge v-if="prs.has(m.id)" color="green" class="ml-2">PR</AppBadge>
            <AppBadge v-if="m.workout_set_id" color="indigo" class="ml-1">Workout</AppBadge>
          </p>
          <p v-if="m.notes" class="truncate text-xs text-slate-500">{{ m.notes }}</p>
        </div>
        <AppButton
          v-if="!readonly"
          variant="ghost"
          :size="touch ? 'touch' : 'sm'"
          icon
          :aria-label="`Delete the ${repsLabel(repsOf(m))} of ${formatDate(m.achieved_on)}`"
          @click="remove(m)"
        >
          <TrashIcon class="size-5" aria-hidden="true" />
        </AppButton>
      </li>
    </ul>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { TrashIcon } from '@heroicons/vue/24/outline'
import { toast } from 'vue-sonner'
import { ApiError } from '../../api'
import AppButton from '../button/AppButton.vue'
import AppBadge from '../ui/AppBadge.vue'
import SelectField from '../form/SelectField.vue'
import TextInput from '../form/TextInput.vue'
import { confirmDialog } from '../../confirm'
import { formatDate, today } from '../../dates'
import { useExercisesStore } from '../../stores/exercises'
import { useLookupsStore } from '../../stores/lookups'
import type { ExerciseMax } from '../../types'
import { num } from '../workout/workoutPlan'
import { current1Rm, effectiveMax, formatLoad, oneRmOf, prIds, repMaxText, repsLabel, repsOf } from './maxes'

// The user's 1RM and rep max history of an exercise, shown in ExerciseMaxDialog: the current 1RM
// (or an estimate from a rep max), the rep maxes, adding an entry (reps 1 = a 1RM) and the entries
// before, the PRs marked per rep count. touch sizes it for the tracker; readonly shows
// only the entries (the tracker's PR list). Needs the load units loaded.
const props = withDefaults(defineProps<{ exerciseId: string; touch?: boolean; readonly?: boolean }>(), {
  touch: false,
  readonly: false,
})
const emit = defineEmits<{ added: [max: ExerciseMax] }>()

const store = useExercisesStore()
const lookups = useLookupsStore()

const history = computed(() => store.maxes.get(props.exerciseId) ?? [])
const unitOf = (id: string) => lookups.loadUnits.find((u) => u.id === id)
const current = computed(() => current1Rm(history.value))
const effective = computed(() => effectiveMax(history.value, unitOf))
const repMaxLine = computed(() => repMaxText(history.value, unitOf))
const weightUnits = computed(() => lookups.loadUnits.filter((u) => u.kg_per_unit))
const prs = computed(() => prIds(history.value, unitOf))
const fieldSize = computed(() => (props.touch ? 'touch' : 'md'))

const reps = ref<number | null>(1)
const value = ref<number | null>(null)
const unit = ref<string | null>(null)
const day = ref(today())
const notes = ref('')
const saving = ref(false)

// A new entry starts in the unit of the current max, else the last load unit if it's a weight.
watch(
  [current, weightUnits],
  () => {
    if (unit.value && weightUnits.value.some((u) => u.id === unit.value)) return
    const last = lookups.defaultUnit('load')
    unit.value = current.value?.load_unit ?? weightUnits.value.find((u) => u.id === last)?.id ?? weightUnits.value[0]?.id ?? null
  },
  { immediate: true },
)

async function add() {
  const v = num(value.value)
  if (v === null || !unit.value) return
  saving.value = true
  try {
    const r = num(reps.value)
    if (r === null) return
    const saved = await store.addMax(props.exerciseId, {
      reps: r,
      load_value: v,
      load_unit: unit.value,
      achieved_on: day.value,
      notes: notes.value.trim(),
    })
    emit('added', saved)
    value.value = null
    reps.value = 1
    notes.value = ''
    day.value = today()
  } catch (e) {
    toast.error(e instanceof ApiError && e.status === 400 ? e.message : 'Could not save the max.')
  } finally {
    saving.value = false
  }
}

async function remove(m: ExerciseMax) {
  const ok = await confirmDialog({
    title: `Delete this ${repsLabel(repsOf(m))}?`,
    message: `${formatLoad(m.load_value, m.load_unit_name)} of ${formatDate(m.achieved_on)} is removed from the history.`,
    confirmLabel: 'Delete',
    danger: true,
  })
  if (!ok) return
  try {
    await store.deleteMax(m)
  } catch {
    toast.error('Could not delete the max.')
  }
}
</script>
