<template>
  <div class="logs-page">
    <div class="page-header">
      <div>
        <h2 class="page-title">{{ t('logs.title') }}</h2>
        <p class="page-description">{{ t('logs.description') }}</p>
      </div>
      <div class="header-actions">
        <button class="btn btn-secondary" @click="exportLogs">
          {{ t('logs.export') }}
        </button>
        <button class="btn btn-secondary" @click="clearLogs">
          {{ t('logs.clear') }}
        </button>
      </div>
    </div>

    <!-- Toolbar -->
    <div class="toolbar">
      <div class="toolbar-left">
        <input
          type="text"
          v-model="filterText"
          :placeholder="t('logs.filter')"
          class="filter-input"
        >
        <select v-model="filterLevel" class="level-select">
          <option value="">{{ t('logs.allLevels') }}</option>
          <option value="debug">{{ t('logs.levels.debug') }}</option>
          <option value="info">{{ t('logs.levels.info') }}</option>
          <option value="warn">{{ t('logs.levels.warn') }}</option>
          <option value="error">{{ t('logs.levels.error') }}</option>
        </select>
      </div>
      <div class="toolbar-right">
        <label class="checkbox-label">
          <input type="checkbox" v-model="autoScroll">
          {{ t('logs.autoScroll') }}
        </label>
        <button class="btn-icon" @click="togglePause">
          {{ paused ? '▶' : '⏸' }}
        </button>
      </div>
    </div>

    <!-- Log Entries -->
    <div class="logs-container" ref="logsContainer">
      <div v-if="filteredLogs.length === 0" class="empty-state">
        <span class="empty-icon">☰</span>
        <p>{{ t('logs.empty') }}</p>
      </div>

      <div v-else class="logs-list">
        <div
          v-for="(log, index) in filteredLogs"
          :key="index"
          class="log-entry"
          :class="log.level"
        >
          <span class="log-time">{{ formatTime(log.timestamp) }}</span>
          <span class="log-level" :class="log.level">{{ log.level.toUpperCase() }}</span>
          <span class="log-source" v-if="log.source">{{ log.source }}</span>
          <span class="log-message">{{ log.message }}</span>
        </div>
      </div>
    </div>

    <!-- Status Bar -->
    <div class="logs-status">
      <span>{{ filteredLogs.length }} {{ t('logs.title').toLowerCase() }}</span>
      <span v-if="paused" class="paused-indicator">{{ t('logs.paused') }}</span>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { ipc, Events } from '../utils/ipc'

const { t } = useI18n()

const logs = ref([])
const filterText = ref('')
const filterLevel = ref('')
const autoScroll = ref(true)
const paused = ref(false)
const logsContainer = ref(null)
const lastTimestamp = ref(0)
let pollInterval = null

const filteredLogs = computed(() => {
  return logs.value.filter(log => {
    // Filter by level
    if (filterLevel.value && log.level !== filterLevel.value) {
      return false
    }
    // Filter by text
    if (filterText.value) {
      const search = filterText.value.toLowerCase()
      return (
        log.message.toLowerCase().includes(search) ||
        (log.source && log.source.toLowerCase().includes(search))
      )
    }
    return true
  })
})

onMounted(() => {
  // Subscribe to push events (may not work in all cases)
  ipc.on(Events.LOG_ENTRY, handleLogEntry)

  // Initial fetch of logs
  fetchLogs()

  // Start polling for new logs every 2 seconds
  pollInterval = setInterval(() => {
    if (!paused.value) {
      fetchLogs()
    }
  }, 2000)
})

onUnmounted(() => {
  ipc.off(Events.LOG_ENTRY, handleLogEntry)

  // Stop polling
  if (pollInterval) {
    clearInterval(pollInterval)
    pollInterval = null
  }
})

watch(filteredLogs, () => {
  if (autoScroll.value && !paused.value) {
    nextTick(() => {
      scrollToBottom()
    })
  }
})

// Fetch logs from backend via polling
const fetchLogs = async () => {
  try {
    const response = await ipc.emit(Events.LOG_LIST, {
      since: lastTimestamp.value,
      limit: 500
    })

    if (response.success && response.data?.logs) {
      const newLogs = response.data.logs
      if (newLogs.length > 0) {
        // Add new logs that we don't already have
        for (const log of newLogs) {
          if (log.timestamp > lastTimestamp.value) {
            logs.value.push({
              timestamp: log.timestamp || Date.now(),
              level: log.level || 'info',
              source: log.source || '',
              message: log.message || ''
            })
            lastTimestamp.value = log.timestamp
          }
        }

        // Keep only last 1000 logs
        if (logs.value.length > 1000) {
          logs.value = logs.value.slice(-1000)
        }
      }
    }
  } catch (err) {
    console.error('[Logs.vue] Failed to fetch logs:', err)
  }
}

// Handle push events (backup method)
const handleLogEntry = (data) => {
  if (paused.value) return

  // Check if we already have this log (by timestamp)
  const exists = logs.value.some(l => l.timestamp === data.timestamp && l.message === data.message)
  if (exists) return

  logs.value.push({
    timestamp: data.timestamp || Date.now(),
    level: data.level || 'info',
    source: data.source || '',
    message: data.message || ''
  })

  // Update last timestamp
  if (data.timestamp > lastTimestamp.value) {
    lastTimestamp.value = data.timestamp
  }

  // Keep only last 1000 logs
  if (logs.value.length > 1000) {
    logs.value = logs.value.slice(-1000)
  }
}

