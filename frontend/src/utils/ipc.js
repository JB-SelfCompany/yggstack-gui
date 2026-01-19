/**
 * IPC Bridge for communication between Vue frontend and Go backend
 * Provides request/response pattern, event subscriptions, and state synchronization
 */

// Event names constants (mirroring Go backend)
export const Events = {
  // Application events
  APP_VERSION: 'app:version',
  APP_READY: 'app:ready',
  APP_PING: 'app:ping',
  APP_QUIT: 'app:quit',

  // Node events
  NODE_START: 'node:start',
  NODE_STOP: 'node:stop',
  NODE_STATUS: 'node:status',
  NODE_STATE_CHANGED: 'node:stateChanged',
  NODE_ERROR: 'node:error',

  // Peer events
  PEERS_LIST: 'peers:list',
  PEERS_ADD: 'peers:add',
  PEERS_REMOVE: 'peers:remove',
  PEERS_UPDATE: 'peers:update',
  PEER_CONNECTED: 'peer:connected',
  PEER_DISCONNECTED: 'peer:disconnected',

  // Session events
  SESSIONS_LIST: 'sessions:list',
  SESSIONS_STATS: 'sessions:stats',

  // Config events
  CONFIG_LOAD: 'config:load',
  CONFIG_SAVE: 'config:save',

  // Settings events
  SETTINGS_GET: 'settings:get',
  SETTINGS_SET: 'settings:set',

  // Proxy events
  PROXY_CONFIG: 'proxy:config',
  PROXY_STATUS: 'proxy:status',
  PROXY_START: 'proxy:start',
  PROXY_STOP: 'proxy:stop',

  // Mapping events
  MAPPING_LIST: 'mapping:list',
  MAPPING_ADD: 'mapping:add',
  MAPPING_REMOVE: 'mapping:remove',

  // State sync events
  STATE_CHANGED: 'state:changed',
  STATE_SYNC: 'state:sync',

  // Stats events
  STATS_UPDATE: 'stats:update',

  // Log events
  LOG_ENTRY: 'log:entry',
  LOG_LIST: 'log:list',
  LOG_CLEAR: 'log:clear'
}

/**
 * IPC Error class
 */
export class IPCError extends Error {
  constructor(code, message, details = null) {
    super(message)
    this.name = 'IPCError'
    this.code = code
    this.details = details
  }
}

/**
 * IPC Bridge class
 */
class IPCBridge {
  constructor() {
    this.handlers = new Map()
    this.pendingRequests = new Map()
    this.requestId = 0
    this.isConnected = false
    this.connectionPromise = null
    this.reconnectAttempts = 0
    this.maxReconnectAttempts = 5
    this.defaultTimeout = 30000
    this.eventListeners = new Map()
    this.stateCache = new Map()
    this.debug = false

    this._initialize()
  }

  /**
   * Initialize the IPC bridge
   */
  _initialize() {
    // Check for Energy IPC
    if (typeof window !== 'undefined' && window.ipc) {
      this.isConnected = true
      this._setupEnergyIPC()
      this._log('Connected to Energy IPC')
    } else {
      this._log('Energy IPC not available, using mock mode')
      this.isConnected = false
    }
  }

  /**
   * Setup Energy IPC event handlers
   */
  _setupEnergyIPC() {
    // Listen for push events from backend
    const pushEvents = [
      Events.NODE_STATE_CHANGED,
      Events.NODE_ERROR,
      Events.PEERS_UPDATE,
      Events.PEER_CONNECTED,
      Events.PEER_DISCONNECTED,
      Events.STATE_CHANGED,
      Events.STATE_SYNC,
      Events.STATS_UPDATE,
      Events.LOG_ENTRY
    ]

    pushEvents.forEach(event => {
      window.ipc.on(event, (data) => {
        this._handlePushEvent(event, data)
      })
    })
  }

