<template>
  <div>
    <div class="page-header">
      <h2>📦 产品包管理</h2>
      <p>上传产品包（.tar.gz / .zip），解析后即可选择应用进行部署</p>
    </div>

    <UploadZone @upload="handleUpload" />

    <div class="mt-6">
      <div v-if="loading" class="empty-state">
        <div class="icon">⏳</div>
        <div>加载中...</div>
      </div>

      <div v-else-if="packages.length === 0" class="empty-state">
        <div class="icon">📭</div>
        <div>暂无产品包，请上传</div>
      </div>

      <div v-else class="card-grid">
        <div v-for="pkg in packages" :key="pkg.id" class="card">
          <div class="flex justify-between items-center">
            <div>
              <div style="font-size: 16px; font-weight: 600;">{{ pkg.project_name || pkg.file_name }}</div>
              <div class="text-sm text-muted" style="margin-top: 4px;">
                v{{ pkg.version }} · {{ pkg.environment }}
              </div>
            </div>
            <span class="tag">{{ pkg.apps?.length || 0 }} 个应用</span>
          </div>

          <div style="margin-top: 12px; font-size: 12px; color: var(--text-muted);">
            <div>📄 {{ pkg.file_name }}</div>
            <div style="margin-top: 2px;">📏 {{ formatSize(pkg.size_bytes) }} · {{ formatDate(pkg.uploaded_at) }}</div>
          </div>

          <div v-if="pkg.apps && pkg.apps.length" style="margin-top: 12px; display: flex; flex-wrap: wrap; gap: 6px;">
            <span v-for="app in pkg.apps" :key="app.key" class="tag">{{ app.name }}</span>
          </div>

          <div style="margin-top: 16px; display: flex; gap: 8px;">
            <router-link :to="`/deploy/${pkg.id}`" class="btn btn-primary btn-sm">
              🚀 部署
            </router-link>
            <button class="btn btn-danger btn-sm" @click="handleDelete(pkg.id)">删除</button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { getPackages, uploadPackage, deletePackage } from '../api'
import UploadZone from '../components/UploadZone.vue'

const packages = ref([])
const loading = ref(true)

onMounted(async () => {
  await loadPackages()
})

async function loadPackages() {
  loading.value = true
  try {
    const { data } = await getPackages()
    packages.value = data.packages || []
  } catch (e) {
    console.error('加载包列表失败', e)
  }
  loading.value = false
}

async function handleUpload(file, onProgress, onDone) {
  try {
    await uploadPackage(file, onProgress)
    onDone()
    await loadPackages()
  } catch (e) {
    onDone()
    alert('上传失败: ' + (e.response?.data?.error || e.message))
  }
}

async function handleDelete(id) {
  if (!confirm('确定要删除此产品包？')) return
  try {
    await deletePackage(id)
    await loadPackages()
  } catch (e) {
    alert('删除失败')
  }
}

function formatSize(bytes) {
  if (!bytes) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB']
  let i = 0
  let val = bytes
  while (val >= 1024 && i < units.length - 1) { val /= 1024; i++ }
  return val.toFixed(1) + ' ' + units[i]
}

function formatDate(d) {
  if (!d) return ''
  return new Date(d).toLocaleString('zh-CN')
}
</script>
