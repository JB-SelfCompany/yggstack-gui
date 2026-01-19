import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { createRouter, createWebHashHistory } from 'vue-router'
import { createI18n } from 'vue-i18n'

import App from './App.vue'
import { en, ru } from './i18n'
import './styles/main.css'

// Routes
const routes = [
  {
    path: '/',
    name: 'dashboard',
    component: () => import('./views/Dashboard.vue')
  },
  {
    path: '/peers',
    name: 'peers',
    component: () => import('./views/Peers.vue')
  },
  {
    path: '/proxy',
    name: 'proxy',
    component: () => import('./views/Proxy.vue')
  },
  {
    path: '/forwarding',
    name: 'forwarding',
    component: () => import('./views/Forwarding.vue')
  },
  {
    path: '/logs',
    name: 'logs',
    component: () => import('./views/Logs.vue')
  },
  {
    path: '/settings',
    name: 'settings',
    component: () => import('./views/Settings.vue')
  }
]

// Router
const router = createRouter({
  history: createWebHashHistory(),
  routes
})

// i18n
const i18n = createI18n({
  legacy: false,
  locale: 'en',
  fallbackLocale: 'en',
  messages: {
    en,
    ru
  }
})

// Create app
const app = createApp(App)

// Use plugins
app.use(createPinia())
app.use(router)
app.use(i18n)

// Mount
app.mount('#app')

// Notify backend that frontend is ready
window.addEventListener('DOMContentLoaded', () => {
  if (window.ipc) {
    window.ipc.emit('app:ready', {})
  }
})
