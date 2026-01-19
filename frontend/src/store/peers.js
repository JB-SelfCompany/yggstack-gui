import { defineStore } from 'pinia'
import { ipc, Events, IPCError } from '../utils/ipc'

export const usePeersStore = defineStore('peers', {
  state: () => ({
    peers: [],
    loading: false,
    error: null,
    lastUpdated: null
  }),

  getters: {
    connectedPeers: (state) => state.peers.filter(p => p.connected),
    disconnectedPeers: (state) => state.peers.filter(p => !p.connected),
    peerCount: (state) => state.peers.length,
    connectedCount: (state) => state.peers.filter(p => p.connected).length,
    totalTraffic: (state) => {
      return state.peers.reduce((acc, p) => ({
        rx: acc.rx + (p.rxBytes || 0),
        tx: acc.tx + (p.txBytes || 0)
      }), { rx: 0, tx: 0 })
    }
  },

  actions: {
    // Initialize store and subscribe to events
    init() {
      // Subscribe to peer updates
      ipc.on(Events.PEERS_UPDATE, (data) => {
        if (Array.isArray(data)) {
          this.peers = data
        } else if (data.peers) {
          this.peers = data.peers
        }
        this.lastUpdated = Date.now()
      })

      // Subscribe to peer connected events
      ipc.on(Events.PEER_CONNECTED, (data) => {
        if (data.peer) {
          const index = this.peers.findIndex(p => p.uri === data.peer.uri)
          if (index !== -1) {
            this.peers[index] = { ...this.peers[index], ...data.peer, connected: true }
          } else {
            this.peers.push({ ...data.peer, connected: true })
          }
        }
      })

      // Subscribe to peer disconnected events
      ipc.on(Events.PEER_DISCONNECTED, (data) => {
        if (data.peer) {
          const index = this.peers.findIndex(p => p.uri === data.peer.uri)
          if (index !== -1) {
            this.peers[index].connected = false
          }
        }
      })

      // Fetch initial peers
      this.fetchPeers()
    },

    // Fetch peers from backend
    async fetchPeers() {
      this.loading = true
      this.error = null

      try {
        const response = await ipc.emit(Events.PEERS_LIST)
        if (response.success) {
          this.peers = response.data || []
          this.lastUpdated = Date.now()
        } else {
          this.error = response.error?.message || 'Failed to fetch peers'
        }
      } catch (err) {
        if (err instanceof IPCError) {
          this.error = err.message
        } else {
          this.error = err.message || 'Failed to fetch peers'
        }
      } finally {
        this.loading = false
      }
    },

    // Add a new peer
    async addPeer(uri) {
      this.error = null

      try {
        const response = await ipc.emit(Events.PEERS_ADD, { uri })
        if (response.success) {
          // Add to local list optimistically
          const exists = this.peers.some(p => p.uri === uri)
          if (!exists) {
            this.peers.push({
              uri,
              connected: false,
              address: '',
              publicKey: ''
            })
          }
          return true
        } else {
          throw new Error(response.error?.message || 'Failed to add peer')
        }
      } catch (err) {
        if (err instanceof IPCError) {
          this.error = err.message
        } else {
          this.error = err.message
        }
        throw err
      }
    },

    // Remove a peer
    async removePeer(uri) {
      this.error = null

      try {
        const response = await ipc.emit(Events.PEERS_REMOVE, { uri })
        if (response.success) {
          // Remove from local list
          const index = this.peers.findIndex(p => p.uri === uri)
          if (index !== -1) {
            this.peers.splice(index, 1)
          }
          return true
        } else {
          throw new Error(response.error?.message || 'Failed to remove peer')
        }
      } catch (err) {
        if (err instanceof IPCError) {
          this.error = err.message
        } else {
          this.error = err.message
        }
        throw err
      }
    },

    // Update a peer's info
    updatePeer(uri, data) {
      const peer = this.peers.find(p => p.uri === uri)
      if (peer) {
        Object.assign(peer, data)
      }
    },

    // Set all peers
    setPeers(peers) {
      this.peers = peers
      this.lastUpdated = Date.now()
    },

    // Clear error
    clearError() {
      this.error = null
    }
  }
})
