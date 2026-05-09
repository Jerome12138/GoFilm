import { defineStore } from 'pinia'
import { ref } from 'vue'

/**
 * 全局 UI 状态：sidebar、主题、loading 计数
 * 由网络层 / 布局组件读写。
 */
export const useUIStore = defineStore('ui', () => {
  const sidebarCollapsed = ref(false)
  const theme = ref<'dark' | 'light'>('dark')

  const loadingCount = ref(0)
  const loading = ref(false)

  function pushLoading(): void {
    loadingCount.value++
    loading.value = true
  }

  function popLoading(): void {
    loadingCount.value = Math.max(0, loadingCount.value - 1)
    if (loadingCount.value === 0) {
      loading.value = false
    }
  }

  function toggleSidebar(): void {
    sidebarCollapsed.value = !sidebarCollapsed.value
  }

  function setTheme(t: 'dark' | 'light'): void {
    theme.value = t
  }

  return {
    sidebarCollapsed,
    theme,
    loading,
    loadingCount,
    pushLoading,
    popLoading,
    toggleSidebar,
    setTheme
  }
})
