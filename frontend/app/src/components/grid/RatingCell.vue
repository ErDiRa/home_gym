<template>
  <!-- click.stop keeps rating from also opening the row. -->
  <div class="flex h-full items-center" @click.stop>
    <StarRating
      :model-value="rating"
      size="sm"
      :label="`Your rating for ${exercise.name}`"
      :disabled="saving"
      :title="saved?.notes || undefined"
      @update:model-value="rate"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import type { ICellRendererParams } from 'ag-grid-community'
import { toast } from 'vue-sonner'
import StarRating from '../exercise/StarRating.vue'
import { useExercisesStore } from '../../stores/exercises'
import type { Exercise } from '../../types'

// AG Grid cell renderer: the user's own rating of the row's exercise as stars; clicking one saves
// it right away, keeping the note (the note shows on hover and is edited in the exercise form).
const props = defineProps<{ params: ICellRendererParams<Exercise> }>()
const store = useExercisesStore()
const saving = ref(false)

const exercise = computed(() => props.params.data!)
const saved = computed(() => store.ratings.get(exercise.value.id))
const rating = computed(() => saved.value?.rating ?? 0)

async function rate(value: number) {
  if (value === rating.value) return
  saving.value = true
  try {
    await store.rateExercise(exercise.value.id, value, saved.value?.notes ?? '')
    // The column sorts and filters by the rating, which the grid caches.
    props.params.api.refreshCells({ columns: ['rating'], force: true })
  } catch {
    toast.error('Could not save the rating.')
  } finally {
    saving.value = false
  }
}
</script>
