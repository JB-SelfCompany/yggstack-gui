import { defineStore } from 'pinia'
import { ipc, Events, IPCError } from '../utils/ipc'

export const useNodeStore = defineStore('node', {
  state: () => ({
    status: 'stopped', // 'stopped' | 'starting' | 'running' | 'stopping'
    info: null,        // { address, subnet, publicKey, coords }
    stats: {
      uptime: 0,
      rxBytes: 0,
      txBytes: 0,
      peerCount: 0,
      sessionCount: 0
    },
    error: null,
    lastUpdated: null
  }),

  getters: {
    isRunning: (state) => state.status === 'running',
    isStopped: (state) => state.status === 'stopped',
    isLoading: (state) => ['starting', 'stopping'].includes(state.status),
    hasError: (state) => state.error !== null,
    displayAddress: (state) => {
      if (!state.info?.address) return '-'
      return state.info.address
    }
  },

  actions: {
    // Initialize store and subscribe to events
    init() {
      // Subscribe to node state changes
      ipc.on(Events.NODE_STATE_CHANGED, (data) => {
        this.handleStateChange(data)
      })

      // Subscribe to stats updates
      ipc.on(Events.STATS_UPDATE, (data) => {
        if (data.peerCount !== undefined) {
          this.stats.peerCount = data.peerCount
        }
        if (data.uptime !== undefined) {
          this.stats.uptime = data.uptime
        }
        if (data.sessionCount !== undefined) {
          this.stats.sessionCount = data.sessionCount
        }
      })

      // Subscribe to errors
      ipc.on(Events.NODE_ERROR, (data) => {
        this.error = data.message || 'Unknown error'
      })

      // Get initial status
      this.fetchStatus()
    },

    // Handle state change from backend
    handleStateChange(data) {
      this.status = data.currentState || data.state
      if (data.nodeInfo) {
        this.info = {
          address: data.nodeInfo.ipv6Address,
          subnet: data.nodeInfo.subnet,
          publicKey: data.nodeInfo.publicKey,
          coords: data.nodeInfo.coords
        }
      }
      if (data.error) {
        this.error = data.error
      }
      this.lastUpdated = Date.now()
    },

    // Fetch current status from backend
    async fetchStatus() {
      try {
        const response = await ipc.emit(Events.NODE_STATUS)
        if (response.success && response.data) {
          this.status = response.data.state || 'stopped'
          if (response.data.ipv6Address) {
            this.info = {
              address: response.data.ipv6Address,
              subnet: response.data.subnet,
              publicKey: response.data.publicKey,
              coords: response.data.coords
            }
          }
          // Handle uptime from top level
          if (response.data.uptime !== undefined) {
            this.stats.uptime = response.data.uptime
          }
          // Handle stats object from backend
          if (response.data.stats) {
            if (response.data.stats.peerCount !== undefined) {
              this.stats.peerCount = response.data.stats.peerCount
            }
            if (response.data.stats.sessionCount !== undefined) {
              this.stats.sessionCount = response.data.stats.sessionCount
            }
            if (response.data.stats.rxBytes !== undefined) {
              this.stats.rxBytes = response.data.stats.rxBytes
            }
            if (response.data.stats.txBytes !== undefined) {
              this.stats.txBytes = response.data.stats.txBytes
            }
          }
        }
      } catch (err) {
        console.error('Failed to fetch node status:', err)
      }
    },

    // Start the node
    async start() {
      if (this.status !== 'stopped') return

      this.status = 'starting'
      this.error = null

      try {
        const response = await ipc.emit(Events.NODE_START)
        if (response.success) {
          this.status = 'running'
          if (response.data) {
            this.info = {
              address: response.data.ipv6Address,
              subnet: response.data.subnet,
              publicKey: response.data.publicKey
            }
          }
        } else {
          this.status = 'stopped'
          this.error = response.error?.message || 'Failed to start node'
        }
      } catch (err) {
        this.status = 'stopped'
        if (err instanceof IPCError) {
          this.error = err.message
        } else {
          this.error = err.message || 'Failed to start node'
        }
      }
    },

    // Stop the node
    async stop() {
      if (this.status !== 'running') return

      this.status = 'stopping'
      this.error = null

      try {
        const response = await ipc.emit(Events.NODE_STOP)
        if (response.success) {
          this.status = 'stopped'
          this.info = null
        } else {
          this.status = 'running'
          this.error = response.error?.message || 'Failed to stop node'
        }
      } catch (err) {
        this.status = 'running'
        if (err instanceof IPCError) {
          this.error = err.message
        } else {
          this.error = err.message || 'Failed to stop node'
        }
      }
    },

    // Restart the node
    async restart() {
      await this.stop()
      // Small delay
      await new Promise(resolve => setTimeout(resolve, 500))
      await this.start()
    },

    // Clear error
    clearError() {
      this.error = null
    },

    // Update stats
    updateStats(stats) {
      this.stats = { ...this.stats, ...stats }
    }
  }
})
