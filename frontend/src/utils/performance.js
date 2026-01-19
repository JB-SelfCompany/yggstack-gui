/**
 * Performance utilities for optimizing Vue components
 * Includes debounce, throttle, memoization, and RAF scheduling
 */

/**
 * Debounce function - delays execution until after wait ms have elapsed
 * since the last time the debounced function was invoked
 * @param {Function} fn - Function to debounce
 * @param {number} wait - Milliseconds to wait
 * @param {object} options - Options: leading, trailing, maxWait
 * @returns {Function} Debounced function with cancel method
 */
export function debounce(fn, wait = 100, options = {}) {
  const { leading = false, trailing = true, maxWait } = options
  let timeoutId = null
  let lastCallTime = 0
  let lastInvokeTime = 0
  let lastArgs = null
  let lastThis = null
  let result = null

  const invokeFunc = (time) => {
    const args = lastArgs
    const thisArg = lastThis
    lastArgs = null
    lastThis = null
    lastInvokeTime = time
    result = fn.apply(thisArg, args)
    return result
  }

  const shouldInvoke = (time) => {
    const timeSinceLastCall = time - lastCallTime
    const timeSinceLastInvoke = time - lastInvokeTime

    return (
      lastCallTime === 0 ||
      timeSinceLastCall >= wait ||
      timeSinceLastCall < 0 ||
      (maxWait !== undefined && timeSinceLastInvoke >= maxWait)
    )
  }

  const trailingEdge = (time) => {
    timeoutId = null
    if (trailing && lastArgs) {
      return invokeFunc(time)
    }
    lastArgs = null
    lastThis = null
    return result
  }

  const timerExpired = () => {
    const time = Date.now()
    if (shouldInvoke(time)) {
      return trailingEdge(time)
    }
    const remaining = wait - (time - lastCallTime)
    timeoutId = setTimeout(timerExpired, remaining)
  }

  const debounced = function (...args) {
    const time = Date.now()
    const isInvoking = shouldInvoke(time)

    lastArgs = args
    lastThis = this
    lastCallTime = time

    if (isInvoking) {
      if (timeoutId === null) {
        lastInvokeTime = time
        timeoutId = setTimeout(timerExpired, wait)
        if (leading) {
          return invokeFunc(time)
        }
      }
      if (maxWait !== undefined) {
        timeoutId = setTimeout(timerExpired, wait)
        return invokeFunc(time)
      }
    }
    if (timeoutId === null) {
      timeoutId = setTimeout(timerExpired, wait)
    }
    return result
  }

  debounced.cancel = () => {
    if (timeoutId !== null) {
      clearTimeout(timeoutId)
      timeoutId = null
    }
    lastInvokeTime = 0
    lastArgs = null
    lastThis = null
    lastCallTime = 0
  }

  debounced.flush = () => {
    if (timeoutId === null) {
      return result
    }
    return trailingEdge(Date.now())
  }

  debounced.pending = () => timeoutId !== null

  return debounced
}

/**
 * Throttle function - ensures function is called at most once per wait ms
 * @param {Function} fn - Function to throttle
 * @param {number} wait - Minimum milliseconds between calls
 * @param {object} options - Options: leading, trailing
 * @returns {Function} Throttled function with cancel method
 */
export function throttle(fn, wait = 100, options = {}) {
  const { leading = true, trailing = true } = options
  return debounce(fn, wait, { leading, trailing, maxWait: wait })
}

/**
 * Request Animation Frame throttle - ensures function runs max once per frame
 * @param {Function} fn - Function to throttle
 * @returns {Function} RAF-throttled function with cancel method
 */
export function rafThrottle(fn) {
  let rafId = null
  let lastArgs = null

  const throttled = function (...args) {
    lastArgs = args

    if (rafId === null) {
      rafId = requestAnimationFrame(() => {
        rafId = null
        fn.apply(this, lastArgs)
      })
    }
  }

  throttled.cancel = () => {
    if (rafId !== null) {
      cancelAnimationFrame(rafId)
      rafId = null
    }
  }

  return throttled
}

/**
 * Memoize function results based on arguments
 * @param {Function} fn - Function to memoize
 * @param {object} options - Options: maxSize, keyFn, ttl
 * @returns {Function} Memoized function with cache controls
 */
export function memoize(fn, options = {}) {
  const {
    maxSize = 100,
    keyFn = (...args) => JSON.stringify(args),
    ttl = 0 // Time to live in ms, 0 = infinite
  } = options

  const cache = new Map()
  const timestamps = new Map()

  const memoized = function (...args) {
    const key = keyFn(...args)
    const now = Date.now()

    // Check if cached and not expired
    if (cache.has(key)) {
      if (ttl === 0 || (now - timestamps.get(key)) < ttl) {
        return cache.get(key)
      }
      // Expired, remove
      cache.delete(key)
      timestamps.delete(key)
    }

    // Compute result
    const result = fn.apply(this, args)

    // Enforce max size (LRU-like)
    if (cache.size >= maxSize) {
      const firstKey = cache.keys().next().value
      cache.delete(firstKey)
      timestamps.delete(firstKey)
    }

    // Store result
    cache.set(key, result)
    timestamps.set(key, now)

    return result
  }

  memoized.cache = cache
  memoized.clear = () => {
    cache.clear()
    timestamps.clear()
  }
  memoized.delete = (key) => {
    cache.delete(key)
    timestamps.delete(key)
  }
  memoized.has = (key) => cache.has(key)

  return memoized
}

