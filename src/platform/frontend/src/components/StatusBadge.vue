<template>
  <span :class="['badge', `badge-${status}`]">
    <span class="badge-dot"></span>
    {{ label }}
  </span>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({ status: String })

const labelMap = {
  running: '部署中',
  success: '成功',
  failed: '失败',
  rolling_back: '回滚中',
  rolled_back: '已回滚'
}

const label = computed(() => labelMap[props.status] || props.status)
</script>
