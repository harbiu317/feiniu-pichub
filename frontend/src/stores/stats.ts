import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from '@/api'

export const useStatsStore = defineStore('stats', () => {
  const data = ref<any>({})

  async function load() {
    try {
      data.value = await api.stats()
    } catch {}
  }

  return { data, load }
})
