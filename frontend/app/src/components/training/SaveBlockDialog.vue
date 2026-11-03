<template>
  <TransitionRoot :show="open" as="template">
    <!-- Focuses the title, not the name: on a phone the keyboard would cover the dialog. -->
    <Dialog :initial-focus="title?.$el" class="relative z-50" @close="emit('close')">
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
            <DialogTitle ref="title" tabindex="-1" class="font-semibold text-slate-900 focus:outline-none">Save block</DialogTitle>
            <p class="mt-1 mb-3 text-sm text-slate-500">
              Keeps the {{ kindName }} of {{ exerciseNames }} to add to other sessions. Sets, reps and load are planned there.
            </p>
            <form @submit.prevent="save">
              <TextInput v-model="name" type="text" maxlength="100" placeholder="Name, e.g. Clean complex" aria-label="Name" />
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
import { computed, ref, watch } from 'vue'
import { Dialog, DialogPanel, DialogTitle, TransitionChild, TransitionRoot } from '@headlessui/vue'
import { toast } from 'vue-sonner'
import { ApiError } from '../../api'
import AppButton from '../button/AppButton.vue'
import TextInput from '../form/TextInput.vue'
import { useBlockTemplatesStore } from '../../stores/blockTemplates'
import { useLookupsStore } from '../../stores/lookups'
import type { BlockRow } from './trainingForm'

// Saves a block's kind and exercises (not its sets, reps or load) as a saved block, named first:
// the block's name, else its exercises.
const props = defineProps<{ open: boolean; block: BlockRow }>()
const emit = defineEmits<{ close: [] }>()
const store = useBlockTemplatesStore()
const lookups = useLookupsStore()

const title = ref<{ $el: HTMLElement } | null>(null)
const name = ref('')
const error = ref('')
const saving = ref(false)

const exerciseNames = computed(() => props.block.exercises.map((e) => e.name).join(', '))
const kindName = computed(() => (lookups.kindOf(props.block.kind)?.name ?? 'block').toLowerCase())

watch(
  () => props.open,
  (open) => {
    if (!open) return
    name.value = (props.block.name.trim() || exerciseNames.value).slice(0, 100)
    error.value = ''
  },
)

async function save() {
  saving.value = true
  error.value = ''
  try {
    const saved = await store.createTemplate({
      name: name.value,
      kind: props.block.kind,
      exercise_ids: props.block.exercises.map((e) => e.exercise_id),
    })
    store.templates = [...store.templates, saved].sort((a, b) => a.name.localeCompare(b.name))
    toast.success(`Saved "${saved.name}".`)
    emit('close')
  } catch (e) {
    error.value = e instanceof ApiError && (e.status === 400 || e.status === 409) ? e.message : 'Could not save the block.'
  } finally {
    saving.value = false
  }
}
</script>
