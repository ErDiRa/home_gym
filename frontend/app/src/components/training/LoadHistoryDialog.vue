<template>
  <TransitionRoot :show="open" as="template">
    <Dialog class="relative z-50" @close="emit('close')">
      <TransitionChild
        as="template"
        enter="duration-150 ease-out"
        enter-from="opacity-0"
        leave="duration-100 ease-in"
        leave-to="opacity-0"
      >
        <div class="fixed inset-0 bg-slate-900/40 transition-opacity" aria-hidden="true" />
      </TransitionChild>

      <!-- A bottom sheet on phones, a centered panel on wider screens. -->
      <div class="fixed inset-0 flex items-end justify-center sm:items-center sm:p-4">
        <TransitionChild
          as="template"
          enter="duration-200 ease-out"
          enter-from="translate-y-full sm:translate-y-0 sm:scale-95 sm:opacity-0"
          leave="duration-150 ease-in"
          leave-to="translate-y-full sm:translate-y-0 sm:scale-95 sm:opacity-0"
        >
          <DialogPanel class="flex max-h-[90dvh] w-full max-w-md flex-col rounded-t-2xl bg-white shadow-2xl transition sm:rounded-2xl">
            <div class="flex items-center gap-2 border-b border-slate-100 py-2 pr-2 pl-4">
              <DialogTitle class="min-w-0 flex-1 truncate font-semibold text-slate-900">Logged loads · {{ name }}</DialogTitle>
              <AppButton variant="ghost" icon aria-label="Close" @click="emit('close')">
                <XMarkIcon class="size-6" aria-hidden="true" />
              </AppButton>
            </div>
            <div class="overflow-y-auto p-4 pb-[calc(1rem+env(safe-area-inset-bottom))]">
              <p v-if="loading" class="text-sm text-slate-500">Loading…</p>
              <p v-else-if="error" class="text-sm text-red-700">{{ error }}</p>
              <p v-else-if="!workouts.length" class="text-sm text-slate-500">No loads logged of this exercise yet.</p>
              <template v-else>
                <p class="mb-3 text-xs text-slate-500">Pick a set to plan its load.</p>
                <!-- A timeline, newest first: one entry per workout with its sets. -->
                <ol class="border-l-2 border-slate-200">
                  <li v-for="w in workouts" :key="w.workout_id" class="relative pb-4 pl-4 last:pb-0">
                    <span class="absolute top-1.5 -left-[5px] size-2 rounded-full bg-indigo-500" aria-hidden="true" />
                    <p class="text-sm">
                      <span class="font-medium text-slate-900">{{ formatDay(w.done_at) }}</span>
                      <span class="text-slate-500"> · {{ w.session_name }}</span>
                    </p>
                    <p class="mb-1.5 text-xs text-slate-500">Top {{ top(w) }}</p>
                    <div class="flex flex-wrap gap-1">
                      <AppButton
                        v-for="s in w.sets"
                        :key="s.set_number"
                        variant="secondary"
                        size="xs"
                        :title="`Plan ${s.load_value} ${s.load_unit_name}`"
                        @click="emit('pick', s)"
                      >
                        {{ setText(s) }}
                      </AppButton>
                    </div>
                  </li>
                </ol>
              </template>
            </div>
          </DialogPanel>
        </TransitionChild>
      </div>
    </Dialog>
  </TransitionRoot>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { Dialog, DialogPanel, DialogTitle, TransitionChild, TransitionRoot } from '@headlessui/vue'
import { XMarkIcon } from '@heroicons/vue/24/outline'
import AppButton from '../button/AppButton.vue'
import { formatDay } from '../../dates'
import { useWorkoutsStore } from '../../stores/workouts'
import type { LoadHistorySet, LoadHistoryWorkout } from '../../types'

// The loads the user logged of an exercise in their newest workouts, to pick the one to plan.
// Loaded each time it opens.
const props = defineProps<{ open: boolean; exerciseId: string; name: string }>()
const emit = defineEmits<{ close: []; pick: [set: LoadHistorySet] }>()
const store = useWorkoutsStore()

const workouts = ref<LoadHistoryWorkout[]>([])
const loading = ref(false)
const error = ref('')

watch(
  () => props.open && props.exerciseId,
  async (id) => {
    if (!id) return
    loading.value = true
    error.value = ''
    try {
      workouts.value = await store.listExerciseLoads(id)
    } catch {
      error.value = 'Could not load the logged loads.'
    } finally {
      loading.value = false
    }
  },
  { immediate: true },
)

// E.g. "5 reps × 60 kg", or only the load when no reps were logged.
function setText(s: LoadHistorySet): string {
  const load = `${s.load_value} ${s.load_unit_name}`
  if (s.reps_value === null) return load
  return `${s.reps_value} ${s.reps_unit_name}${s.per_side ? ' per side' : ''} × ${load}`
}

// The heaviest load of a workout among the sets in its first set's unit.
function top(w: LoadHistoryWorkout): string {
  const unit = w.sets[0].load_unit
  const same = w.sets.filter((s) => s.load_unit === unit)
  const best = same.reduce((a, b) => (b.load_value > a.load_value ? b : a))
  return `${best.load_value} ${best.load_unit_name}`
}
</script>
