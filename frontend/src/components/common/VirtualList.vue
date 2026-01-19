<template>
  <div
    ref="containerRef"
    class="virtual-list"
    :style="containerStyle"
    @scroll="onScroll"
  >
    <div class="virtual-list-spacer" :style="spacerStyle">
      <div
        class="virtual-list-content"
        :style="contentStyle"
      >
        <slot
          v-for="item in visibleItems"
          :key="getItemKey(item)"
          :item="item.data"
          :index="item.index"
        />
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'

const props = defineProps({
  // Array of items to render
  items: {
    type: Array,
    required: true
  },
  // Height of each item in pixels
  itemHeight: {
    type: Number,
    required: true
  },
  // Number of items to render above/below visible area
  overscan: {
    type: Number,
    default: 5
  },
  // Key function for item identification
  keyField: {
    type: String,
    default: 'id'
  },
  // Container height (auto if not specified)
  height: {
    type: [Number, String],
    default: 'auto'
  },
  // Minimum items to trigger virtualization
  minItemsForVirtualization: {
    type: Number,
    default: 50
  }
})

const emit = defineEmits(['scroll', 'visible-range-change'])

const containerRef = ref(null)
const scrollTop = ref(0)
const containerHeight = ref(0)

// Whether to use virtualization
const useVirtualization = computed(() =>
  props.items.length >= props.minItemsForVirtualization
)

// Total height of all items
const totalHeight = computed(() =>
  props.items.length * props.itemHeight
)

// Calculate visible range
const visibleRange = computed(() => {
  if (!useVirtualization.value) {
    return { start: 0, end: props.items.length }
  }

  const start = Math.max(0, Math.floor(scrollTop.value / props.itemHeight) - props.overscan)
  const visibleCount = Math.ceil(containerHeight.value / props.itemHeight)
  const end = Math.min(
    props.items.length,
    start + visibleCount + props.overscan * 2
  )

  return { start, end }
})

// Visible items with their indices
const visibleItems = computed(() => {
  const { start, end } = visibleRange.value
  return props.items.slice(start, end).map((data, i) => ({
    data,
    index: start + i
  }))
})

// Container style
const containerStyle = computed(() => ({
  height: typeof props.height === 'number' ? `${props.height}px` : props.height,
  overflow: 'auto'
}))

// Spacer style (creates scroll area)
const spacerStyle = computed(() => ({
  height: useVirtualization.value ? `${totalHeight.value}px` : 'auto',
  position: 'relative'
}))

// Content positioning
const contentStyle = computed(() => {
  if (!useVirtualization.value) {
    return {}
  }
  return {
    position: 'absolute',
    top: `${visibleRange.value.start * props.itemHeight}px`,
    left: 0,
    right: 0
  }
})

// Get unique key for item
const getItemKey = (item) => {
  if (typeof props.keyField === 'function') {
    return props.keyField(item.data)
  }
  return item.data[props.keyField] ?? item.index
}

// Scroll handler with RAF throttling
let scrollRAF = null
const onScroll = (e) => {
  if (scrollRAF) return

  scrollRAF = requestAnimationFrame(() => {
    scrollTop.value = e.target.scrollTop
    emit('scroll', {
      scrollTop: scrollTop.value,
      scrollHeight: e.target.scrollHeight,
      clientHeight: e.target.clientHeight
    })
    scrollRAF = null
  })
}

// Update container height
const updateContainerHeight = () => {
  if (containerRef.value) {
    containerHeight.value = containerRef.value.clientHeight
  }
}

// Scroll to specific index
const scrollToIndex = (index, behavior = 'auto') => {
  if (containerRef.value) {
    const offset = index * props.itemHeight
    containerRef.value.scrollTo({
      top: offset,
      behavior
    })
  }
}

// Scroll to top
const scrollToTop = (behavior = 'auto') => {
  scrollToIndex(0, behavior)
}

// Scroll to bottom
const scrollToBottom = (behavior = 'auto') => {
  if (containerRef.value) {
    containerRef.value.scrollTo({
      top: totalHeight.value,
      behavior
    })
  }
}

// Resize observer
let resizeObserver = null

onMounted(() => {
  updateContainerHeight()

  // Watch for container resize
  if (typeof ResizeObserver !== 'undefined') {
    resizeObserver = new ResizeObserver(() => {
      updateContainerHeight()
    })
    if (containerRef.value) {
      resizeObserver.observe(containerRef.value)
    }
  }
})

onUnmounted(() => {
  if (scrollRAF) {
    cancelAnimationFrame(scrollRAF)
  }
  if (resizeObserver) {
    resizeObserver.disconnect()
  }
})

// Watch for visible range changes
watch(visibleRange, (newRange) => {
  emit('visible-range-change', newRange)
}, { deep: true })

// Re-measure on items change
watch(() => props.items.length, () => {
  nextTick(updateContainerHeight)
})

// Expose methods
defineExpose({
  scrollToIndex,
  scrollToTop,
  scrollToBottom,
  getVisibleRange: () => visibleRange.value
})
</script>

<style scoped>
.virtual-list {
  will-change: transform;
  -webkit-overflow-scrolling: touch;
}

.virtual-list-spacer {
  min-height: 100%;
}

.virtual-list-content {
  will-change: transform;
}
</style>
