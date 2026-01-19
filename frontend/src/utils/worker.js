/**
 * Web Worker wrapper for Vue components
 * Provides promise-based API for worker operations
 */

class WorkerPool {
  constructor(workerFactory, poolSize = navigator.hardwareConcurrency || 4) {
    this.workerFactory = workerFactory
    this.poolSize = Math.min(poolSize, 4) // Cap at 4 workers
    this.workers = []
    this.taskQueue = []
    this.requestId = 0
    this.pendingRequests = new Map()
    this.initialized = false
  }

  /**
   * Initialize workers lazily
   */
  _init() {
    if (this.initialized) return

    for (let i = 0; i < this.poolSize; i++) {
      const worker = this.workerFactory()
      worker.busy = false
      worker.onmessage = (e) => this._handleMessage(worker, e)
      worker.onerror = (e) => this._handleError(worker, e)
      this.workers.push(worker)
    }

    this.initialized = true
  }

  /**
   * Handle worker message
   */
  _handleMessage(worker, e) {
    const { id, success, result, error, duration } = e.data
    const pending = this.pendingRequests.get(id)

    if (pending) {
      this.pendingRequests.delete(id)
      if (success) {
        pending.resolve({ result, duration })
      } else {
        pending.reject(new Error(error))
      }
    }

    worker.busy = false
    this._processQueue()
  }

  /**
   * Handle worker error
   */
  _handleError(worker, e) {
    console.error('Worker error:', e)
    worker.busy = false
    this._processQueue()
  }

  /**
   * Process task queue
   */
  _processQueue() {
    if (this.taskQueue.length === 0) return

    const availableWorker = this.workers.find(w => !w.busy)
    if (!availableWorker) return

    const task = this.taskQueue.shift()
    availableWorker.busy = true
    availableWorker.postMessage(task.message)
  }

  /**
   * Execute operation on worker
   * @param {string} type - Operation type
   * @param {object} data - Operation data
   * @param {object} options - Options
   * @returns {Promise<object>} Result with duration
   */
  execute(type, data, options = {}) {
    this._init()

    const { timeout = 30000, priority = 'normal' } = options
    const id = String(++this.requestId)

    return new Promise((resolve, reject) => {
      const message = { id, type, data }

      // Set timeout
      const timeoutId = setTimeout(() => {
        this.pendingRequests.delete(id)
        reject(new Error(`Worker operation timed out after ${timeout}ms`))
      }, timeout)

      this.pendingRequests.set(id, {
        resolve: (result) => {
          clearTimeout(timeoutId)
          resolve(result)
        },
        reject: (error) => {
          clearTimeout(timeoutId)
          reject(error)
        }
      })

      // Add to queue
      const task = { message, priority }
      if (priority === 'high') {
        this.taskQueue.unshift(task)
      } else {
        this.taskQueue.push(task)
      }

      this._processQueue()
    })
  }

  /**
   * Terminate all workers
   */
  terminate() {
    this.workers.forEach(w => w.terminate())
    this.workers = []
    this.initialized = false
    this.pendingRequests.clear()
    this.taskQueue = []
  }
}

// Data worker pool singleton
let dataWorkerPool = null

/**
 * Get data worker pool (lazy initialization)
 * @returns {WorkerPool}
 */
export function getDataWorkerPool() {
  if (!dataWorkerPool) {
    dataWorkerPool = new WorkerPool(() =>
      new Worker(new URL('../workers/data.worker.js', import.meta.url), {
        type: 'module'
      })
    )
  }
  return dataWorkerPool
}

/**
 * Execute data operation on worker
 * Falls back to main thread if workers unavailable
 * @param {string} type - Operation type
 * @param {object} data - Operation data
 * @param {object} options - Options
 * @returns {Promise<any>} Operation result
 */
export async function executeOnWorker(type, data, options = {}) {
  // For small datasets, run on main thread
  const itemCount = data.items?.length || 0
  const threshold = options.threshold || 1000

  if (itemCount < threshold) {
    return executeOnMainThread(type, data)
  }

  try {
    const pool = getDataWorkerPool()
    const { result } = await pool.execute(type, data, options)
    return result
  } catch (error) {
    console.warn('Worker execution failed, falling back to main thread:', error)
    return executeOnMainThread(type, data)
  }
}

/**
 * Execute operation on main thread (fallback)
 */
function executeOnMainThread(type, data) {
  switch (type) {
    case 'sort':
      return sortOnMainThread(data)
    case 'filter':
      return filterOnMainThread(data)
    case 'search':
      return searchOnMainThread(data)
    case 'paginate':
      return paginateOnMainThread(data)
    default:
      throw new Error(`Unknown operation type: ${type}`)
  }
}

function sortOnMainThread({ items, field, direction = 'asc' }) {
  const multiplier = direction === 'desc' ? -1 : 1
  return [...items].sort((a, b) => {
    const valA = a[field]
    const valB = b[field]
    if (valA == null && valB == null) return 0
    if (valA == null) return multiplier
    if (valB == null) return -multiplier
    if (typeof valA === 'string') return multiplier * valA.localeCompare(valB)
    return multiplier * (valA - valB)
  })
}

function filterOnMainThread({ items, predicates }) {
  return items.filter(item =>
    predicates.every(p => {
      const val = item[p.field]
      switch (p.operator) {
        case 'eq': return val === p.value
        case 'contains': return String(val).toLowerCase().includes(String(p.value).toLowerCase())
        default: return true
      }
    })
  )
}

function searchOnMainThread({ items, query, fields }) {
  if (!query) return items
  const q = query.toLowerCase()
  return items.filter(item =>
    fields.some(f => String(item[f] || '').toLowerCase().includes(q))
  )
}

function paginateOnMainThread({ items, page = 1, pageSize = 50 }) {
  const offset = (page - 1) * pageSize
  return {
    items: items.slice(offset, offset + pageSize),
    page,
    pageSize,
    totalItems: items.length,
    totalPages: Math.ceil(items.length / pageSize),
    hasNext: page * pageSize < items.length,
    hasPrev: page > 1
  }
}

// Cleanup on page unload
if (typeof window !== 'undefined') {
  window.addEventListener('unload', () => {
    if (dataWorkerPool) {
      dataWorkerPool.terminate()
    }
  })
}

export default {
  getDataWorkerPool,
  executeOnWorker
}
