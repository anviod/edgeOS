import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src'),
    },
  },
  server: {
    port: 3000,
    proxy: {
      '/api': {
        // 与 internal/config/config.go 的 cfg.Node.Listen 默认值保持一致（":80"）
        // 如后端改到其它端口，用 EDGEOS_API 环境变量覆盖，例如：
        //   EDGEOS_API=http://localhost:8000 npm run dev
        target: process.env.EDGEOS_API || 'http://localhost:80',
        changeOrigin: true,
        configure: (proxy, options) => {
          proxy.on('proxyReq', (proxyReq, req, res) => {
            console.log('Proxying request:', req.url);
          });
          proxy.on('proxyRes', (proxyRes, req, res) => {
            console.log('Proxy response status:', proxyRes.statusCode);
          });
        }
      },
    },
  },
  build: {
    outDir: 'dist',
    sourcemap: false,
  },
  test: {
    environment: 'jsdom',
    include: ['src/**/*.test.ts'],
    clearMocks: true,
    restoreMocks: true,
  },
})