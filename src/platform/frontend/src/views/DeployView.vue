<template>
  <div>
    <div class="page-header">
      <h2>🚀 部署控制台</h2>
      <p v-if="pkg">{{ pkg.project_name }} v{{ pkg.version }} · {{ pkg.environment }}</p>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="empty-state">
      <div class="icon">⏳</div>
      <div>加载包信息...</div>
    </div>

    <!-- Select & Deploy -->
    <template v-else-if="!deploying && !finished">
      <AppSelector
        v-if="pkg"
        :apps="pkg.apps"
        v-model="selectedApps"
      />

      <div class="mt-6 flex gap-3">
        <button
          class="btn btn-primary"
          :disabled="selectedApps.length === 0"
          @click="doDeploy"
        >
          🚀 开始部署 ({{ selectedApps.length }} 个应用)
        </button>
        <router-link to="/packages" class="btn btn-ghost">← 返回</router-link>
      </div>
    </template>

    <!-- Deploying / Finished -->
    <template v-if="deploying || finished">
      <div class="flex justify-between items-center mb-4">
        <div class="flex items-center gap-3">
          <StatusBadge :status="deployStatus" />
          <span class="text-sm text-muted" v-if="deploying">
            正在部署中...
          </span>
        </div>
        <div class="flex gap-3" v-if="finished">
          <router-link to="/packages" class="btn btn-ghost btn-sm">← 返回</router-link>
          <router-link to="/history" class="btn btn-ghost btn-sm">查看历史</router-link>
        </div>
      </div>

      <LogTerminal :logs="logs" :title="`deploy — ${pkg?.project_name || ''}`" />
    </template>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import { getPackage, startDeploy, connectWS } from '../api'
import AppSelector from '../components/AppSelector.vue'
import LogTerminal from '../components/LogTerminal.vue'
import StatusBadge from '../components/StatusBadge.vue'

const route = useRoute()
const packageId = route.params.id

const pkg = ref(null)
const loading = ref(true)
const selectedApps = ref([])
const deploying = ref(false)
const finished = ref(false)
const deployStatus = ref('running')
const logs = ref([])

let ws = null

onMounted(async () => {
  try {
    const { data } = await getPackage(packageId)
    pkg.value = data.package
    // Auto-select all enabled
    selectedApps.value = (pkg.value.apps || [])
      .filter(a => a.enabled)
      .map(a => a.key)
  } catch (e) {
    alert('加载包失败')
  }
  loading.value = false
})

onUnmounted(() => {
  if (ws) ws.close()
})

async function doDeploy() {
  deploying.value = true
  logs.value = []
  deployStatus.value = 'running'

  try {
    const { data } = await startDeploy(packageId, selectedApps.value)
    const taskId = data.task_id

    ws = connectWS(
      taskId,
      (msg) => {
        if (msg.type === 'log') {
          logs.value.push(msg.data)
        } else if (msg.type === 'done') {
          deployStatus.value = msg.data
          deploying.value = false
          finished.value = true
        } else if (msg.type === 'status') {
          deployStatus.value = msg.data
        }
      },
      () => {
        if (deploying.value) {
          deploying.value = false
          finished.value = true
          if (deployStatus.value === 'running') {
            deployStatus.value = 'failed'
          }
        }
      }
    )
  } catch (e) {
    alert('启动部署失败: ' + (e.response?.data?.error || e.message))
    deploying.value = false
  }
}
</script>
