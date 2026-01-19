/**
 * Memory leak detection and cleanup utilities
 * Helps identify and prevent memory leaks in Vue components
 */

/**
 * WeakRef-based cache that allows garbage collection
 * Use for caching large objects that can be recreated if needed
 */
export class WeakCache {
  constructor() {
    this.cache = new Map()
    this.finalizer = new FinalizationRegistry((key) => {
      this.cache.delete(key)
    })
  }

  /**
   * Get cached value
   * @param {string} key - Cache key
   * @returns {any|undefined}
   */
  get(key) {
    const ref = this.cache.get(key)
    if (ref) {
      const value = ref.deref()
      if (value !== undefined) {
        return value
      }
      this.cache.delete(key)
    }
    return undefined
  }

  /**
   * Set cached value (must be an object)
   * @param {string} key - Cache key
   * @param {object} value - Value to cache
   */
  set(key, value) {
    if (typeof value !== 'object' || value === null) {
      throw new Error('WeakCache can only store objects')
    }
    this.cache.set(key, new WeakRef(value))
    this.finalizer.register(value, key)
  }

  /**
   * Check if key exists and value hasn't been collected
   * @param {string} key - Cache key
   * @returns {boolean}
   */
  has(key) {
    return this.get(key) !== undefined
  }

  /**
   * Delete cached value
   * @param {string} key - Cache key
   */
  delete(key) {
    this.cache.delete(key)
  }

  /**
   * Clear all cached values
   */
  clear() {
    this.cache.clear()
  }

  /**
   * Get current cache size (may include collected entries)
   * @returns {number}
   */
  get size() {
    return this.cache.size
  }
}

/**
 * Resource tracker for cleanup management
 * Tracks subscriptions, timers, and other resources that need cleanup
 */
export class ResourceTracker {
  constructor(name = 'anonymous') {
    this.name = name
    this.subscriptions = new Set()
    this.timers = new Set()
    this.animationFrames = new Set()
    this.abortControllers = new Set()
    this.eventListeners = []
    this.cleanupFns = []
    this.isDisposed = false
  }

  /**
   * Track a subscription with unsubscribe function
   * @param {Function} unsubscribe - Unsubscribe function
   * @returns {Function} The unsubscribe function
   */
  trackSubscription(unsubscribe) {
    if (this.isDisposed) {
      unsubscribe()
      return unsubscribe
    }
    this.subscriptions.add(unsubscribe)
    return unsubscribe
  }

  /**
   * Track a setTimeout
   * @param {Function} fn - Callback function
   * @param {number} delay - Delay in ms
   * @returns {number} Timer ID
   */
  setTimeout(fn, delay) {
    if (this.isDisposed) return -1
    const id = setTimeout(() => {
      this.timers.delete(id)
      fn()
    }, delay)
    this.timers.add(id)
    return id
  }

  /**
   * Track a setInterval
   * @param {Function} fn - Callback function
   * @param {number} interval - Interval in ms
   * @returns {number} Timer ID
   */
  setInterval(fn, interval) {
    if (this.isDisposed) return -1
    const id = setInterval(fn, interval)
    this.timers.add(id)
    return id
  }

  /**
   * Track a requestAnimationFrame
   * @param {Function} fn - Callback function
   * @returns {number} RAF ID
   */
  requestAnimationFrame(fn) {
    if (this.isDisposed) return -1
    const id = requestAnimationFrame((time) => {
      this.animationFrames.delete(id)
      fn(time)
    })
    this.animationFrames.add(id)
    return id
  }

  /**
   * Create and track an AbortController
   * @returns {AbortController}
   */
  createAbortController() {
    if (this.isDisposed) {
      const controller = new AbortController()
      controller.abort()
      return controller
    }
    const controller = new AbortController()
    this.abortControllers.add(controller)
    return controller
  }

  /**
   * Track an event listener
   * @param {EventTarget} target - Event target
   * @param {string} event - Event name
   * @param {Function} handler - Event handler
   * @param {object} options - addEventListener options
   */
  addEventListener(target, event, handler, options) {
    if (this.isDisposed) return
    target.addEventListener(event, handler, options)
    this.eventListeners.push({ target, event, handler, options })
  }

