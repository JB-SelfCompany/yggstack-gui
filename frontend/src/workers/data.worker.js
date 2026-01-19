/**
 * Data processing Web Worker
 * Handles heavy operations off the main thread:
 * - Sorting large lists
 * - Filtering with complex predicates
 * - Data aggregation
 * - Search/indexing
 */

// Message handlers
const handlers = {
  /**
   * Sort array of objects by field
   * @param {object} data - { items, field, direction, locale }
   */
  sort({ items, field, direction = 'asc', locale = 'en' }) {
    const multiplier = direction === 'desc' ? -1 : 1
    const collator = new Intl.Collator(locale, { sensitivity: 'base' })

    const sorted = [...items].sort((a, b) => {
      const valA = getNestedValue(a, field)
      const valB = getNestedValue(b, field)

      // Handle null/undefined
      if (valA == null && valB == null) return 0
      if (valA == null) return multiplier
      if (valB == null) return -multiplier

      // String comparison
      if (typeof valA === 'string' && typeof valB === 'string') {
        return multiplier * collator.compare(valA, valB)
      }

      // Number comparison
      if (typeof valA === 'number' && typeof valB === 'number') {
        return multiplier * (valA - valB)
      }

      // Boolean comparison
      if (typeof valA === 'boolean' && typeof valB === 'boolean') {
        return multiplier * (valA === valB ? 0 : valA ? -1 : 1)
      }

      // Fallback to string comparison
      return multiplier * String(valA).localeCompare(String(valB))
    })

    return sorted
  },

  /**
   * Filter array with predicate
   * @param {object} data - { items, predicates }
   */
  filter({ items, predicates }) {
    return items.filter(item => {
      return predicates.every(predicate => evaluatePredicate(item, predicate))
    })
  },

  /**
   * Search items by text in multiple fields
   * @param {object} data - { items, query, fields, caseSensitive }
   */
  search({ items, query, fields, caseSensitive = false }) {
    if (!query || query.trim() === '') {
      return items
    }

    const searchQuery = caseSensitive ? query : query.toLowerCase()
    const searchTerms = searchQuery.split(/\s+/).filter(Boolean)

    return items.filter(item => {
      const searchableText = fields
        .map(field => getNestedValue(item, field))
        .filter(Boolean)
        .map(val => caseSensitive ? String(val) : String(val).toLowerCase())
        .join(' ')

      return searchTerms.every(term => searchableText.includes(term))
    })
  },

  /**
   * Group items by field
   * @param {object} data - { items, field }
   */
  groupBy({ items, field }) {
    const groups = new Map()

    items.forEach(item => {
      const key = getNestedValue(item, field) ?? '__null__'
      if (!groups.has(key)) {
        groups.set(key, [])
      }
      groups.get(key).push(item)
    })

    return Object.fromEntries(groups)
  },

  /**
   * Aggregate numeric values
   * @param {object} data - { items, field, operations }
   */
  aggregate({ items, field, operations = ['sum', 'avg', 'min', 'max', 'count'] }) {
    const values = items
      .map(item => getNestedValue(item, field))
      .filter(v => typeof v === 'number' && !isNaN(v))

    const result = {}

    if (operations.includes('count')) {
      result.count = values.length
    }
    if (operations.includes('sum')) {
      result.sum = values.reduce((a, b) => a + b, 0)
    }
    if (operations.includes('avg')) {
      result.avg = values.length > 0 ? values.reduce((a, b) => a + b, 0) / values.length : 0
    }
    if (operations.includes('min')) {
      result.min = values.length > 0 ? Math.min(...values) : null
    }
    if (operations.includes('max')) {
      result.max = values.length > 0 ? Math.max(...values) : null
    }

    return result
  },

  /**
   * Compute diff between two arrays
   * @param {object} data - { oldItems, newItems, keyField }
   */
  diff({ oldItems, newItems, keyField = 'id' }) {
    const oldMap = new Map(oldItems.map(item => [item[keyField], item]))
    const newMap = new Map(newItems.map(item => [item[keyField], item]))

    const added = []
    const removed = []
    const updated = []
    const unchanged = []

    // Find added and updated
    newItems.forEach(item => {
      const key = item[keyField]
      const oldItem = oldMap.get(key)

      if (!oldItem) {
        added.push(item)
      } else if (!deepEqual(oldItem, item)) {
        updated.push({ old: oldItem, new: item })
      } else {
        unchanged.push(item)
      }
    })

    // Find removed
    oldItems.forEach(item => {
      const key = item[keyField]
      if (!newMap.has(key)) {
        removed.push(item)
      }
    })

    return { added, removed, updated, unchanged }
  },

  /**
   * Paginate items
   * @param {object} data - { items, page, pageSize }
   */
  paginate({ items, page = 1, pageSize = 50 }) {
    const totalItems = items.length
    const totalPages = Math.ceil(totalItems / pageSize)
    const offset = (page - 1) * pageSize
    const pageItems = items.slice(offset, offset + pageSize)

    return {
      items: pageItems,
      page,
      pageSize,
      totalItems,
      totalPages,
      hasNext: page < totalPages,
      hasPrev: page > 1
    }
  },

  /**
   * Create index for fast lookup
   * @param {object} data - { items, fields }
   */
  createIndex({ items, fields }) {
    const indices = {}

    fields.forEach(field => {
      indices[field] = new Map()
      items.forEach((item, idx) => {
        const value = getNestedValue(item, field)
        if (value != null) {
          if (!indices[field].has(value)) {
            indices[field].set(value, [])
          }
          indices[field].get(value).push(idx)
        }
      })
      // Convert Map to object for transfer
      indices[field] = Object.fromEntries(indices[field])
    })

    return indices
  }
}

