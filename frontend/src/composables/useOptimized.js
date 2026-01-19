/**
 * Composables for optimized Vue component patterns
 * Combines performance utilities for easy use in components
 */

import { ref, computed, watch, onMounted, onUnmounted, shallowRef, triggerRef } from 'vue'
import { debounce, throttle, memoize, rafThrottle } from '../utils/performance'
import { executeOnWorker } from '../utils/worker'
import { useResourceTracker } from '../utils/memory'
import { metrics } from '../utils/metrics'

/**
 * Use debounced ref
 * Updates are debounced before triggering reactivity
 * @param {any} initialValue - Initial value
 * @param {number} wait - Debounce wait in ms
 * @returns {object} { value, immediate, pending }
 */
export function useDebouncedRef(initialValue, wait = 300) {
  const value = ref(initialValue)
  const pending = ref(false)

  const debouncedUpdate = debounce((newValue) => {
    value.value = newValue
    pending.value = false
  }, wait)

  return {
    get value() {
      return value.value
    },
    set value(newValue) {
      pending.value = true
      debouncedUpdate(newValue)
    },
    immediate(newValue) {
      debouncedUpdate.cancel()
      value.value = newValue
      pending.value = false
    },
    pending: computed(() => pending.value),
    cancel() {
      debouncedUpdate.cancel()
      pending.value = false
    }
  }
}

/**
 * Use throttled callback
 * @param {Function} fn - Function to throttle
 * @param {number} wait - Throttle wait in ms
 * @returns {Function} Throttled function
 */
export function useThrottledFn(fn, wait = 100) {
  const { tracker, dispose } = useResourceTracker('useThrottledFn')
  const throttled = throttle(fn, wait)

  onUnmounted(() => {
    throttled.cancel()
    dispose()
  })

  return throttled
}

/**
 * Use RAF-throttled callback (max once per frame)
 * @param {Function} fn - Function to throttle
 * @returns {Function} RAF-throttled function
 */
export function useRAFThrottle(fn) {
  const { dispose } = useResourceTracker('useRAFThrottle')
  const throttled = rafThrottle(fn)

  onUnmounted(() => {
    throttled.cancel()
    dispose()
  })

  return throttled
}

/**
 * Use memoized computed value
 * Caches computed results based on dependencies
 * @param {Function} getter - Getter function
 * @param {object} options - Memoize options
 * @returns {ComputedRef}
 */
export function useMemoizedComputed(getter, options = {}) {
  const memoizedGetter = memoize(getter, options)

  onUnmounted(() => {
    memoizedGetter.clear()
  })

  return computed(memoizedGetter)
}

/**
 * Use async data with caching and loading states
 * @param {Function} fetcher - Async data fetcher
 * @param {object} options - Options
 * @returns {object} { data, loading, error, refresh }
 */
export function useAsyncData(fetcher, options = {}) {
  const {
    immediate = true,
    cache = false,
    cacheKey = 'default',
    transform = (data) => data
  } = options

  const data = ref(null)
  const loading = ref(false)
  const error = ref(null)

  const { tracker, dispose } = useResourceTracker('useAsyncData')
  const cache$ = cache ? new Map() : null

  const execute = async (force = false) => {
    // Check cache
    if (cache$ && !force && cache$.has(cacheKey)) {
      data.value = cache$.get(cacheKey)
      return data.value
    }

    loading.value = true
    error.value = null

    try {
      const result = await fetcher()
      const transformed = transform(result)
      data.value = transformed

      if (cache$) {
        cache$.set(cacheKey, transformed)
      }

      return transformed
    } catch (e) {
      error.value = e
      throw e
    } finally {
      loading.value = false
    }
  }

  const refresh = () => execute(true)

  if (immediate) {
    onMounted(execute)
  }

  onUnmounted(dispose)

  return {
    data: computed(() => data.value),
    loading: computed(() => loading.value),
    error: computed(() => error.value),
    execute,
    refresh
  }
}

/**
 * Use filtered/sorted list with worker offloading
 * @param {Ref<Array>} items - Reactive items array
 * @param {object} options - Filter/sort options
 * @returns {object} { result, filter, sort, search }
 */
