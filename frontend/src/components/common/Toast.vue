<template>
  <Teleport to="body">
    <TransitionGroup
      name="toast"
      tag="div"
      class="toast-container"
    >
      <div
        v-for="notification in notifications"
        :key="notification.id"
        :class="['toast', `toast-${notification.type}`]"
      >
        <span class="toast-icon">{{ getIcon(notification.type) }}</span>
        <span class="toast-message">{{ notification.message }}</span>
        <button class="toast-close" @click="dismiss(notification.id)">&times;</button>
      </div>
    </TransitionGroup>
  </Teleport>
</template>

<script setup>
import { computed } from 'vue'
import { useUiStore } from '../../store/ui'

const uiStore = useUiStore()

const notifications = computed(() => uiStore.notifications)

const getIcon = (type) => {
  switch (type) {
    case 'success': return '\u2713'
    case 'error': return '\u2717'
    case 'warning': return '\u26A0'
    case 'info':
    default: return '\u2139'
  }
}

const dismiss = (id) => {
  uiStore.removeNotification(id)
}
</script>

<style scoped>
.toast-container {
  position: fixed;
  bottom: 24px;
  right: 24px;
  display: flex;
  flex-direction: column;
  gap: 8px;
  z-index: 9999;
  max-width: 400px;
}

.toast {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  border-radius: 8px;
  background-color: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  min-width: 280px;
}

.toast-icon {
  font-size: 18px;
  flex-shrink: 0;
}

.toast-message {
  flex: 1;
  font-size: 14px;
  color: var(--color-text-primary);
  line-height: 1.4;
}

.toast-close {
  background: none;
  border: none;
  color: var(--color-text-secondary);
  font-size: 18px;
  cursor: pointer;
  padding: 0 4px;
  flex-shrink: 0;
  transition: color 0.2s ease;
}

.toast-close:hover {
  color: var(--color-text-primary);
}

/* Toast types */
.toast-success {
  border-color: var(--color-success);
}

.toast-success .toast-icon {
  color: var(--color-success);
}

.toast-error {
  border-color: var(--color-danger);
}

.toast-error .toast-icon {
  color: var(--color-danger);
}

.toast-warning {
  border-color: var(--color-warning);
}

.toast-warning .toast-icon {
  color: var(--color-warning);
}

.toast-info {
  border-color: var(--color-accent);
}

.toast-info .toast-icon {
  color: var(--color-accent);
}

/* Animations */
.toast-enter-active {
  animation: toast-in 0.3s ease;
}

.toast-leave-active {
  animation: toast-out 0.2s ease;
}

@keyframes toast-in {
  from {
    opacity: 0;
    transform: translateX(100%);
  }
  to {
    opacity: 1;
    transform: translateX(0);
  }
}

@keyframes toast-out {
  from {
    opacity: 1;
    transform: translateX(0);
  }
  to {
    opacity: 0;
    transform: translateX(100%);
  }
}

/* Dark theme adjustments */
[data-theme="dark"] .toast {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.4);
}
</style>
