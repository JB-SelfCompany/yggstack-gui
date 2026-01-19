/**
 * IPC Batching and Delta Sync tests
 */

import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { batcher, deltaSync, deduplicator } from './ipc-batch'
import { ipc } from './ipc'

// Mock metrics
vi.mock('./metrics', () => ({
  metrics: {
    recordIPCLatency: vi.fn()
  }
}))

describe('IPCBatcher', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    batcher.pendingBatch = []
    batcher.processing = false
    batcher.batchTimeout = null
  })

  afterEach(() => {
    vi.useRealTimers()
    batcher.clear()
  })

  describe('queue', () => {
    it('should add request to pending batch', () => {
      batcher.queue('test:event', { data: 1 })
      expect(batcher.pendingBatch).toHaveLength(1)
    })

    it('should schedule flush after batchWindow', () => {
      vi.spyOn(ipc, 'emit').mockResolvedValue({ success: true, data: {} })

      batcher.queue('test:event', { data: 1 })

      expect(batcher.batchTimeout).not.toBeNull()
    })

    it('should flush immediately when batch is full', async () => {
      vi.spyOn(ipc, 'emit').mockResolvedValue({ success: true, data: {} })

      // Queue up to maxBatchSize
      const promises = []
      for (let i = 0; i < batcher.maxBatchSize; i++) {
        promises.push(batcher.queue('test:event', { data: i }))
      }

      // Batch should be flushed immediately
      expect(batcher.pendingBatch.length).toBeLessThan(batcher.maxBatchSize)
    })
  })

  describe('_flush', () => {
    it('should send single request directly', async () => {
      const emitSpy = vi.spyOn(ipc, 'emit').mockResolvedValue({
        success: true,
        data: { result: 'ok' }
      })

      const promise = batcher.queue('test:single', { data: 'value' })
      vi.advanceTimersByTime(20)
      await promise

      expect(emitSpy).toHaveBeenCalledWith('test:single', { data: 'value' })
    })

    it('should batch multiple requests of same event', async () => {
      const emitSpy = vi.spyOn(ipc, 'emit').mockResolvedValue({
        success: true,
        data: ['result1', 'result2']
      })

      const p1 = batcher.queue('test:batch', { id: 1 })
      const p2 = batcher.queue('test:batch', { id: 2 })

      vi.advanceTimersByTime(20)
      await Promise.all([p1, p2])

      expect(emitSpy).toHaveBeenCalledWith('test:batch:batch', {
        items: [{ id: 1 }, { id: 2 }]
      })
    })

    it('should reject all on error', async () => {
      vi.spyOn(ipc, 'emit').mockRejectedValue(new Error('Network error'))

      const promise = batcher.queue('test:error', {})
      vi.advanceTimersByTime(20)

      await expect(promise).rejects.toThrow('Network error')
    })
  })

  describe('clear', () => {
    it('should reject pending requests and clear batch', () => {
      const promise = batcher.queue('test:clear', {})
      batcher.clear()

      expect(promise).rejects.toThrow('Batch cleared')
      expect(batcher.pendingBatch).toHaveLength(0)
    })
  })
})

