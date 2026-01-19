import { defineStore } from 'pinia'
import { ipc, Events, IPCError } from '../utils/ipc'

export const useUiStore = defineStore('ui', {
  state: () => ({
    language: 'en',
    theme: 'system', // 'light' | 'dark' | 'system'
    systemTheme: 'dark', // Detected system theme
    sidebarCollapsed: false,
    loading: false,
    notifications: [],
    notificationId: 0
  }),

  getters: {
    // Resolve actual theme based on user preference
    resolvedTheme: (state) => {
      if (state.theme === 'system') {
        return state.systemTheme
      }
      return state.theme
    },
    isDarkTheme() {
      return this.resolvedTheme === 'dark'
    },
    hasNotifications: (state) => state.notifications.length > 0
  },

  actions: {
    // Initialize store and subscribe to state sync
    init() {
      // Listen for backend state changes
      ipc.on(Events.STATE_CHANGED, (data) => {
        if (data.key === 'language') {
          this.language = data.value
        } else if (data.key === 'theme') {
          this.theme = data.value
        }
      })

      // Detect system theme
      this.detectSystemTheme()

      // Listen for system theme changes
      if (typeof window !== 'undefined' && window.matchMedia) {
        const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
        mediaQuery.addEventListener('change', (e) => {
          this.systemTheme = e.matches ? 'dark' : 'light'
        })
      }
    },

    // Detect current system theme preference
    detectSystemTheme() {
      if (typeof window !== 'undefined' && window.matchMedia) {
        const isDark = window.matchMedia('(prefers-color-scheme: dark)').matches
        this.systemTheme = isDark ? 'dark' : 'light'
      }
    },

    // Load settings from backend
    async loadSettings() {
      this.loading = true
      try {
        const response = await ipc.emit(Events.SETTINGS_GET)
        if (response.success && response.data) {
          this.language = response.data.language || 'en'
          this.theme = response.data.theme || 'system'
        }
      } catch (err) {
        console.error('Failed to load settings:', err)
      } finally {
        this.loading = false
      }
    },

    // Save settings to backend
    async saveSettings() {
      try {
        await ipc.emit(Events.SETTINGS_SET, {
          language: this.language,
          theme: this.theme
        })
      } catch (err) {
        console.error('Failed to save settings:', err)
        this.addNotification('error', 'Failed to save settings')
      }
    },

    // Set language
    async setLanguage(lang) {
      this.language = lang
      await this.saveSettings()
    },

    // Set theme
    async setTheme(theme) {
      this.theme = theme
      await this.saveSettings()
    },

    // Toggle sidebar
    toggleSidebar() {
      this.sidebarCollapsed = !this.sidebarCollapsed
    },

    // Add notification
    addNotification(type, message, timeout = 5000) {
      const id = ++this.notificationId
      this.notifications.push({
        id,
        type, // 'info' | 'success' | 'warning' | 'error'
        message,
        timestamp: Date.now()
      })

      // Auto-remove after timeout
      if (timeout > 0) {
        setTimeout(() => {
          this.removeNotification(id)
        }, timeout)
      }

      return id
    },

    // Remove notification
    removeNotification(id) {
      const index = this.notifications.findIndex(n => n.id === id)
      if (index !== -1) {
        this.notifications.splice(index, 1)
      }
    },

    // Clear all notifications
    clearNotifications() {
      this.notifications = []
    }
  }
})