  /**
   * Handle push events from backend
   */
  _handlePushEvent(event, data) {
    try {
      const parsed = typeof data === 'string' ? JSON.parse(data) : data
      this._log(`Received push event: ${event}`, parsed)

      // Update state cache if it's a state event
      if (event === Events.STATE_CHANGED && parsed.data) {
        this.stateCache.set(parsed.data.key, parsed.data.value)
      } else if (event === Events.STATE_SYNC && parsed.data) {
        Object.entries(parsed.data).forEach(([key, value]) => {
          this.stateCache.set(key, value)
        })
      }

      // Notify listeners
      this._notifyListeners(event, parsed.data || parsed)
    } catch (err) {
      this._log(`Error handling push event ${event}:`, err)
    }
  }

  /**
   * Notify all listeners for an event
   */
  _notifyListeners(event, data) {
    const listeners = this.eventListeners.get(event)
    if (listeners) {
      listeners.forEach(callback => {
        try {
          callback(data)
        } catch (err) {
          console.error(`Error in event listener for ${event}:`, err)
        }
      })
    }

    // Also notify wildcard listeners
    const wildcardListeners = this.eventListeners.get('*')
    if (wildcardListeners) {
      wildcardListeners.forEach(callback => {
        try {
          callback(event, data)
        } catch (err) {
          console.error('Error in wildcard event listener:', err)
        }
      })
    }
  }

  /**
   * Subscribe to an event
   * @param {string} event - Event name or '*' for all events
   * @param {function} callback - Callback function
   * @returns {function} Unsubscribe function
   */
  on(event, callback) {
    if (!this.eventListeners.has(event)) {
      this.eventListeners.set(event, new Set())
    }
    this.eventListeners.get(event).add(callback)

    // Return unsubscribe function
    return () => {
      const listeners = this.eventListeners.get(event)
      if (listeners) {
        listeners.delete(callback)
      }
    }
  }

  /**
   * Subscribe to an event once
   * @param {string} event - Event name
   * @param {function} callback - Callback function
   */
  once(event, callback) {
    const unsubscribe = this.on(event, (data) => {
      unsubscribe()
      callback(data)
    })
  }

  /**
   * Unsubscribe from an event
   * @param {string} event - Event name
   * @param {function} callback - Callback function (optional, removes all if not provided)
   */
  off(event, callback) {
    if (callback) {
      const listeners = this.eventListeners.get(event)
      if (listeners) {
        listeners.delete(callback)
      }
    } else {
      this.eventListeners.delete(event)
    }
  }

  /**
   * Emit an event to the backend
   * @param {string} event - Event name
   * @param {object} payload - Event payload
   * @param {object} options - Options (timeout, etc.)
   * @returns {Promise<object>} Response from backend
   */
  async emit(event, payload = {}, options = {}) {
    const requestId = String(++this.requestId)
    const timeout = options.timeout || this.defaultTimeout

    // If not connected, return mock response
    if (!this.isConnected) {
      return this._mockResponse(event, payload)
    }

    return new Promise((resolve, reject) => {
      // Set timeout
      const timeoutId = setTimeout(() => {
        this.pendingRequests.delete(requestId)
        reject(new IPCError('TIMEOUT', `Request timed out after ${timeout}ms`))
      }, timeout)

      // Store pending request
      this.pendingRequests.set(requestId, {
        resolve: (data) => {
          clearTimeout(timeoutId)
          resolve(data)
        },
        reject: (err) => {
          clearTimeout(timeoutId)
          reject(err)
        },
        event,
        timestamp: Date.now()
      })

      // Prepare message for backend
      const message = JSON.stringify({
        requestId,
        payload,
        timestamp: Date.now()
      })

      this._log(`Emitting: ${event}`, { requestId, payload })

      // Use Energy IPC with callback
      // Energy IPC format: ipc.emit(event, argument, callback)
      try {
        window.ipc.emit(event, message, (result) => {
          this._handleResponse(requestId, result)
        })
      } catch (err) {
        this.pendingRequests.delete(requestId)
        clearTimeout(timeoutId)
        reject(new IPCError('IPC_ERROR', err.message))
      }
    })
  }

