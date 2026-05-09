import { expect, type Page } from '@playwright/test'

/**
 * 等待"应用就绪"——网络空闲 + #app 可见 + 至少一次拦截器响应。
 * 不依赖具体 selector，留给各 spec 用 expect 断言后续元素。
 */
export async function waitAppReady(page: Page): Promise<void> {
  await expect(page.locator('#app')).toBeVisible()
  // 让 mock 拦截器和 base toast 至少跑过一轮
  await page.waitForLoadState('networkidle', { timeout: 10_000 }).catch(() => {})
}

/**
 * 模拟登录：写真实结构（utils/token.ts 用 'auth' key + JSON 包装）
 *  { key: 'auth-token', value: '<token>' }
 */
export async function fakeLogin(page: Page): Promise<void> {
  await page.addInitScript(() => {
    const auth = { key: 'auth-token', value: 'mock-token-e2e' }
    localStorage.setItem('auth', JSON.stringify(auth))
  })
}

/** 把视图模式写到 localStorage（强制 TV 等） */
export async function setViewMode(
  page: Page,
  mode: 'mobile' | 'desktop' | 'tv'
): Promise<void> {
  await page.addInitScript((m) => {
    localStorage.setItem('gf-mode', m)
  }, mode)
}

/** 把首屏图片加载策略改快：插入 CSS 隐藏所有 img onload 等待，仅做 DOM 可见性测试 */
export async function muteImages(page: Page): Promise<void> {
  await page.addStyleTag({
    content: `img { transition: none !important; opacity: 1 !important; }`
  })
}

/** 是否当前 project 视口属于"小屏" */
export function isMobile(viewport: { width: number; height: number } | null): boolean {
  if (!viewport) return false
  return viewport.width < 768
}
