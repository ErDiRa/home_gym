import { defineStore } from 'pinia'
import { ref } from 'vue'
import { createTraining, deleteTraining, duplicateTraining, getTraining, listTrainings, updateTraining } from '../api'
import type { Training } from '../types'

export const useTrainingsStore = defineStore('trainings', () => {
  const trainings = ref<Training[]>([])
  const loading = ref(false)

  async function loadTrainings() {
    loading.value = true
    try {
      trainings.value = await listTrainings()
    } finally {
      loading.value = false
    }
  }

  return {
    trainings,
    loading,
    loadTrainings,
    getTraining,
    createTraining,
    updateTraining,
    deleteTraining,
    duplicateTraining,
  }
})
