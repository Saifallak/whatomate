import { fileURLToPath, URL } from 'node:url'
import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), 'VITE_')

  return {
    base: './',
    plugins: [
      vue(),
      {
        name: 'html-transform',
        transformIndexHtml(html) {
          return html
            .replace(/__VITE_APP_NAME__/g, env.VITE_APP_NAME || 'Whatomate')
            .replace(/__VITE_APP_DESCRIPTION__/g, env.VITE_APP_DESCRIPTION || 'WhatsApp Business Platform')
            .replace(/__VITE_APP_ICON__/g, env.VITE_APP_ICON || '/favicon.svg')
        }
      }
    ],
    resolve: {
      alias: {
        '@': fileURLToPath(new URL('./src', import.meta.url))
      }
    },
    server: {
      port: 3000,
      allowedHosts: ['roland-min-assessments-lesser.trycloudflare.com', '.trycloudflare.com'],
      proxy: {
        '/api': {
          target: 'http://localhost:8080',
          changeOrigin: true
        },
        '/ws': {
          target: 'ws://localhost:8080',
          ws: true
        }
      }
    }
  }
})
