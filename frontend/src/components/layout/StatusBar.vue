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
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useNodeStore } from '../../store/node'
import { ipc, Events } from '../../utils/ipc'

const { t } = useI18n()
const nodeStore = useNodeStore()

const version = ref('...')
const nodeInfo = computed(() => nodeStore.info || {})

onMounted(async () => {
  try {
    const response = await ipc.emit(Events.APP_VERSION)
    if (response.success && response.data) {
      version.value = response.data.version
    }
  } catch (err) {
    console.error('Failed to load version:', err)
    version.value = 'unknown'
  }
})
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
