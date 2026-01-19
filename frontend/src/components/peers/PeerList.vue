<template>
  <div class="peer-list">
    <!-- Search and filter bar -->
    <div class="peer-list-toolbar" v-if="peers.length > 0">
      <input
        type="text"
        v-model="searchQuery"
        :placeholder="t('peers.search')"
        class="search-input"
      />
      <div class="peer-count">
        {{ filteredPeers.length }} / {{ peers.length }}
      </div>
    </div>

    <div v-if="peers.length === 0" class="empty-state">
      <p>{{ t('peers.empty') }}</p>
    </div>

    <!-- Use virtual scrolling for large lists -->
    <template v-else-if="filteredPeers.length > 50">
      <div class="peers-table-header">
        <div class="header-cell uri-col" @click="toggleSort('uri')">
          {{ t('peers.uri') }}
          <span v-if="sortField === 'uri'" class="sort-indicator">
            {{ sortDirection === 'asc' ? '▲' : '▼' }}
          </span>
        </div>
        <div class="header-cell status-col" @click="toggleSort('connected')">
          {{ t('peers.status.label') }}
          <span v-if="sortField === 'connected'" class="sort-indicator">
            {{ sortDirection === 'asc' ? '▲' : '▼' }}
          </span>
        </div>
        <div class="header-cell actions-col">{{ t('peers.actions') }}</div>
      </div>
      <VirtualList
        :items="sortedPeers"
        :item-height="48"
        :height="400"
        key-field="uri"
        :overscan="5"
        class="virtual-peer-list"
      >
        <template #default="{ item }">
          <PeerItemRow
            :peer="item"
            @remove="removePeer"
          />
        </template>
      </VirtualList>
    </template>

    <!-- Regular table for smaller lists -->
    <table v-else class="peers-table">
      <thead>
        <tr>
          <th @click="toggleSort('uri')" class="sortable">
            {{ t('peers.uri') }}
            <span v-if="sortField === 'uri'" class="sort-indicator">
              {{ sortDirection === 'asc' ? '▲' : '▼' }}
            </span>
          </th>
          <th @click="toggleSort('connected')" class="sortable">
            {{ t('peers.status.label') }}
            <span v-if="sortField === 'connected'" class="sort-indicator">
              {{ sortDirection === 'asc' ? '▲' : '▼' }}
            </span>
          </th>
          <th>{{ t('peers.actions') }}</th>
        </tr>
      </thead>
      <tbody>
        <PeerItem
          v-for="peer in sortedPeers"
          :key="peer.uri"
          :peer="peer"
          @remove="removePeer"
        />
      </tbody>
    </table>
  </div>
</template>

<script setup>
import { ref, computed, h } from 'vue'
import { useI18n } from 'vue-i18n'
import { usePeersStore } from '../../store/peers'
import { useDebouncedRef } from '../../composables/useOptimized'
import PeerItem from './PeerItem.vue'
import VirtualList from '../common/VirtualList.vue'

const { t } = useI18n()
const peersStore = usePeersStore()

// Search with debounce
const searchQuery = ref('')

// Sorting state
const sortField = ref(null)
const sortDirection = ref('asc')

const peers = computed(() => peersStore.peers)

// Filter peers by search query
const filteredPeers = computed(() => {
  if (!searchQuery.value) return peers.value

  const query = searchQuery.value.toLowerCase()
  return peers.value.filter(peer =>
    peer.uri?.toLowerCase().includes(query) ||
    peer.address?.toLowerCase().includes(query)
  )
})

// Sort peers
const sortedPeers = computed(() => {
  if (!sortField.value) return filteredPeers.value

  return [...filteredPeers.value].sort((a, b) => {
    const mult = sortDirection.value === 'desc' ? -1 : 1
    const valA = a[sortField.value]
    const valB = b[sortField.value]

    if (valA == null) return mult
    if (valB == null) return -mult

    if (typeof valA === 'boolean') {
      return mult * (valA === valB ? 0 : valA ? -1 : 1)
    }

    if (typeof valA === 'string') {
      return mult * valA.localeCompare(valB)
    }

    return mult * (valA - valB)
  })
})

