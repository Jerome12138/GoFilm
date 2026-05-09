import { createApp } from 'vue'
import { createPinia } from 'pinia'

import 'virtual:uno.css'
import '@/assets/styles/reset.css'
import '@/assets/styles/theme.css'
import '@/assets/styles/iconfont.css'

import App from './App.vue'
import router from './router'
import { installViewMode } from '@/composables/useViewMode'

const app = createApp(App)
const pinia = createPinia()

app.use(pinia)
app.use(router)

// 全局安装一次 viewMode（resize / storage 监听绑定到 window 生命周期）
installViewMode()

app.mount('#app')
