/**
 * Vitest setup file
 * Configures global mocks and test environment
 */

import { vi } from 'vitest'

// Mock window.matchMedia for theme detection
Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: vi.fn().mockImplementation(query => ({
    matches: query === '(prefers-color-scheme: dark)',
    media: query,
    onchange: null,
    addListener: vi.fn(),
    removeListener: vi.fn(),
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn()
  }))
})

// Mock Energy IPC bridge
const mockIpcListeners = new Map()

window.ipc = {
  emitSync: vi.fn((event, message) => {
    // Return mock response based on event
    const payload = JSON.parse(message)
    return JSON.stringify({
      success: true,
      data: getMockResponseData(event),
      requestId: payload.requestId,
      timestamp: Date.now()
    })
  }),
  on: vi.fn((event, callback) => {
    if (!mockIpcListeners.has(event)) {
      mockIpcListeners.set(event, new Set())
    }
    mockIpcListeners.get(event).add(callback)
    return () => {
      mockIpcListeners.get(event)?.delete(callback)
    }
  }),
  off: vi.fn((event, callback) => {
    mockIpcListeners.get(event)?.delete(callback)
  })
}

// Helper to get mock response data
function getMockResponseData(event) {
  const responses = {
    'app:version': { version: '0.1.0-test' },
    'app:ping': { pong: true, timestamp: Date.now() },
    'node:status': { state: 'stopped', peerCount: 0, uptime: 0 },
    'node:start': { ipv6Address: '200::1', subnet: '300::/64', publicKey: 'test-key' },
    'node:stop': {},
    'peers:list': [],
    'peers:add': { uri: 'test://peer' },
    'peers:remove': {},
    'settings:get': { language: 'en', theme: 'dark' },
    'settings:set': {}
  }
  return responses[event] || {}
}

// Helper to simulate IPC push events from backend
export function simulateIpcEvent(event, data) {
  const listeners = mockIpcListeners.get(event)
  if (listeners) {
    listeners.forEach(callback => {
      callback(JSON.stringify({ data }))
    })
  }
}

// Reset mocks between tests
beforeEach(() => {
  vi.clearAllMocks()
  mockIpcListeners.clear()
})

// Global test utilities
global.flushPromises = () => new Promise(resolve => setTimeout(resolve, 0))

// Console error suppression for expected errors in tests
const originalError = console.error
beforeAll(() => {
  console.error = (...args) => {
    if (
      typeof args[0] === 'string' &&
      (args[0].includes('Expected error') ||
       args[0].includes('Failed to fetch'))
    ) {
      return
    }
    originalError.apply(console, args)
  }
})

afterAll(() => {
  console.error = originalError
})
