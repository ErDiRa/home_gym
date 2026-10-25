<template>
  <div class="space-y-3">
    <p
      v-if="!goals.length"
      class="rounded-lg border border-dashed border-slate-300 px-4 py-6 text-center text-sm text-slate-500"
    >
      No goals yet, e.g. "Better hips" or "Peak for the competition in week 10".
    </p>
    <div v-for="(goal, i) in goals" :key="goal.key" class="space-y-2 rounded-lg border border-slate-200 bg-white p-3">
      <div class="flex items-start gap-2">
        <span class="mt-2 w-5 shrink-0 text-sm font-semibold text-slate-400">{{ i + 1 }}</span>
        <div class="grid min-w-0 flex-1 gap-2 sm:grid-cols-2">
          <label class="text-xs font-medium text-slate-600">
            Goal<span class="text-red-600" aria-hidden="true"> *</span>
            <TextInput v-model="goal.title" required maxlength="200" placeholder="e.g. Better hips" class="mt-1" />
          </label>
          <label class="text-xs font-medium text-slate-600">
            Target
            <TextInput v-model="goal.target" maxlength="200" placeholder="e.g. Deep squat hold 2 min" class="mt-1" />
          </label>
        </div>
        <MoveRemoveButtons
          :index="i"
          :count="goals.length"
          :label="goal.title || `goal ${i + 1}`"
          class="mt-6"
          @move="move(goals, i, $event)"
          @remove="goals.splice(i, 1)"
        />
      </div>
      <div class="flex flex-wrap items-start gap-3 pl-7">
        <SegmentedControl v-model="goal.status" :options="statusOptions" label="Status" size="xs" />
        <TextInput v-model="goal.notes" multiline rows="1" placeholder="Notes" class="min-w-48 flex-1" />
      </div>
    </div>
    <AppButton variant="secondary" size="md" :disabled="goals.length >= MAX_GOALS" @click="goals.push(newGoal())">
      <PlusIcon class="size-4" aria-hidden="true" />
      Add goal
    </AppButton>
  </div>
</template>

<script setup lang="ts">
import { PlusIcon } from '@heroicons/vue/24/outline'
import AppButton from '../button/AppButton.vue'
import SegmentedControl from '../form/SegmentedControl.vue'
import TextInput from '../form/TextInput.vue'
import MoveRemoveButtons from '../ui/MoveRemoveButtons.vue'
import { move, nextKey } from '../training/trainingForm'
import { goalStatusLabels, type GoalStatus } from '../../types'
import { MAX_GOALS, type GoalRow } from './cycleForm'

// The editable goals of a cycle, in order: title, target, status and notes.
const goals = defineModel<GoalRow[]>({ required: true })

const statusOptions = (Object.keys(goalStatusLabels) as GoalStatus[]).map((s) => ({ value: s, label: goalStatusLabels[s] }))

const newGoal = (): GoalRow => ({ key: nextKey(), title: '', target: '', status: 'open', notes: '' })
</script>
