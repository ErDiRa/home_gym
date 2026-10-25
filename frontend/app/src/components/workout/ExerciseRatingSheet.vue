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
              <DialogTitle class="min-w-0 flex-1 truncate font-semibold text-slate-900">Rate {{ name }}</DialogTitle>
              <AppButton variant="ghost" size="touch" icon aria-label="Close" @click="emit('close')">
                <XMarkIcon class="size-6" aria-hidden="true" />
              </AppButton>
            </div>
            <div class="space-y-4 p-4 pb-[calc(1rem+env(safe-area-inset-bottom))]">
              <div>
                <p class="text-sm font-medium text-slate-800">How helpful is it for you?</p>
                <StarRating v-model="rating" :label="`Rating for ${name}`" class="-ml-2" />
              </div>
              <label class="block text-sm font-medium text-slate-800">
                Note <span class="font-normal text-slate-500">(optional)</span>
                <TextInput v-model="notes" multiline size="touch" rows="2" maxlength="2000" class="mt-1" placeholder="e.g. Knees feel good" />
              </label>
              <AppButton size="xl" class="w-full" :disabled="!rating || saving" @click="save">
                {{ saving ? 'Saving…' : 'Save rating' }}
              </AppButton>
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
import { toast } from 'vue-sonner'
import AppButton from '../button/AppButton.vue'
import StarRating from '../exercise/StarRating.vue'
import TextInput from '../form/TextInput.vue'
import { useExercisesStore } from '../../stores/exercises'

// Rates an exercise (1-5 and a note) in the middle of a workout, saved right away: the user's own
// rating, prefilled with the one given before.
const props = defineProps<{ open: boolean; exerciseId: string; name: string }>()
const emit = defineEmits<{ close: [] }>()

const store = useExercisesStore()
const rating = ref(0)
const notes = ref('')
const saving = ref(false)

// Every opening starts from the saved rating.
watch(
  () => props.open,
  (open) => {
    if (!open) return
    const saved = store.ratings.get(props.exerciseId)
    rating.value = saved?.rating ?? 0
    notes.value = saved?.notes ?? ''
  },
  { immediate: true },
)

async function save() {
  saving.value = true
  try {
    await store.rateExercise(props.exerciseId, rating.value, notes.value.trim())
    toast.success(`Rated ${props.name} ${rating.value}/5.`)
    emit('close')
  } catch {
    toast.error('Could not save the rating.')
  } finally {
    saving.value = false
  }
}
</script>
