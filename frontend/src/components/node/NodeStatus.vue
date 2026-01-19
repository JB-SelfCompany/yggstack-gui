<template>
  <div class="node-status">
    <div class="status-badge" :class="statusClass">
      <span class="status-dot"></span>
      <span class="status-label">{{ t(`node.status.${status}`) }}</span>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useNodeStore } from '../../store/node'

const { t } = useI18n()
const nodeStore = useNodeStore()

const status = computed(() => nodeStore.status)
const statusClass = computed(() => `status-${status.value}`)
</script>

<style scoped>
.node-status {
  margin-bottom: 16px;
}

.status-badge {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  border-radius: 20px;
  font-size: 14px;
  font-weight: 500;
}

.status-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
}

.status-stopped {
  background-color: rgba(220, 53, 69, 0.1);
  color: var(--color-danger);
}

.status-stopped .status-dot {
  background-color: var(--color-danger);
}

.status-starting {
  background-color: rgba(255, 193, 7, 0.1);
  color: #ffc107;
}

.status-starting .status-dot {
  background-color: #ffc107;
  animation: pulse 1s infinite;
}

.status-running {
  background-color: rgba(40, 167, 69, 0.1);
  color: var(--color-success);
}

.status-running .status-dot {
  background-color: var(--color-success);
}

.status-stopping {
  background-color: rgba(255, 193, 7, 0.1);
  color: #ffc107;
}

.status-stopping .status-dot {
  background-color: #ffc107;
  animation: pulse 1s infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}
</style>
