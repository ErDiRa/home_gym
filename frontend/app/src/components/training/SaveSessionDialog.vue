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

      <div class="fixed inset-0 flex items-end justify-center sm:items-center sm:p-4">
        <TransitionChild
          as="template"
          enter="duration-200 ease-out"
          enter-from="translate-y-full sm:translate-y-0 sm:scale-95 sm:opacity-0"
          leave="duration-150 ease-in"
          leave-to="translate-y-full sm:translate-y-0 sm:scale-95 sm:opacity-0"
        >
          <DialogPanel class="w-full max-w-md rounded-t-2xl bg-white p-4 pb-[calc(1rem+env(safe-area-inset-bottom))] shadow-2xl transition sm:rounded-2xl">
            <DialogTitle class="font-semibold text-slate-900">Save to library</DialogTitle>
            <p class="mt-1 mb-3 text-sm text-slate-500">
              Saves a copy of this session with its blocks to your library, to add to other trainings. Changing one later doesn't change the
              other.
            </p>
            <form @submit.prevent="save">
              <TextInput v-model="name" type="text" maxlength="100" placeholder="Name, e.g. Lower body A" aria-label="Name" />
              <p v-if="error" class="mt-2 text-sm text-red-700">{{ error }}</p>
              <div class="mt-4 flex justify-end gap-2">
                <AppButton variant="secondary" :disabled="saving" @click="emit('close')">Cancel</AppButton>
                <AppButton type="submit" :disabled="saving || !name.trim()">{{ saving ? 'Saving…' : 'Save' }}</AppButton>
              </div>
            </form>
          </DialogPanel>
        </TransitionChild>
      </div>
    </Dialog>
  </TransitionRoot>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { Dialog, DialogPanel, DialogTitle, TransitionChild, TransitionRoot } from '@headlessui/vue'
import { toast } from 'vue-sonner'
import { ApiError } from '../../api'
import AppButton from '../button/AppButton.vue'
import TextInput from '../form/TextInput.vue'
import { useSessionsStore } from '../../stores/sessions'
import { toSessionInput, type SessionRow } from './trainingForm'

// Saves a training's session as a new template of the library, named first (the session's name).
const props = defineProps<{ open: boolean; session: SessionRow }>()
const emit = defineEmits<{ close: [] }>()
const store = useSessionsStore()

const name = ref('')
const error = ref('')
const saving = ref(false)

watch(
  () => props.open,
  (open) => {
    if (!open) return
    name.value = props.session.name.trim().slice(0, 100)
    error.value = ''
  },
)

async function save() {
  saving.value = true
  error.value = ''
  try {
    // A new template: without the training session's id.
    const saved = await store.createSession({ ...toSessionInput(props.session), id: undefined, name: name.value })
    toast.success(`Saved "${saved.name}" to your library.`)
    emit('close')
    // So "From library" offers it right away.
    store.loadSessions().catch(() => {})
  } catch (e) {
    error.value = e instanceof ApiError && e.status === 400 ? e.message : 'Could not save the session.'
  } finally {
    saving.value = false
  }
}
</script>
