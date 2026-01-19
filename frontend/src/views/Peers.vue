<template>
  <div class="peers-page">
    <div class="page-header">
      <h2 class="page-title">{{ t('peers.title') }}</h2>
      <div class="header-actions">
        <button class="btn btn-secondary" @click="refreshPeers" :disabled="loading">
          {{ t('peers.refresh') }}
        </button>
        <button class="btn btn-primary" @click="showAddDialog = true">
          {{ t('peers.add') }}
        </button>
      </div>
    </div>

    <!-- Stats Bar -->
    <div class="stats-bar">
      <div class="stat">
        <span class="stat-value">{{ peerCount }}</span>
        <span class="stat-label">{{ t('dashboard.peers') }}</span>
      </div>
      <div class="stat">
        <span class="stat-value connected">{{ connectedCount }}</span>
        <span class="stat-label">{{ t('peers.status.up') }}</span>
      </div>
      <div class="stat">
        <span class="stat-value disconnected">{{ disconnectedCount }}</span>
        <span class="stat-label">{{ t('peers.status.down') }}</span>
      </div>
    </div>

    <!-- Peers List -->
    <div class="card">
      <div v-if="loading" class="loading-state">
        {{ t('common.loading') }}
      </div>
      <div v-else-if="peers.length === 0" class="empty-state">
        <p>{{ t('peers.empty') }}</p>
        <button class="btn btn-primary" @click="showAddDialog = true">
          {{ t('peers.add') }}
        </button>
      </div>
      <div v-else class="peers-list">
        <div
          v-for="peer in sortedPeers"
          :key="peer.uri"
          class="peer-card"
          :class="{ 'peer-connected': peer.connected }"
        >
          <div class="peer-header">
            <span class="peer-status-indicator" :class="{ connected: peer.connected }"></span>
            <span class="peer-uri">{{ peer.uri }}</span>
            <span class="peer-direction" v-if="peer.inbound !== undefined">
              {{ peer.inbound ? t('peers.inbound') : t('peers.outbound') }}
            </span>
          </div>

          <div class="peer-details" v-if="peer.connected">
            <div class="detail-row">
              <span class="detail-label">{{ t('node.info.address') }}:</span>
              <span class="detail-value mono">{{ peer.address || '-' }}</span>
            </div>
            <div class="detail-row" v-if="peer.latency">
              <span class="detail-label">Latency:</span>
              <span class="detail-value">{{ peer.latency.toFixed(1) }} ms</span>
            </div>
            <div class="detail-row" v-if="peer.uptime">
              <span class="detail-label">{{ t('dashboard.uptime') }}:</span>
              <span class="detail-value">{{ formatUptime(peer.uptime) }}</span>
            </div>
          </div>

          <div class="peer-traffic" v-if="peer.rxBytes || peer.txBytes">
            <div class="traffic-item">
              <span class="traffic-arrow">↓</span>
              <span class="traffic-value">{{ formatBytes(peer.rxBytes) }}</span>
            </div>
            <div class="traffic-item">
              <span class="traffic-arrow">↑</span>
              <span class="traffic-value">{{ formatBytes(peer.txBytes) }}</span>
            </div>
          </div>

          <div class="peer-actions">
            <button
              class="btn-icon btn-danger-icon"
              @click="confirmRemovePeer(peer)"
              :title="t('peers.remove')"
            >
              ×
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Add Peer Dialog -->
    <Modal v-if="showAddDialog" @close="showAddDialog = false">
      <template #header>{{ t('peers.add') }}</template>
      <template #default>
        <AddPeerDialog @close="handlePeerAdded" />
      </template>
    </Modal>

    <!-- Confirm Remove Dialog -->
    <Modal v-if="showRemoveDialog" @close="showRemoveDialog = false">
      <template #header>{{ t('peers.remove') }}</template>
      <template #default>
        <div class="confirm-dialog">
          <p>{{ t('peers.confirmRemove') }}</p>
          <p class="peer-to-remove">{{ peerToRemove?.uri }}</p>
          <div class="dialog-actions">
            <button class="btn btn-secondary" @click="showRemoveDialog = false">
              {{ t('common.cancel') }}
            </button>
            <button class="btn btn-danger" @click="removePeer">
              {{ t('common.delete') }}
            </button>
          </div>
        </div>
      </template>
    </Modal>

  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { usePeersStore } from '../store/peers'
import { useUiStore } from '../store/ui'
import Modal from '../components/common/Modal.vue'
import AddPeerDialog from '../components/peers/AddPeerDialog.vue'

const { t } = useI18n()
const peersStore = usePeersStore()
const uiStore = useUiStore()

// State
const showAddDialog = ref(false)
const showRemoveDialog = ref(false)
const peerToRemove = ref(null)

// Computed
const peers = computed(() => peersStore.peers)
const loading = computed(() => peersStore.loading)
const peerCount = computed(() => peersStore.peerCount)
const connectedCount = computed(() => peersStore.connectedCount)
const disconnectedCount = computed(() => peerCount.value - connectedCount.value)

const sortedPeers = computed(() => {
  return [...peers.value].sort((a, b) => {
    // Connected first
    if (a.connected !== b.connected) return b.connected ? 1 : -1
    // Then by URI
    return a.uri.localeCompare(b.uri)
  })
})

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

  if (days > 0) return `${days}d ${hours}h`
  if (hours > 0) return `${hours}h ${mins}m`
  return `${mins}m`
}