  /**
   * Handle response from backend
   */
  _handleResponse(requestId, result) {
    const pending = this.pendingRequests.get(requestId)
    if (!pending) {
      this._log(`No pending request for ${requestId}`)
      return
    }

    this.pendingRequests.delete(requestId)

    try {
      const response = typeof result === 'string' ? JSON.parse(result) : result

      if (response.success) {
        pending.resolve(response)
      } else {
        const error = response.error || { code: 'UNKNOWN', message: 'Unknown error' }
        pending.reject(new IPCError(error.code, error.message, error.details))
      }
    } catch (err) {
      pending.reject(new IPCError('PARSE_ERROR', 'Failed to parse response'))
    }
  }

  /**
   * Send a ping to check connection
   * @returns {Promise<boolean>}
   */
  async ping() {
    try {
      const response = await this.emit(Events.APP_PING, {}, { timeout: 5000 })
      return response.success && response.data?.pong
    } catch {
      return false
    }
  }

  /**
   * Get cached state value
   * @param {string} key - State key
   * @returns {any} State value
   */
  getState(key) {
    return this.stateCache.get(key)
  }

  /**
   * Request full state sync from backend
   */
  async requestStateSync() {
    try {
      const response = await this.emit('state:request')
      if (response.success && response.data) {
        Object.entries(response.data).forEach(([key, value]) => {
          this.stateCache.set(key, value)
        })
      }
    } catch (err) {
      this._log('State sync failed:', err)
    }
  }

  /**
   * Enable/disable debug logging
   */
  setDebug(enabled) {
    this.debug = enabled
  }

  /**
   * Internal logging
   */
  _log(...args) {
    if (this.debug) {
      console.log('[IPC]', ...args)
    }
  }

  /**
   * Mock responses for development mode
   */
  _mockResponse(event, payload) {
    const mockResponses = {
      [Events.APP_VERSION]: { success: true, data: { version: 'dev-mock' } },
      [Events.APP_READY]: { success: true, data: { acknowledged: true } },
      [Events.APP_PING]: { success: true, data: { pong: true, timestamp: Date.now() } },
      [Events.NODE_START]: {
        success: false,
        error: { code: 'DEV_MODE', message: 'Running in development mode' }
      },
      [Events.NODE_STOP]: {
        success: false,
        error: { code: 'DEV_MODE', message: 'Running in development mode' }
      },
      [Events.NODE_STATUS]: {
        success: true,
        data: { state: 'stopped', peerCount: 0 }
      },
      [Events.PEERS_LIST]: { success: true, data: [] },
      [Events.PEERS_ADD]: { success: true, data: {} },
      [Events.PEERS_REMOVE]: { success: true, data: {} },
      [Events.CONFIG_LOAD]: { success: true, data: {} },
      [Events.CONFIG_SAVE]: { success: true, data: {} },
      [Events.SETTINGS_GET]: {
        success: true,
        data: { language: 'en', theme: 'dark' }
      },
      [Events.SETTINGS_SET]: { success: true, data: payload },
      [Events.PROXY_STATUS]: {
        success: true,
        data: { enabled: false, listenAddress: '127.0.0.1:1080' }
      }
    }

    const response = mockResponses[event] || {
      success: false,
      error: { code: 'UNKNOWN_EVENT', message: `Unknown event: ${event}` }
    }

    return Promise.resolve(response)
  }
}

// Create and export singleton instance
export const ipcBridge = new IPCBridge()

// Convenience methods
export const ipc = {
  emit: (event, payload, options) => ipcBridge.emit(event, payload, options),
  on: (event, callback) => ipcBridge.on(event, callback),
  once: (event, callback) => ipcBridge.once(event, callback),
  off: (event, callback) => ipcBridge.off(event, callback),
  ping: () => ipcBridge.ping(),
  getState: (key) => ipcBridge.getState(key),
  requestStateSync: () => ipcBridge.requestStateSync(),
  setDebug: (enabled) => ipcBridge.setDebug(enabled)
}

export default ipc