/**
 * Create a shallow comparison function for objects
 * @param {object} objA - First object
 * @param {object} objB - Second object
 * @returns {boolean} True if shallow equal
 */
export function shallowEqual(objA, objB) {
  if (objA === objB) return true
  if (!objA || !objB) return false
  if (typeof objA !== 'object' || typeof objB !== 'object') return false

  const keysA = Object.keys(objA)
  const keysB = Object.keys(objB)

  if (keysA.length !== keysB.length) return false

  for (const key of keysA) {
    if (!Object.prototype.hasOwnProperty.call(objB, key)) return false
    if (objA[key] !== objB[key]) return false
  }

  return true
}

/**
 * Batch multiple updates into a single operation
 * @param {Function} fn - Update function
 * @param {number} wait - Batch window in ms
 * @returns {Function} Batching function
 */
export function batchUpdates(fn, wait = 16) {
  let updates = []
  let timeoutId = null

  const flush = () => {
    if (updates.length > 0) {
      const batch = updates
      updates = []
      fn(batch)
    }
    timeoutId = null
  }

  const batched = (update) => {
    updates.push(update)

    if (timeoutId === null) {
      timeoutId = setTimeout(flush, wait)
    }
  }

  batched.flush = () => {
    if (timeoutId !== null) {
      clearTimeout(timeoutId)
      flush()
    }
  }

  batched.cancel = () => {
    if (timeoutId !== null) {
      clearTimeout(timeoutId)
      timeoutId = null
      updates = []
    }
  }

  return batched
}

/**
 * Schedule work during idle time
 * @param {Function} fn - Work function
 * @param {object} options - Options: timeout
 * @returns {number} Handle for cancellation
 */
export function scheduleIdle(fn, options = {}) {
  const { timeout = 5000 } = options

  if (typeof requestIdleCallback !== 'undefined') {
    return requestIdleCallback(fn, { timeout })
  }

  // Fallback for browsers without requestIdleCallback
  return setTimeout(fn, 1)
}

/**
 * Cancel scheduled idle work
 * @param {number} handle - Handle from scheduleIdle
 */
export function cancelIdle(handle) {
  if (typeof cancelIdleCallback !== 'undefined') {
    cancelIdleCallback(handle)
  } else {
    clearTimeout(handle)
  }
}

/**
 * Create an async queue with concurrency control
 * @param {number} concurrency - Max concurrent operations
 * @returns {object} Queue with add method
 */
export function createAsyncQueue(concurrency = 3) {
  const queue = []
  let running = 0
  const results = new Map()

  const runNext = async () => {
    if (running >= concurrency || queue.length === 0) return

    const { id, fn, resolve, reject } = queue.shift()
    running++

    try {
      const result = await fn()
      results.set(id, { success: true, result })
      resolve(result)
    } catch (error) {
      results.set(id, { success: false, error })
      reject(error)
    } finally {
      running--
      runNext()
    }
  }

  let nextId = 0

  return {
    add(fn) {
      const id = nextId++
      return new Promise((resolve, reject) => {
        queue.push({ id, fn, resolve, reject })
        runNext()
      })
    },
    get pending() {
      return queue.length
    },
    get running() {
      return running
    },
    clear() {
      queue.length = 0
    }
  }
}

/**
 * Measure execution time of a function
 * @param {Function} fn - Function to measure
 * @param {string} label - Label for console output
 * @returns {Function} Wrapped function
 */
export function measureTime(fn, label = 'Function') {
  return function (...args) {
    const start = performance.now()
    const result = fn.apply(this, args)

    if (result instanceof Promise) {
      return result.finally(() => {
        const duration = performance.now() - start
        console.debug(`[Performance] ${label}: ${duration.toFixed(2)}ms`)
      })
    }

    const duration = performance.now() - start
    console.debug(`[Performance] ${label}: ${duration.toFixed(2)}ms`)
    return result
  }
}

/**
 * Create a deferred promise
 * @returns {object} Object with promise, resolve, reject
 */
export function createDeferred() {
  let resolve, reject
  const promise = new Promise((res, rej) => {
    resolve = res
    reject = rej
  })
  return { promise, resolve, reject }
}

/**
 * Wait for condition to be true
 * @param {Function} condition - Condition function
 * @param {object} options - Options: interval, timeout
 * @returns {Promise} Resolves when condition is true
 */
export function waitFor(condition, options = {}) {
  const { interval = 100, timeout = 10000 } = options

  return new Promise((resolve, reject) => {
    const startTime = Date.now()

    const check = () => {
      if (condition()) {
        resolve()
        return
      }

      if (Date.now() - startTime >= timeout) {
        reject(new Error('waitFor timeout'))
        return
      }

      setTimeout(check, interval)
    }

    check()
  })
}

export default {
  debounce,
  throttle,
  rafThrottle,
  memoize,
  shallowEqual,
  batchUpdates,
  scheduleIdle,
  cancelIdle,
  createAsyncQueue,
  measureTime,
  createDeferred,
  waitFor
}
