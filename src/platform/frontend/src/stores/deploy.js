import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useDeployStore = defineStore('deploy', () => {
  const currentTaskId = ref(null)
  const logs = ref([])
  const status = ref(null) // running, success, failed, rolling_back, rolled_back

  function reset() {
    currentTaskId.value = null
    logs.value = []
    status.value = null
  }

  function addLog(line) {
    logs.value.push(line)
    // Keep last 5000 lines
    if (logs.value.length > 5000) {
      logs.value = logs.value.slice(-5000)
    }
  }

  function setStatus(s) {
    status.value = s
  }

  return { currentTaskId, logs, status, reset, addLog, setStatus }
})
