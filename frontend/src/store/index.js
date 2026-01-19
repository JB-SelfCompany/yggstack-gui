import { createPinia } from 'pinia'

export const pinia = createPinia()

export { useNodeStore } from './node'
export { usePeersStore } from './peers'
export { useUiStore } from './ui'
