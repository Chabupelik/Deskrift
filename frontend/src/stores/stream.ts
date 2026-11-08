import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useStreamStore = defineStore('stream', () => {
  const connected = ref(false)
  const pcOnline = ref(false)
  const remoteWidth = ref(1920)
  const remoteHeight = ref(1080)
  const quality = ref<'high' | 'medium' | 'low'>('high')

  function reset() {
    connected.value = false
    pcOnline.value = false
  }

  return {
    connected,
    pcOnline,
    remoteWidth,
    remoteHeight,
    quality,
    reset,
  }
})
