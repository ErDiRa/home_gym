<template>
  <span
    :class="
      rating.rating >= 4
        ? 'border-amber-200 bg-amber-50 text-amber-700'
        : rating.rating <= 2
          ? 'border-slate-200 bg-slate-100 text-slate-500'
          : 'border-slate-200 bg-white text-slate-700'
    "
    :title="rating.notes ? `Rated ${rating.rating}/5: ${rating.notes}` : `Rated ${rating.rating}/5`"
    class="inline-flex shrink-0 items-center gap-0.5 rounded-full border px-2 py-0.5 text-xs font-semibold"
  >
    <StarIcon class="size-3" aria-hidden="true" />
    {{ rating.rating }}
    <span class="sr-only">of 5</span>
  </span>
</template>

<script setup lang="ts">
import { StarIcon } from '@heroicons/vue/16/solid'
import type { ExerciseRating } from '../../types'

// The client's rating of an exercise; the notes show on hover.
defineProps<{ rating: ExerciseRating }>()
</script>
