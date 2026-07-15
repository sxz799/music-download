
<template>
  <div id="app">
    <h1>音乐下载器</h1>
    
    <div class="json-input-section">
      <h2>输入 JSON 数据</h2>
      <textarea 
        v-model="jsonInput" 
        placeholder="在此处粘贴或输入 JSON 数据，例如：[{&quot;name&quot;:&quot;歌曲名&quot;,&quot;url&quot;:&quot;下载链接&quot;,&quot;time&quot;:&quot;时间&quot;,&quot;downloaded&quot;:false}]"
      ></textarea>
      <button @click="parseJson">解析并添加</button>
      <button @click="loadExample" class="btn-secondary">加载示例数据</button>
    </div>

    <div v-if="musicList.length > 0" class="table-section">
      <div class="table-header">
        <h2>音乐列表 ({{ musicList.length }} 首)</h2>
        <div class="batch-actions">
          <label class="select-all">
            <input type="checkbox" v-model="selectAll" @change="toggleSelectAll" />
            全选
          </label>
          <button v-if="selectedCount > 0" @click="startBatchDownload" class="btn-batch-download" :disabled="isBatchDownloading">
            {{ batchDownloadBtnText }}
          </button>
          <button v-if="selectedCount > 0" @click="batchDelete" class="btn-batch-delete">
            批量删除 ({{ selectedCount }})
          </button>
          <button v-if="selectedCount > 0" @click="clearSelection" class="btn-clear-selection">
            清除选择
          </button>
        </div>
      </div>
      <table>
        <thead>
          <tr>
            <th>选择</th>
            <th>序号</th>
            <th>歌曲名称</th>
            <th>添加时间</th>
            <th>下载状态</th>
            <th>进度</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(music, index) in musicList" :key="music.id" :class="{ 'downloading': music.status === 'downloading' }">
            <td>
              <input 
                type="checkbox" 
                v-model="selectedIds" 
                :value="music.id"
                :disabled="music.status === 'downloading' || music.status === 'completed'"
              />
            </td>
            <td>{{ index + 1 }}</td>
            <td>
              <div class="song-name">
                {{ music.name }}
                <div v-if="music.fileName" class="file-name">保存为: {{ music.fileName }}</div>
              </div>
            </td>
            <td>{{ music.time || '-' }}</td>
            <td>
              <span :class="getStatusClass(music.status)">
                {{ getStatusText(music.status) }}
              </span>
            </td>
            <td>
              <div v-if="music.status === 'downloading'" class="progress-bar">
                <div class="progress-fill" :style="{ width: music.progress + '%' }"></div>
                <span class="progress-text">{{ music.progress }}%</span>
              </div>
              <span v-else-if="music.progress > 0">{{ music.progress }}%</span>
              <span v-else>-</span>
            </td>
            <td>
              <div class="actions">
                <template v-if="music.status === 'pending' || music.status === 'error' || music.status === 'cancelled'">
                  <input 
                    v-if="!music.fileName"
                    v-model="music.fileName" 
                    type="text" 
                    placeholder="文件名(可选)"
                    class="file-name-input"
                  />
                  <button @click="startDownload(music)" class="btn-download">下载</button>
                </template>
                <template v-else-if="music.status === 'downloading'">
                  <button @click="cancelDownload(music)" class="btn-cancel">取消</button>
                </template>
                <template v-else-if="music.status === 'completed'">
                  <span class="success-badge">✓ 已完成</span>
                </template>
                <button @click="removeMusic(music.id)" class="btn-delete">删除</button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-if="errorMessage" class="error">
      ❌ {{ errorMessage }}
    </div>

    <div v-if="successMessage" class="success">
      ✅ {{ successMessage }}
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'

const jsonInput = ref('')
const musicList = ref([])
const selectedIds = ref([])
const errorMessage = ref('')
const successMessage = ref('')
const isBatchDownloading = ref(false)
let eventSource = null

