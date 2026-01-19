/**
 * TypeScript type definitions for IPC communication
 */

// Node states
export type NodeState = 'stopped' | 'starting' | 'running' | 'stopping'

// IPC Response
export interface IPCResponse<T = any> {
  success: boolean
  data?: T
  error?: IPCErrorInfo
  requestId: string
  timestamp: number
}

// IPC Error
export interface IPCErrorInfo {
  code: string
  message: string
  details?: string
}

// IPC Request
export interface IPCRequest<T = any> {
  requestId: string
  payload: T
  timestamp?: number
}

// Push Event
export interface IPCEvent<T = any> {
  type: string
  data: T
  timestamp: number
}

// Node status
export interface NodeStatus {
  state: NodeState
  ipv6Address?: string
  subnet?: string
  publicKey?: string
  coords?: number[]
  uptime?: number
  peerCount?: number
}

// Node state change event
export interface NodeStateChange {
  previousState: NodeState
  currentState: NodeState
  nodeInfo?: NodeStatus
  error?: string
  timestamp: number
}

// Peer info
export interface PeerInfo {
  uri: string
  address: string
  publicKey: string
  connected: boolean
  inbound?: boolean
  latency?: number
  rxBytes?: number
  txBytes?: number
  uptime?: number
}

// Peer event
export interface PeerEvent {
  type: 'connected' | 'disconnected'
  peer: PeerInfo
  timestamp: number
}

// Session info
export interface SessionInfo {
  address: string
  publicKey: string
  rxBytes: number
  txBytes: number
  uptime: number
}

// App settings
export interface AppSettings {
  language: 'en' | 'ru'
  theme: 'light' | 'dark' | 'system'
  minimizeToTray: boolean
  startMinimized: boolean
  autostart: boolean
  logLevel: 'debug' | 'info' | 'warn' | 'error'
}

// Proxy config
export interface ProxyConfig {
  enabled: boolean
  listenAddress: string
  nameserver?: string
}

// Proxy status
export interface ProxyStatus {
  enabled: boolean
  listenAddress: string
  activeConnections: number
  totalConnections: number
  bytesIn: number
  bytesOut: number
}

// Port mapping types
export type MappingType = 'local-tcp' | 'remote-tcp' | 'local-udp' | 'remote-udp'

// Port mapping
export interface PortMapping {
  id: string
  type: MappingType
  source: string
  target: string
  enabled: boolean
  active?: boolean
  bytesIn?: number
  bytesOut?: number
}

// Network stats
export interface NetworkStats {
  totalRxBytes: number
  totalTxBytes: number
  rxBytesPerSec: number
  txBytesPerSec: number
  peerCount: number
  sessionCount: number
  uptime: number
  timestamp: number
}

// State change event
export interface StateChangeEvent {
  key: string
  value: any
  previous?: any
  timestamp: number
}

// Log entry
export interface LogEntry {
  level: 'debug' | 'info' | 'warn' | 'error'
  message: string
  fields?: Record<string, any>
  timestamp: string
}

// Notification
export interface Notification {
  id: number
  type: 'info' | 'success' | 'warning' | 'error'
  message: string
  timestamp: number
}

// Request payloads
export interface AddPeerPayload {
  uri: string
}

export interface RemovePeerPayload {
  uri: string
}

export interface AddMappingPayload {
  type: MappingType
  source: string
  target: string
  enabled: boolean
}

export interface RemoveMappingPayload {
  id: string
}

// IPC Bridge interface
export interface IIPCBridge {
  emit<T = any>(event: string, payload?: any, options?: { timeout?: number }): Promise<IPCResponse<T>>
  on(event: string, callback: (data: any) => void): () => void
  once(event: string, callback: (data: any) => void): void
  off(event: string, callback?: (data: any) => void): void
  ping(): Promise<boolean>
  getState(key: string): any
  requestStateSync(): Promise<void>
  setDebug(enabled: boolean): void
}

// Event constants type
export interface IPCEvents {
  APP_VERSION: string
  APP_READY: string
  APP_PING: string
  APP_QUIT: string
  NODE_START: string
  NODE_STOP: string
  NODE_STATUS: string
  NODE_STATE_CHANGED: string
  NODE_ERROR: string
  PEERS_LIST: string
  PEERS_ADD: string
  PEERS_REMOVE: string
  PEERS_UPDATE: string
  PEER_CONNECTED: string
  PEER_DISCONNECTED: string
  SESSIONS_LIST: string
  SESSIONS_STATS: string
  CONFIG_LOAD: string
  CONFIG_SAVE: string
  SETTINGS_GET: string
  SETTINGS_SET: string
  PROXY_CONFIG: string
  PROXY_STATUS: string
  PROXY_START: string
  PROXY_STOP: string
  MAPPING_LIST: string
  MAPPING_ADD: string
  MAPPING_REMOVE: string
  STATE_CHANGED: string
  STATE_SYNC: string
  STATS_UPDATE: string
}

// Declare module exports
declare module '@/utils/ipc' {
  export const Events: IPCEvents
  export class IPCError extends Error {
    code: string
    details: string | null
    constructor(code: string, message: string, details?: string | null)
  }
  export const ipcBridge: IIPCBridge
  export const ipc: IIPCBridge
  export default ipc
}
