import { defineStore } from 'pinia'
import { ref } from 'vue'
import { createExercise, getExercise, listExercises, updateExercise } from '../api'
import type { Exercise } from '../types'

export const useExercisesStore = defineStore('exercises', () => {
  const exercises = ref<Exercise[]>([])
  const loading = ref(false)

  async function loadExercises() {
    loading.value = true
    try {
      exercises.value = await listExercises()
    } finally {
      loading.value = false
    }
  }

  return { exercises, loading, loadExercises, getExercise, createExercise, updateExercise }
})