const exampleData = [
  {
    "name": "老街 - 李荣浩",
    "url": "https://other-er.kuwo.cn/9765575eced074d8b30c660334ca9bb1/6a55ddaa/resource/30106/trackmedia/F000004U927G0splK9.flac",
    "time": "2026/7/14 14:56:43",
    "downloaded": false
  },
  {
    "name": "模特 - 李荣浩",
    "url": "https://other-er.kuwo.cn/73ee5baabad446aec24c7dcb7641e222/6a55dda5/resource/1307392909/trackmedia/F000001nujtF0jebVT.flac",
    "time": "2026/7/14 14:56:37",
    "downloaded": false
  }
]

const selectAll = computed({
  get: () => {
    const pendingSongs = musicList.value.filter(m => 
      m.status === 'pending' || m.status === 'error' || m.status === 'cancelled')
    return pendingSongs.length > 0 && pendingSongs.every(m => 
      selectedIds.value.includes(getNumericId(m.id))
    )
  },
  set: (value) => {
    if (value) {
      selectedIds.value = musicList.value
        .filter(m => m.status === 'pending' || m.status === 'error' || m.status === 'cancelled')
        .map(m => getNumericId(m.id))
    } else {
      selectedIds.value = []
    }
  }
})

const selectedCount = computed(() => selectedIds.value.length)

const batchDownloadBtnText = computed(() => {
  if (isBatchDownloading.value) {
    return '批量下载中...'
  } else {
    return '批量下载 (' + selectedCount.value + ')'
  }
})

const loadExample = () => {
  jsonInput.value = JSON.stringify(exampleData, null, 2)
  clearMessages()
}

const clearMessages = () => {
  errorMessage.value = ''
  successMessage.value = ''
}

const toggleSelectAll = () => {
  // 计算属性会自动处理
}

const getNumericId = (id) => {
  if (typeof id === 'string') {
    return parseInt(id, 10)
  }
  return id
}

const clearSelection = () => {
  selectedIds.value = []
}

const parseJson = async () => {
  clearMessages()
  
  const input = jsonInput.value.trim()
  
  if (!input) {
    errorMessage.value = '请输入 JSON 数据'
    return
  }

  try {
    console.log('开始解析 JSON...')
    const parsed = JSON.parse(input)
    console.log('解析成功:', parsed)

    let songs = []
    if (Array.isArray(parsed)) {
      songs = parsed
    } else if (typeof parsed === 'object') {
      songs = [parsed]
    } else {
      throw new Error('JSON 数据格式不正确，需要是对象或数组')
    }

    if (songs.length === 0) {
      throw new Error('解析结果为空数组')
    }

    for (const song of songs) {
      await addMusicToBackend(song)
    }

    successMessage.value = `成功添加 ${songs.length} 首歌曲`
    jsonInput.value = ''
    
    setTimeout(() => {
      successMessage.value = ''
    }, 3000)

  } catch (e) {
    console.error('解析错误:', e)
    errorMessage.value = 'JSON 解析错误: ' + e.message
  }
}


const addMusicToBackend = async (music) => {
  try {
    const response = await fetch('/api/music', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(music)
    })
    
    if (!response.ok) {
      throw new Error('Failed to add music')
    }
    
    const result = await response.json()
    musicList.value.push(result)
  } catch (err) {
    console.error('Error adding music:', err)
  }
}

const startDownload = async (music) => {
  try {
    const response = await fetch('/api/download', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        id: music.id,
        fileName: music.fileName
      })
    })
    
    if (!response.ok) {
      throw new Error('Failed to start download')
    }
    
    successMessage.value = `开始下载: ${music.name}`
    setTimeout(() => {
      successMessage.value = ''
    }, 2000)
  } catch (err) {
    console.error('Error starting download:', err)
    errorMessage.value = '启动下载失败'
  }
}

const startBatchDownload = async () => {
  if (selectedIds.value.length === 0) {
    errorMessage.value = '请先选择要下载的歌曲'
    return
  }

  isBatchDownloading.value = true

  try {
    for (const id of selectedIds.value) {
      const music = musicList.value.find(m => m.id === id)
      if (music && (music.status === 'pending' || music.status === 'error' || music.status === 'cancelled')) {
        await startDownload(music)
        // 等待一小段时间，避免同时发起太多请求
        await new Promise(resolve => setTimeout(resolve, 200))
      }
    }
    successMessage.value = `已启动批量下载，共 ${selectedIds.value.length} 首歌曲`
    setTimeout(() => {
      successMessage.value = ''
    }, 3000)
  } catch (err) {
    console.error('Error in batch download:', err)
    errorMessage.value = '批量下载启动失败'
  } finally {
    isBatchDownloading.value = false
    selectedIds.value = []
  }
}

