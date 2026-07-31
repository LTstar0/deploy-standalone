<template>
  <div>
    <div class="flex justify-between items-center mb-4">
      <span class="text-sm text-muted">
        已选 {{ selected.length }} / {{ apps.length }} 个应用
      </span>
      <button class="btn btn-ghost btn-sm" @click="toggleAll">
        {{ selected.length === apps.length ? '取消全选' : '全选' }}
      </button>
    </div>
    <div class="app-select-list">
      <div
        v-for="app in apps"
        :key="app.key"
        class="app-select-card"
        :class="{ selected: selected.includes(app.key) }"
        @click="toggle(app.key)"
      >
        <div class="app-check">
          <span v-if="selected.includes(app.key)">✓</span>
        </div>
        <div class="app-name">{{ app.name }}</div>
        <div class="app-desc">{{ app.description }}</div>
        <div class="app-meta">
          <span>顺序: {{ app.deploy_order }}</span>
          <span v-if="app.dependencies && app.dependencies.length">
            依赖: {{ app.dependencies.join(', ') }}
          </span>
          <span v-if="app.has_health_check">✓ 健康检查</span>
          <span v-if="app.rollback_support">✓ 可回滚</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
const props = defineProps({
  apps: { type: Array, default: () => [] },
  modelValue: { type: Array, default: () => [] }
})

const emit = defineEmits(['update:modelValue'])

const selected = computed(() => props.modelValue)

import { computed } from 'vue'

function toggle(key) {
  const arr = [...props.modelValue]
  const idx = arr.indexOf(key)
  if (idx >= 0) {
    arr.splice(idx, 1)
  } else {
    arr.push(key)
  }
  emit('update:modelValue', arr)
}

function toggleAll() {
  if (props.modelValue.length === props.apps.length) {
    emit('update:modelValue', [])
  } else {
    emit('update:modelValue', props.apps.map(a => a.key))
  }
}
</script>
