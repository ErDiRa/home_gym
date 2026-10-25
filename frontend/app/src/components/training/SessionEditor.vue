<template>
  <div class="rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
    <div class="flex items-center gap-2" :class="{ 'mb-2': !collapsed }">
      <AppButton
        v-if="collapsible"
        variant="ghost"
        size="sm"
        icon
        :aria-expanded="!collapsed"
        :aria-controls="bodyId"
        :aria-label="`${collapsed ? 'Expand' : 'Collapse'} ${session.name || label}`"
        @click="collapsed = !collapsed"
      >
        <ChevronRightIcon class="size-5 transition-transform" :class="{ 'rotate-90': !collapsed }" aria-hidden="true" />
      </AppButton>
      <TextInput
        v-model="session.name"
        type="text"
        required
        maxlength="100"
        placeholder="Session name, e.g. Session 1"
        :aria-label="`${label} name`"
        class="flex-1 text-base! font-semibold!"
      />
      <MoveRemoveButtons
        v-if="count !== undefined"
        :index="index ?? 0"
        :count="count"
        :label="session.name || label"
        @move="emit('move', $event)"
        @remove="emit('remove')"
      />
    </div>
    <p v-if="collapsed" class="mt-1 ml-10 text-xs text-slate-500">
      {{ summary }}
      <span v-if="problem" class="ml-1 text-amber-700">· {{ problem }}</span>
    </p>
    <div v-show="!collapsed" :id="bodyId">
      <TextInput v-model="session.notes" type="text" placeholder="Notes for this session" :aria-label="`${label} notes`" class="mb-3" />
      <div class="space-y-3">
        <TrainingBlockEditor
          v-for="(block, b) in session.blocks"
          :key="block.key"
          v-model="session.blocks[b]"
          :index="b"
          :count="session.blocks.length"
          @move="move(session.blocks, b, $event)"
          @remove="session.blocks.splice(b, 1)"
        />
      </div>
      <AppButton variant="secondary" size="sm" class="mt-3" @click="session.blocks.push(newBlock())">
        <PlusIcon class="size-4" aria-hidden="true" />
        Add block
      </AppButton>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, useId } from 'vue'
import { ChevronRightIcon, PlusIcon } from '@heroicons/vue/24/outline'
import AppButton from '../button/AppButton.vue'
import TextInput from '../form/TextInput.vue'
import MoveRemoveButtons from '../ui/MoveRemoveButtons.vue'
import TrainingBlockEditor from './TrainingBlockEditor.vue'
import { move, newBlock, sessionProblem, type SessionRow } from './trainingForm'

// A session with its blocks, as a training or the library edits it. With count set it also
// shows the controls to move or remove it within the training's list; collapsible lets it fold
// to its name and a summary (startCollapsed: folded at first).
const props = withDefaults(
  defineProps<{ index?: number; count?: number; label?: string; collapsible?: boolean; startCollapsed?: boolean }>(),
  { label: 'Session' },
)
const emit = defineEmits<{ move: [by: -1 | 1]; remove: [] }>()
// Edited in place; the parent owns the rows.
const session = defineModel<SessionRow>({ required: true })

const collapsed = ref(props.collapsible && props.startCollapsed)
const bodyId = useId()
// E.g. "3 blocks · 7 exercises", shown while folded.
const summary = computed(() => {
  const blocks = session.value.blocks.length
  const exercises = session.value.blocks.reduce((n, b) => n + b.exercises.length, 0)
  return `${blocks} ${blocks === 1 ? 'block' : 'blocks'} · ${exercises} ${exercises === 1 ? 'exercise' : 'exercises'}`
})
// Shown while folded, so a problem isn't hidden.
const problem = computed(() => (collapsed.value ? sessionProblem(session.value) : ''))
</script>
