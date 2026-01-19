<template>
  <tr class="peer-item">
    <td class="peer-uri">{{ peer.uri }}</td>
    <td class="peer-status">
      <span class="status-badge" :class="statusClass">
        {{ t(`peers.status.${peer.connected ? 'up' : 'down'}`) }}
      </span>
    </td>
    <td class="peer-actions">
      <button class="btn-icon btn-danger-icon" @click="$emit('remove', peer.uri)">
        ×
      </button>
    </td>
  </tr>
</template>

<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const props = defineProps({
  peer: {
    type: Object,
    required: true
  }
})

defineEmits(['remove'])

const statusClass = computed(() => props.peer.connected ? 'status-up' : 'status-down')
</script>

<style scoped>
.peer-item td {
  padding: 12px;
  border-bottom: 1px solid var(--color-border);
}

.peer-uri {
  font-family: monospace;
  font-size: 13px;
  color: var(--color-text-primary);
  max-width: 400px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.peer-status {
  width: 120px;
}

.status-badge {
  display: inline-block;
  padding: 4px 12px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: 500;
}

.status-up {
  background-color: rgba(40, 167, 69, 0.1);
  color: var(--color-success);
}

.status-down {
  background-color: rgba(220, 53, 69, 0.1);
  color: var(--color-danger);
}

.peer-actions {
  width: 60px;
  text-align: center;
}

.btn-icon {
  width: 28px;
  height: 28px;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 18px;
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
</style>
