<template>
  <!-- Editing: the note, saved or (empty) removed. -->
  <div v-if="editing" class="mt-3 space-y-2">
    <TextInput
      ref="input"
      v-model="draft"
      multiline
      rows="2"
      size="touch"
      maxlength="1000"
      placeholder="e.g. BikeErg instead, avg HR 132"
      :aria-label="`Note on ${card.name}`"
    />
    <AppButton size="touch" class="w-full" :disabled="saving" @click="save">{{ saving ? 'Saving…' : 'Save' }}</AppButton>
    <AppButton variant="secondary" size="touch" class="w-full" :disabled="saving" @click="editing = false">Cancel</AppButton>
  </div>
  <!-- The note as saved; tapping it edits it. -->
  <button
    v-else-if="card.note"
    type="button"
    class="mt-3 flex w-full items-start gap-2 rounded-xl bg-slate-50 px-3 py-2 text-left text-sm text-slate-700 active:bg-slate-100"
    :aria-label="`Edit the note on ${card.name}`"
    @click="edit"
  >
    <ChatBubbleBottomCenterTextIcon class="mt-0.5 size-4 shrink-0 text-slate-400" aria-hidden="true" />
    <span class="min-w-0 flex-1 whitespace-pre-line">{{ card.note.notes }}</span>
  </button>
</template>

<script setup lang="ts">
import { nextTick, ref, useTemplateRef } from 'vue'
import { ChatBubbleBottomCenterTextIcon } from '@heroicons/vue/24/outline'
import { toast } from 'vue-sonner'
import { ApiError } from '../../api'
import { useWorkoutsStore } from '../../stores/workouts'
import AppButton from '../button/AppButton.vue'
import TextInput, { type TextInputExposed } from '../form/TextInput.vue'
import type { ExerciseCard } from './workoutPlan'

// The user's note on an exercise in this workout (e.g. a machine swapped, how it felt): shown when
// there is one, edited after tapping it or calling edit() (the card's "Add note" button). Saved
// right away; saving it empty removes it.
const props = defineProps<{ workoutId: string }>()
const card = defineModel<ExerciseCard>({ required: true })

const workouts = useWorkoutsStore()
const editing = ref(false)
const saving = ref(false)
const draft = ref('')
const input = useTemplateRef<TextInputExposed>('input')

async function edit() {
  draft.value = card.value.note?.notes ?? ''
  editing.value = true
  await nextTick()
  input.value?.focus()
}
defineExpose({ edit, editing })

async function save() {
  const text = draft.value.trim()
  saving.value = true
  try {
    if (text) {
      card.value.note = await workouts.noteExercise(props.workoutId, {
        exercise_id: card.value.exerciseId,
        block_position: card.value.blockPosition,
        exercise_position: card.value.position,
        notes: text,
      })
    } else if (card.value.note) {
      await workouts.deleteExerciseNote(props.workoutId, card.value.note.id)
      card.value.note = null
    }
    editing.value = false
  } catch (e) {
    toast.error(e instanceof ApiError && e.status === 400 ? e.message : 'Could not save the note.')
  } finally {
    saving.value = false
  }
}
</script>
