<template>
  <div>
    <div class="page-header">
      <h2>📋 部署历史</h2>
      <p>查看所有部署和回滚记录</p>
    </div>

    <div v-if="loading" class="empty-state">
      <div class="icon">⏳</div>
      <div>加载中...</div>
    </div>

    <div v-else-if="tasks.length === 0" class="empty-state">
      <div class="icon">📭</div>
      <div>暂无部署记录</div>
    </div>

    <div v-else class="flex flex-col gap-3">
      <div v-for="task in tasks" :key="task.id" class="card" style="cursor: pointer;" @click="toggleExpand(task.id)">
        <div class="flex justify-between items-center">
          <div>
            <div style="font-weight: 600; font-size: 15px;">
              {{ task.package_name || '未知包' }}
              <span style="font-weight: 400; color: var(--text-dim);">v{{ task.version }}</span>
            </div>
            <div class="text-sm text-muted" style="margin-top: 4px;">
              {{ task.environment }} · {{ formatDate(task.started_at) }}
              <span v-if="task.finished_at"> · 耗时 {{ duration(task.started_at, task.finished_at) }}</span>
            </div>
          </div>
          <div class="flex items-center gap-3">
            <StatusBadge :status="task.status" />
            <button
              v-if="task.status === 'failed' || task.status === 'success'"
              class="btn btn-ghost btn-sm"
              @click.stop="doRollback(task.id)"
            >
              ⏪ 回滚
            </button>
          </div>
        </div>

        <div v-if="task.deployed_apps && task.deployed_apps.length" style="margin-top: 10px; display: flex; flex-wrap: wrap; gap: 6px;">
          <span v-for="app in task.deployed_apps" :key="app" class="tag">{{ app }}</span>
          <span v-if="task.failed_app" class="tag" style="background: var(--red-soft); color: var(--red);">
            ✖ {{ task.failed_app }}
          </span>
        </div>

        <!-- Expanded logs -->
        <div v-if="expandedId === task.id && task.logs && task.logs.length" style="margin-top: 16px;">
          <LogTerminal :logs="task.logs" :title="`history — ${task.id}`" />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { getHistory, getHistoryDetail, rollback } from '../api'
import StatusBadge from '../components/StatusBadge.vue'
import LogTerminal from '../components/LogTerminal.vue'

const tasks = ref([])
const loading = ref(true)
const expandedId = ref(null)

onMounted(async () => {
  await loadHistory()
})

async function loadHistory() {
  loading.value = true
  try {
    const { data } = await getHistory()
    tasks.value = data.tasks || []
  } catch (e) {
    console.error(e)
  }
  loading.value = false
}

async function toggleExpand(id) {
  if (expandedId.value === id) {
    expandedId.value = null
    return
  }
  // Load full detail (with logs)
  try {
    const { data } = await getHistoryDetail(id)
    const idx = tasks.value.findIndex(t => t.id === id)
    if (idx >= 0) {
      tasks.value[idx] = data.task
    }
    expandedId.value = id
  } catch (e) {
    console.error(e)
  }
}

async function doRollback(id) {
  if (!confirm('确定要回滚此次部署？')) return
  try {
    const { data } = await rollback(id)
    alert('回滚已启动，任务ID: ' + data.task_id)
    await loadHistory()
  } catch (e) {
    alert('回滚失败: ' + (e.response?.data?.error || e.message))
  }
}

function formatDate(d) {
  if (!d) return ''
  return new Date(d).toLocaleString('zh-CN')
}

function duration(start, end) {
  if (!start || !end) return ''
  const ms = new Date(end) - new Date(start)
  const s = Math.floor(ms / 1000)
  if (s < 60) return s + '秒'
  return Math.floor(s / 60) + '分' + (s % 60) + '秒'
}
</script>