const formatTime = (timestamp) => {
  const date = new Date(timestamp)
  return date.toLocaleTimeString('en-US', {
    hour12: false,
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    fractionalSecondDigits: 3
  })
}

const scrollToBottom = () => {
  if (logsContainer.value) {
    logsContainer.value.scrollTop = logsContainer.value.scrollHeight
  }
}

const togglePause = () => {
  paused.value = !paused.value
}

const clearLogs = async () => {
  logs.value = []
  lastTimestamp.value = 0

  // Clear logs on backend too
  try {
    await ipc.emit(Events.LOG_CLEAR)
  } catch (err) {
    console.error('[Logs.vue] Failed to clear logs on backend:', err)
  }
}

const exportLogs = () => {
  const content = logs.value.map(log => {
    const time = new Date(log.timestamp).toISOString()
    return `[${time}] [${log.level.toUpperCase()}] ${log.source ? `[${log.source}] ` : ''}${log.message}`
  }).join('\n')

  const blob = new Blob([content], { type: 'text/plain' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `yggstack-gui-logs-${new Date().toISOString().split('T')[0]}.txt`
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}
</script>

<style scoped>
.logs-page {
  display: flex;
  flex-direction: column;
  height: calc(100vh - 140px);
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 16px;
}

.page-title {
  margin: 0 0 8px 0;
  font-size: 24px;
  font-weight: 600;
  color: var(--color-text-primary);
}

.page-description {
  margin: 0;
  font-size: 14px;
  color: var(--color-text-secondary);
}

.header-actions {
  display: flex;
  gap: 8px;
}

/* Toolbar */
.toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px;
  background-color: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
  border-radius: 8px 8px 0 0;
  gap: 16px;
}

.toolbar-left {
  display: flex;
  gap: 12px;
  flex: 1;
}

.toolbar-right {
  display: flex;
  align-items: center;
  gap: 16px;
}

.filter-input {
  flex: 1;
  max-width: 300px;
  padding: 8px 12px;
  font-size: 13px;
  color: var(--color-text-primary);
  background-color: var(--color-bg-primary);
  border: 1px solid var(--color-border);
  border-radius: 6px;
}

.filter-input:focus {
  outline: none;
  border-color: var(--color-accent);
}

.level-select {
  padding: 8px 12px;
  font-size: 13px;
  color: var(--color-text-primary);
  background-color: var(--color-bg-primary);
  border: 1px solid var(--color-border);
  border-radius: 6px;
  cursor: pointer;
}

.checkbox-label {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: var(--color-text-secondary);
  cursor: pointer;
}

.checkbox-label input {
  width: 16px;
  height: 16px;
}

.btn-icon {
  padding: 8px 12px;
  background-color: var(--color-bg-primary);
  border: 1px solid var(--color-border);
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
}

.btn-icon:hover {
  background-color: var(--color-bg-secondary);
}

/* Logs Container */
.logs-container {
  flex: 1;
  overflow-y: auto;
  background-color: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
  border-top: none;
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  font-size: 12px;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 200px;
  color: var(--color-text-secondary);
}

.empty-icon {
  font-size: 48px;
  margin-bottom: 16px;
  opacity: 0.3;
}

.logs-list {
  padding: 8px;
}

.log-entry {
  display: flex;
  gap: 12px;
  padding: 4px 8px;
  border-radius: 4px;
  line-height: 1.5;
}

.log-entry:hover {
  background-color: var(--color-bg-primary);
}

.log-time {
  color: var(--color-text-secondary);
  flex-shrink: 0;
}

.log-level {
  width: 50px;
  flex-shrink: 0;
  font-weight: 600;
  text-transform: uppercase;
}

.log-level.debug {
  color: var(--color-text-secondary);
}

.log-level.info {
  color: var(--color-accent);
}

.log-level.warn {
  color: #f39c12;
}

.log-level.error {
  color: var(--color-danger);
}

.log-source {
  color: var(--color-accent);
  flex-shrink: 0;
}

.log-source::before {
  content: '[';
}

.log-source::after {
  content: ']';
}

.log-message {
  color: var(--color-text-primary);
  word-break: break-word;
}

/* Status Bar */
.logs-status {
  display: flex;
  justify-content: space-between;
  padding: 8px 12px;
  background-color: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
  border-top: none;
  border-radius: 0 0 8px 8px;
  font-size: 12px;
  color: var(--color-text-secondary);
}

.paused-indicator {
  color: #f39c12;
  font-weight: 600;
}

/* Buttons */
.btn {
  padding: 8px 16px;
  font-size: 13px;
  font-weight: 500;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.btn-secondary {
  background-color: var(--color-bg-secondary);
  color: var(--color-text-primary);
  border: 1px solid var(--color-border);
}

.btn-secondary:hover {
  background-color: var(--color-bg-primary);
}

@media (max-width: 600px) {
  .page-header {
    flex-direction: column;
    gap: 16px;
  }

  .toolbar {
    flex-direction: column;
    align-items: stretch;
  }

  .toolbar-left, .toolbar-right {
    width: 100%;
  }

  .filter-input {
    max-width: none;
  }
}
</style>