// Helper functions
function getNestedValue(obj, path) {
  if (!path) return obj
  const parts = path.split('.')
  let value = obj
  for (const part of parts) {
    if (value == null) return undefined
    value = value[part]
  }
  return value
}

function evaluatePredicate(item, predicate) {
  const { field, operator, value } = predicate
  const itemValue = getNestedValue(item, field)

  switch (operator) {
    case 'eq':
      return itemValue === value
    case 'neq':
      return itemValue !== value
    case 'gt':
      return itemValue > value
    case 'gte':
      return itemValue >= value
    case 'lt':
      return itemValue < value
    case 'lte':
      return itemValue <= value
    case 'contains':
      return String(itemValue).toLowerCase().includes(String(value).toLowerCase())
    case 'startsWith':
      return String(itemValue).toLowerCase().startsWith(String(value).toLowerCase())
    case 'endsWith':
      return String(itemValue).toLowerCase().endsWith(String(value).toLowerCase())
    case 'in':
      return Array.isArray(value) && value.includes(itemValue)
    case 'notIn':
      return Array.isArray(value) && !value.includes(itemValue)
    case 'exists':
      return itemValue != null
    case 'notExists':
      return itemValue == null
    case 'regex':
      return new RegExp(value, 'i').test(String(itemValue))
    default:
      return true
  }
}

function deepEqual(a, b) {
  if (a === b) return true
  if (a == null || b == null) return false
  if (typeof a !== 'object' || typeof b !== 'object') return false

  const keysA = Object.keys(a)
  const keysB = Object.keys(b)

  if (keysA.length !== keysB.length) return false

  for (const key of keysA) {
    if (!Object.prototype.hasOwnProperty.call(b, key)) return false
    if (!deepEqual(a[key], b[key])) return false
  }

  return true
}

// Message listener
self.onmessage = function (e) {
  const { id, type, data } = e.data

  try {
    const handler = handlers[type]
    if (!handler) {
      throw new Error(`Unknown operation type: ${type}`)
    }

    const startTime = performance.now()
    const result = handler(data)
    const duration = performance.now() - startTime

    self.postMessage({
      id,
      success: true,
      result,
      duration
    })
  } catch (error) {
    self.postMessage({
      id,
      success: false,
      error: error.message
    })
  }
}
