import { defineStore } from 'pinia'
import { ref } from 'vue'
import { createBlockTemplate, deleteBlockTemplate, getBlockTemplate, listBlockTemplates, updateBlockTemplate } from '../api'
import type { BlockTemplate } from '../types'

// Saved blocks: combinations of exercises (a complex, a superset) that sessions copy.
export const useBlockTemplatesStore = defineStore('blockTemplates', () => {
  const templates = ref<BlockTemplate[]>([])
  const loading = ref(false)

  async function loadTemplates() {
    loading.value = true
    try {
      templates.value = await listBlockTemplates()
    } finally {
      loading.value = false
    }
  }

  return {
    templates,
    loading,
    loadTemplates,
    getTemplate: getBlockTemplate,
    createTemplate: createBlockTemplate,
    updateTemplate: updateBlockTemplate,
    deleteTemplate: deleteBlockTemplate,
  }
})
