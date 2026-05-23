import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import ManageSidebar from './ManageSidebar.vue'

const router = createRouter({
  history: createMemoryHistory(),
  routes: [{ path: '/:pathMatch(.*)*', component: { template: '<div />' } }]
})

function mountSidebar(props: Record<string, unknown> = {}) {
  return mount(ManageSidebar, {
    props,
    global: { plugins: [router] }
  })
}

describe('ManageSidebar variant', () => {
  beforeEach(() => setActivePinia(createPinia()))

  it('variant=drawer + open=true → 遮罩 div 存在 (含 fixed inset-0 class)', () => {
    const w = mountSidebar({ variant: 'drawer', open: true })
    // 遮罩 div 是 aside 同级前置元素, 含 fixed + inset-0 + bg-black
    const overlay = w.findAll('div').find(d => {
      const cls = d.classes()
      return cls.includes('fixed') && cls.includes('inset-0')
    })
    expect(overlay).toBeTruthy()
  })

  it('variant=drawer + open=false → 无遮罩', () => {
    const w = mountSidebar({ variant: 'drawer', open: false })
    const overlay = w.findAll('div').find(d => {
      const cls = d.classes()
      return cls.includes('fixed') && cls.includes('inset-0')
    })
    expect(overlay).toBeUndefined()
  })

  it('drawer + open=false → aside -translate-x-full', () => {
    const w = mountSidebar({ variant: 'drawer', open: false })
    expect(w.find('aside').classes()).toContain('-translate-x-full')
  })

  it('drawer + open=true → aside translate-x-0', () => {
    const w = mountSidebar({ variant: 'drawer', open: true })
    expect(w.find('aside').classes()).toContain('translate-x-0')
  })

  it('variant=icon-rail → 强制 w-[64px]', () => {
    const w = mountSidebar({ variant: 'icon-rail' })
    expect(w.find('aside').classes()).toContain('w-[64px]')
  })

  it('variant=full + uiStore.sidebarCollapsed 默认 false → w-[220px]', () => {
    const w = mountSidebar({ variant: 'full' })
    expect(w.find('aside').classes()).toContain('w-[220px]')
  })

  it('drawer 模式点击遮罩 emit close', async () => {
    const w = mountSidebar({ variant: 'drawer', open: true })
    const overlay = w.findAll('div').find(d => {
      const cls = d.classes()
      return cls.includes('fixed') && cls.includes('inset-0')
    })
    await overlay!.trigger('click')
    expect(w.emitted('close')).toBeTruthy()
  })

  it('drawer 模式点击菜单 item emit close', async () => {
    const w = mountSidebar({ variant: 'drawer', open: true })
    const firstLink = w.find('a')
    await firstLink.trigger('click')
    expect(w.emitted('close')).toBeTruthy()
  })

  it('full 模式点击菜单 item 不 emit close', async () => {
    const w = mountSidebar({ variant: 'full' })
    const firstLink = w.find('a')
    await firstLink.trigger('click')
    expect(w.emitted('close')).toBeUndefined()
  })
})
