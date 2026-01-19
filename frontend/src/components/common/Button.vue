<template>
  <button
    :class="['btn', `btn-${variant}`, `btn-${size}`, { 'btn-loading': loading, 'btn-icon-only': iconOnly }]"
    :disabled="disabled || loading"
    :type="type"
    @click="handleClick"
  >
    <span v-if="loading" class="btn-spinner"></span>
    <span v-if="icon && !loading" class="btn-icon">{{ icon }}</span>
    <span v-if="$slots.default && !iconOnly" class="btn-text">
      <slot></slot>
    </span>
  </button>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  variant: {
    type: String,
    default: 'primary',
    validator: (v) => ['primary', 'secondary', 'danger', 'ghost', 'success'].includes(v)
  },
  size: {
    type: String,
    default: 'medium',
    validator: (v) => ['small', 'medium', 'large'].includes(v)
  },
  type: {
    type: String,
    default: 'button'
  },
  disabled: {
    type: Boolean,
    default: false
  },
  loading: {
    type: Boolean,
    default: false
  },
  icon: {
    type: String,
    default: null
  },
  iconOnly: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['click'])

const handleClick = (event) => {
  if (!props.disabled && !props.loading) {
    emit('click', event)
  }
}
</script>

<style scoped>
.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  border: 1px solid transparent;
  border-radius: 6px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
  white-space: nowrap;
}

.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* Sizes */
.btn-small {
  padding: 6px 12px;
  font-size: 12px;
}

.btn-medium {
  padding: 8px 16px;
  font-size: 14px;
}

.btn-large {
  padding: 12px 24px;
  font-size: 16px;
}

.btn-icon-only.btn-small {
  padding: 6px;
  min-width: 28px;
}

.btn-icon-only.btn-medium {
  padding: 8px;
  min-width: 36px;
}

.btn-icon-only.btn-large {
  padding: 12px;
  min-width: 44px;
}

/* Variants */
.btn-primary {
  background-color: var(--color-accent);
  color: white;
  border-color: var(--color-accent);
}

.btn-primary:hover:not(:disabled) {
  background-color: var(--color-accent-hover, #3a7bc8);
  border-color: var(--color-accent-hover, #3a7bc8);
}

.btn-secondary {
  background-color: var(--color-bg-secondary);
  color: var(--color-text-primary);
  border-color: var(--color-border);
}

.btn-secondary:hover:not(:disabled) {
  background-color: var(--color-bg-primary);
  border-color: var(--color-accent);
}

.btn-danger {
  background-color: var(--color-danger);
  color: white;
  border-color: var(--color-danger);
}

.btn-danger:hover:not(:disabled) {
  background-color: #c82333;
  border-color: #c82333;
}

.btn-success {
  background-color: var(--color-success);
  color: white;
  border-color: var(--color-success);
}

.btn-success:hover:not(:disabled) {
  background-color: #218838;
  border-color: #218838;
}

.btn-ghost {
  background-color: transparent;
  color: var(--color-text-primary);
  border-color: transparent;
}

.btn-ghost:hover:not(:disabled) {
  background-color: var(--color-bg-secondary);
}

/* Loading state */
.btn-loading {
  position: relative;
  pointer-events: none;
}

.btn-spinner {
  width: 14px;
  height: 14px;
  border: 2px solid transparent;
  border-top-color: currentColor;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

.btn-large .btn-spinner {
  width: 18px;
  height: 18px;
}

.btn-small .btn-spinner {
  width: 12px;
  height: 12px;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

/* Icon */
.btn-icon {
  font-size: 1.1em;
  line-height: 1;
}

/* Focus state */
.btn:focus-visible {
  outline: 2px solid var(--color-accent);
  outline-offset: 2px;
}
</style>
