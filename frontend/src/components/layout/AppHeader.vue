<template>
  <header class="app-header">
    <div class="header-title">
      <h1>Yggstack-GUI</h1>
    </div>
    <div class="header-status">
      <span
        class="status-indicator"
        :class="{ 'status-running': isRunning, 'status-stopped': !isRunning }"
      ></span>
      <span class="status-text">{{ statusText }}</span>
    </div>
  </header>
</template>

<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useNodeStore } from '../../store/node'

const { t } = useI18n()
const nodeStore = useNodeStore()

const isRunning = computed(() => nodeStore.status === 'running')
const statusText = computed(() => t(`node.status.${nodeStore.status}`))
</script>

<style scoped>
.app-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 24px;
  height: 56px;
  background-color: var(--color-bg-secondary);
  border-bottom: 1px solid var(--color-border);
  -webkit-app-region: drag;
}

.header-title h1 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: var(--color-text-primary);
}

.header-status {
  display: flex;
  align-items: center;
  gap: 8px;
  -webkit-app-region: no-drag;
}

.status-indicator {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background-color: var(--color-text-secondary);
}

.status-indicator.status-running {
  background-color: var(--color-success);
}

.status-indicator.status-stopped {
  background-color: var(--color-danger);
}

.status-text {
  font-size: 14px;
  color: var(--color-text-secondary);
}
</style>
