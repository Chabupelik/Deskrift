import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useStreamStore = defineStore('stream', () => {
  const connected = ref(false)
  const quality = ref<'high' | 'medium' | 'low'>('high')
  const pcOnline = ref(false)

  return { connected, quality, pcOnline }
})

