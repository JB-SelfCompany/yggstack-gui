<template>
  <footer class="status-bar">
    <div class="status-left">
      <span v-if="nodeInfo.address" class="status-item">
        {{ t('node.info.address') }}: {{ nodeInfo.address }}
      </span>
    </div>
    <div class="status-right">
      <span class="status-item">v{{ version }}</span>
    </div>
  </footer>
</template>

<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useNodeStore } from '../../store/node'

const { t } = useI18n()
const nodeStore = useNodeStore()

const version = '0.1.0'
const nodeInfo = computed(() => nodeStore.info || {})
</script>

<style scoped>
.status-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 16px;
  height: 28px;
  background-color: var(--color-bg-secondary);
  border-top: 1px solid var(--color-border);
  font-size: 12px;
  color: var(--color-text-secondary);
}

.status-left,
.status-right {
  display: flex;
  align-items: center;
  gap: 16px;
}

.status-item {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
</style>
