<template>
  <div class="forwarding-page">
    <div class="page-header">
      <div>
        <h2 class="page-title">{{ t('forwarding.title') }}</h2>
        <p class="page-description">{{ t('forwarding.description') }}</p>
      </div>
      <button class="btn btn-primary" @click="showAddDialog = true">
        {{ t('forwarding.addMapping') }}
      </button>
    </div>

    <!-- Tabs -->
    <div class="tabs">
      <button
        v-for="tab in tabs"
        :key="tab.id"
        class="tab"
        :class="{ active: activeTab === tab.id }"
        @click="activeTab = tab.id"
      >
        {{ t(tab.labelKey) }}
        <span class="tab-count">{{ getMappingsCount(tab.id) }}</span>
      </button>
    </div>

    <!-- Mappings List -->
    <div class="mappings-container">
      <div v-if="currentMappings.length === 0" class="empty-state">
        <span class="empty-icon">↹</span>
        <p>{{ t('forwarding.noMappings') }}</p>
      </div>

      <div v-else class="mappings-list">
        <div
          v-for="mapping in currentMappings"
          :key="mapping.id"
          class="mapping-card"
        >
          <div class="mapping-info">
            <div class="mapping-direction">
              <span class="mapping-source">{{ mapping.source }}</span>
              <span class="mapping-arrow">→</span>
              <span class="mapping-target">{{ mapping.target }}</span>
            </div>
            <div class="mapping-meta">
              <span class="mapping-type">{{ getTypeLabel(activeTab) }}</span>
              <span class="mapping-status" :class="mapping.enabled ? 'enabled' : 'disabled'">
                {{ mapping.enabled ? t('forwarding.enabled') : t('forwarding.disabled') }}
              </span>
            </div>
          </div>
          <div class="mapping-actions">
            <label class="toggle">
              <input type="checkbox" v-model="mapping.enabled" @change="toggleMapping(mapping)">
              <span class="toggle-slider"></span>
            </label>
            <button class="btn-icon" @click="removeMapping(mapping)">✕</button>
          </div>
        </div>
      </div>
    </div>

    <!-- Add Mapping Dialog -->
    <Modal v-if="showAddDialog" @close="closeAddDialog">
      <template #title>{{ t('forwarding.addMapping') }}</template>
      <template #content>
        <div class="dialog-form">
          <div class="form-group">
            <label>{{ t('forwarding.protocol') }}</label>
            <select v-model="newMapping.protocol" class="select">
              <option value="tcp">TCP</option>
              <option value="udp">UDP</option>
            </select>
          </div>
          <div class="form-group">
            <label>{{ t('forwarding.direction') }}</label>
            <select v-model="newMapping.direction" class="select">
              <option value="local">{{ t('forwarding.local') }} (→ Yggdrasil)</option>
              <option value="remote">{{ t('forwarding.remote') }} (← Yggdrasil)</option>
            </select>
          </div>
          <div class="form-group">
            <label>{{ t('forwarding.source') }}</label>
            <input
              type="text"
              v-model="newMapping.source"
              :placeholder="t('forwarding.sourcePlaceholder')"
              class="input"
            >
          </div>
          <div class="form-group">
            <label>{{ t('forwarding.target') }}</label>
            <input
              type="text"
              v-model="newMapping.target"
              :placeholder="t('forwarding.targetPlaceholder')"
              class="input"
            >
          </div>
        </div>
      </template>
      <template #actions>
        <button class="btn btn-secondary" @click="closeAddDialog">
          {{ t('common.cancel') }}
        </button>
        <button class="btn btn-primary" @click="addMapping" :disabled="!isValidMapping">
          {{ t('common.save') }}
        </button>
      </template>
    </Modal>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useUiStore } from '../store/ui'
import { ipc, Events } from '../utils/ipc'
import Modal from '../components/common/Modal.vue'

const { t } = useI18n()
const uiStore = useUiStore()

const activeTab = ref('localTcp')
const showAddDialog = ref(false)
const mappings = ref({
  localTcp: [],
  remoteTcp: [],
  localUdp: [],
  remoteUdp: []
})

