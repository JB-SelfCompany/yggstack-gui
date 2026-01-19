<template>
  <div class="add-peer-dialog">
    <div class="form-group">
      <label class="form-label">{{ t('peers.uri') }}</label>
      <input
        v-model="peerUri"
        type="text"
        class="form-input"
        :placeholder="t('peers.uriPlaceholder')"
        @keyup.enter="addPeer"
      />
      <p v-if="error" class="form-error">{{ error }}</p>
    </div>
    <div class="dialog-actions">
      <button class="btn btn-secondary" @click="$emit('close')">
        {{ t('common.cancel') }}
      </button>
      <button class="btn btn-primary" :disabled="!isValid" @click="addPeer">
        {{ t('peers.add') }}
      </button>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { usePeersStore } from '../../store/peers'

const { t } = useI18n()
const peersStore = usePeersStore()

const emit = defineEmits(['close'])

const peerUri = ref('')
const error = ref('')

// URI validation for all supported protocols: tcp, tls, quic, ws, wss, unix
const isValid = computed(() => {
  const uri = peerUri.value.trim()
  if (uri.length === 0) return false

  const supportedProtocols = ['tcp://', 'tls://', 'quic://', 'ws://', 'wss://', 'unix://']
  return supportedProtocols.some(protocol => uri.startsWith(protocol))
})

const addPeer = async () => {
  if (!isValid.value) {
    error.value = t('peers.invalidUri')
    return
  }

  try {
    await peersStore.addPeer(peerUri.value.trim())
    emit('close')
  } catch (e) {
    error.value = e.message
  }
}
</script>

<style scoped>
.add-peer-dialog {
  padding: 8px 0;
}

.form-group {
  margin-bottom: 20px;
}

.form-label {
  display: block;
  margin-bottom: 8px;
  font-size: 14px;
  font-weight: 500;
  color: var(--color-text-primary);
}

.form-input {
  width: 100%;
  padding: 10px 12px;
  font-size: 14px;
  color: var(--color-text-primary);
  background-color: var(--color-bg-primary);
  border: 1px solid var(--color-border);
  border-radius: 6px;
  box-sizing: border-box;
}

.form-input:focus {
  outline: none;
  border-color: var(--color-accent);
}

.form-error {
  margin-top: 8px;
  font-size: 13px;
  color: var(--color-danger);
}

.dialog-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
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

.btn-primary {
  background-color: var(--color-accent);
  color: #ffffff;
}

.btn-secondary {
  background-color: var(--color-bg-primary);
  color: var(--color-text-primary);
  border: 1px solid var(--color-border);
}
</style>