const cancelDownload = async (music) => {
  try {
    await fetch(`/api/music/${music.id}`, {
      method: 'DELETE'
    })
  } catch (err) {
    console.error('Error cancelling download:', err)
  }
}

const removeMusic = async (id) => {
  try {
    await fetch(`/api/music/${id}`, {
      method: 'DELETE'
    })
    musicList.value = musicList.value.filter(m => m.id !== id)
    selectedIds.value = selectedIds.value.filter(i => i !== id)
  } catch (err) {
    console.error('Error removing music:', err)
  }
}

const batchDelete = async () => {
  if (selectedIds.value.length === 0) {
    errorMessage.value = '请先选择要删除的歌曲'
    return
  }

  if (!confirm(`确定要删除选中的 ${selectedIds.value.length} 首歌曲吗？`)) {
    return
  }

  try {
    const deletePromises = selectedIds.value.map(id => 
      fetch(`/api/music/${id}`, { method: 'DELETE' })
    )
    await Promise.all(deletePromises)

    musicList.value = musicList.value.filter(m => !selectedIds.value.includes(m.id))
    selectedIds.value = []
    successMessage.value = '批量删除成功'
    setTimeout(() => {
      successMessage.value = ''
    }, 2000)
  } catch (err) {
    console.error('Error in batch delete:', err)
    errorMessage.value = '批量删除失败'
  }
}

const getStatusClass = (status) => {
  const classes = {
    'pending': 'status-pending',
    'downloading': 'status-downloading',
    'completed': 'status-completed',
    'error': 'status-error',
    'cancelled': 'status-cancelled'
  }
  return classes[status] || 'status-pending'
}

const getStatusText = (status) => {
  const texts = {
    'pending': '等待下载',
    'downloading': '下载中',
    'completed': '已完成',
    'error': '下载失败',
    'cancelled': '已取消'
  }
  return texts[status] || '等待下载'
}

const connectProgress = () => {
  if (typeof EventSource !== 'undefined') {
    eventSource = new EventSource('/api/progress')
    
    eventSource.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data)
        // 确保所有的 id 都是数字类型
        musicList.value = data.map(item => ({
          ...item,
          id: getNumericId(item.id)
        }))
      } catch (e) {
        console.error('Failed to parse progress data:', e)
        musicList.value = []
      }
    }
    
    eventSource.onerror = (err) => {
      console.error('EventSource failed:', err)
    }
  }
}

const loadInitialData = async () => {
  try {
    const response = await fetch('/api/music')
    if (response.ok) {
      const data = await response.json()
      // 确保所有的 id 都是数字类型
      musicList.value = data.map(item => ({
        ...item,
        id: getNumericId(item.id)
      }))
    }
  } catch (err) {
    console.error('Failed to load initial data:', err)
  }
}

onMounted(() => {
  loadInitialData()
  connectProgress()
})

onUnmounted(() => {
  if (eventSource) {
    eventSource.close()
  }
})
</script>

<style scoped>
#app {
  max-width: 1500px;
  margin: 0 auto;
  padding: 20px;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
}

h1 {
  text-align: center;
  color: #333;
  margin-bottom: 30px;
}

h2 {
  color: #555;
  margin: 0;
}

.json-input-section {
  margin-bottom: 30px;
}

.table-section {
  margin-top: 20px;
}

.table-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 15px;
  flex-wrap: wrap;
  gap: 15px;
}

.batch-actions {
  display: flex;
  gap: 10px;
  align-items: center;
  flex-wrap: wrap;
}

.select-all {
  display: flex;
  align-items: center;
  gap: 6px;
  cursor: pointer;
  color: #555;
  font-size: 14px;
}

.select-all input {
  width: 16px;
  height: 16px;
  cursor: pointer;
}

.btn-batch-download {
  background-color: #9C27B0;
  padding: 10px 20px;
  font-size: 14px;
}

