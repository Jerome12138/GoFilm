import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { publicRoutes } from './routes.public'
import { manageRoutes } from './routes.manage'
import { registerGuards } from './guards'

/** 扩展 RouteMeta 类型 */
declare module 'vue-router' {
  interface RouteMeta {
    /** 布局壳 */
    layout?: 'public' | 'manage' | 'auth' | 'minimal'
    /** 是否需要登录 */
    requiresAuth?: boolean
    /** 浏览器标题 */
    title?: string
    /** keepAlive 缓存 */
    keepAlive?: boolean
  }
}

const routes: RouteRecordRaw[] = [
  ...publicRoutes,
  ...manageRoutes,
  {
    path: '/:pathMatch(.*)*',
    name: 'not-found',
    component: () => import('@/views/error/NotFoundView.vue'),
    meta: { layout: 'minimal', title: '页面不存在' }
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes,
  scrollBehavior(to, from, savedPosition) {
    if (savedPosition) {
      return savedPosition
    }
    if (to.hash) {
      return { el: to.hash, behavior: 'smooth' }
    }
    return { top: 0 }
  }
})

registerGuards(router)

export default router
