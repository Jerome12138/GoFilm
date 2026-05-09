import type { Router } from 'vue-router'
import { getToken } from '@/utils/token'

const APP_TITLE = (import.meta.env.VITE_APP_TITLE as string | undefined) ?? 'GoFilm'

/** 注册全局前后置守卫 */
export function registerGuards(router: Router): void {
  router.beforeEach((to) => {
    if (to.meta.requiresAuth) {
      const token = getToken()
      if (!token || !token.value) {
        return {
          path: '/login',
          query: { redirect: to.fullPath }
        }
      }
    }

    // TV 模式禁止访问 /manage（在 useViewMode install 后生效）
    if (typeof document !== 'undefined') {
      const mode = document.documentElement.getAttribute('data-mode')
      if (mode === 'tv' && to.path.startsWith('/manage')) {
        return { path: '/index' }
      }
    }

    return true
  })

  router.afterEach((to) => {
    const t = (to.meta.title as string | undefined) ?? ''
    document.title = t ? `${t} - ${APP_TITLE}` : APP_TITLE
  })
}
