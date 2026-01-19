<template>
  <div :class="['spinner-container', { 'spinner-overlay': overlay }]">
    <div :class="['spinner', `spinner-${size}`]" :style="spinnerStyle">
      <div class="spinner-ring"></div>
    </div>
    <span v-if="text" class="spinner-text">{{ text }}</span>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  size: {
    type: String,
    default: 'medium',
    validator: (v) => ['small', 'medium', 'large'].includes(v)
  },
  color: {
    type: String,
    default: null
  },
  text: {
    type: String,
    default: null
  },
  overlay: {
    type: Boolean,
    default: false
  }
})

const spinnerStyle = computed(() => {
  if (props.color) {
    return { '--spinner-color': props.color }
  }
  return {}
})
</script>

<style scoped>
.spinner-container {
  display: inline-flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
}

.spinner-overlay {
  position: absolute;
  inset: 0;
  background-color: rgba(var(--color-bg-primary-rgb, 255, 255, 255), 0.8);
  z-index: 10;
}

[data-theme="dark"] .spinner-overlay {
  background-color: rgba(26, 26, 46, 0.8);
}

.spinner {
  --spinner-color: var(--color-accent);
  position: relative;
}

.spinner-small {
  width: 20px;
  height: 20px;
}

.spinner-medium {
  width: 32px;
  height: 32px;
}

.spinner-large {
  width: 48px;
  height: 48px;
}

.spinner-ring {
  width: 100%;
  height: 100%;
  border: 3px solid var(--color-border);
  border-top-color: var(--spinner-color);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

.spinner-small .spinner-ring {
  border-width: 2px;
}

.spinner-large .spinner-ring {
  border-width: 4px;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.spinner-text {
  font-size: 14px;
  color: var(--color-text-secondary);
}

.spinner-small + .spinner-text {
  font-size: 12px;
}

.spinner-large + .spinner-text {
  font-size: 16px;
}
</style>
