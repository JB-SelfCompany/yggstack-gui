<template>
  <div class="proxy-page">
    <h2 class="page-title">{{ t('proxy.title') }}</h2>
    <p class="page-description">{{ t('proxy.description') }}</p>

    <div class="proxy-grid">
      <!-- Configuration Section -->
      <section class="proxy-section">
        <h3 class="section-title">{{ t('settings.general') }}</h3>

        <!-- Listen Address -->
        <div class="card">
          <div class="setting-item">
            <div class="setting-info">
              <span class="setting-label">{{ t('proxy.listenAddress') }}</span>
              <span class="setting-description">{{ t('proxy.listenAddressDesc') }}</span>
            </div>
            <input
              type="text"
              v-model="listenAddress"
              :placeholder="t('proxy.listenPlaceholder')"
              class="input"
            >
          </div>
        </div>

        <!-- Nameserver -->
        <div class="card">
          <div class="setting-item">
            <div class="setting-info">
              <span class="setting-label">{{ t('proxy.nameserver') }}</span>
              <span class="setting-description">{{ t('proxy.nameserverDesc') }}</span>
            </div>
            <input
              type="text"
              v-model="nameserver"
              :placeholder="t('proxy.nameserverPlaceholder')"
              class="input"
            >
          </div>
        </div>

        <div class="actions">
          <button class="btn btn-primary" @click="saveConfig">
            {{ t('common.save') }}
          </button>
        </div>
      </section>

    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useUiStore } from '../store/ui'
import { ipc, Events } from '../utils/ipc'

const { t } = useI18n()
const uiStore = useUiStore()

const listenAddress = ref('')
const nameserver = ref('')

onMounted(async () => {
  await loadSettings()
})

const loadSettings = async () => {
  try {
    const response = await ipc.emit(Events.SETTINGS_GET)
    if (response.success && response.data?.proxy) {
      listenAddress.value = response.data.proxy.listenAddress || ''
      nameserver.value = response.data.proxy.nameserver || ''
    }
  } catch (err) {
    console.error('Failed to load proxy settings:', err)
  }
}

const saveConfig = async () => {
  try {
    await ipc.emit(Events.SETTINGS_SET, {
      proxy: {
        listenAddress: listenAddress.value,
        nameserver: nameserver.value
      }
    })
    uiStore.addNotification('success', t('settings.saved'))
  } catch (err) {
    uiStore.addNotification('error', t('settings.saveFailed'))
  }
}
</script>

<style scoped>
.proxy-page {
  max-width: 900px;
}

.page-title {
  margin: 0 0 8px 0;
  font-size: 24px;
  font-weight: 600;
  color: var(--color-text-primary);
}

.page-description {
  margin: 0 0 24px 0;
  font-size: 14px;
  color: var(--color-text-secondary);
}

.proxy-grid {
  display: flex;
  flex-direction: column;
  gap: 32px;
}

.proxy-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.section-title {
  margin: 0;
  font-size: 13px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--color-text-secondary);
}

.card {
  background-color: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
  border-radius: 8px;
  padding: 16px 20px;
}

.setting-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.setting-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.setting-label {
  font-size: 14px;
  font-weight: 500;
  color: var(--color-text-primary);
}

.setting-description {
  font-size: 12px;
  color: var(--color-text-secondary);
}

.input {
  padding: 8px 12px;
  font-size: 14px;
  color: var(--color-text-primary);
  background-color: var(--color-bg-primary);
  border: 1px solid var(--color-border);
  border-radius: 6px;
  min-width: 200px;
}

.input:focus {
  outline: none;
  border-color: var(--color-accent);
}

.input:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

/* Toggle Switch */
.toggle {
  position: relative;
  display: inline-block;
  width: 48px;
  height: 26px;
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
  border-radius: 26px;
}

.toggle-slider:before {
  position: absolute;
  content: "";
  height: 20px;
  width: 20px;
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
  transform: translateX(22px);
}

/* Status Grid */
.status-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;
}

.status-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.status-label {
  font-size: 12px;
  color: var(--color-text-secondary);
}

.status-value {
  font-size: 16px;
  font-weight: 600;
  color: var(--color-text-primary);
}

.status-running {
  color: var(--color-success);
}

.status-stopped {
  color: var(--color-text-secondary);
}

.actions {
  display: flex;
  justify-content: flex-end;
  margin-top: 8px;
}

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

.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

@media (max-width: 600px) {
  .setting-item {
    flex-direction: column;
    align-items: flex-start;
  }

  .input {
    width: 100%;
  }

  .status-grid {
    grid-template-columns: 1fr;
  }
}
</style>
