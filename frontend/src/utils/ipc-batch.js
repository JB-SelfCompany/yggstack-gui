/**
 * IPC Batching and Delta Sync utilities
 * Optimizes IPC communication by batching requests and syncing only changes
 */

import { ipc, Events } from './ipc'
import { metrics } from './metrics'

/**
 * Batch multiple IPC requests into a single call
 */
class IPCBatcher {
  constructor(options = {}) {
    this.batchWindow = options.batchWindow || 16 // ~1 frame at 60fps
    this.maxBatchSize = options.maxBatchSize || 50
    this.pendingBatch = []
    this.batchTimeout = null
    this.processing = false
  }

  /**
   * Add request to batch queue
   * @param {string} event - Event name
   * @param {object} payload - Event payload
   * @returns {Promise<object>} Response
   */
  async queue(event, payload) {
    return new Promise((resolve, reject) => {
      this.pendingBatch.push({ event, payload, resolve, reject })

      // Immediate flush if batch is full
      if (this.pendingBatch.length >= this.maxBatchSize) {
        this._flush()
        return
      }

      // Schedule flush
      if (!this.batchTimeout) {
        this.batchTimeout = setTimeout(() => this._flush(), this.batchWindow)
      }
    })
  }

  /**
   * Flush pending batch
   */
  async _flush() {
    if (this.batchTimeout) {
      clearTimeout(this.batchTimeout)
      this.batchTimeout = null
    }

    if (this.pendingBatch.length === 0 || this.processing) return

    this.processing = true
    const batch = this.pendingBatch
    this.pendingBatch = []

    // Group by event type
    const grouped = new Map()
    batch.forEach(item => {
      if (!grouped.has(item.event)) {
        grouped.set(item.event, [])
      }
      grouped.get(item.event).push(item)
    })

    // Process each group
    for (const [event, items] of grouped) {
      try {
        const startTime = performance.now()

        // If single item, send directly
        if (items.length === 1) {
          const response = await ipc.emit(event, items[0].payload)
          items[0].resolve(response)
        } else {
          // Batch send
          const payloads = items.map(i => i.payload)
          const response = await ipc.emit(`${event}:batch`, { items: payloads })

          if (response.success && Array.isArray(response.data)) {
            items.forEach((item, idx) => {
              item.resolve({
                success: true,
                data: response.data[idx]
              })
            })
          } else {
            // Fallback: send individually
            for (const item of items) {
              const res = await ipc.emit(event, item.payload)
              item.resolve(res)
            }
          }
        }

        const latency = performance.now() - startTime
        metrics.recordIPCLatency(event, latency)
      } catch (error) {
        items.forEach(item => item.reject(error))
      }
    }

    this.processing = false

    // Process any new items that came in during processing
    if (this.pendingBatch.length > 0) {
      this._flush()
    }
  }

  /**
   * Clear pending batch
   */
  clear() {
    if (this.batchTimeout) {
      clearTimeout(this.batchTimeout)
      this.batchTimeout = null
    }
    this.pendingBatch.forEach(item =>
      item.reject(new Error('Batch cleared'))
    )
    this.pendingBatch = []
  }
}

/**
 * Delta sync manager for state synchronization
 * Only syncs changed fields instead of full state
 */
class DeltaSyncManager {
  constructor() {
    this.stateCache = new Map()
    this.subscribers = new Map()
    this.pendingUpdates = new Map()
    this.flushTimeout = null
  }

  /**
   * Get cached state for key
   * @param {string} key - State key
   * @returns {any}
   */
  getState(key) {
    return this.stateCache.get(key)
  }

  /**
   * Apply delta update to cached state
   * @param {string} key - State key
   * @param {object} delta - Delta changes
   * @returns {object} New state
   */
  applyDelta(key, delta) {
    const current = this.stateCache.get(key) || {}
    const updated = this._deepMerge(current, delta)
    this.stateCache.set(key, updated)
    this._notifySubscribers(key, updated, delta)
    return updated
  }

  /**
   * Set full state (replaces cached value)
   * @param {string} key - State key
   * @param {any} value - New value
   */
  setState(key, value) {
    this.stateCache.set(key, value)
    this._notifySubscribers(key, value, null)
  }

  /**
   * Compute delta between old and new state
   * @param {object} oldState - Previous state
   * @param {object} newState - New state
   * @returns {object|null} Delta or null if no changes
   */
  computeDelta(oldState, newState) {
    if (oldState === newState) return null
    if (!oldState || !newState) return newState

    const delta = {}
    let hasChanges = false

    // Check for changed/added keys
    for (const key of Object.keys(newState)) {
      const oldVal = oldState[key]
      const newVal = newState[key]

      if (oldVal !== newVal) {
        if (typeof oldVal === 'object' && typeof newVal === 'object' &&
            oldVal !== null && newVal !== null && !Array.isArray(newVal)) {
          const nestedDelta = this.computeDelta(oldVal, newVal)
          if (nestedDelta) {
            delta[key] = nestedDelta
            hasChanges = true
          }
        } else {
          delta[key] = newVal
          hasChanges = true
        }
      }
    }

    // Check for removed keys
    for (const key of Object.keys(oldState)) {
      if (!(key in newState)) {
        delta[key] = undefined
        hasChanges = true
      }
    }

    return hasChanges ? delta : null
  }

