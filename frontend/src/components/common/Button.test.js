/**
 * Button component tests
 */

import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import Button from './Button.vue'

describe('Button', () => {
  describe('rendering', () => {
    it('renders with default props', () => {
      const wrapper = mount(Button, {
        slots: { default: 'Click me' }
      })

      expect(wrapper.find('button').exists()).toBe(true)
      expect(wrapper.text()).toContain('Click me')
      expect(wrapper.classes()).toContain('btn')
      expect(wrapper.classes()).toContain('btn-primary')
      expect(wrapper.classes()).toContain('btn-medium')
    })

    it('renders with different variants', () => {
      const variants = ['primary', 'secondary', 'danger', 'ghost', 'success']

      variants.forEach(variant => {
        const wrapper = mount(Button, {
          props: { variant },
          slots: { default: 'Button' }
        })
        expect(wrapper.classes()).toContain(`btn-${variant}`)
      })
    })

    it('renders with different sizes', () => {
      const sizes = ['small', 'medium', 'large']

      sizes.forEach(size => {
        const wrapper = mount(Button, {
          props: { size },
          slots: { default: 'Button' }
        })
        expect(wrapper.classes()).toContain(`btn-${size}`)
      })
    })

    it('renders icon when provided', () => {
      const wrapper = mount(Button, {
        props: { icon: '★' },
        slots: { default: 'Star' }
      })

      expect(wrapper.find('.btn-icon').exists()).toBe(true)
      expect(wrapper.find('.btn-icon').text()).toBe('★')
    })

    it('renders as icon-only button', () => {
      const wrapper = mount(Button, {
        props: { icon: '★', iconOnly: true }
      })

      expect(wrapper.classes()).toContain('btn-icon-only')
      expect(wrapper.find('.btn-text').exists()).toBe(false)
    })
  })

  describe('loading state', () => {
    it('shows spinner when loading', () => {
      const wrapper = mount(Button, {
        props: { loading: true },
        slots: { default: 'Loading' }
      })

      expect(wrapper.find('.btn-spinner').exists()).toBe(true)
      expect(wrapper.classes()).toContain('btn-loading')
    })

    it('hides icon when loading', () => {
      const wrapper = mount(Button, {
        props: { icon: '★', loading: true },
        slots: { default: 'Loading' }
      })

      expect(wrapper.find('.btn-icon').exists()).toBe(false)
      expect(wrapper.find('.btn-spinner').exists()).toBe(true)
    })

    it('disables button when loading', () => {
      const wrapper = mount(Button, {
        props: { loading: true },
        slots: { default: 'Loading' }
      })

      expect(wrapper.find('button').attributes('disabled')).toBeDefined()
    })
  })

  describe('disabled state', () => {
    it('disables button when disabled prop is true', () => {
      const wrapper = mount(Button, {
        props: { disabled: true },
        slots: { default: 'Disabled' }
      })

      expect(wrapper.find('button').attributes('disabled')).toBeDefined()
    })

    it('does not emit click when disabled', async () => {
      const wrapper = mount(Button, {
        props: { disabled: true },
        slots: { default: 'Disabled' }
      })

      await wrapper.trigger('click')

      expect(wrapper.emitted('click')).toBeUndefined()
    })
  })

  describe('click handling', () => {
    it('emits click event when clicked', async () => {
      const wrapper = mount(Button, {
        slots: { default: 'Click me' }
      })

      await wrapper.trigger('click')

      expect(wrapper.emitted('click')).toBeTruthy()
      expect(wrapper.emitted('click')).toHaveLength(1)
    })

    it('does not emit click when loading', async () => {
      const wrapper = mount(Button, {
        props: { loading: true },
        slots: { default: 'Loading' }
      })

      await wrapper.trigger('click')

      expect(wrapper.emitted('click')).toBeUndefined()
    })

    it('passes event object to click handler', async () => {
      const wrapper = mount(Button, {
        slots: { default: 'Click' }
      })

      await wrapper.trigger('click')

      expect(wrapper.emitted('click')[0][0]).toBeInstanceOf(MouseEvent)
    })
  })

  describe('button type', () => {
    it('has default type of button', () => {
      const wrapper = mount(Button, {
        slots: { default: 'Button' }
      })

      expect(wrapper.find('button').attributes('type')).toBe('button')
    })

    it('allows custom type', () => {
      const wrapper = mount(Button, {
        props: { type: 'submit' },
        slots: { default: 'Submit' }
      })

      expect(wrapper.find('button').attributes('type')).toBe('submit')
    })
  })
})