const refreshPeers = async () => {
  await peersStore.fetchPeers()
}

const handlePeerAdded = () => {
  showAddDialog.value = false
  uiStore.addNotification('success', t('peers.addSuccess'))
}

const confirmRemovePeer = (peer) => {
  peerToRemove.value = peer
  showRemoveDialog.value = true
}

const removePeer = async () => {
  if (!peerToRemove.value) return

  try {
    await peersStore.removePeer(peerToRemove.value.uri)
    uiStore.addNotification('success', t('peers.removeSuccess'))
  } catch (err) {
    uiStore.addNotification('error', t('peers.removeFailed'))
  } finally {
    showRemoveDialog.value = false
    peerToRemove.value = null
  }
}

onMounted(() => {
  peersStore.fetchPeers()
})
</script>

<style scoped>
.peers-page {
  max-width: 1200px;
}

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 24px;
}

.page-title {
  margin: 0;
  font-size: 24px;
  font-weight: 600;
  color: var(--color-text-primary);
}

.header-actions {
  display: flex;
  gap: 12px;
}

/* Stats Bar */
.stats-bar {
  display: flex;
  gap: 24px;
  margin-bottom: 24px;
  padding: 16px 24px;
  background-color: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
  border-radius: 8px;
}

.stat {
  display: flex;
  align-items: center;
  gap: 8px;
}

.stat-value {
  font-size: 20px;
  font-weight: 700;
  color: var(--color-text-primary);
}

.stat-value.connected {
  color: var(--color-success);
}

.stat-value.disconnected {
  color: var(--color-danger);
}

.stat-label {
  font-size: 14px;
  color: var(--color-text-secondary);
}

/* Card */
.card {
  background-color: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
  border-radius: 12px;
  padding: 20px;
}

.loading-state,
.empty-state {
  text-align: center;
  padding: 40px;
  color: var(--color-text-secondary);
}

.empty-state .btn {
  margin-top: 16px;
}

/* Peers List */
.peers-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.peer-card {
  display: grid;
  grid-template-columns: 1fr auto auto;
  align-items: center;
  gap: 16px;
  padding: 16px;
  background-color: var(--color-bg-primary);
  border: 1px solid var(--color-border);
  border-radius: 8px;
  transition: border-color 0.2s ease;
}

.peer-card.peer-connected {
  border-left: 3px solid var(--color-success);
}

.peer-card:not(.peer-connected) {
  border-left: 3px solid var(--color-danger);
  opacity: 0.7;
}

.peer-header {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}

.peer-status-indicator {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background-color: var(--color-danger);
  flex-shrink: 0;
}

.peer-status-indicator.connected {
  background-color: var(--color-success);
}

.peer-uri {
  font-family: monospace;
  font-size: 14px;
  color: var(--color-text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.peer-direction {
  font-size: 11px;
  padding: 2px 8px;
  background-color: var(--color-bg-secondary);
  border-radius: 10px;
  color: var(--color-text-secondary);
  flex-shrink: 0;
}

.peer-details {
  display: flex;
  flex-direction: column;
  gap: 4px;
  grid-column: 1 / -1;
  padding-top: 12px;
  border-top: 1px solid var(--color-border);
}

.detail-row {
  display: flex;
  gap: 8px;
  font-size: 13px;
}

.detail-label {
  color: var(--color-text-secondary);
}

.detail-value {
  color: var(--color-text-primary);
}

.detail-value.mono {
  font-family: monospace;
}

.peer-traffic {
  display: flex;
  gap: 16px;
}

.traffic-item {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  color: var(--color-text-secondary);
}

.traffic-arrow {
  font-weight: bold;
}

.peer-actions {
  display: flex;
  gap: 8px;
}

/* Buttons */
.btn {
  padding: 10px 20px;
  border: none;
  border-radius: 6px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
}

.btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.btn-primary {
  background-color: var(--color-accent);
  color: #ffffff;
}

.btn-primary:hover:not(:disabled) {
  opacity: 0.9;
}

.btn-secondary {
  background-color: var(--color-bg-primary);
  color: var(--color-text-primary);
  border: 1px solid var(--color-border);
}

.btn-secondary:hover:not(:disabled) {
  background-color: var(--color-border);
}

.btn-danger {
  background-color: var(--color-danger);
  color: #ffffff;
}

.btn-danger:hover:not(:disabled) {
  opacity: 0.9;
}

.btn-icon {
  width: 32px;
  height: 32px;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  font-size: 20px;
  line-height: 1;
  transition: all 0.2s ease;
}

.btn-danger-icon {
  background-color: transparent;
  color: var(--color-danger);
}

.btn-danger-icon:hover {
  background-color: rgba(220, 53, 69, 0.1);
}

/* Confirm Dialog */
.confirm-dialog {
  text-align: center;
}

.confirm-dialog p {
  margin-bottom: 16px;
  color: var(--color-text-primary);
}

.peer-to-remove {
  font-family: monospace;
  padding: 12px;
  background-color: var(--color-bg-primary);
  border-radius: 6px;
  word-break: break-all;
}

.dialog-actions {
  display: flex;
  justify-content: center;
  gap: 12px;
  margin-top: 24px;
}
</style>