describe('DeltaSyncManager', () => {
  beforeEach(() => {
    deltaSync.clear()
  })

  describe('state management', () => {
    it('getState should return cached value', () => {
      deltaSync.setState('key1', { value: 123 })
      expect(deltaSync.getState('key1')).toEqual({ value: 123 })
    })

    it('setState should notify subscribers', () => {
      const callback = vi.fn()
      deltaSync.subscribe('myKey', callback)

      deltaSync.setState('myKey', { data: 'test' })

      expect(callback).toHaveBeenCalledWith({ data: 'test' }, null)
    })
  })

  describe('applyDelta', () => {
    it('should merge delta into existing state', () => {
      deltaSync.setState('state', { a: 1, b: 2 })

      const result = deltaSync.applyDelta('state', { b: 3, c: 4 })

      expect(result).toEqual({ a: 1, b: 3, c: 4 })
      expect(deltaSync.getState('state')).toEqual({ a: 1, b: 3, c: 4 })
    })

    it('should handle nested objects', () => {
      deltaSync.setState('nested', { outer: { inner: 1, other: 2 } })

      deltaSync.applyDelta('nested', { outer: { inner: 10 } })

      expect(deltaSync.getState('nested')).toEqual({
        outer: { inner: 10, other: 2 }
      })
    })

    it('should remove keys with undefined value', () => {
      deltaSync.setState('remove', { keep: 1, remove: 2 })

      deltaSync.applyDelta('remove', { remove: undefined })

      expect(deltaSync.getState('remove')).toEqual({ keep: 1 })
    })

    it('should notify subscribers with delta', () => {
      const callback = vi.fn()
      deltaSync.setState('notify', { a: 1 })
      deltaSync.subscribe('notify', callback)

      deltaSync.applyDelta('notify', { b: 2 })

      expect(callback).toHaveBeenCalledWith(
        { a: 1, b: 2 },
        { b: 2 }
      )
    })
  })

  describe('computeDelta', () => {
    it('should return null for identical states', () => {
      const state = { a: 1, b: 2 }
      expect(deltaSync.computeDelta(state, state)).toBeNull()
    })

    it('should detect added keys', () => {
      const delta = deltaSync.computeDelta({ a: 1 }, { a: 1, b: 2 })
      expect(delta).toEqual({ b: 2 })
    })

    it('should detect changed values', () => {
      const delta = deltaSync.computeDelta({ a: 1 }, { a: 2 })
      expect(delta).toEqual({ a: 2 })
    })

    it('should detect removed keys', () => {
      const delta = deltaSync.computeDelta({ a: 1, b: 2 }, { a: 1 })
      expect(delta).toEqual({ b: undefined })
    })

    it('should handle nested changes', () => {
      const old = { outer: { a: 1, b: 2 } }
      const updated = { outer: { a: 1, b: 3 } }

      const delta = deltaSync.computeDelta(old, updated)

      expect(delta).toEqual({ outer: { b: 3 } })
    })

    it('should return new state if old is null', () => {
      const delta = deltaSync.computeDelta(null, { a: 1 })
      expect(delta).toEqual({ a: 1 })
    })
  })

  describe('subscriptions', () => {
    it('should support wildcard subscriptions', () => {
      const callback = vi.fn()
      deltaSync.subscribe('*', callback)

      deltaSync.setState('any:key', { data: 'value' })

      expect(callback).toHaveBeenCalledWith('any:key', { data: 'value' }, null)
    })

    it('unsubscribe should remove callback', () => {
      const callback = vi.fn()
      const unsubscribe = deltaSync.subscribe('key', callback)

      unsubscribe()
      deltaSync.setState('key', { data: 'value' })

      expect(callback).not.toHaveBeenCalled()
    })
  })
})

describe('RequestDeduplicator', () => {
  beforeEach(() => {
    deduplicator.pending.clear()
    deduplicator.cache.clear()
  })

  describe('execute', () => {
    it('should execute function and cache result', async () => {
      const fn = vi.fn().mockResolvedValue({ data: 'result' })

      const result = await deduplicator.execute('key1', fn)

      expect(result).toEqual({ data: 'result' })
      expect(fn).toHaveBeenCalledTimes(1)
    })

    it('should return cached result within TTL', async () => {
      const fn = vi.fn().mockResolvedValue({ data: 'result' })

      await deduplicator.execute('cached', fn)
      const result = await deduplicator.execute('cached', fn)

      expect(result).toEqual({ data: 'result' })
      expect(fn).toHaveBeenCalledTimes(1)
    })

    it('should deduplicate concurrent requests', async () => {
      let resolvePromise
      const fn = vi.fn().mockReturnValue(new Promise(r => { resolvePromise = r }))

      const p1 = deduplicator.execute('concurrent', fn)
      const p2 = deduplicator.execute('concurrent', fn)

      resolvePromise({ data: 'shared' })

      const [r1, r2] = await Promise.all([p1, p2])

      expect(r1).toEqual({ data: 'shared' })
      expect(r2).toEqual({ data: 'shared' })
      expect(fn).toHaveBeenCalledTimes(1)
    })

    it('should skip cache when useCache is false', async () => {
      const fn = vi.fn().mockResolvedValue({ data: 'fresh' })

      await deduplicator.execute('nocache', fn, { useCache: false })
      await deduplicator.execute('nocache', fn, { useCache: false })

      expect(fn).toHaveBeenCalledTimes(2)
    })

    it('should refetch when TTL expired', async () => {
      vi.useFakeTimers()
      const fn = vi.fn().mockResolvedValue({ data: 'result' })

      await deduplicator.execute('ttl', fn, { ttl: 100 })

      vi.advanceTimersByTime(150)

      await deduplicator.execute('ttl', fn, { ttl: 100 })

      expect(fn).toHaveBeenCalledTimes(2)
      vi.useRealTimers()
    })
  })

  describe('invalidate', () => {
    it('should remove cached entry', async () => {
      const fn = vi.fn().mockResolvedValue({ data: 'result' })

      await deduplicator.execute('invalidate', fn)
      deduplicator.invalidate('invalidate')
      await deduplicator.execute('invalidate', fn)

      expect(fn).toHaveBeenCalledTimes(2)
    })
  })

  describe('clearCache', () => {
    it('should clear all cached entries', async () => {
      const fn = vi.fn().mockResolvedValue({ data: 'result' })

      await deduplicator.execute('key1', fn)
      await deduplicator.execute('key2', fn)

      deduplicator.clearCache()

      expect(deduplicator.cache.size).toBe(0)
    })
  })
})
