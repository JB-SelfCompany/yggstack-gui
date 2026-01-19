/**
 * NodeStatus component tests
 */

import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import NodeStatus from './NodeStatus.vue'
import { useNodeStore } from '../../store/node'

const i18n = createI18n({
  legacy: false,
  locale: 'en',
  messages: {
    en: {
      node: {
        status: {
          stopped: 'Stopped',
          starting: 'Starting...',
          running: 'Running',
          stopping: 'Stopping...'
        }
      }
    }
  }
})

describe('NodeStatus', () => {
  let pinia

  beforeEach(() => {
    pinia = createPinia()
    setActivePinia(pinia)
  })

  const mountComponent = () => {
    return mount(NodeStatus, {
      global: {
        plugins: [pinia, i18n]
      }
    })
  }

  describe('rendering', () => {
    it('renders status badge', () => {
      const wrapper = mountComponent()
      expect(wrapper.find('.status-badge').exists()).toBe(true)
    })

    it('renders status dot', () => {
      const wrapper = mountComponent()
      expect(wrapper.find('.status-dot').exists()).toBe(true)
    })

    it('renders status label', () => {
      const wrapper = mountComponent()
      expect(wrapper.find('.status-label').exists()).toBe(true)
    })
  })

  describe('status states', () => {
    it('shows stopped status correctly', () => {
      const wrapper = mountComponent()
      const nodeStore = useNodeStore()
      nodeStore.status = 'stopped'

      expect(wrapper.find('.status-badge').classes()).toContain('status-stopped')
      expect(wrapper.text()).toContain('Stopped')
    })

    it('shows starting status correctly', async () => {
      const nodeStore = useNodeStore()
      nodeStore.status = 'starting'
      const wrapper = mountComponent()

      expect(wrapper.find('.status-badge').classes()).toContain('status-starting')
      expect(wrapper.text()).toContain('Starting...')
    })

    it('shows running status correctly', async () => {
      const nodeStore = useNodeStore()
      nodeStore.status = 'running'
      const wrapper = mountComponent()

      expect(wrapper.find('.status-badge').classes()).toContain('status-running')
      expect(wrapper.text()).toContain('Running')
    })

    it('shows stopping status correctly', async () => {
      const nodeStore = useNodeStore()
      nodeStore.status = 'stopping'
      const wrapper = mountComponent()

      expect(wrapper.find('.status-badge').classes()).toContain('status-stopping')
      expect(wrapper.text()).toContain('Stopping...')
    })
  })

  describe('reactivity', () => {
    it('updates when store status changes', async () => {
      const nodeStore = useNodeStore()
      nodeStore.status = 'stopped'
      const wrapper = mountComponent()

      expect(wrapper.text()).toContain('Stopped')

      nodeStore.status = 'running'
      await wrapper.vm.$nextTick()

      expect(wrapper.text()).toContain('Running')
    })
  })
})
