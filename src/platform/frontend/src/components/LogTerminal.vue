<template>
  <div class="terminal">
    <div class="terminal-header">
      <div class="terminal-dots">
        <span></span><span></span><span></span>
      </div>
      <div class="terminal-title">{{ title || 'deploy-log' }}</div>
    </div>
    <div class="terminal-body" ref="bodyRef">
      <div v-for="(line, i) in logs" :key="i" class="log-line">
        <span class="log-time">{{ formatTime(line.time) }}</span>
        <span :class="['log-msg', line.level]">{{ line.message }}</span>
      </div>
      <div v-if="logs.length === 0" style="color: var(--text-muted); padding: 20px 0; text-align: center;">
        等待日志输出...
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, watch, nextTick } from 'vue'

const props = defineProps({
  logs: { type: Array, default: () => [] },
  title: { type: String, default: '' }
})

const bodyRef = ref(null)

watch(
  () => props.logs.length,
  async () => {
    await nextTick()
    if (bodyRef.value) {
      bodyRef.value.scrollTop = bodyRef.value.scrollHeight
    }
  }
)

function formatTime(t) {
  if (!t) return ''
  const d = new Date(t)
  return d.toLocaleTimeString('zh-CN', { hour12: false })
}
</script>
