import { defineStore } from 'pinia'
import { ref } from 'vue'
import {
  deleteResult,
  deleteSet,
  deleteWorkout,
  finishWorkout,
  getWeek,
  getWorkout,
  listWorkouts,
  logResult,
  logSet,
  startWorkout,
} from '../api'
import { nextWeek, toIso, weekStart } from '../dates'
import type { Week } from '../types'

// The tracker's data: the current week, and the api functions for workouts, their sets and the
// scores of their WOD blocks.
export const useWorkoutsStore = defineStore('workouts', () => {
  const week = ref<Week | null>(null)
  const loading = ref(false)

  // Loads the current local week (Monday to Sunday) as one of the active trainings plans it
  // (the first without one).
  async function loadWeek(trainingId?: string) {
    loading.value = true
    try {
      const start = weekStart()
      week.value = await getWeek(toIso(start), start, nextWeek(start), trainingId)
    } finally {
      loading.value = false
    }
  }

  return {
    week,
    loading,
    loadWeek,
    listWorkouts,
    getWorkout,
    startWorkout,
    logSet,
    deleteSet,
    logResult,
    deleteResult,
    finishWorkout,
    deleteWorkout,
  }
})
