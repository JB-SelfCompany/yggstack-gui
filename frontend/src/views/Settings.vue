<template>
  <div class="settings-page">
    <h2 class="page-title">{{ t('settings.title') }}</h2>

    <div class="settings-grid">
      <!-- General Section -->
      <section class="settings-section">
        <h3 class="section-title">{{ t('settings.general') }}</h3>

        <!-- Language Setting -->
        <div class="card">
          <div class="setting-item">
            <div class="setting-info">
              <span class="setting-label">{{ t('settings.language') }}</span>
              <span class="setting-description">{{ t('settings.languageDesc') }}</span>
            </div>
            <select v-model="language" class="select" @change="updateLanguage">
              <option value="en">English</option>
              <option value="ru">Русский</option>
            </select>
          </div>
        </div>

        <!-- Theme Setting -->
        <div class="card">
          <div class="setting-item">
            <div class="setting-info">
              <span class="setting-label">{{ t('settings.theme') }}</span>
              <span class="setting-description">{{ t('settings.themeDesc') }}</span>
            </div>
            <select v-model="theme" class="select" @change="updateTheme">
              <option value="system">{{ t('settings.themes.system') }}</option>
              <option value="light">{{ t('settings.themes.light') }}</option>
              <option value="dark">{{ t('settings.themes.dark') }}</option>
            </select>
          </div>
          <div v-if="theme === 'system'" class="theme-preview">
            <span class="theme-indicator" :class="uiStore.resolvedTheme"></span>
            <span class="theme-current">{{ t('settings.currentTheme') }}: {{ t(`settings.themes.${uiStore.resolvedTheme}`) }}</span>
          </div>
        </div>

        <!-- Minimize to Tray -->
        <div class="card">
          <div class="setting-item">
            <div class="setting-info">
              <span class="setting-label">{{ t('settings.minimizeToTray') }}</span>
            </div>
            <label class="toggle">
              <input type="checkbox" v-model="minimizeToTray" @change="updateSetting('minimizeToTray', minimizeToTray)">
              <span class="toggle-slider"></span>
            </label>
          </div>
        </div>

        <!-- Start Minimized -->
        <div class="card">
          <div class="setting-item">
            <div class="setting-info">
              <span class="setting-label">{{ t('settings.startMinimized') }}</span>
            </div>
            <label class="toggle">
              <input type="checkbox" v-model="startMinimized" @change="updateSetting('startMinimized', startMinimized)">
              <span class="toggle-slider"></span>
            </label>
          </div>
        </div>

        <!-- Log Level -->
        <div class="card">
          <div class="setting-item">
            <div class="setting-info">
              <span class="setting-label">{{ t('settings.logLevel') }}</span>
            </div>
            <select v-model="logLevel" class="select" @change="updateSetting('logLevel', logLevel)">
              <option value="debug">{{ t('settings.logLevels.debug') }}</option>
              <option value="info">{{ t('settings.logLevels.info') }}</option>
              <option value="warn">{{ t('settings.logLevels.warn') }}</option>
              <option value="error">{{ t('settings.logLevels.error') }}</option>
            </select>
          </div>
        </div>
      </section>

      <!-- About Section -->
      <section class="settings-section">
        <h3 class="section-title">{{ t('settings.about') }}</h3>
        <div class="card">
          <div class="about-info">
            <div class="about-row">
              <span class="about-label">{{ t('settings.version') }}</span>
              <span class="about-value">{{ appVersion || '...' }}</span>
            </div>
            <div class="about-row">
              <span class="about-label">{{ t('settings.framework') }}</span>
              <span class="about-value">Energy + Vue.js 3</span>
            </div>
            <div class="about-row">
              <span class="about-label">{{ t('settings.repository') }}</span>
              <a href="#" class="about-link" @click.prevent="openRepo">
                github.com/JB-SelfCompany/yggstack-gui
              </a>
            </div>
          </div>
        </div>
      </section>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useUiStore } from '../store/ui'
import { ipc, Events } from '../utils/ipc'

const { t, locale } = useI18n()
const uiStore = useUiStore()

const language = ref('en')
const theme = ref('system')
const minimizeToTray = ref(true)
const startMinimized = ref(false)
const logLevel = ref('info')
const appVersion = ref('')

