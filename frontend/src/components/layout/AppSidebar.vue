<template>
  <nav class="app-sidebar">
    <ul class="nav-list">
      <li v-for="item in navItems" :key="item.path">
        <router-link
          :to="item.path"
          class="nav-link"
          :class="{ 'nav-link-active': isActive(item.path) }"
        >
          <span class="nav-icon">{{ item.icon }}</span>
          <span class="nav-label">{{ t(item.labelKey) }}</span>
        </router-link>
      </li>
    </ul>
  </nav>
</template>

<script setup>
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'

const { t } = useI18n()
const route = useRoute()

const navItems = [
  { path: '/', labelKey: 'nav.dashboard', icon: '◉' },
  { path: '/peers', labelKey: 'nav.peers', icon: '⬡' },
  { path: '/proxy', labelKey: 'nav.proxy', icon: '⇄' },
  { path: '/forwarding', labelKey: 'nav.forwarding', icon: '↹' },
  { path: '/logs', labelKey: 'nav.logs', icon: '☰' },
  { path: '/settings', labelKey: 'nav.settings', icon: '⚙' }
]

const isActive = (path) => {
  if (path === '/') {
    return route.path === '/'
  }
  return route.path.startsWith(path)
}
</script>

<style scoped>
.app-sidebar {
  width: 200px;
  background-color: var(--color-bg-secondary);
  border-right: 1px solid var(--color-border);
  padding: 16px 0;
}

.nav-list {
  list-style: none;
  margin: 0;
  padding: 0;
}

.nav-link {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 24px;
  color: var(--color-text-secondary);
  text-decoration: none;
  transition: all 0.2s ease;
}

.nav-link:hover {
  background-color: var(--color-bg-primary);
  color: var(--color-text-primary);
}

.nav-link-active {
  background-color: var(--color-accent);
  color: #ffffff;
}

.nav-link-active:hover {
  background-color: var(--color-accent);
  color: #ffffff;
}

.nav-icon {
  font-size: 18px;
}

.nav-label {
  font-size: 14px;
  font-weight: 500;
}
</style>
