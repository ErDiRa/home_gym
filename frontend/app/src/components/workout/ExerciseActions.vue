<template>
  <!-- The user's own rating of the exercise, set any time: filled with the stars once rated. -->
  <AppButton
    variant="ghost"
    size="touch"
    class="-mt-2 gap-0.5! px-2!"
    :class="aligned ? 'w-14 justify-start!' : { '-mr-2': !videoUrls.length }"
    :aria-label="rating ? `Rated ${rating} of 5, change the rating of ${name}` : `Rate ${name}`"
    @click="ratingOpen = true"
  >
    <SolidStarIcon v-if="rating" class="size-5 text-amber-400" aria-hidden="true" />
    <StarIcon v-else class="size-5 text-slate-400" aria-hidden="true" />
    <span v-if="rating" class="text-sm font-semibold text-slate-700">{{ rating }}</span>
  </AppButton>
  <ExerciseRatingSheet :open="ratingOpen" :exercise-id="exerciseId" :name="name" @close="ratingOpen = false" />
  <AppButton
    v-if="videoUrls.length"
    variant="ghost"
    size="touch"
    icon
    class="-mt-2 -mr-2"
    :aria-label="videoUrls.length > 1 ? `Videos for ${name}` : `Video for ${name}`"
    @click="videosOpen = true"
  >
    <VideoCameraIcon class="size-6" aria-hidden="true" />
  </AppButton>
  <!-- In a list of exercises the video button's place stays, so the stars line up. -->
  <span v-else-if="aligned" class="-mr-2 size-11 shrink-0" aria-hidden="true" />
  <ExerciseVideoSheet :open="videosOpen" :name="name" :urls="videoUrls" @close="videosOpen = false" />
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { StarIcon, VideoCameraIcon } from '@heroicons/vue/24/outline'
import { StarIcon as SolidStarIcon } from '@heroicons/vue/24/solid'
import AppButton from '../button/AppButton.vue'
import ExerciseRatingSheet from './ExerciseRatingSheet.vue'
import ExerciseVideoSheet from './ExerciseVideoSheet.vue'
import { useExercisesStore } from '../../stores/exercises'

// The buttons beside an exercise of a workout: its rating and its videos, each opening a sheet.
// aligned gives the rating a fixed width and keeps the place of a missing video button, so the
// stars of exercises listed one below the other line up.
const props = defineProps<{ exerciseId: string; name: string; aligned?: boolean }>()
const exercises = useExercisesStore()

// The exercise's videos; only http(s) links count.
const videoUrls = computed(() =>
  (exercises.exercises.find((e) => e.id === props.exerciseId)?.video_urls ?? []).filter((u) => /^https?:\/\//i.test(u)),
)
const videosOpen = ref(false)

// The user's rating of the exercise (0 = not rated).
const rating = computed(() => exercises.ratings.get(props.exerciseId)?.rating ?? 0)
const ratingOpen = ref(false)
</script>
