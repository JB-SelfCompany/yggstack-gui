<template>
  <div id="app" :data-theme="resolvedTheme">
    <AppHeader />
    <div class="app-container">
      <AppSidebar />
      <main class="app-main">
        <router-view />
      </main>
    </div>
    <StatusBar />
    <Toast />
  </div>
</template>

<script setup>
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useUiStore } from './store/ui'
import { useNodeStore } from './store/node'
import { usePeersStore } from './store/peers'
import AppHeader from './components/layout/AppHeader.vue'
import AppSidebar from './components/layout/AppSidebar.vue'
import StatusBar from './components/layout/StatusBar.vue'
import Toast from './components/common/Toast.vue'
import { ipc, Events } from './utils/ipc'

const { locale } = useI18n()
const uiStore = useUiStore()
const nodeStore = useNodeStore()
const peersStore = usePeersStore()

// Use resolvedTheme for system theme support
const resolvedTheme = computed(() => uiStore.resolvedTheme)

onMounted(async () => {
  // Initialize stores with event subscriptions
  uiStore.init()
  nodeStore.init()
  peersStore.init()

  // Load settings from backend
  await uiStore.loadSettings()

  // Sync locale with stored language
  locale.value = uiStore.language

  // Notify backend that frontend is ready
  try {
    await ipc.emit(Events.APP_READY)
  } catch (err) {
    console.warn('Failed to notify backend:', err)
  }
})
</script>

<style>
#app {
  display: flex;
  flex-direction: column;
  height: 100vh;
  background-color: var(--color-bg-primary);
  color: var(--color-text-primary);
}

.app-container {
  display: flex;
  flex: 1;
  overflow: hidden;
}

.app-main {
  flex: 1;
  overflow-y: auto;
  padding: 24px;
}
</style>
