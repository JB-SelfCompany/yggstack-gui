<template>
  <div class="node-info">
    <div v-if="isRunning" class="info-list">
      <div class="info-item">
        <span class="info-label">{{ t('node.info.address') }}</span>
        <div class="info-value-row">
          <span class="info-value">{{ info.address || '-' }}</span>
          <button
            v-if="info.address"
            class="copy-btn"
            @click="copyAddress"
            :title="t('dashboard.copyAddress')"
          >
            <span v-if="!copied">📋</span>
            <span v-else>✓</span>
          </button>
        </div>
      </div>
      <div class="info-item">
        <span class="info-label">{{ t('node.info.subnet') }}</span>
        <span class="info-value">{{ info.subnet || '-' }}</span>
      </div>
      <div class="info-item">
        <span class="info-label">{{ t('node.info.publicKey') }}</span>
        <span class="info-value info-key">{{ truncateKey(info.publicKey) }}</span>
      </div>
    </div>
    <div v-else class="info-placeholder">
      <p>{{ t('node.info.notRunning') }}</p>
    </div>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useNodeStore } from '../../store/node'
import { useUiStore } from '../../store/ui'

const { t } = useI18n()
const nodeStore = useNodeStore()
const uiStore = useUiStore()

const info = computed(() => nodeStore.info || {})
const isRunning = computed(() => nodeStore.status === 'running')
const copied = ref(false)

const truncateKey = (key) => {
  if (!key) return '-'
  if (key.length <= 20) return key
  return `${key.slice(0, 10)}...${key.slice(-10)}`
}

const copyAddress = async () => {
  const address = info.value.address
  if (address) {
    try {
      await navigator.clipboard.writeText(address)
      copied.value = true
      uiStore.addNotification('success', t('dashboard.addressCopied'))
      setTimeout(() => {
        copied.value = false
      }, 2000)
    } catch {
      uiStore.addNotification('error', t('dashboard.copyFailed'))
    }
  }
}
</script>

<style scoped>
.node-info {
  font-size: 14px;
}

.info-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.info-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.info-label {
  font-size: 12px;
  color: var(--color-text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.info-value-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.info-value {
  color: var(--color-text-primary);
  font-family: monospace;
  word-break: break-all;
}

.info-key {
  font-size: 13px;
}

.copy-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  padding: 0;
  background-color: var(--color-bg-primary);
  border: 1px solid var(--color-border);
  border-radius: 4px;
  cursor: pointer;
  font-size: 14px;
  transition: all 0.2s ease;
  flex-shrink: 0;
}

.copy-btn:hover {
  border-color: var(--color-accent);
  background-color: rgba(74, 144, 217, 0.1);
}

.info-placeholder {
  color: var(--color-text-secondary);
  text-align: center;
  padding: 20px;
}
</style>
