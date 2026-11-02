import { defineStore } from 'pinia'
import { ref } from 'vue'
import {
  deleteBodyWeight,
  deleteDayMacros,
  getNutritionProfile,
  listBodyWeights,
  listNutritionDays,
  saveBodyWeight,
  saveDayMacros,
  saveNutritionProfile,
} from '../api'
import type { BodyWeight, BodyWeightInput, NutritionProfile, NutritionProfileInput } from '../types'

// The user's body (profile and weight history) and the api functions for the days' macros.
export const useNutritionStore = defineStore('nutrition', () => {
  const profile = ref<NutritionProfile | null>(null)
  const weights = ref<BodyWeight[]>([])
  // Whether the profile was loaded, as null means none was entered.
  const profileLoaded = ref(false)

  async function loadProfile() {
    profile.value = await getNutritionProfile()
    profileLoaded.value = true
  }

  async function saveProfile(input: NutritionProfileInput) {
    profile.value = await saveNutritionProfile(input)
  }

  async function loadWeights() {
    weights.value = await listBodyWeights()
  }

  async function logWeight(input: BodyWeightInput) {
    await saveBodyWeight(input)
    await loadWeights()
  }

  async function removeWeight(id: string) {
    await deleteBodyWeight(id)
    weights.value = weights.value.filter((w) => w.id !== id)
  }

  return {
    profile,
    profileLoaded,
    loadProfile,
    saveProfile,
    weights,
    loadWeights,
    logWeight,
    removeWeight,
    listNutritionDays,
    saveDayMacros,
    deleteDayMacros,
  }
})
