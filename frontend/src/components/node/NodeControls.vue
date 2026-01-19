<template>
  <div class="node-controls">
    <button
      v-if="canStart"
      class="btn btn-success"
      :disabled="isLoading"
      @click="startNode"
    >
      {{ t('node.actions.start') }}
    </button>
    <button
      v-if="canStop"
      class="btn btn-danger"
      :disabled="isLoading"
      @click="stopNode"
    >
      {{ t('node.actions.stop') }}
    </button>
    <button
      v-if="isRunning"
      class="btn btn-secondary"
      :disabled="isLoading"
      @click="restartNode"
    >
      {{ t('node.actions.restart') }}
    </button>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useNodeStore } from '../../store/node'

const { t } = useI18n()
const nodeStore = useNodeStore()

const status = computed(() => nodeStore.status)
const isLoading = computed(() => ['starting', 'stopping'].includes(status.value))
const isRunning = computed(() => status.value === 'running')
const canStart = computed(() => status.value === 'stopped')
const canStop = computed(() => status.value === 'running')

const startNode = async () => {
  await nodeStore.start()
}

const stopNode = async () => {
  await nodeStore.stop()
}

const restartNode = async () => {
  await nodeStore.stop()
  await nodeStore.start()
}
</script>

<style scoped>
.node-controls {
  display: flex;
  gap: 12px;
  margin-top: 16px;
}

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

.btn-success {
  background-color: var(--color-success);
  color: #ffffff;
}

.btn-success:hover:not(:disabled) {
  opacity: 0.9;
}

.btn-danger {
  background-color: var(--color-danger);
  color: #ffffff;
}

.btn-danger:hover:not(:disabled) {
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
</style>
