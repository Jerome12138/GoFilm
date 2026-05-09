import { expect, test } from '@playwright/test'
import { fakeLogin, muteImages, waitAppReady } from './helpers'

/**
 * 管理端：登录 + 仪表盘
 *  - LoginView：未登录访问 /manage/index 应跳 /login
 *  - LoginView：填表单提交 → 跳转 manage/index
 *  - 已登录访问 /login 应自动跳走（兜底逻辑）
 *  - DashboardView：能渲染（mock DASHBOARD_STAT）
 */

test.describe('管理端鉴权 + 仪表盘', () => {
  test('未登录访问 /manage/index → 重定向 /login', async ({ page }) => {
    await muteImages(page)
    await page.goto('/manage/index')
    await page.waitForURL(/\/login/, { timeout: 8_000 })
    expect(page.url()).toContain('/login')
  })

  test('登录页表单提交后跳转管理首页', async ({ page }) => {
    await muteImages(page)
    await page.goto('/login')
    await waitAppReady(page)

    await page.locator('input[autocomplete="username"]').fill('admin')
    await page.locator('input[autocomplete="current-password"]').fill('123456')
    await page.locator('button[type="submit"]').click()

    await page.waitForURL(/\/manage\/index/, { timeout: 8_000 })
    expect(page.url()).toContain('/manage/index')
  })

  test('已登录访问 /login 兜底跳转', async ({ page }) => {
    await fakeLogin(page)
    await page.goto('/login')
    await page.waitForURL(/\/manage\/index/, { timeout: 8_000 })
    expect(page.url()).toContain('/manage/index')
  })

  test('Dashboard 页面渲染', async ({ page }) => {
    await fakeLogin(page)
    await muteImages(page)
    await page.goto('/manage/index')
    await waitAppReady(page)
    // 期望页面里有"仪表"或"控制台"或类似标题；至少 ManageHeader 可见
    const header = page.locator('header').first()
    await expect(header).toBeVisible({ timeout: 5_000 })
  })

  test('密码错误（mock 无校验）也能登录 — 因为是演示，仅作 smoke', async ({ page }) => {
    await muteImages(page)
    await page.goto('/login')
    await waitAppReady(page)
    await page.locator('input[autocomplete="username"]').fill('demo')
    await page.locator('input[autocomplete="current-password"]').fill('any')
    await page.locator('button[type="submit"]').click()
    await page.waitForURL(/\/manage\/index/, { timeout: 8_000 })
  })
})
