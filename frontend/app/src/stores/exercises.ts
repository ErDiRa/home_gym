import { defineStore } from 'pinia'
import { ref } from 'vue'
import { createExercise, getExercise, listExerciseRatings, listExercises, rateExercise as apiRateExercise, updateExercise } from '../api'
import type { Exercise, ExerciseRating } from '../types'

export const useExercisesStore = defineStore('exercises', () => {
  const exercises = ref<Exercise[]>([])
  const loading = ref(false)
  // The user's ratings by exercise id, for the exercise picker and the tracker.
  const ratings = ref(new Map<string, ExerciseRating>())

  async function loadExercises() {
    loading.value = true
    try {
      exercises.value = await listExercises()
    } finally {
      loading.value = false
    }
  }

  async function loadRatings() {
    const loaded = await listExerciseRatings()
    ratings.value = new Map(loaded.map((r) => [r.exercise_id, r]))
  }

  // Rates an exercise, replacing the earlier rating.
  async function rateExercise(id: string, rating: number, notes: string) {
    const saved = await apiRateExercise(id, rating, notes)
    ratings.value = new Map(ratings.value).set(id, saved)
  }

  return { exercises, loading, ratings, loadExercises, loadRatings, rateExercise, getExercise, createExercise, updateExercise }
})
