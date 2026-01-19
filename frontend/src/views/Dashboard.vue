<template>
  <div class="dashboard">
    <h2 class="page-title">{{ t('nav.dashboard') }}</h2>

    <div class="dashboard-grid">
      <!-- Node Status Card -->
      <div class="card card-status">
        <div class="card-header">
          <h3 class="card-title">{{ t('node.status.title') }}</h3>
          <NodeStatus />
        </div>
        <div class="card-body">
          <NodeInfo />
        </div>
        <div class="card-footer">
          <NodeControls />
        </div>
      </div>

      <!-- Quick Stats Card -->
      <div class="card card-stats">
        <h3 class="card-title">{{ t('dashboard.stats') }}</h3>
        <div class="stats-grid">
          <div class="stat-item">
            <span class="stat-value">{{ stats.peerCount }}</span>
            <span class="stat-label">{{ t('dashboard.peers') }}</span>
          </div>
          <div class="stat-item">
            <span class="stat-value">{{ stats.sessionCount }}</span>
            <span class="stat-label">{{ t('dashboard.sessions') }}</span>
          </div>
          <div class="stat-item">
            <span class="stat-value">{{ formatBytes(stats.rxBytes) }}</span>
            <span class="stat-label">{{ t('dashboard.received') }}</span>
          </div>
          <div class="stat-item">
            <span class="stat-value">{{ formatBytes(stats.txBytes) }}</span>
            <span class="stat-label">{{ t('dashboard.sent') }}</span>
          </div>
        </div>
        <div class="uptime" v-if="isRunning">
          <span class="uptime-label">{{ t('dashboard.uptime') }}:</span>
          <span class="uptime-value">{{ formatUptime(stats.uptime) }}</span>
        </div>
      </div>

      <!-- Connected Peers Preview -->
      <div class="card card-peers">
        <div class="card-header-row">
          <h3 class="card-title">{{ t('dashboard.connectedPeers') }}</h3>
          <router-link to="/peers" class="view-all">
            {{ t('dashboard.viewAll') }}
          </router-link>
        </div>
        <div class="peers-preview" v-if="connectedPeers.length > 0">
          <div
            v-for="peer in connectedPeers.slice(0, 5)"
            :key="peer.uri"
            class="peer-preview-item"
          >
            <span class="peer-status-dot"></span>
            <span class="peer-uri">{{ truncateUri(peer.uri) }}</span>
            <span class="peer-latency" v-if="peer.latency">
              {{ peer.latency.toFixed(0) }}ms
            </span>
          </div>
        </div>
        <div v-else class="empty-peers">
          <p>{{ t('dashboard.noPeers') }}</p>
        </div>
      </div>
    </div>

    <!-- Error Alert -->
    <div v-if="error" class="error-alert">
      <span class="error-icon">⚠</span>
      <span class="error-message">{{ error }}</span>
      <button class="error-dismiss" @click="clearError">×</button>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useNodeStore } from '../store/node'
import { usePeersStore } from '../store/peers'
import NodeStatus from '../components/node/NodeStatus.vue'
import NodeInfo from '../components/node/NodeInfo.vue'
import NodeControls from '../components/node/NodeControls.vue'

const { t } = useI18n()
const nodeStore = useNodeStore()
const peersStore = usePeersStore()

// Computed
const isRunning = computed(() => nodeStore.isRunning)
const error = computed(() => nodeStore.error)
const connectedPeers = computed(() => peersStore.connectedPeers)

const stats = computed(() => ({
  peerCount: peersStore.connectedCount,
  sessionCount: nodeStore.stats.sessionCount,
  rxBytes: peersStore.totalTraffic.rx,
  txBytes: peersStore.totalTraffic.tx,
  uptime: nodeStore.stats.uptime
}))

// Methods
const formatBytes = (bytes) => {
  if (!bytes || bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${units[i]}`
}

const formatUptime = (seconds) => {
  if (!seconds || seconds === 0) return '-'
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  const mins = Math.floor((seconds % 3600) / 60)

  if (days > 0) return `${days}d ${hours}h ${mins}m`
  if (hours > 0) return `${hours}h ${mins}m`
  return `${mins}m`
}

const truncateUri = (uri) => {
  if (!uri) return '-'
  if (uri.length <= 40) return uri
  return uri.substring(0, 37) + '...'
}

const clearError = () => nodeStore.clearError()

// Polling for stats update
let statsInterval = null

onMounted(() => {
  // Poll for updates every 5 seconds when running
  statsInterval = setInterval(() => {
    if (nodeStore.isRunning) {
      nodeStore.fetchStatus()
      peersStore.fetchPeers()
    }
  }, 5000)
})

onUnmounted(() => {
  if (statsInterval) {
    clearInterval(statsInterval)
  }
})
</script>

<style scoped>
.dashboard {
  max-width: 1400px;
}

.page-title {
  margin: 0 0 24px 0;
  font-size: 24px;
  font-weight: 600;
  color: var(--color-text-primary);
}

.dashboard-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 24px;
}

@media (max-width: 900px) {
  .dashboard-grid {
    grid-template-columns: 1fr;
  }
}

.card {
  background-color: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
  border-radius: 12px;
  padding: 20px;
}

.card-title {
  margin: 0 0 16px 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--color-text-primary);
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}

.card-header .card-title {
  margin: 0;
}

.card-body {
  margin-bottom: 16px;
}

.card-footer {
  padding-top: 16px;
  border-top: 1px solid var(--color-border);
}

/* Stats Card */
.stats-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;
}

.stat-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 16px;
  background-color: var(--color-bg-primary);
  border-radius: 8px;
}

.stat-value {
  font-size: 24px;
  font-weight: 700;
  color: var(--color-accent);
}

.stat-label {
  font-size: 12px;
  color: var(--color-text-secondary);
  margin-top: 4px;
}

.uptime {
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid var(--color-border);
  text-align: center;
}

.uptime-label {
  color: var(--color-text-secondary);
  font-size: 14px;
}

.uptime-value {
  color: var(--color-text-primary);
  font-weight: 600;
  margin-left: 8px;
}

/* Peers Preview Card */
.card-header-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}

.card-header-row .card-title {
  margin: 0;
}

.view-all {
  font-size: 13px;
  color: var(--color-accent);
  text-decoration: none;
}

.view-all:hover {
  text-decoration: underline;
}

.peers-preview {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.peer-preview-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  background-color: var(--color-bg-primary);
  border-radius: 6px;
  font-size: 13px;
}

.peer-status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background-color: var(--color-success);
}

.peer-uri {
  flex: 1;
  font-family: monospace;
  color: var(--color-text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.peer-latency {
  color: var(--color-text-secondary);
  font-size: 12px;
}

.empty-peers {
  text-align: center;
  padding: 24px;
  color: var(--color-text-secondary);
}

/* Error Alert */
.error-alert {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-top: 24px;
  padding: 16px;
  background-color: rgba(220, 53, 69, 0.1);
  border: 1px solid var(--color-danger);
  border-radius: 8px;
  color: var(--color-danger);
}

.error-icon {
  font-size: 20px;
}

.error-message {
  flex: 1;
}

.error-dismiss {
  background: none;
  border: none;
  color: var(--color-danger);
  font-size: 20px;
  cursor: pointer;
  padding: 0 8px;
}

.error-dismiss:hover {
  opacity: 0.7;
}
</style>
