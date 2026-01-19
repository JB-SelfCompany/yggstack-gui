/**
 * Performance metrics collection and monitoring
 * Tracks FPS, memory usage, IPC latency, and component render times
 */

class PerformanceMetrics {
  constructor() {
    this.enabled = false
    this.frameCount = 0
    this.lastFrameTime = 0
    this.fps = 60
    this.fpsHistory = []
    this.maxHistorySize = 60

    this.ipcLatencies = []
    this.renderTimes = new Map()
    this.componentMounts = new Map()

    this.memoryHistory = []
    this.listeners = new Set()

    this._rafId = null
    this._reportInterval = null
  }

  /**
   * Enable metrics collection
   */
  enable() {
    if (this.enabled) return
    this.enabled = true
    this._startFPSMonitor()
    this._startMemoryMonitor()
  }

  /**
   * Disable metrics collection
   */
  disable() {
    if (!this.enabled) return
    this.enabled = false
    this._stopFPSMonitor()
    this._stopMemoryMonitor()
  }

  /**
   * Start FPS monitoring
   */
  _startFPSMonitor() {
    const measureFPS = (time) => {
      if (!this.enabled) return

      this.frameCount++

      if (this.lastFrameTime !== 0) {
        const delta = time - this.lastFrameTime
        if (delta > 0) {
          const currentFPS = 1000 / delta
          // Smooth FPS with exponential moving average
          this.fps = this.fps * 0.9 + currentFPS * 0.1
        }
      }

      this.lastFrameTime = time

      // Record FPS every second
      if (this.frameCount % 60 === 0) {
        this.fpsHistory.push({
          timestamp: Date.now(),
          fps: Math.round(this.fps)
        })
        if (this.fpsHistory.length > this.maxHistorySize) {
          this.fpsHistory.shift()
        }
      }

      this._rafId = requestAnimationFrame(measureFPS)
    }

    this._rafId = requestAnimationFrame(measureFPS)
  }

  /**
   * Stop FPS monitoring
   */
  _stopFPSMonitor() {
    if (this._rafId) {
      cancelAnimationFrame(this._rafId)
      this._rafId = null
    }
  }

  /**
   * Start memory monitoring
   */
  _startMemoryMonitor() {
    this._reportInterval = setInterval(() => {
      if (!this.enabled) return

      const memory = this.getMemoryInfo()
      if (memory) {
        this.memoryHistory.push({
          timestamp: Date.now(),
          ...memory
        })
        if (this.memoryHistory.length > this.maxHistorySize) {
          this.memoryHistory.shift()
        }
      }

      // Notify listeners
      this._notifyListeners()
    }, 1000)
  }

  /**
   * Stop memory monitoring
   */
  _stopMemoryMonitor() {
    if (this._reportInterval) {
      clearInterval(this._reportInterval)
      this._reportInterval = null
    }
  }

  /**
   * Get current memory info
   * @returns {object|null} Memory info
   */
  getMemoryInfo() {
    if (typeof performance !== 'undefined' && performance.memory) {
      const { usedJSHeapSize, totalJSHeapSize, jsHeapSizeLimit } = performance.memory
      return {
        usedMB: Math.round(usedJSHeapSize / 1024 / 1024 * 100) / 100,
        totalMB: Math.round(totalJSHeapSize / 1024 / 1024 * 100) / 100,
        limitMB: Math.round(jsHeapSizeLimit / 1024 / 1024 * 100) / 100,
        usagePercent: Math.round(usedJSHeapSize / jsHeapSizeLimit * 100)
      }
    }
    return null
  }

  /**
   * Record IPC latency
   * @param {string} event - Event name
   * @param {number} latency - Latency in ms
   */
  recordIPCLatency(event, latency) {
    if (!this.enabled) return

    this.ipcLatencies.push({
      event,
      latency,
      timestamp: Date.now()
    })

    // Keep last 100 entries
    if (this.ipcLatencies.length > 100) {
      this.ipcLatencies.shift()
    }
  }

  /**
   * Start measuring component render time
   * @param {string} componentName - Component name
   * @returns {Function} End measurement function
   */
  startRenderMeasure(componentName) {
    if (!this.enabled) return () => {}

    const startTime = performance.now()

    return () => {
      const duration = performance.now() - startTime
      const times = this.renderTimes.get(componentName) || []
      times.push(duration)
      if (times.length > 50) times.shift()
      this.renderTimes.set(componentName, times)
    }
  }

  /**
   * Record component mount
   * @param {string} componentName - Component name
   */
  recordMount(componentName) {
    if (!this.enabled) return

    const count = (this.componentMounts.get(componentName) || 0) + 1
    this.componentMounts.set(componentName, count)
  }

  /**
   * Get average FPS
   * @returns {number}
   */
  getAverageFPS() {
    if (this.fpsHistory.length === 0) return 60
    const sum = this.fpsHistory.reduce((a, b) => a + b.fps, 0)
    return Math.round(sum / this.fpsHistory.length)
  }

