<template>
  <!-- The options stay rendered while the dialog fades out, so its text doesn't vanish first. -->
  <TransitionRoot :show="!!pendingConfirm" as="template" @after-leave="shown = null">
    <Dialog class="relative z-50" @close="answerConfirm(false)">
      <TransitionChild
        as="template"
        enter="duration-150 ease-out"
        enter-from="opacity-0"
        leave="duration-100 ease-in"
        leave-to="opacity-0"
      >
        <div class="fixed inset-0 bg-slate-900/40 transition-opacity" aria-hidden="true" />
      </TransitionChild>

      <div
        class="fixed inset-0 flex items-end justify-center p-4 pb-[calc(1rem+env(safe-area-inset-bottom))] sm:items-start sm:pt-[15vh]"
      >
        <TransitionChild
          as="template"
          enter="duration-150 ease-out"
          enter-from="translate-y-4 opacity-0 sm:translate-y-0 sm:scale-95"
          leave="duration-100 ease-in"
          leave-to="translate-y-4 opacity-0 sm:translate-y-0 sm:scale-95"
        >
          <DialogPanel class="w-full max-w-md rounded-2xl bg-white p-5 shadow-2xl transition sm:rounded-xl">
            <div class="flex gap-3">
              <span
                v-if="shown?.danger"
                class="flex size-10 shrink-0 items-center justify-center rounded-full bg-red-50 text-red-600"
                aria-hidden="true"
              >
                <ExclamationTriangleIcon class="size-5" />
              </span>
              <div class="min-w-0 pt-0.5">
                <DialogTitle class="text-base font-semibold text-slate-900">{{ shown?.title }}</DialogTitle>
                <DialogDescription v-if="shown?.message" class="mt-1 text-sm text-slate-600">
                  {{ shown.message }}
                </DialogDescription>
                <AppCheckbox v-if="shown?.option" v-model="option" class="mt-3">{{ shown.option }}</AppCheckbox>
              </div>
            </div>
            <!-- Cancel comes first, so it gets the focus: Enter never confirms by accident. On phones
                 the buttons stack full width, the confirming one on top. -->
            <div class="mt-5 flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
              <AppButton variant="secondary" class="max-sm:h-11 max-sm:w-full" @click="answerConfirm(false)">
                {{ shown?.cancelLabel ?? 'Cancel' }}
              </AppButton>
              <AppButton
                :variant="shown?.danger ? 'danger' : 'primary'"
                class="max-sm:h-11 max-sm:w-full"
                @click="answerConfirm(true, option)"
              >
                {{ shown?.confirmLabel ?? 'OK' }}
              </AppButton>
            </div>
          </DialogPanel>
        </TransitionChild>
      </div>
    </Dialog>
  </TransitionRoot>
</template>

<script setup lang="ts">
import { ref, shallowRef, watch } from 'vue'
import { Dialog, DialogDescription, DialogPanel, DialogTitle, TransitionChild, TransitionRoot } from '@headlessui/vue'
import { ExclamationTriangleIcon } from '@heroicons/vue/24/outline'
import { answerConfirm, pendingConfirm, type ConfirmOptions } from '../../confirm'
import AppButton from '../button/AppButton.vue'
import AppCheckbox from '../form/AppCheckbox.vue'

// The app's one confirmation modal; open it with confirmDialog() from confirm.ts.
// The question on screen; kept after answering until the fade-out ends.
const shown = shallowRef<ConfirmOptions | null>(null)
// The state of the question's checkbox, if it has one; unchecked for every new question.
const option = ref(false)
watch(pendingConfirm, (pending) => {
  if (pending) {
    shown.value = pending
    option.value = false
  }
})
</script>