const tabs = [
  { id: 'localTcp', labelKey: 'forwarding.localTcp' },
  { id: 'remoteTcp', labelKey: 'forwarding.remoteTcp' },
  { id: 'localUdp', labelKey: 'forwarding.localUdp' },
  { id: 'remoteUdp', labelKey: 'forwarding.remoteUdp' }
]

const newMapping = ref({
  protocol: 'tcp',
  direction: 'local',
  source: '',
  target: '',
  enabled: true
})

const currentMappings = computed(() => mappings.value[activeTab.value] || [])

const getMappingsCount = (tabId) => mappings.value[tabId]?.length || 0

const getTypeLabel = (tabId) => {
  const labels = {
    localTcp: 'Local TCP',
    remoteTcp: 'Remote TCP',
    localUdp: 'Local UDP',
    remoteUdp: 'Remote UDP'
  }
  return labels[tabId]
}

const isValidMapping = computed(() => {
  return newMapping.value.source.trim() !== '' && newMapping.value.target.trim() !== ''
})

onMounted(async () => {
  await loadMappings()
})

const loadMappings = async () => {
  try {
    const response = await ipc.emit(Events.SETTINGS_GET)
    if (response.success && response.data?.mappings) {
      // Mappings would be stored in settings
    }
  } catch (err) {
    console.error('Failed to load mappings:', err)
  }
}

const addMapping = async () => {
  if (!isValidMapping.value) return

  const mapping = {
    id: Date.now().toString(),
    source: newMapping.value.source,
    target: newMapping.value.target,
    enabled: newMapping.value.enabled
  }

  // Convert to backend format: "local-tcp", "remote-tcp", "local-udp", "remote-udp"
  const backendType = `${newMapping.value.direction}-${newMapping.value.protocol}`
  // Frontend tab key: "localTcp", "remoteTcp", etc.
  const frontendKey = `${newMapping.value.direction}${newMapping.value.protocol.charAt(0).toUpperCase() + newMapping.value.protocol.slice(1)}`

  try {
    const response = await ipc.emit(Events.MAPPING_ADD, {
      type: backendType,
      ...mapping
    })

    if (response.success) {
      if (!mappings.value[frontendKey]) {
        mappings.value[frontendKey] = []
      }
      mappings.value[frontendKey].push(mapping)
      uiStore.addNotification('success', t('forwarding.addSuccess'))
      closeAddDialog()
    } else {
      uiStore.addNotification('error', response.error?.message || t('forwarding.addFailed'))
    }
  } catch (err) {
    uiStore.addNotification('error', t('forwarding.addFailed'))
  }
}

const removeMapping = async (mapping) => {
  if (!confirm(t('forwarding.confirmRemove'))) return

  try {
    const response = await ipc.emit(Events.MAPPING_REMOVE, { id: mapping.id })

    if (response.success) {
      const index = mappings.value[activeTab.value].findIndex(m => m.id === mapping.id)
      if (index !== -1) {
        mappings.value[activeTab.value].splice(index, 1)
      }
      uiStore.addNotification('success', t('forwarding.removeSuccess'))
    } else {
      uiStore.addNotification('error', response.error?.message || t('forwarding.removeFailed'))
    }
  } catch (err) {
    uiStore.addNotification('error', t('forwarding.removeFailed'))
  }
}

// Convert frontend tab key to backend type format
const tabToBackendType = (tabKey) => {
  const mapping = {
    localTcp: 'local-tcp',
    remoteTcp: 'remote-tcp',
    localUdp: 'local-udp',
    remoteUdp: 'remote-udp'
  }
  return mapping[tabKey] || tabKey
}

const toggleMapping = async (mapping) => {
  try {
    await ipc.emit(Events.MAPPING_ADD, {
      type: tabToBackendType(activeTab.value),
      ...mapping
    })
  } catch (err) {
    mapping.enabled = !mapping.enabled
    uiStore.addNotification('error', 'Failed to toggle mapping')
  }
}

const closeAddDialog = () => {
  showAddDialog.value = false
  newMapping.value = {
    protocol: 'tcp',
    direction: 'local',
    source: '',
    target: '',
    enabled: true
  }
}
</script>

