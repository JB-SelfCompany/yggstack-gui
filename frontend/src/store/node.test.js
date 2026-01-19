/**
 * Node store tests
 */

import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useNodeStore } from './node'
import { ipcBridge } from '../utils/ipc'

describe('Node Store', () => {
  let store

  beforeEach(() => {
    setActivePinia(createPinia())
    store = useNodeStore()
  })

  describe('initial state', () => {
    it('should have correct initial values', () => {
      expect(store.status).toBe('stopped')
      expect(store.info).toBeNull()
      expect(store.error).toBeNull()
      expect(store.stats).toEqual({
        uptime: 0,
        rxBytes: 0,
        txBytes: 0,
        peerCount: 0,
        sessionCount: 0
      })
    })
  })

  describe('getters', () => {
    it('isRunning should return true when status is running', () => {
      expect(store.isRunning).toBe(false)
      store.status = 'running'
      expect(store.isRunning).toBe(true)
    })

    it('isStopped should return true when status is stopped', () => {
      expect(store.isStopped).toBe(true)
      store.status = 'running'
      expect(store.isStopped).toBe(false)
    })

    it('isLoading should return true during starting/stopping', () => {
      expect(store.isLoading).toBe(false)
      store.status = 'starting'
      expect(store.isLoading).toBe(true)
      store.status = 'stopping'
      expect(store.isLoading).toBe(true)
      store.status = 'running'
      expect(store.isLoading).toBe(false)
    })

    it('hasError should return true when error exists', () => {
      expect(store.hasError).toBe(false)
      store.error = 'Test error'
      expect(store.hasError).toBe(true)
    })

    it('displayAddress should return address or placeholder', () => {
      expect(store.displayAddress).toBe('-')
      store.info = { address: '200::1' }
      expect(store.displayAddress).toBe('200::1')
    })
  })

  describe('actions', () => {
    describe('start', () => {
      it('should not start if not stopped', async () => {
        store.status = 'running'
        await store.start()
        expect(store.status).toBe('running')
      })

      it('should set status to starting then running on success', async () => {
        await store.start()
        expect(store.status).toBe('running')
        expect(store.info).toBeTruthy()
      })

      it('should handle errors and reset to stopped', async () => {
        vi.spyOn(ipcBridge, 'emit').mockRejectedValueOnce(new Error('Start failed'))
        await store.start()
        expect(store.status).toBe('stopped')
        expect(store.error).toBe('Start failed')
      })
    })

    describe('stop', () => {
      it('should not stop if not running', async () => {
        store.status = 'stopped'
        await store.stop()
        expect(store.status).toBe('stopped')
      })

      it('should set status to stopping then stopped on success', async () => {
        store.status = 'running'
        store.info = { address: '200::1' }
        await store.stop()
        expect(store.status).toBe('stopped')
        expect(store.info).toBeNull()
      })

      it('should handle errors and reset to running', async () => {
        store.status = 'running'
        vi.spyOn(ipcBridge, 'emit').mockRejectedValueOnce(new Error('Stop failed'))
        await store.stop()
        expect(store.status).toBe('running')
        expect(store.error).toBe('Stop failed')
      })
    })

    describe('handleStateChange', () => {
      it('should update status from state change event', () => {
        store.handleStateChange({
          currentState: 'running',
          nodeInfo: {
            ipv6Address: '200::1',
            subnet: '300::/64',
            publicKey: 'test-key',
            coords: '[1,2,3]'
          }
        })
        expect(store.status).toBe('running')
        expect(store.info.address).toBe('200::1')
        expect(store.info.subnet).toBe('300::/64')
        expect(store.lastUpdated).toBeTruthy()
      })

      it('should handle error in state change', () => {
        store.handleStateChange({
          state: 'stopped',
          error: 'Connection failed'
        })
        expect(store.status).toBe('stopped')
        expect(store.error).toBe('Connection failed')
      })
    })

    describe('updateStats', () => {
      it('should merge new stats with existing', () => {
        store.stats = { uptime: 100, rxBytes: 500, txBytes: 0, peerCount: 0, sessionCount: 0 }
        store.updateStats({ txBytes: 200, peerCount: 5 })
        expect(store.stats.uptime).toBe(100)
        expect(store.stats.rxBytes).toBe(500)
        expect(store.stats.txBytes).toBe(200)
        expect(store.stats.peerCount).toBe(5)
      })
    })

    describe('clearError', () => {
      it('should clear error state', () => {
        store.error = 'Some error'
        store.clearError()
        expect(store.error).toBeNull()
      })
    })
  })
})