export function useOptimizedList(items, options = {}) {
  const {
    searchFields = [],
    defaultSort = null,
    workerThreshold = 500
  } = options

  const searchQuery = ref('')
  const sortField = ref(defaultSort?.field || null)
  const sortDirection = ref(defaultSort?.direction || 'asc')
  const filters = ref([])

  const result = shallowRef([])
  const processing = ref(false)

  const { tracker, dispose } = useResourceTracker('useOptimizedList')

  // Process list with optional worker offloading
  const processItems = async () => {
    const itemsArray = items.value
    if (!itemsArray || itemsArray.length === 0) {
      result.value = []
      return
    }

    processing.value = true
    const startTime = performance.now()

    try {
      let processed = [...itemsArray]

      // Search
      if (searchQuery.value && searchFields.length > 0) {
        if (processed.length > workerThreshold) {
          processed = await executeOnWorker('search', {
            items: processed,
            query: searchQuery.value,
            fields: searchFields
          })
        } else {
          const query = searchQuery.value.toLowerCase()
          processed = processed.filter(item =>
            searchFields.some(field =>
              String(item[field] || '').toLowerCase().includes(query)
            )
          )
        }
      }

      // Filter
      if (filters.value.length > 0) {
        if (processed.length > workerThreshold) {
          processed = await executeOnWorker('filter', {
            items: processed,
            predicates: filters.value
          })
        } else {
          processed = processed.filter(item =>
            filters.value.every(f => {
              const val = item[f.field]
              switch (f.operator) {
                case 'eq': return val === f.value
                case 'neq': return val !== f.value
                case 'contains': return String(val).includes(f.value)
                default: return true
              }
            })
          )
        }
      }

      // Sort
      if (sortField.value) {
        if (processed.length > workerThreshold) {
          processed = await executeOnWorker('sort', {
            items: processed,
            field: sortField.value,
            direction: sortDirection.value
          })
        } else {
          const mult = sortDirection.value === 'desc' ? -1 : 1
          processed.sort((a, b) => {
            const valA = a[sortField.value]
            const valB = b[sortField.value]
            if (valA == null) return mult
            if (valB == null) return -mult
            if (typeof valA === 'string') return mult * valA.localeCompare(valB)
            return mult * (valA - valB)
          })
        }
      }

      result.value = processed
      triggerRef(result)

      const duration = performance.now() - startTime
      metrics.recordIPCLatency('list-processing', duration)
    } catch (e) {
      console.error('List processing error:', e)
      result.value = items.value
    } finally {
      processing.value = false
    }
  }

  // Debounced search
  const debouncedProcess = debounce(processItems, 150)

  // Watch for changes
  watch(items, processItems, { immediate: true })
  watch([searchQuery, sortField, sortDirection, filters], () => {
    debouncedProcess()
  }, { deep: true })

  onUnmounted(() => {
    debouncedProcess.cancel()
    dispose()
  })

  return {
    result: computed(() => result.value),
    processing: computed(() => processing.value),
    // Search
    searchQuery,
    setSearch(query) {
      searchQuery.value = query
    },
    // Sort
    sortField,
    sortDirection,
    setSort(field, direction = 'asc') {
      sortField.value = field
      sortDirection.value = direction
    },
    toggleSort(field) {
      if (sortField.value === field) {
        sortDirection.value = sortDirection.value === 'asc' ? 'desc' : 'asc'
      } else {
        sortField.value = field
        sortDirection.value = 'asc'
      }
    },
    // Filters
    filters,
    addFilter(filter) {
      filters.value.push(filter)
    },
    removeFilter(field) {
      filters.value = filters.value.filter(f => f.field !== field)
    },
    clearFilters() {
      filters.value = []
    }
  }
}

/**
 * Use intersection observer for lazy loading
 * @param {object} options - IntersectionObserver options
 * @returns {object} { targetRef, isIntersecting }
 */
export function useIntersectionObserver(options = {}) {
  const targetRef = ref(null)
  const isIntersecting = ref(false)

  const { threshold = 0.1, rootMargin = '50px' } = options

  let observer = null

  onMounted(() => {
    observer = new IntersectionObserver((entries) => {
      isIntersecting.value = entries[0]?.isIntersecting || false
    }, { threshold, rootMargin })

    if (targetRef.value) {
      observer.observe(targetRef.value)
    }
  })

  onUnmounted(() => {
    if (observer) {
      observer.disconnect()
    }
  })

  // Watch for ref changes
  watch(targetRef, (newTarget, oldTarget) => {
    if (oldTarget && observer) {
      observer.unobserve(oldTarget)
    }
    if (newTarget && observer) {
      observer.observe(newTarget)
    }
  })

  return {
    targetRef,
    isIntersecting: computed(() => isIntersecting.value)
  }
}

/**
 * Use resize observer
 * @param {Ref} targetRef - Element ref
 * @returns {object} { width, height }
 */
export function useResizeObserver(targetRef) {
  const width = ref(0)
  const height = ref(0)

  let observer = null

  const updateSize = rafThrottle((entry) => {
    width.value = entry.contentRect.width
    height.value = entry.contentRect.height
  })

  onMounted(() => {
    if (typeof ResizeObserver !== 'undefined') {
      observer = new ResizeObserver((entries) => {
        if (entries[0]) {
          updateSize(entries[0])
        }
      })

      if (targetRef.value) {
        observer.observe(targetRef.value)
      }
    }
  })

  onUnmounted(() => {
    if (observer) {
      observer.disconnect()
    }
    updateSize.cancel()
  })

  return {
    width: computed(() => width.value),
    height: computed(() => height.value)
  }
}

export default {
  useDebouncedRef,
  useThrottledFn,
  useRAFThrottle,
  useMemoizedComputed,
  useAsyncData,
  useOptimizedList,
  useIntersectionObserver,
  useResizeObserver
}