onMounted(async () => {
  // Load app version
  try {
    const versionResponse = await ipc.emit(Events.APP_VERSION)
    if (versionResponse.success && versionResponse.data) {
      appVersion.value = versionResponse.data.version
    }
  } catch (err) {
    console.error('Failed to load version:', err)
    appVersion.value = 'unknown'
  }

  // Load settings from backend
  try {
    const response = await ipc.emit(Events.SETTINGS_GET)
    if (response.success && response.data) {
      language.value = response.data.language || 'en'
      theme.value = response.data.theme || 'system'
      minimizeToTray.value = response.data.minimizeToTray ?? true
      startMinimized.value = response.data.startMinimized ?? false
      logLevel.value = response.data.logLevel || 'info'

      // Sync with UI store
      uiStore.language = language.value
      uiStore.theme = theme.value
    }
  } catch (err) {
    console.error('Failed to load settings:', err)
    // Use UI store values as fallback
    language.value = uiStore.language
    theme.value = uiStore.theme
  }
})

// Sync with store changes
watch(() => uiStore.language, (val) => {
  language.value = val
})

watch(() => uiStore.theme, (val) => {
  theme.value = val
})

const updateLanguage = () => {
  uiStore.setLanguage(language.value)
  locale.value = language.value
}

const updateTheme = () => {
  uiStore.setTheme(theme.value)
}

const updateSetting = async (key, value) => {
  try {
    await ipc.emit(Events.SETTINGS_SET, { [key]: value })
  } catch (err) {
    console.error(`Failed to update setting ${key}:`, err)
    uiStore.addNotification('error', t('settings.saveFailed'))
  }
}

const openRepo = () => {
  if (window.ipc) {
    window.ipc.emit('app:openUrl', { url: 'https://github.com/JB-SelfCompany/yggstack-gui' })
  }
}
</script>

<style scoped>
.settings-page {
  max-width: 800px;
}

.page-title {
  margin: 0 0 24px 0;
  font-size: 24px;
  font-weight: 600;
  color: var(--color-text-primary);
}

.settings-grid {
  display: flex;
  flex-direction: column;
  gap: 32px;
}

.settings-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.section-title {
  margin: 0;
  font-size: 13px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--color-text-secondary);
}

.card {
  background-color: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
  border-radius: 8px;
  padding: 16px 20px;
}

.setting-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.setting-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.setting-label {
  font-size: 14px;
  font-weight: 500;
  color: var(--color-text-primary);
}

.setting-description {
  font-size: 12px;
  color: var(--color-text-secondary);
}

.select {
  padding: 8px 12px;
  font-size: 14px;
  color: var(--color-text-primary);
  background-color: var(--color-bg-primary);
  border: 1px solid var(--color-border);
  border-radius: 6px;
  cursor: pointer;
  min-width: 160px;
}

.select:focus {
  outline: none;
  border-color: var(--color-accent);
}

.theme-preview {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid var(--color-border);
}

.theme-indicator {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  border: 2px solid var(--color-border);
}

.theme-indicator.dark {
  background-color: #1a1a2e;
}

.theme-indicator.light {
  background-color: #ffffff;
}

.theme-current {
  font-size: 12px;
  color: var(--color-text-secondary);
}

/* About Section */
.about-info {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.about-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--color-border);
}

.about-row:last-child {
  padding-bottom: 0;
  border-bottom: none;
}

.about-label {
  font-size: 14px;
  color: var(--color-text-secondary);
}

.about-value {
  font-size: 14px;
  font-weight: 500;
  color: var(--color-text-primary);
}

.about-link {
  font-size: 14px;
  color: var(--color-accent);
  text-decoration: none;
}

.about-link:hover {
  text-decoration: underline;
}

/* Toggle Switch */
.toggle {
  position: relative;
  display: inline-block;
  width: 48px;
  height: 26px;
}

.toggle input {
  opacity: 0;
  width: 0;
  height: 0;
}

.toggle-slider {
  position: absolute;
  cursor: pointer;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: var(--color-border);
  transition: 0.3s;
  border-radius: 26px;
}

.toggle-slider:before {
  position: absolute;
  content: "";
  height: 20px;
  width: 20px;
  left: 3px;
  bottom: 3px;
  background-color: white;
  transition: 0.3s;
  border-radius: 50%;
}

.toggle input:checked + .toggle-slider {
  background-color: var(--color-success);
}

.toggle input:checked + .toggle-slider:before {
  transform: translateX(22px);
}

@media (max-width: 600px) {
  .setting-item {
    flex-direction: column;
    align-items: flex-start;
  }

  .select {
    width: 100%;
  }
}
</style>
