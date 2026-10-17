import { defineStore } from 'pinia'
import { ref } from 'vue'
import { createClient, deleteClient, getClient, listClientExerciseRatings, listClients, updateClient } from '../api'
import type { Client, ExerciseRating } from '../types'

export const useClientsStore = defineStore('clients', () => {
  const clients = ref<Client[]>([])
  const loading = ref(false)
  // How one client rated exercises, by exercise id; null while no client is picked.
  const ratings = ref<Map<string, ExerciseRating> | null>(null)
  let ratingsFor: string | null = null

  async function loadClients() {
    loading.value = true
    try {
      clients.value = await listClients()
    } finally {
      loading.value = false
    }
  }

  // Loads the ratings of a client (or clears them for null).
  async function loadRatings(clientId: string | null) {
    ratings.value = null
    ratingsFor = clientId
    if (!clientId) return
    const loaded = await listClientExerciseRatings(clientId)
    // Ignore the answer if another client was picked meanwhile.
    if (ratingsFor === clientId) ratings.value = new Map(loaded.map((r) => [r.exercise_id, r]))
  }

  return {
    clients,
    loading,
    ratings,
    loadClients,
    loadRatings,
    getClient,
    createClient,
    updateClient,
    deleteClient,
  }
})