<style scoped>
.forwarding-page {
  max-width: 900px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 24px;
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

/* Tabs */
.tabs {
  display: flex;
  gap: 4px;
  margin-bottom: 20px;
  border-bottom: 1px solid var(--color-border);
  padding-bottom: 0;
}

.tab {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  font-size: 14px;
  font-weight: 500;
  color: var(--color-text-secondary);
  background: none;
  border: none;
  border-bottom: 2px solid transparent;
  cursor: pointer;
  transition: all 0.2s ease;
}

.tab:hover {
  color: var(--color-text-primary);
}

.tab.active {
  color: var(--color-accent);
  border-bottom-color: var(--color-accent);
}

.tab-count {
  padding: 2px 8px;
  font-size: 12px;
  background-color: var(--color-bg-secondary);
  border-radius: 10px;
}

/* Mappings List */
.mappings-container {
  min-height: 200px;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
  color: var(--color-text-secondary);
}

.empty-icon {
  font-size: 48px;
  margin-bottom: 16px;
  opacity: 0.3;
}

.mappings-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.mapping-card {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  background-color: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
  border-radius: 8px;
}

.mapping-info {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.mapping-direction {
  display: flex;
  align-items: center;
  gap: 12px;
  font-family: monospace;
  font-size: 14px;
}

.mapping-source, .mapping-target {
  color: var(--color-text-primary);
}

.mapping-arrow {
  color: var(--color-accent);
}

.mapping-meta {
  display: flex;
  gap: 12px;
  font-size: 12px;
}

.mapping-type {
  color: var(--color-text-secondary);
}

.mapping-status {
  padding: 2px 8px;
  border-radius: 4px;
}

.mapping-status.enabled {
  background-color: rgba(40, 167, 69, 0.1);
  color: var(--color-success);
}

.mapping-status.disabled {
  background-color: rgba(108, 117, 125, 0.1);
  color: var(--color-text-secondary);
}

.mapping-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.btn-icon {
  padding: 8px;
  background: none;
  border: none;
  color: var(--color-text-secondary);
  cursor: pointer;
  border-radius: 4px;
  transition: all 0.2s ease;
}

.btn-icon:hover {
  background-color: var(--color-bg-primary);
  color: var(--color-danger);
}

/* Toggle Switch */
.toggle {
  position: relative;
  display: inline-block;
  width: 40px;
  height: 22px;
}

.toggle input {
  opacity: 0;
  width: 0;
  height: 0;
}

.toggle-slider {
  position: absolute;
  cursor: pointer;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: var(--color-border);
  transition: 0.3s;
  border-radius: 22px;
}

.toggle-slider:before {
  position: absolute;
  content: "";
  height: 16px;
  width: 16px;
  left: 3px;
  bottom: 3px;
  background-color: white;
  transition: 0.3s;
  border-radius: 50%;
}

.toggle input:checked + .toggle-slider {
  background-color: var(--color-success);
}

.toggle input:checked + .toggle-slider:before {
  transform: translateX(18px);
}

/* Dialog Form */
.dialog-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 400px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.form-group label {
  font-size: 13px;
  font-weight: 500;
  color: var(--color-text-secondary);
}

.select, .input {
  padding: 10px 12px;
  font-size: 14px;
  color: var(--color-text-primary);
  background-color: var(--color-bg-primary);
  border: 1px solid var(--color-border);
  border-radius: 6px;
}

.select:focus, .input:focus {
  outline: none;
  border-color: var(--color-accent);
}

/* Buttons */
.btn {
  padding: 10px 20px;
  font-size: 14px;
  font-weight: 500;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.btn-primary {
  background-color: var(--color-accent);
  color: white;
}

.btn-primary:hover:not(:disabled) {
  filter: brightness(1.1);
}

.btn-secondary {
  background-color: var(--color-bg-secondary);
  color: var(--color-text-primary);
  border: 1px solid var(--color-border);
}

.btn-secondary:hover {
  background-color: var(--color-bg-primary);
}

.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

@media (max-width: 600px) {
  .page-header {
    flex-direction: column;
    gap: 16px;
  }

  .tabs {
    overflow-x: auto;
  }

  .dialog-form {
    min-width: auto;
  }
}
</style>
