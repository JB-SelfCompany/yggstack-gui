/**
 * Peers store tests
 */

import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { usePeersStore } from './peers'
import { ipcBridge, IPCError } from '../utils/ipc'

describe('Peers Store', () => {
  let store

  beforeEach(() => {
    setActivePinia(createPinia())
    store = usePeersStore()
  })

  describe('initial state', () => {
    it('should have empty initial state', () => {
      expect(store.peers).toEqual([])
      expect(store.loading).toBe(false)
      expect(store.error).toBeNull()
      expect(store.lastUpdated).toBeNull()
    })
  })

  describe('getters', () => {
    const testPeers = [
      { uri: 'tcp://peer1:1234', connected: true, rxBytes: 100, txBytes: 50 },
      { uri: 'tcp://peer2:1234', connected: false, rxBytes: 200, txBytes: 100 },
      { uri: 'tcp://peer3:1234', connected: true, rxBytes: 300, txBytes: 150 }
    ]

    beforeEach(() => {
      store.peers = testPeers
    })

    it('connectedPeers should filter connected peers', () => {
      expect(store.connectedPeers).toHaveLength(2)
      expect(store.connectedPeers.every(p => p.connected)).toBe(true)
    })

    it('disconnectedPeers should filter disconnected peers', () => {
      expect(store.disconnectedPeers).toHaveLength(1)
      expect(store.disconnectedPeers[0].uri).toBe('tcp://peer2:1234')
    })

    it('peerCount should return total peers count', () => {
      expect(store.peerCount).toBe(3)
    })

    it('connectedCount should return connected peers count', () => {
      expect(store.connectedCount).toBe(2)
    })

    it('totalTraffic should calculate sum of rx/tx bytes', () => {
      expect(store.totalTraffic).toEqual({ rx: 600, tx: 300 })
    })

    it('totalTraffic should handle missing values', () => {
      store.peers = [{ uri: 'test', connected: true }]
      expect(store.totalTraffic).toEqual({ rx: 0, tx: 0 })
    })
  })

  describe('actions', () => {
    describe('fetchPeers', () => {
      it('should fetch and store peers', async () => {
        const mockPeers = [
          { uri: 'tcp://peer1:1234', connected: true },
          { uri: 'tcp://peer2:1234', connected: false }
        ]
        vi.spyOn(ipcBridge, 'emit').mockResolvedValueOnce({
          success: true,
          data: mockPeers
        })

        await store.fetchPeers()

        expect(store.peers).toEqual(mockPeers)
        expect(store.loading).toBe(false)
        expect(store.lastUpdated).toBeTruthy()
      })

      it('should handle fetch error', async () => {
        vi.spyOn(ipcBridge, 'emit').mockResolvedValueOnce({
          success: false,
          error: { message: 'Failed to fetch peers' }
        })

        await store.fetchPeers()

        expect(store.error).toBe('Failed to fetch peers')
        expect(store.loading).toBe(false)
      })

      it('should handle IPC error', async () => {
        vi.spyOn(ipcBridge, 'emit').mockRejectedValueOnce(
          new IPCError('TIMEOUT', 'Request timed out')
        )

        await store.fetchPeers()

        expect(store.error).toBe('Request timed out')
        expect(store.loading).toBe(false)
      })
    })

    describe('addPeer', () => {
      it('should add peer optimistically on success', async () => {
        vi.spyOn(ipcBridge, 'emit').mockResolvedValueOnce({ success: true })

        const result = await store.addPeer('tcp://newpeer:1234')

        expect(result).toBe(true)
        expect(store.peers).toHaveLength(1)
        expect(store.peers[0].uri).toBe('tcp://newpeer:1234')
        expect(store.peers[0].connected).toBe(false)
      })

      it('should not add duplicate peer', async () => {
        store.peers = [{ uri: 'tcp://existing:1234', connected: true }]
        vi.spyOn(ipcBridge, 'emit').mockResolvedValueOnce({ success: true })

        await store.addPeer('tcp://existing:1234')

        expect(store.peers).toHaveLength(1)
      })

      it('should handle add error and throw', async () => {
        vi.spyOn(ipcBridge, 'emit').mockResolvedValueOnce({
          success: false,
          error: { message: 'Invalid peer URI' }
        })

        await expect(store.addPeer('invalid')).rejects.toThrow('Invalid peer URI')
        expect(store.error).toBe('Invalid peer URI')
      })
    })

    describe('removePeer', () => {
      beforeEach(() => {
        store.peers = [
          { uri: 'tcp://peer1:1234', connected: true },
          { uri: 'tcp://peer2:1234', connected: false }
        ]
      })

      it('should remove peer from list on success', async () => {
        vi.spyOn(ipcBridge, 'emit').mockResolvedValueOnce({ success: true })

        const result = await store.removePeer('tcp://peer1:1234')

        expect(result).toBe(true)
        expect(store.peers).toHaveLength(1)
        expect(store.peers[0].uri).toBe('tcp://peer2:1234')
      })

      it('should handle remove error and throw', async () => {
        vi.spyOn(ipcBridge, 'emit').mockResolvedValueOnce({
          success: false,
          error: { message: 'Peer not found' }
        })

        await expect(store.removePeer('tcp://unknown:1234')).rejects.toThrow('Peer not found')
      })
    })

    describe('updatePeer', () => {
      it('should update existing peer data', () => {
        store.peers = [{ uri: 'tcp://peer:1234', connected: false }]

        store.updatePeer('tcp://peer:1234', { connected: true, latency: 50 })

        expect(store.peers[0].connected).toBe(true)
        expect(store.peers[0].latency).toBe(50)
      })

      it('should do nothing for unknown peer', () => {
        store.peers = [{ uri: 'tcp://peer:1234', connected: false }]

        store.updatePeer('tcp://unknown:1234', { connected: true })

        expect(store.peers[0].connected).toBe(false)
      })
    })

    describe('setPeers', () => {
      it('should replace all peers and update timestamp', () => {
        const newPeers = [
          { uri: 'tcp://new1:1234', connected: true },
          { uri: 'tcp://new2:1234', connected: true }
        ]

        store.setPeers(newPeers)

        expect(store.peers).toEqual(newPeers)
        expect(store.lastUpdated).toBeTruthy()
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
