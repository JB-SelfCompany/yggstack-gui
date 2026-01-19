/**
 * UI store tests
 */

import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useUiStore } from './ui'
import { ipcBridge } from '../utils/ipc'

describe('UI Store', () => {
  let store

  beforeEach(() => {
    setActivePinia(createPinia())
    store = useUiStore()
  })

  describe('initial state', () => {
    it('should have correct initial values', () => {
      expect(store.language).toBe('en')
      expect(store.theme).toBe('system')
      expect(store.systemTheme).toBe('dark')
      expect(store.sidebarCollapsed).toBe(false)
      expect(store.loading).toBe(false)
      expect(store.notifications).toEqual([])
    })
  })

  describe('getters', () => {
    it('resolvedTheme should return system theme when theme is system', () => {
      store.theme = 'system'
      store.systemTheme = 'dark'
      expect(store.resolvedTheme).toBe('dark')

      store.systemTheme = 'light'
      expect(store.resolvedTheme).toBe('light')
    })

    it('resolvedTheme should return explicit theme when not system', () => {
      store.theme = 'dark'
      store.systemTheme = 'light'
      expect(store.resolvedTheme).toBe('dark')

      store.theme = 'light'
      expect(store.resolvedTheme).toBe('light')
    })

    it('isDarkTheme should return true when resolved theme is dark', () => {
      store.theme = 'dark'
      expect(store.isDarkTheme).toBe(true)

      store.theme = 'light'
      expect(store.isDarkTheme).toBe(false)
    })

    it('hasNotifications should return true when notifications exist', () => {
      expect(store.hasNotifications).toBe(false)
      store.notifications = [{ id: 1, message: 'test' }]
      expect(store.hasNotifications).toBe(true)
    })
  })

  describe('actions', () => {
    describe('detectSystemTheme', () => {
      it('should detect dark theme from matchMedia', () => {
        window.matchMedia = vi.fn().mockImplementation(query => ({
          matches: query === '(prefers-color-scheme: dark)',
          media: query,
          addEventListener: vi.fn()
        }))

        store.detectSystemTheme()
        expect(store.systemTheme).toBe('dark')
      })

      it('should detect light theme from matchMedia', () => {
        window.matchMedia = vi.fn().mockImplementation(() => ({
          matches: false,
          media: '',
          addEventListener: vi.fn()
        }))

        store.detectSystemTheme()
        expect(store.systemTheme).toBe('light')
      })
    })

    describe('loadSettings', () => {
      it('should load settings from backend', async () => {
        vi.spyOn(ipcBridge, 'emit').mockResolvedValueOnce({
          success: true,
          data: { language: 'ru', theme: 'light' }
        })

        await store.loadSettings()

        expect(store.language).toBe('ru')
        expect(store.theme).toBe('light')
        expect(store.loading).toBe(false)
      })

      it('should keep defaults on failed load', async () => {
        vi.spyOn(ipcBridge, 'emit').mockRejectedValueOnce(new Error('Load failed'))

        await store.loadSettings()

        expect(store.language).toBe('en')
        expect(store.theme).toBe('system')
      })
    })

    describe('saveSettings', () => {
      it('should save settings to backend', async () => {
        const emitSpy = vi.spyOn(ipcBridge, 'emit').mockResolvedValueOnce({ success: true })
        store.language = 'ru'
        store.theme = 'dark'

        await store.saveSettings()

        expect(emitSpy).toHaveBeenCalledWith('settings:set', {
          language: 'ru',
          theme: 'dark'
        })
      })

      it('should add notification on save error', async () => {
        vi.spyOn(ipcBridge, 'emit').mockRejectedValueOnce(new Error('Save failed'))

        await store.saveSettings()

        expect(store.notifications).toHaveLength(1)
        expect(store.notifications[0].type).toBe('error')
      })
    })

    describe('setLanguage', () => {
      it('should set language and save', async () => {
        const saveSpy = vi.spyOn(store, 'saveSettings').mockResolvedValueOnce()

        await store.setLanguage('ru')

        expect(store.language).toBe('ru')
        expect(saveSpy).toHaveBeenCalled()
      })
    })

    describe('setTheme', () => {
      it('should set theme and save', async () => {
        const saveSpy = vi.spyOn(store, 'saveSettings').mockResolvedValueOnce()

        await store.setTheme('light')

        expect(store.theme).toBe('light')
        expect(saveSpy).toHaveBeenCalled()
      })
    })

    describe('toggleSidebar', () => {
      it('should toggle sidebar collapsed state', () => {
        expect(store.sidebarCollapsed).toBe(false)
        store.toggleSidebar()
        expect(store.sidebarCollapsed).toBe(true)
        store.toggleSidebar()
        expect(store.sidebarCollapsed).toBe(false)
      })
    })

    describe('notifications', () => {
      it('addNotification should add and return id', () => {
        const id = store.addNotification('info', 'Test message', 0)

        expect(store.notifications).toHaveLength(1)
        expect(store.notifications[0]).toEqual({
          id,
          type: 'info',
          message: 'Test message',
          timestamp: expect.any(Number)
        })
      })

      it('addNotification should auto-remove after timeout', async () => {
        vi.useFakeTimers()
        store.addNotification('info', 'Test message', 1000)

        expect(store.notifications).toHaveLength(1)

        vi.advanceTimersByTime(1000)
        expect(store.notifications).toHaveLength(0)

        vi.useRealTimers()
      })

      it('removeNotification should remove by id', () => {
        const id = store.addNotification('info', 'Test', 0)
        store.addNotification('error', 'Another', 0)

        store.removeNotification(id)

        expect(store.notifications).toHaveLength(1)
        expect(store.notifications[0].type).toBe('error')
      })

      it('clearNotifications should remove all', () => {
        store.addNotification('info', 'Test1', 0)
        store.addNotification('error', 'Test2', 0)

        store.clearNotifications()

        expect(store.notifications).toHaveLength(0)
      })
    })
  })
})
