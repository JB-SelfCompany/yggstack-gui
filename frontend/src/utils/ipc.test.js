/**
 * IPC Bridge tests
 */

import { describe, it, expect, beforeEach, vi } from 'vitest'
import { IPCError, Events, ipc, ipcBridge } from './ipc'

describe('IPCError', () => {
  it('should create error with code and message', () => {
    const error = new IPCError('TIMEOUT', 'Request timed out')
    expect(error.code).toBe('TIMEOUT')
    expect(error.message).toBe('Request timed out')
    expect(error.name).toBe('IPCError')
  })

  it('should include details when provided', () => {
    const error = new IPCError('VALIDATION', 'Invalid input', { field: 'uri' })
    expect(error.details).toEqual({ field: 'uri' })
  })
})

describe('Events', () => {
  it('should define all required event names', () => {
    expect(Events.APP_VERSION).toBe('app:version')
    expect(Events.APP_READY).toBe('app:ready')
    expect(Events.NODE_START).toBe('node:start')
    expect(Events.NODE_STOP).toBe('node:stop')
    expect(Events.NODE_STATUS).toBe('node:status')
    expect(Events.PEERS_LIST).toBe('peers:list')
    expect(Events.PEERS_ADD).toBe('peers:add')
    expect(Events.PEERS_REMOVE).toBe('peers:remove')
    expect(Events.SETTINGS_GET).toBe('settings:get')
    expect(Events.SETTINGS_SET).toBe('settings:set')
  })
})

describe('IPCBridge', () => {
  beforeEach(() => {
    // Reset bridge state
    ipcBridge.eventListeners.clear()
    ipcBridge.pendingRequests.clear()
    ipcBridge.stateCache.clear()
    ipcBridge.requestId = 0
  })

  describe('event subscriptions', () => {
    it('on should add listener and return unsubscribe function', () => {
      const callback = vi.fn()
      const unsubscribe = ipc.on('test:event', callback)

      expect(ipcBridge.eventListeners.has('test:event')).toBe(true)
      expect(ipcBridge.eventListeners.get('test:event').has(callback)).toBe(true)

      unsubscribe()
      expect(ipcBridge.eventListeners.get('test:event').has(callback)).toBe(false)
    })

    it('once should remove listener after first call', () => {
      const callback = vi.fn()
      ipc.once('test:once', callback)

      // Simulate event
      ipcBridge._notifyListeners('test:once', { data: 'test' })

      expect(callback).toHaveBeenCalledTimes(1)
      expect(ipcBridge.eventListeners.get('test:once')?.size || 0).toBe(0)
    })

    it('off should remove specific callback', () => {
      const callback1 = vi.fn()
      const callback2 = vi.fn()

      ipc.on('test:multi', callback1)
      ipc.on('test:multi', callback2)

      ipc.off('test:multi', callback1)

      expect(ipcBridge.eventListeners.get('test:multi').has(callback1)).toBe(false)
      expect(ipcBridge.eventListeners.get('test:multi').has(callback2)).toBe(true)
    })

    it('off without callback should remove all listeners', () => {
      ipc.on('test:all', vi.fn())
      ipc.on('test:all', vi.fn())

      ipc.off('test:all')

      expect(ipcBridge.eventListeners.has('test:all')).toBe(false)
    })

    it('wildcard listener should receive all events', () => {
      const wildcardCallback = vi.fn()
      ipc.on('*', wildcardCallback)

      ipcBridge._notifyListeners('any:event', { data: 'test' })

      expect(wildcardCallback).toHaveBeenCalledWith('any:event', { data: 'test' })
    })
  })

  describe('emit', () => {
    it('should return mock response when not connected', async () => {
      // Temporarily set not connected
      const wasConnected = ipcBridge.isConnected
      ipcBridge.isConnected = false

      const response = await ipc.emit(Events.APP_PING)

      expect(response.success).toBe(true)
      expect(response.data.pong).toBe(true)

      ipcBridge.isConnected = wasConnected
    })

    it('should send message with requestId', async () => {
      const response = await ipc.emit(Events.NODE_STATUS)

      expect(response.success).toBe(true)
      expect(response.data.state).toBe('stopped')
    })

    it('should increment requestId for each call', async () => {
      const initialId = ipcBridge.requestId
      await ipc.emit(Events.APP_PING)
      await ipc.emit(Events.APP_PING)

      expect(ipcBridge.requestId).toBe(initialId + 2)
    })
  })

  describe('ping', () => {
    it('should return true on successful ping', async () => {
      const result = await ipc.ping()
      expect(result).toBe(true)
    })
  })

  describe('state cache', () => {
    it('getState should return cached value', () => {
      ipcBridge.stateCache.set('test:key', { value: 123 })
      expect(ipc.getState('test:key')).toEqual({ value: 123 })
    })

    it('getState should return undefined for missing key', () => {
      expect(ipc.getState('missing:key')).toBeUndefined()
    })
  })

  describe('debug mode', () => {
    it('setDebug should enable/disable logging', () => {
      ipc.setDebug(true)
      expect(ipcBridge.debug).toBe(true)

      ipc.setDebug(false)
      expect(ipcBridge.debug).toBe(false)
    })
  })

  describe('mock responses', () => {
    beforeEach(() => {
      ipcBridge.isConnected = false
    })

    afterEach(() => {
      ipcBridge.isConnected = true
    })

    it('should return mock version', async () => {
      const response = await ipc.emit(Events.APP_VERSION)
      expect(response.data.version).toBe('0.1.0-dev')
    })

    it('should return mock settings', async () => {
      const response = await ipc.emit(Events.SETTINGS_GET)
      expect(response.data.language).toBe('en')
      expect(response.data.theme).toBe('dark')
    })

    it('should return error for unknown event', async () => {
      const response = await ipc.emit('unknown:event')
      expect(response.success).toBe(false)
      expect(response.error.code).toBe('UNKNOWN_EVENT')
    })
  })

  describe('_handlePushEvent', () => {
    it('should parse JSON data and notify listeners', () => {
      const callback = vi.fn()
      ipc.on(Events.NODE_STATE_CHANGED, callback)

      ipcBridge._handlePushEvent(Events.NODE_STATE_CHANGED, JSON.stringify({
        data: { state: 'running' }
      }))

      expect(callback).toHaveBeenCalledWith({ state: 'running' })
    })

    it('should update state cache on STATE_CHANGED', () => {
      ipcBridge._handlePushEvent(Events.STATE_CHANGED, JSON.stringify({
        data: { key: 'test', value: 'updated' }
      }))

      expect(ipcBridge.stateCache.get('test')).toBe('updated')
    })

    it('should update state cache on STATE_SYNC', () => {
      ipcBridge._handlePushEvent(Events.STATE_SYNC, JSON.stringify({
        data: { key1: 'value1', key2: 'value2' }
      }))

      expect(ipcBridge.stateCache.get('key1')).toBe('value1')
      expect(ipcBridge.stateCache.get('key2')).toBe('value2')
    })
  })
})
