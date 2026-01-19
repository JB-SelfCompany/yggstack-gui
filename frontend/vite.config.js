import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

export default defineConfig(({ mode }) => ({
  plugins: [
    vue({
      script: {
        defineModel: true,
        propsDestructure: true
      }
    })
  ],
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src')
    }
  },
  server: {
    port: 5173,
    strictPort: true,
    host: 'localhost'
  },
  build: {
    outDir: 'dist',
    assetsDir: 'assets',
    emptyOutDir: true,
    // Optimize for production
    target: 'esnext',
    minify: 'esbuild',
    // Enable source maps only in development
    sourcemap: mode === 'development',
    // Chunk size warning threshold (500kb)
    chunkSizeWarningLimit: 500,
    rollupOptions: {
      output: {
        // Optimize chunk splitting
        manualChunks(id) {
          // Vue core
          if (id.includes('node_modules/vue/') ||
              id.includes('node_modules/@vue/')) {
            return 'vue-core'
          }
          // Vue ecosystem
          if (id.includes('node_modules/vue-router/') ||
              id.includes('node_modules/pinia/')) {
            return 'vue-ecosystem'
          }
          // i18n
          if (id.includes('node_modules/vue-i18n/') ||
              id.includes('node_modules/@intlify/')) {
            return 'i18n'
          }
          // Utilities (keep separate for caching)
          if (id.includes('/src/utils/')) {
            return 'utils'
          }
          // Workers
          if (id.includes('/src/workers/')) {
            return 'workers'
          }
        },
        // Consistent chunk names for caching
        chunkFileNames: (chunkInfo) => {
          const facadeModuleId = chunkInfo.facadeModuleId
          if (facadeModuleId && facadeModuleId.includes('/views/')) {
            return 'assets/views/[name]-[hash].js'
          }
          return 'assets/[name]-[hash].js'
        },
        assetFileNames: (assetInfo) => {
          const ext = assetInfo.name?.split('.').pop()
          if (ext === 'css') {
            return 'assets/css/[name]-[hash].[ext]'
          }
          if (['png', 'jpg', 'jpeg', 'gif', 'svg', 'ico'].includes(ext)) {
            return 'assets/images/[name]-[hash].[ext]'
          }
          return 'assets/[name]-[hash].[ext]'
        }
      },
      // Tree-shaking optimizations
      treeshake: {
        moduleSideEffects: false,
        propertyReadSideEffects: false
      }
    },
    // CSS optimization
    cssCodeSplit: true,
    cssMinify: true
  },
  // Optimize dependencies
  optimizeDeps: {
    include: [
      'vue',
      'vue-router',
      'pinia',
      'vue-i18n'
    ],
    exclude: []
  },
  // Define globals
  define: {
    __VUE_OPTIONS_API__: false, // Disable Options API for smaller bundle
    __VUE_PROD_DEVTOOLS__: false,
    __VUE_PROD_HYDRATION_MISMATCH_DETAILS__: false
  },
  // Enable esbuild optimizations
  esbuild: {
    legalComments: 'none',
    treeShaking: true,
    minifyIdentifiers: mode === 'production',
    minifySyntax: mode === 'production',
    minifyWhitespace: mode === 'production'
  }
}))
