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
          <DialogPanel class="w-full max-w-md rounded-t-2xl bg-white shadow-2xl transition sm:rounded-2xl">
            <div class="flex items-center gap-2 border-b border-slate-100 py-2 pr-2 pl-4">
              <DialogTitle class="min-w-0 flex-1 truncate font-semibold text-slate-900">
                Macros of {{ formatDate(date, { weekday: true, year: false }) }}
              </DialogTitle>
              <AppButton variant="ghost" size="touch" icon aria-label="Close" @click="emit('close')">
                <XMarkIcon class="size-6" aria-hidden="true" />
              </AppButton>
            </div>
            <form class="space-y-4 p-4 pb-[calc(1rem+env(safe-area-inset-bottom))]" @submit.prevent="save">
              <div class="grid grid-cols-3 gap-3">
                <label v-for="m in MACROS" :key="m.key" class="block text-sm font-medium text-slate-800">
                  {{ m.label }} (g)
                  <TextInput
                    v-model.number="draft[m.key]"
                    type="number"
                    inputmode="decimal"
                    min="0"
                    max="2000"
                    step="any"
                    size="touch"
                    class="mt-1"
                    :placeholder="target ? formatAmount(target[m.key]) : ''"
                  />
                </label>
              </div>
              <p class="text-sm text-slate-600">
                <span class="font-semibold text-slate-900">{{ formatAmount(kcal) }}</span>
                <template v-if="target"> of {{ formatAmount(target.kcal) }}</template> kcal
                <span v-if="target" class="text-slate-500"> · the targets are the placeholders</span>
              </p>
              <label class="block text-sm font-medium text-slate-800">
                Note <span class="font-normal text-slate-500">(optional)</span>
                <TextInput v-model="draft.notes" multiline size="touch" rows="2" maxlength="1000" class="mt-1" placeholder="e.g. Ate out" />
              </label>
              <AppButton type="submit" size="xl" class="w-full" :disabled="!filled || saving">
                {{ saving ? 'Saving…' : 'Save' }}
              </AppButton>
              <AppButton v-if="macros" variant="ghost" size="touch" class="w-full text-red-700!" :disabled="saving" @click="remove">
                Clear the day
              </AppButton>
            </form>
          </DialogPanel>
        </TransitionChild>
      </div>
    </Dialog>
  </TransitionRoot>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { Dialog, DialogPanel, DialogTitle, TransitionChild, TransitionRoot } from '@headlessui/vue'
import { XMarkIcon } from '@heroicons/vue/24/outline'
import { toast } from 'vue-sonner'
import { ApiError } from '../../api'
import { formatDate } from '../../dates'
import { useNutritionStore } from '../../stores/nutrition'
import type { DayMacros, MacroTarget } from '../../types'
import AppButton from '../button/AppButton.vue'
import TextInput from '../form/TextInput.vue'
import { MACROS, formatAmount, kcalOf } from './macros'

// Logs the totals eaten on a day (protein, fat, carbs; each may be left empty, not all), prefilled
// with what was logged before; the day's target shows as the placeholders.
const props = defineProps<{ open: boolean; date: string; macros: DayMacros | null; target: MacroTarget | null }>()
const emit = defineEmits<{ close: []; saved: [macros: DayMacros | null] }>()

const store = useNutritionStore()
const draft = reactive<{ protein_g: number | null; fat_g: number | null; carbs_g: number | null; notes: string }>({
  protein_g: null,
  fat_g: null,
  carbs_g: null,
  notes: '',
})
const saving = ref(false)

const kcal = computed(() => kcalOf(draft))
const filled = computed(() => MACROS.some(({ key }) => draft[key] !== null))

// Every opening starts from what was logged.
watch(
  () => props.open,
  (open) => {
    if (!open) return
    draft.protein_g = props.macros?.protein_g ?? null
    draft.fat_g = props.macros?.fat_g ?? null
    draft.carbs_g = props.macros?.carbs_g ?? null
    draft.notes = props.macros?.notes ?? ''
  },
  { immediate: true },
)

async function save() {
  if (!filled.value) return
  saving.value = true
  try {
    const saved = await store.saveDayMacros(props.date, { ...draft, notes: draft.notes.trim() })
    emit('saved', saved)
    emit('close')
  } catch (e) {
    toast.error(e instanceof ApiError && e.status === 400 ? e.message : 'Could not save the macros.')
  } finally {
    saving.value = false
  }
}

async function remove() {
  saving.value = true
  try {
    await store.deleteDayMacros(props.date)
    emit('saved', null)
    emit('close')
  } catch {
    toast.error('Could not clear the day.')
  } finally {
    saving.value = false
  }
}
</script>
