import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

// 开发期把 /api 代理到后端，浏览器看到的是同源请求，因此后端不需要 CORS。
// 目标写 127.0.0.1 而不是 localhost：Node 17+ 在 Windows 上常把 localhost 解析为 IPv6 ::1，
// 而后端只监听 IPv4，会导致代理报 ECONNREFUSED。
export default defineConfig({
  plugins: [react()],
  server: {
    host: '127.0.0.1',
    proxy: {
      '/api': 'http://127.0.0.1:8080',
    },
  },
})
