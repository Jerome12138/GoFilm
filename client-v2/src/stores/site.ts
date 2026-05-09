import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { SiteBasic } from '@/types/manage'

/**
 * 站点信息（siteName / logo 等），仅首次加载，全应用共享
 */
export const useSiteStore = defineStore('site', () => {
  const basic = ref<SiteBasic | null>(null)
  const loaded = ref(false)
  const loading = ref(false)

  async function ensureLoaded(): Promise<void> {
    if (loaded.value || loading.value) {
      return
    }
    loading.value = true
    try {
      const { getSiteBasic } = await import('@/api/film')
      basic.value = await getSiteBasic()
      loaded.value = true
    } finally {
      loading.value = false
    }
  }

  function reset(): void {
    basic.value = null
    loaded.value = false
  }

  return { basic, loaded, loading, ensureLoaded, reset }
})
