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
              <DialogTitle class="min-w-0 flex-1 truncate font-semibold text-slate-900">1RM · {{ name }}</DialogTitle>
              <AppButton variant="ghost" :size="touch ? 'touch' : 'md'" icon aria-label="Close" @click="emit('close')">
                <XMarkIcon class="size-6" aria-hidden="true" />
              </AppButton>
            </div>
            <div class="overflow-y-auto p-4 pb-[calc(1rem+env(safe-area-inset-bottom))]">
              <MaxHistory :exercise-id="exerciseId" :touch="touch" @added="emit('added', $event)" />
            </div>
          </DialogPanel>
        </TransitionChild>
      </div>
    </Dialog>
  </TransitionRoot>
</template>

<script setup lang="ts">
import { Dialog, DialogPanel, DialogTitle, TransitionChild, TransitionRoot } from '@headlessui/vue'
import { XMarkIcon } from '@heroicons/vue/24/outline'
import AppButton from '../button/AppButton.vue'
import MaxHistory from './MaxHistory.vue'
import type { ExerciseMax } from '../../types'

// The user's 1RM of an exercise: the current max, its history and adding one, saved right away.
// A bottom sheet in a workout (touch), a centered dialog on the dashboard's PR page.
withDefaults(defineProps<{ open: boolean; exerciseId: string; name: string; touch?: boolean }>(), { touch: false })
const emit = defineEmits<{ close: []; added: [max: ExerciseMax] }>()
</script>
