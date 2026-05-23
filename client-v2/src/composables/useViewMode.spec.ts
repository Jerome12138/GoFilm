import { describe, it, expect, beforeEach, vi } from 'vitest'

describe('useViewMode 四档检测', () => {
  beforeEach(() => {
    localStorage.removeItem('gf-mode')
    document.documentElement.removeAttribute('data-mode')
    vi.resetModules()
  })

  it('视口 600px → mobile', async () => {
    Object.defineProperty(window, 'innerWidth', { value: 600, configurable: true })
    const m = await import('@/composables/useViewMode')
    m.installViewMode()
    const v = m.useViewMode()
    expect(v.mode.value).toBe('mobile')
    expect(v.isMobile.value).toBe(true)
    expect(v.isTablet.value).toBe(false)
    expect(v.isNarrow.value).toBe(true)
  })

  it('视口 900px → tablet', async () => {
    Object.defineProperty(window, 'innerWidth', { value: 900, configurable: true })
    const m = await import('@/composables/useViewMode')
    m.installViewMode()
    const v = m.useViewMode()
    expect(v.mode.value).toBe('tablet')
    expect(v.isTablet.value).toBe(true)
    expect(v.isNarrow.value).toBe(true)
  })

  it('视口 1280px → desktop', async () => {
    Object.defineProperty(window, 'innerWidth', { value: 1280, configurable: true })
    const m = await import('@/composables/useViewMode')
    m.installViewMode()
    const v = m.useViewMode()
    expect(v.mode.value).toBe('desktop')
    expect(v.isNarrow.value).toBe(false)
  })
})