const toggleSort = (field) => {
  if (sortField.value === field) {
    sortDirection.value = sortDirection.value === 'asc' ? 'desc' : 'asc'
  } else {
    sortField.value = field
    sortDirection.value = 'asc'
  }
}

const removePeer = async (uri) => {
  await peersStore.removePeer(uri)
}

// Row component for virtual list
const PeerItemRow = {
  props: ['peer'],
  emits: ['remove'],
  setup(props, { emit }) {
    return () => h('div', { class: 'peer-row' }, [
      h('div', { class: 'peer-cell uri-col', title: props.peer.uri }, props.peer.uri),
      h('div', { class: 'peer-cell status-col' }, [
        h('span', {
          class: ['status-badge', props.peer.connected ? 'connected' : 'disconnected']
        }, props.peer.connected ? t('peers.status.up') : t('peers.status.down'))
      ]),
      h('div', { class: 'peer-cell actions-col' }, [
        h('button', {
          class: 'remove-btn',
          onClick: () => emit('remove', props.peer.uri)
        }, t('peers.remove'))
      ])
    ])
  }
}
</script>

<style scoped>
.peer-list {
  width: 100%;
}

.peer-list-toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
}

.search-input {
  flex: 1;
  max-width: 300px;
  padding: 8px 12px;
  border: 1px solid var(--color-border);
  border-radius: 4px;
  background: var(--color-bg-secondary);
  color: var(--color-text-primary);
  font-size: 14px;
}

.search-input:focus {
  outline: none;
  border-color: var(--color-accent);
}

.peer-count {
  font-size: 12px;
  color: var(--color-text-secondary);
}

.empty-state {
  text-align: center;
  padding: 40px;
  color: var(--color-text-secondary);
}

.peers-table {
  width: 100%;
  border-collapse: collapse;
}

.peers-table th {
  text-align: left;
  padding: 12px;
  font-size: 12px;
  font-weight: 600;
  color: var(--color-text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.5px;
  border-bottom: 1px solid var(--color-border);
}

.peers-table th.sortable {
  cursor: pointer;
  user-select: none;
}

.peers-table th.sortable:hover {
  color: var(--color-text-primary);
}

.sort-indicator {
  margin-left: 4px;
  font-size: 10px;
}

/* Virtual list styles */
.peers-table-header {
  display: flex;
  padding: 12px;
  border-bottom: 1px solid var(--color-border);
}

.header-cell {
  font-size: 12px;
  font-weight: 600;
  color: var(--color-text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.5px;
  cursor: pointer;
  user-select: none;
}

.header-cell:hover {
  color: var(--color-text-primary);
}

.virtual-peer-list {
  border: 1px solid var(--color-border);
  border-radius: 4px;
}

.peer-row {
  display: flex;
  align-items: center;
  padding: 12px;
  border-bottom: 1px solid var(--color-border);
}

.peer-row:last-child {
  border-bottom: none;
}

.peer-cell {
  font-size: 14px;
}

.uri-col {
  flex: 2;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.status-col {
  flex: 1;
}

.actions-col {
  flex: 0 0 100px;
  text-align: right;
}

.status-badge {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 500;
}

.status-badge.connected {
  background: rgba(40, 167, 69, 0.2);
  color: var(--color-success);
}

.status-badge.disconnected {
  background: rgba(220, 53, 69, 0.2);
  color: var(--color-danger);
}

.remove-btn {
  padding: 4px 8px;
  border: 1px solid var(--color-danger);
  border-radius: 4px;
  background: transparent;
  color: var(--color-danger);
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s;
}

.remove-btn:hover {
  background: var(--color-danger);
  color: white;
}
</style>
