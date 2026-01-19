<template>
  <div :class="['input-wrapper', { 'input-error': error, 'input-disabled': disabled }]">
    <label v-if="label" :for="inputId" class="input-label">
      {{ label }}
      <span v-if="required" class="input-required">*</span>
    </label>
    <div class="input-container">
      <span v-if="icon" class="input-icon">{{ icon }}</span>
      <input
        :id="inputId"
        :type="type"
        :value="modelValue"
        :placeholder="placeholder"
        :disabled="disabled"
        :readonly="readonly"
        :required="required"
        :autocomplete="autocomplete"
        :class="['input', { 'input-with-icon': icon }]"
        @input="handleInput"
        @blur="handleBlur"
        @focus="handleFocus"
        @keydown.enter="handleEnter"
      />
      <button
        v-if="clearable && modelValue"
        type="button"
        class="input-clear"
        @click="handleClear"
      >
        &times;
      </button>
    </div>
    <p v-if="error" class="input-error-message">{{ error }}</p>
    <p v-else-if="hint" class="input-hint">{{ hint }}</p>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  modelValue: {
    type: [String, Number],
    default: ''
  },
  type: {
    type: String,
    default: 'text'
  },
  label: {
    type: String,
    default: null
  },
  placeholder: {
    type: String,
    default: ''
  },
  error: {
    type: String,
    default: null
  },
  hint: {
    type: String,
    default: null
  },
  icon: {
    type: String,
    default: null
  },
  disabled: {
    type: Boolean,
    default: false
  },
  readonly: {
    type: Boolean,
    default: false
  },
  required: {
    type: Boolean,
    default: false
  },
  clearable: {
    type: Boolean,
    default: false
  },
  autocomplete: {
    type: String,
    default: 'off'
  },
  id: {
    type: String,
    default: null
  }
})

const emit = defineEmits(['update:modelValue', 'blur', 'focus', 'enter', 'clear'])

// Generate unique ID if not provided
const inputId = computed(() => props.id || `input-${Math.random().toString(36).substr(2, 9)}`)

const handleInput = (event) => {
  emit('update:modelValue', event.target.value)
}

const handleBlur = (event) => {
  emit('blur', event)
}

const handleFocus = (event) => {
  emit('focus', event)
}

const handleEnter = (event) => {
  emit('enter', event)
}

const handleClear = () => {
  emit('update:modelValue', '')
  emit('clear')
}
</script>

<style scoped>
.input-wrapper {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.input-label {
  font-size: 14px;
  font-weight: 500;
  color: var(--color-text-primary);
}

.input-required {
  color: var(--color-danger);
  margin-left: 2px;
}

.input-container {
  position: relative;
  display: flex;
  align-items: center;
}

.input {
  width: 100%;
  padding: 10px 12px;
  font-size: 14px;
  font-family: inherit;
  color: var(--color-text-primary);
  background-color: var(--color-bg-primary);
  border: 1px solid var(--color-border);
  border-radius: 6px;
  transition: border-color 0.2s ease, box-shadow 0.2s ease;
}

.input::placeholder {
  color: var(--color-text-secondary);
}

.input:focus {
  outline: none;
  border-color: var(--color-accent);
  box-shadow: 0 0 0 3px rgba(74, 144, 217, 0.15);
}

.input:disabled {
  background-color: var(--color-bg-secondary);
  cursor: not-allowed;
  opacity: 0.6;
}

.input:read-only {
  background-color: var(--color-bg-secondary);
}

.input-with-icon {
  padding-left: 36px;
}

.input-icon {
  position: absolute;
  left: 12px;
  color: var(--color-text-secondary);
  font-size: 14px;
  pointer-events: none;
}

.input-clear {
  position: absolute;
  right: 8px;
  width: 20px;
  height: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-bg-secondary);
  border: none;
  border-radius: 50%;
  color: var(--color-text-secondary);
  font-size: 14px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.input-clear:hover {
  background-color: var(--color-border);
  color: var(--color-text-primary);
}

/* Error state */
.input-error .input {
  border-color: var(--color-danger);
}

.input-error .input:focus {
  box-shadow: 0 0 0 3px rgba(220, 53, 69, 0.15);
}

.input-error-message {
  margin: 0;
  font-size: 12px;
  color: var(--color-danger);
}

.input-hint {
  margin: 0;
  font-size: 12px;
  color: var(--color-text-secondary);
}

/* Disabled state */
.input-disabled {
  opacity: 0.6;
}
</style>
