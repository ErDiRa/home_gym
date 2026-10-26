<template>
  <div>
    <p v-if="!readonly" class="text-sm text-slate-600">
      <template v-if="current">
        Current 1RM <span class="font-semibold text-slate-900">{{ formatLoad(current.load_value, current.load_unit_name) }}</span>
        · {{ formatDate(current.achieved_on) }}
      </template>
      <template v-else>No 1RM yet. Loads planned in % 1RM are calculated from it.</template>
    </p>

    <!-- A new entry: it becomes the current max, the older ones stay as the history. -->
    <form v-if="!readonly" class="mt-3 grid grid-cols-[1fr_6rem] gap-2" @submit.prevent="add">
      <TextInput
        v-model.number="value"
        type="number"
        inputmode="decimal"
        min="0"
        step="any"
        required
        placeholder="1RM"
        aria-label="1RM"
        :size="fieldSize"
      />
      <SelectField v-model="unit" :options="weightUnits" :searchable="false" aria-label="1RM unit" :size="fieldSize" />
      <TextInput v-model="day" type="date" :max="today()" required aria-label="Lifted on" :size="fieldSize" class="col-span-2" />
      <TextInput v-model="notes" maxlength="1000" placeholder="Note (optional)" aria-label="Note" :size="fieldSize" class="col-span-2" />
      <AppButton type="submit" :size="touch ? 'xl' : 'md'" class="col-span-2" :disabled="saving || num(value) === null || !unit">
        {{ saving ? 'Saving…' : 'Add 1RM' }}
      </AppButton>
    </form>

    <ul v-if="history.length" :class="['divide-y divide-slate-100', { 'mt-3': !readonly }]" aria-label="1RM history">
      <li v-for="m in history" :key="m.id" class="flex items-center gap-2 py-1.5">
        <div class="min-w-0 flex-1">
          <p class="text-sm text-slate-800">
            <span class="font-semibold">{{ formatLoad(m.load_value, m.load_unit_name) }}</span>
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
          :aria-label="`Delete the 1RM of ${formatDate(m.achieved_on)}`"
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
import { formatLoad, prIds } from './maxes'

// The user's 1RM history of an exercise, shown in ExerciseMaxDialog: the current max, adding a
// new one and the entries before, the PRs marked. touch sizes it for the tracker; readonly shows
// only the entries (the tracker's PR list). Needs the load units loaded.
const props = withDefaults(defineProps<{ exerciseId: string; touch?: boolean; readonly?: boolean }>(), {
  touch: false,
  readonly: false,
})
const emit = defineEmits<{ added: [max: ExerciseMax] }>()

const store = useExercisesStore()
const lookups = useLookupsStore()

const history = computed(() => store.maxes.get(props.exerciseId) ?? [])
const current = computed(() => history.value[0] ?? null)
const weightUnits = computed(() => lookups.loadUnits.filter((u) => u.kg_per_unit))
const prs = computed(() => prIds(history.value, (id) => lookups.loadUnits.find((u) => u.id === id)))
const fieldSize = computed(() => (props.touch ? 'touch' : 'md'))

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
    const saved = await store.addMax(props.exerciseId, {
      load_value: v,
      load_unit: unit.value,
      achieved_on: day.value,
      notes: notes.value.trim(),
    })
    emit('added', saved)
    value.value = null
    notes.value = ''
    day.value = today()
  } catch (e) {
    toast.error(e instanceof ApiError && e.status === 400 ? e.message : 'Could not save the 1RM.')
  } finally {
    saving.value = false
  }
}

async function remove(m: ExerciseMax) {
  const ok = await confirmDialog({
    title: 'Delete this 1RM?',
    message: `${formatLoad(m.load_value, m.load_unit_name)} of ${formatDate(m.achieved_on)} is removed from the history.`,
    confirmLabel: 'Delete',
    danger: true,
  })
  if (!ok) return
  try {
    await store.deleteMax(m)
  } catch {
    toast.error('Could not delete the 1RM.')
  }
}
</script>
