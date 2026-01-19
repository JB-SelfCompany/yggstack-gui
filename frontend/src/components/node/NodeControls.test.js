/**
 * NodeControls component tests
 */

import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import NodeControls from './NodeControls.vue'
import { useNodeStore } from '../../store/node'

const i18n = createI18n({
  legacy: false,
  locale: 'en',
  messages: {
    en: {
      node: {
        actions: {
          start: 'Start',
          stop: 'Stop',
          restart: 'Restart'
        }
      }
    }
  }
})

describe('NodeControls', () => {
  let pinia
  let nodeStore

  beforeEach(() => {
    pinia = createPinia()
    setActivePinia(pinia)
    nodeStore = useNodeStore()
  })

  const mountComponent = () => {
    return mount(NodeControls, {
      global: {
        plugins: [pinia, i18n]
      }
    })
  }

  describe('button visibility', () => {
    it('shows only Start button when stopped', () => {
      nodeStore.status = 'stopped'
      const wrapper = mountComponent()

      expect(wrapper.text()).toContain('Start')
      expect(wrapper.text()).not.toContain('Stop')
      expect(wrapper.text()).not.toContain('Restart')
    })

    it('shows Stop and Restart buttons when running', () => {
      nodeStore.status = 'running'
      const wrapper = mountComponent()

      expect(wrapper.text()).not.toContain('Start')
      expect(wrapper.text()).toContain('Stop')
      expect(wrapper.text()).toContain('Restart')
    })

    it('hides all action buttons during starting', () => {
      nodeStore.status = 'starting'
      const wrapper = mountComponent()

      // Start button hidden because status !== 'stopped'
      // Stop button hidden because status !== 'running'
      const buttons = wrapper.findAll('.btn')
      buttons.forEach(btn => {
        expect(btn.attributes('disabled')).toBeDefined()
      })
    })

    it('disables buttons during stopping', () => {
      nodeStore.status = 'stopping'
      const wrapper = mountComponent()

      const buttons = wrapper.findAll('.btn')
      buttons.forEach(btn => {
        expect(btn.attributes('disabled')).toBeDefined()
      })
    })
  })

  describe('button actions', () => {
    it('calls nodeStore.start when Start clicked', async () => {
      nodeStore.status = 'stopped'
      const startSpy = vi.spyOn(nodeStore, 'start').mockResolvedValue()

      const wrapper = mountComponent()
      await wrapper.find('.btn-success').trigger('click')

      expect(startSpy).toHaveBeenCalled()
    })

    it('calls nodeStore.stop when Stop clicked', async () => {
      nodeStore.status = 'running'
      const stopSpy = vi.spyOn(nodeStore, 'stop').mockResolvedValue()

      const wrapper = mountComponent()
      await wrapper.find('.btn-danger').trigger('click')

      expect(stopSpy).toHaveBeenCalled()
    })

    it('calls stop then start for restart', async () => {
      nodeStore.status = 'running'
      const stopSpy = vi.spyOn(nodeStore, 'stop').mockResolvedValue()
      const startSpy = vi.spyOn(nodeStore, 'start').mockResolvedValue()

      const wrapper = mountComponent()
      await wrapper.find('.btn-secondary').trigger('click')

      expect(stopSpy).toHaveBeenCalled()
      expect(startSpy).toHaveBeenCalled()
    })
  })

  describe('loading states', () => {
    it('disables Start button during starting', () => {
      nodeStore.status = 'starting'
      const wrapper = mountComponent()

      const buttons = wrapper.findAll('.btn')
      buttons.forEach(btn => {
        expect(btn.attributes('disabled')).toBeDefined()
      })
    })

    it('disables Stop button during stopping', () => {
      nodeStore.status = 'stopping'
      const wrapper = mountComponent()

      const buttons = wrapper.findAll('.btn')
      buttons.forEach(btn => {
        expect(btn.attributes('disabled')).toBeDefined()
      })
    })
  })
})