.btn-batch-download:hover:not(:disabled) {
  background-color: #7B1FA2;
}

.btn-batch-download:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.btn-clear-selection {
  background-color: #795548;
  padding: 10px 20px;
  font-size: 14px;
}

.btn-clear-selection:hover {
  background-color: #5D4037;
}

.btn-batch-delete {
  background-color: #f44336;
  padding: 10px 20px;
  font-size: 14px;
}

.btn-batch-delete:hover {
  background-color: #d32f2f;
}

textarea {
  width: 100%;
  min-height: 200px;
  padding: 12px;
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 14px;
  line-height: 1.5;
  border: 2px solid #ddd;
  border-radius: 8px;
  resize: vertical;
  box-sizing: border-box;
}

textarea:focus {
  outline: none;
  border-color: #4CAF50;
}

button {
  margin-top: 10px;
  margin-right: 10px;
  padding: 12px 24px;
  background-color: #4CAF50;
  color: white;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  font-size: 16px;
  font-weight: 500;
  transition: background-color 0.2s;
}

button:hover {
  background-color: #45a049;
}

.btn-secondary {
  background-color: #2196F3;
}

.btn-secondary:hover {
  background-color: #0b7dda;
}

.btn-push {
  background-color: #9C27B0;
}

.btn-push:hover {
  background-color: #7B1FA2;
}

table {
  width: 100%;
  border-collapse: collapse;
  margin-top: 10px;
  border-radius: 8px;
  overflow: hidden;
  box-shadow: 0 2px 8px rgba(0,0,0,0.1);
}

th, td {
  border: 1px solid #e0e0e0;
  padding: 14px;
  text-align: left;
}

th {
  background-color: #f8f9fa;
  font-weight: 600;
  color: #333;
}

tr.downloading {
  background-color: #f0f9ff;
}

tr:hover {
  background-color: #f5f5f5;
}

tr.downloading:hover {
  background-color: #e6f3ff;
}

.song-name {
  font-weight: 500;
}

.file-name {
  font-size: 12px;
  color: #666;
  margin-top: 4px;
}

.file-name-input {
  padding: 6px 10px;
  border: 1px solid #ddd;
  border-radius: 4px;
  font-size: 14px;
  margin-right: 8px;
  width: 150px;
}

.status-pending {
  color: #FF9800;
  font-weight: 500;
}

.status-downloading {
  color: #2196F3;
  font-weight: 500;
}

.status-completed {
  color: #4CAF50;
  font-weight: 500;
}

.status-error {
  color: #f44336;
  font-weight: 500;
}

.status-cancelled {
  color: #9e9e9e;
  font-weight: 500;
}

.progress-bar {
  position: relative;
  width: 120px;
  height: 24px;
  background-color: #e0e0e0;
  border-radius: 12px;
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  background: linear-gradient(90deg, #4CAF50, #8BC34A);
  transition: width 0.3s ease;
}

.progress-text {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  font-size: 12px;
  font-weight: 600;
  color: #333;
  text-shadow: 0 1px 2px rgba(255,255,255,0.8);
}

.actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.btn-download {
  background-color: #4CAF50;
  padding: 8px 16px;
  font-size: 14px;
  margin: 0;
}

.btn-download:hover {
  background-color: #45a049;
}

.btn-cancel {
  background-color: #ff9800;
  padding: 8px 16px;
  font-size: 14px;
  margin: 0;
}

.btn-cancel:hover {
  background-color: #f57c00;
}

.btn-delete {
  background-color: #f44336;
  padding: 8px 16px;
  font-size: 14px;
  margin: 0;
}

.btn-delete:hover {
  background-color: #d32f2f;
}

.success-badge {
  color: #4CAF50;
  font-weight: 600;
}

.error {
  color: #d32f2f;
  margin-top: 15px;
  padding: 12px 16px;
  background-color: #ffebee;
  border-radius: 6px;
  border-left: 4px solid #d32f2f;
}

.success {
  color: #2e7d32;
  margin-top: 15px;
  padding: 12px 16px;
  background-color: #e8f5e9;
  border-radius: 6px;
  border-left: 4px solid #2e7d32;
}
</style>