  /**
   * Add a custom cleanup function
   * @param {Function} fn - Cleanup function
   */
  onCleanup(fn) {
    if (this.isDisposed) {
      fn()
      return
    }
    this.cleanupFns.push(fn)
  }

  /**
   * Clean up all tracked resources
   */
  dispose() {
    if (this.isDisposed) return
    this.isDisposed = true

    // Unsubscribe all subscriptions
    this.subscriptions.forEach(unsub => {
      try { unsub() } catch (e) { console.warn('Cleanup error:', e) }
    })
    this.subscriptions.clear()

    // Clear all timers
    this.timers.forEach(id => clearTimeout(id))
    this.timers.clear()

    // Cancel all animation frames
    this.animationFrames.forEach(id => cancelAnimationFrame(id))
    this.animationFrames.clear()

    // Abort all pending requests
    this.abortControllers.forEach(controller => controller.abort())
    this.abortControllers.clear()

    // Remove all event listeners
    this.eventListeners.forEach(({ target, event, handler, options }) => {
      try {
        target.removeEventListener(event, handler, options)
      } catch (e) {
        console.warn('Failed to remove event listener:', e)
      }
    })
    this.eventListeners = []

    // Run custom cleanup functions
    this.cleanupFns.forEach(fn => {
      try { fn() } catch (e) { console.warn('Custom cleanup error:', e) }
    })
    this.cleanupFns = []
  }
}

/**
 * Memory leak detector
 * Tracks component instances and detects potential leaks
 */
class MemoryLeakDetector {
  constructor() {
    this.instances = new Map()
    this.mountCounts = new Map()
    this.unmountCounts = new Map()
    this.warnings = []
    this.enabled = false
  }

  /**
   * Enable leak detection
   */
  enable() {
    this.enabled = true
  }

  /**
   * Disable leak detection
   */
  disable() {
    this.enabled = false
  }

  /**
   * Record component mount
   * @param {string} componentName - Component name
   * @param {object} instance - Component instance
   */
  recordMount(componentName, instance) {
    if (!this.enabled) return

    // Track instance with WeakRef
    if (!this.instances.has(componentName)) {
      this.instances.set(componentName, new Set())
    }
    this.instances.get(componentName).add(new WeakRef(instance))

    // Update mount count
    const mounts = (this.mountCounts.get(componentName) || 0) + 1
    this.mountCounts.set(componentName, mounts)
  }

  /**
   * Record component unmount
   * @param {string} componentName - Component name
   */
  recordUnmount(componentName) {
    if (!this.enabled) return

    const unmounts = (this.unmountCounts.get(componentName) || 0) + 1
    this.unmountCounts.set(componentName, unmounts)
  }

  /**
   * Check for potential leaks
   * @returns {Array} Array of leak warnings
   */
  checkForLeaks() {
    const leaks = []

    this.mountCounts.forEach((mounts, componentName) => {
      const unmounts = this.unmountCounts.get(componentName) || 0
      const diff = mounts - unmounts

      if (diff > 10) {
        leaks.push({
          component: componentName,
          mounts,
          unmounts,
          leaked: diff,
          severity: diff > 50 ? 'critical' : 'warning'
        })
      }
    })

    // Check for live instances that should have been collected
    this.instances.forEach((refs, componentName) => {
      let liveCount = 0
      const cleanRefs = new Set()

      refs.forEach(ref => {
        const instance = ref.deref()
        if (instance) {
          liveCount++
          cleanRefs.add(ref)
        }
      })

      this.instances.set(componentName, cleanRefs)

      const expectedLive = (this.mountCounts.get(componentName) || 0) -
                          (this.unmountCounts.get(componentName) || 0)

      if (liveCount > expectedLive + 5) {
        leaks.push({
          component: componentName,
          liveInstances: liveCount,
          expectedLive,
          severity: 'warning',
          message: 'More live instances than expected'
        })
      }
    })

    return leaks
  }

