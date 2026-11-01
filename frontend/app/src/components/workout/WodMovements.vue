<template>
  <div class="pb-1">
    <p v-if="notes" class="pt-2 text-sm text-slate-500">{{ notes }}</p>
    <!-- The WOD as written, one short line per step; rating and video only on an exercise's first. -->
    <ul class="py-2">
      <li v-for="l in lines" :key="l.key" class="flex min-h-7 items-center gap-1">
        <span class="min-w-0 flex-1 text-base text-slate-900">
          <span class="font-semibold">{{ l.reps }}</span> {{ l.name }}<span v-if="l.load" class="text-sm text-slate-500"> · {{ l.load }}</span>
        </span>
        <ExerciseActions v-if="l.first" :exercise-id="l.exerciseId" :name="l.name" compact aligned />
      </li>
    </ul>
  </div>
</template>

<script setup lang="ts">
import ExerciseActions from './ExerciseActions.vue'
import type { WodLine } from './workoutPlan'

// The exercises of a WOD block in plan order, like the programming reads (18 Hang Clean, 1 Rope
// Climb, 15 Hang Clean…), with the rating and video buttons once per exercise. The score is
// entered below, see WodResult.
defineProps<{ lines: WodLine[]; notes: string }>()
</script>
