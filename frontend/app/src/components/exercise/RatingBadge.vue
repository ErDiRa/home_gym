<template>
  <AppBadge
    :color="rating.rating >= 4 ? 'yellow' : 'gray'"
    :title="rating.notes ? `Rated ${rating.rating}/5: ${rating.notes}` : `Rated ${rating.rating}/5`"
  >
    <StarIcon class="-mx-0.5 size-3" aria-hidden="true" />
    {{ rating.rating }}
    <span class="sr-only">of 5</span>
  </AppBadge>
</template>

<script setup lang="ts">
import { StarIcon } from '@heroicons/vue/16/solid'
import AppBadge from '../ui/AppBadge.vue'
import type { ExerciseRating } from '../../types'

// The user's rating of an exercise; the notes show on hover.
defineProps<{ rating: ExerciseRating }>()
</script>