  /**
   * Get average IPC latency
   * @param {string} event - Optional event filter
   * @returns {number}
   */
  getAverageIPCLatency(event = null) {
    let latencies = this.ipcLatencies
    if (event) {
      latencies = latencies.filter(l => l.event === event)
    }
    if (latencies.length === 0) return 0
    const sum = latencies.reduce((a, b) => a + b.latency, 0)
    return Math.round(sum / latencies.length * 100) / 100
  }

  /**
   * Get average render time for component
   * @param {string} componentName - Component name
   * @returns {number}
   */
  getAverageRenderTime(componentName) {
    const times = this.renderTimes.get(componentName) || []
    if (times.length === 0) return 0
    const sum = times.reduce((a, b) => a + b, 0)
    return Math.round(sum / times.length * 100) / 100
  }

  /**
   * Get comprehensive metrics report
   * @returns {object}
   */
  getReport() {
    const memory = this.getMemoryInfo()
    const componentStats = {}

    this.renderTimes.forEach((times, name) => {
      componentStats[name] = {
        avgRenderTime: this.getAverageRenderTime(name),
        mountCount: this.componentMounts.get(name) || 0,
        renderCount: times.length
      }
    })

    return {
      timestamp: Date.now(),
      fps: {
        current: Math.round(this.fps),
        average: this.getAverageFPS(),
        history: this.fpsHistory.slice(-10)
      },
      memory: memory ? {
        current: memory,
        history: this.memoryHistory.slice(-10)
      } : null,
      ipc: {
        averageLatency: this.getAverageIPCLatency(),
        recentLatencies: this.ipcLatencies.slice(-10)
      },
      components: componentStats
    }
  }

  /**
   * Add listener for metrics updates
   * @param {Function} callback
   * @returns {Function} Unsubscribe function
   */
  subscribe(callback) {
    this.listeners.add(callback)
    return () => this.listeners.delete(callback)
  }

  /**
   * Notify all listeners
   */
  _notifyListeners() {
    const report = this.getReport()
    this.listeners.forEach(callback => {
      try {
        callback(report)
      } catch (e) {
        console.error('Metrics listener error:', e)
      }
    })
  }

  /**
   * Check for performance issues
   * @returns {Array} Array of warnings
   */
  checkPerformance() {
    const warnings = []
    const report = this.getReport()

    // Check FPS
    if (report.fps.average < 30) {
      warnings.push({
        type: 'fps',
        severity: 'critical',
        message: `Low FPS detected: ${report.fps.average} fps (target: 60)`
      })
    } else if (report.fps.average < 50) {
      warnings.push({
        type: 'fps',
        severity: 'warning',
        message: `Below target FPS: ${report.fps.average} fps (target: 60)`
      })
    }

    // Check memory
    if (report.memory && report.memory.current.usagePercent > 80) {
      warnings.push({
        type: 'memory',
        severity: 'critical',
        message: `High memory usage: ${report.memory.current.usagePercent}%`
      })
    } else if (report.memory && report.memory.current.usagePercent > 60) {
      warnings.push({
        type: 'memory',
        severity: 'warning',
        message: `Elevated memory usage: ${report.memory.current.usagePercent}%`
      })
    }

    // Check IPC latency
    if (report.ipc.averageLatency > 100) {
      warnings.push({
        type: 'ipc',
        severity: 'warning',
        message: `High IPC latency: ${report.ipc.averageLatency}ms (target: <10ms)`
      })
    }

    // Check component render times
    Object.entries(report.components).forEach(([name, stats]) => {
      if (stats.avgRenderTime > 16) {
        warnings.push({
          type: 'render',
          severity: 'warning',
          message: `Slow component render: ${name} (${stats.avgRenderTime}ms)`
        })
      }
    })

    return warnings
  }

  /**
   * Reset all metrics
   */
  reset() {
    this.frameCount = 0
    this.lastFrameTime = 0
    this.fps = 60
    this.fpsHistory = []
    this.ipcLatencies = []
    this.renderTimes.clear()
    this.componentMounts.clear()
    this.memoryHistory = []
  }
}

// Singleton instance
export const metrics = new PerformanceMetrics()

/**
 * Vue directive for measuring render time
 * Usage: v-measure-render="'ComponentName'"
 */
export const measureRenderDirective = {
  created(el, binding) {
    el._endMeasure = metrics.startRenderMeasure(binding.value)
  },
  mounted(el, binding) {
    if (el._endMeasure) {
      el._endMeasure()
    }
    metrics.recordMount(binding.value)
  },
  updated(el, binding) {
    if (el._endMeasure) {
      el._endMeasure()
    }
    el._endMeasure = metrics.startRenderMeasure(binding.value)
  }
}

/**
 * Vue composable for accessing metrics
 */
export function useMetrics() {
  return {
    enable: () => metrics.enable(),
    disable: () => metrics.disable(),
    getReport: () => metrics.getReport(),
    checkPerformance: () => metrics.checkPerformance(),
    subscribe: (callback) => metrics.subscribe(callback),
    recordIPCLatency: (event, latency) => metrics.recordIPCLatency(event, latency),
    isEnabled: () => metrics.enabled
  }
}

export default metrics