  /**
   * Get memory report
   * @returns {object}
   */
  getReport() {
    const components = {}

    this.mountCounts.forEach((mounts, name) => {
      const unmounts = this.unmountCounts.get(name) || 0
      const refs = this.instances.get(name) || new Set()
      let liveCount = 0

      refs.forEach(ref => {
        if (ref.deref()) liveCount++
      })

      components[name] = {
        mounts,
        unmounts,
        expectedLive: mounts - unmounts,
        actualLive: liveCount
      }
    })

    return {
      components,
      leaks: this.checkForLeaks(),
      memory: typeof performance !== 'undefined' && performance.memory ? {
        usedMB: Math.round(performance.memory.usedJSHeapSize / 1024 / 1024),
        totalMB: Math.round(performance.memory.totalJSHeapSize / 1024 / 1024)
      } : null
    }
  }

  /**
   * Reset all tracking data
   */
  reset() {
    this.instances.clear()
    this.mountCounts.clear()
    this.unmountCounts.clear()
    this.warnings = []
  }
}

// Singleton instance
export const leakDetector = new MemoryLeakDetector()

/**
 * Vue composable for resource management
 * Automatically cleans up on component unmount
 */
export function useResourceTracker(name) {
  const tracker = new ResourceTracker(name)

  // Auto-record for leak detection if enabled
  if (leakDetector.enabled) {
    leakDetector.recordMount(name, tracker)
  }

  // Return cleanup function for manual use
  const dispose = () => {
    if (leakDetector.enabled) {
      leakDetector.recordUnmount(name)
    }
    tracker.dispose()
  }

  return {
    tracker,
    dispose,
    // Convenience methods
    trackSubscription: (unsub) => tracker.trackSubscription(unsub),
    setTimeout: (fn, delay) => tracker.setTimeout(fn, delay),
    setInterval: (fn, interval) => tracker.setInterval(fn, interval),
    requestAnimationFrame: (fn) => tracker.requestAnimationFrame(fn),
    createAbortController: () => tracker.createAbortController(),
    addEventListener: (target, event, handler, options) =>
      tracker.addEventListener(target, event, handler, options),
    onCleanup: (fn) => tracker.onCleanup(fn)
  }
}

/**
 * Force garbage collection hint (not guaranteed)
 * Only works if --expose-gc flag is set
 */
export function hintGC() {
  if (typeof window !== 'undefined' && window.gc) {
    window.gc()
    return true
  }
  return false
}

/**
 * Get memory snapshot for comparison
 * @returns {object|null}
 */
export function getMemorySnapshot() {
  if (typeof performance !== 'undefined' && performance.memory) {
    return {
      timestamp: Date.now(),
      usedJSHeapSize: performance.memory.usedJSHeapSize,
      totalJSHeapSize: performance.memory.totalJSHeapSize,
      jsHeapSizeLimit: performance.memory.jsHeapSizeLimit
    }
  }
  return null
}

/**
 * Compare two memory snapshots
 * @param {object} before - Before snapshot
 * @param {object} after - After snapshot
 * @returns {object}
 */
export function compareMemorySnapshots(before, after) {
  if (!before || !after) return null

  const usedDiff = after.usedJSHeapSize - before.usedJSHeapSize
  const totalDiff = after.totalJSHeapSize - before.totalJSHeapSize

  return {
    timeDelta: after.timestamp - before.timestamp,
    usedDiffMB: Math.round(usedDiff / 1024 / 1024 * 100) / 100,
    totalDiffMB: Math.round(totalDiff / 1024 / 1024 * 100) / 100,
    usedDiffPercent: Math.round(usedDiff / before.usedJSHeapSize * 100),
    possibleLeak: usedDiff > 10 * 1024 * 1024 // More than 10MB increase
  }
}

export default {
  WeakCache,
  ResourceTracker,
  leakDetector,
  useResourceTracker,
  hintGC,
  getMemorySnapshot,
  compareMemorySnapshots
}
