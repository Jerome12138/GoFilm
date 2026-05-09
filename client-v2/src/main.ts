import { createApp } from 'vue'
import { createPinia } from 'pinia'

import 'virtual:uno.css'
import '@/assets/styles/reset.css'
import '@/assets/styles/theme.css'
import '@/assets/styles/iconfont.css'

import App from './App.vue'
import router from './router'
import { installViewMode } from '@/composables/useViewMode'

// Mock 适配器：仅 dev 且 VITE_USE_MOCK 为真时挂载（不影响生产构建）
// 用 top-level await 等装载完成，避免 App.vue 首屏 API 与拦截器装载抢跑
if (import.meta.env.DEV && import.meta.env.VITE_USE_MOCK) {
  const { installMockAdapter } = await import('@/mock/install')
  installMockAdapter()
}

const app = createApp(App)
const pinia = createPinia()

app.use(pinia)
app.use(router)

// 全局安装一次 viewMode（resize / storage 监听绑定到 window 生命周期）
installViewMode()

app.mount('#app')