  /**
   * Queue update to be synced
   * @param {string} key - State key
   * @param {object} update - Update data
   */
  queueUpdate(key, update) {
    if (!this.pendingUpdates.has(key)) {
      this.pendingUpdates.set(key, {})
    }

    const pending = this.pendingUpdates.get(key)
    Object.assign(pending, update)

    if (!this.flushTimeout) {
      this.flushTimeout = setTimeout(() => this._flushUpdates(), 50)
    }
  }

  /**
   * Flush queued updates
   */
  async _flushUpdates() {
    this.flushTimeout = null

    if (this.pendingUpdates.size === 0) return

    const updates = Array.from(this.pendingUpdates.entries())
    this.pendingUpdates.clear()

    // Send batched updates to backend
    try {
      await ipc.emit(Events.STATE_SYNC, {
        updates: updates.map(([key, data]) => ({ key, data }))
      })
    } catch (error) {
      console.error('Failed to sync updates:', error)
      // Re-queue failed updates
      updates.forEach(([key, data]) => {
        this.queueUpdate(key, data)
      })
    }
  }

  /**
   * Subscribe to state changes
   * @param {string} key - State key (or '*' for all)
   * @param {Function} callback - Callback(newState, delta)
   * @returns {Function} Unsubscribe function
   */
  subscribe(key, callback) {
    if (!this.subscribers.has(key)) {
      this.subscribers.set(key, new Set())
    }
    this.subscribers.get(key).add(callback)

    return () => {
      const subs = this.subscribers.get(key)
      if (subs) {
        subs.delete(callback)
      }
    }
  }

  /**
   * Notify subscribers of state change
   */
  _notifySubscribers(key, state, delta) {
    // Specific key subscribers
    const subs = this.subscribers.get(key)
    if (subs) {
      subs.forEach(cb => {
        try {
          cb(state, delta)
        } catch (e) {
          console.error('Subscriber error:', e)
        }
      })
    }

    // Wildcard subscribers
    const wildcardSubs = this.subscribers.get('*')
    if (wildcardSubs) {
      wildcardSubs.forEach(cb => {
        try {
          cb(key, state, delta)
        } catch (e) {
          console.error('Wildcard subscriber error:', e)
        }
      })
    }
  }

  /**
   * Deep merge objects
   */
  _deepMerge(target, source) {
    if (!source) return target
    if (typeof source !== 'object') return source

    const result = { ...target }

    for (const key of Object.keys(source)) {
      const sourceVal = source[key]

      if (sourceVal === undefined) {
        delete result[key]
      } else if (typeof sourceVal === 'object' && sourceVal !== null &&
                 !Array.isArray(sourceVal)) {
        result[key] = this._deepMerge(result[key] || {}, sourceVal)
      } else {
        result[key] = sourceVal
      }
    }

    return result
  }

  /**
   * Clear all cached state
   */
  clear() {
    this.stateCache.clear()
    this.pendingUpdates.clear()
    if (this.flushTimeout) {
      clearTimeout(this.flushTimeout)
      this.flushTimeout = null
    }
  }
}

/**
 * Request deduplicator
 * Prevents duplicate concurrent requests for same data
 */
class RequestDeduplicator {
  constructor() {
    this.pending = new Map()
    this.cache = new Map()
    this.cacheTTL = 5000 // 5 second cache
  }

  /**
   * Execute request with deduplication
   * @param {string} key - Request key
   * @param {Function} fn - Request function
   * @param {object} options - Options
   * @returns {Promise<any>}
   */
  async execute(key, fn, options = {}) {
    const { useCache = true, ttl = this.cacheTTL } = options

    // Check cache
    if (useCache && this.cache.has(key)) {
      const cached = this.cache.get(key)
      if (Date.now() - cached.timestamp < ttl) {
        return cached.data
      }
      this.cache.delete(key)
    }

    // Check for pending request
    if (this.pending.has(key)) {
      return this.pending.get(key)
    }

    // Execute request
    const promise = fn().then(result => {
      this.pending.delete(key)
      if (useCache) {
        this.cache.set(key, { data: result, timestamp: Date.now() })
      }
      return result
    }).catch(error => {
      this.pending.delete(key)
      throw error
    })

    this.pending.set(key, promise)
    return promise
  }

  /**
   * Invalidate cache entry
   * @param {string} key - Cache key
   */
  invalidate(key) {
    this.cache.delete(key)
  }

  /**
   * Clear all cache
   */
  clearCache() {
    this.cache.clear()
  }
}

// Export singletons
export const batcher = new IPCBatcher()
export const deltaSync = new DeltaSyncManager()
export const deduplicator = new RequestDeduplicator()

// Convenience functions
export const batchRequest = (event, payload) => batcher.queue(event, payload)
export const dedupRequest = (key, fn, options) => deduplicator.execute(key, fn, options)

export default {
  batcher,
  deltaSync,
  deduplicator,
  batchRequest,
  dedupRequest
}
