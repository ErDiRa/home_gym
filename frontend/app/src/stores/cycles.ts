import { defineStore } from 'pinia'
import { ref } from 'vue'
import { createCycle, deleteCycle, getCycle, importCycle, listCycles, updateCycle } from '../api'
import type { CycleSummary } from '../types'

export const useCyclesStore = defineStore('cycles', () => {
  const cycles = ref<CycleSummary[]>([])
  const loading = ref(false)

  async function loadCycles() {
    loading.value = true
    try {
      cycles.value = await listCycles()
    } finally {
      loading.value = false
    }
  }

  return { cycles, loading, loadCycles, getCycle, createCycle, updateCycle, deleteCycle, importCycle }
})
