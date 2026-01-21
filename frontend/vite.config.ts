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
            .replace(/%VITE_APP_NAME%/g, env.VITE_APP_NAME || 'Whatomate')
            .replace(/%VITE_APP_DESCRIPTION%/g, env.VITE_APP_DESCRIPTION || 'WhatsApp Business Platform')
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
