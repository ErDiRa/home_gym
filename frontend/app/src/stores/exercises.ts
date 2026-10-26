import { defineStore } from 'pinia'
import { ref } from 'vue'
import {
  addExerciseMax,
  createExercise,
  deleteExerciseMax,
  getExercise,
  listExerciseMaxes,
  listExerciseRatings,
  listExercises,
  rateExercise as apiRateExercise,
  updateExercise,
} from '../api'
import { newestFirst } from '../components/exercise/maxes'
import type { Exercise, ExerciseMax, ExerciseMaxInput, ExerciseRating } from '../types'

export const useExercisesStore = defineStore('exercises', () => {
  const exercises = ref<Exercise[]>([])
  const loading = ref(false)
  // The user's ratings by exercise id, for the exercise picker and the tracker.
  const ratings = ref(new Map<string, ExerciseRating>())
  // The user's 1RM history by exercise id, newest (the current max) first.
  const maxes = ref(new Map<string, ExerciseMax[]>())

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

  async function loadMaxes() {
    const byExercise = new Map<string, ExerciseMax[]>()
    for (const m of await listExerciseMaxes()) byExercise.set(m.exercise_id, [...(byExercise.get(m.exercise_id) ?? []), m])
    maxes.value = new Map([...byExercise].map(([id, list]) => [id, newestFirst(list)]))
  }

  // Adds a 1RM to an exercise's history.
  async function addMax(exerciseId: string, input: ExerciseMaxInput) {
    const saved = await addExerciseMax(exerciseId, input)
    maxes.value = new Map(maxes.value).set(exerciseId, newestFirst([...(maxes.value.get(exerciseId) ?? []), saved]))
    return saved
  }

  async function deleteMax(max: ExerciseMax) {
    await deleteExerciseMax(max.id)
    const left = (maxes.value.get(max.exercise_id) ?? []).filter((m) => m.id !== max.id)
    maxes.value = new Map(maxes.value).set(max.exercise_id, left)
  }

  // An exercise's current max (its newest entry), or null.
  const currentMax = (exerciseId: string) => maxes.value.get(exerciseId)?.[0] ?? null

  return {
    exercises,
    loading,
    ratings,
    maxes,
    loadExercises,
    loadRatings,
    rateExercise,
    loadMaxes,
    addMax,
    deleteMax,
    currentMax,
    getExercise,
    createExercise,
    updateExercise,
  }
})
