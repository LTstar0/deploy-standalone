<template>
  <div
    class="upload-zone"
    :class="{ 'drag-over': dragOver }"
    @click="$refs.fileInput.click()"
    @dragover.prevent="dragOver = true"
    @dragleave="dragOver = false"
    @drop.prevent="onDrop"
  >
    <input ref="fileInput" type="file" accept=".tar.gz,.tgz,.zip" hidden @change="onSelect" />
    <div v-if="!uploading">
      <div class="upload-icon">📁</div>
      <div class="upload-text">拖拽产品包到此处，或点击选择文件</div>
      <div class="upload-hint">支持 .tar.gz / .tgz / .zip 格式</div>
    </div>
    <div v-else>
      <div class="upload-icon">⏳</div>
      <div class="upload-text">上传中... {{ progress }}%</div>
      <div class="upload-progress">
        <div class="progress-bar">
          <div class="progress-fill" :style="{ width: progress + '%' }"></div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'

const emit = defineEmits(['upload'])
const dragOver = ref(false)
const uploading = ref(false)
const progress = ref(0)

function onSelect(e) {
  const file = e.target.files[0]
  if (file) doUpload(file)
}

function onDrop(e) {
  dragOver.value = false
  const file = e.dataTransfer.files[0]
  if (file) doUpload(file)
}

function doUpload(file) {
  uploading.value = true
  progress.value = 0
  emit('upload', file, (p) => { progress.value = p }, () => {
    uploading.value = false
    progress.value = 0
  })
}
</script>
