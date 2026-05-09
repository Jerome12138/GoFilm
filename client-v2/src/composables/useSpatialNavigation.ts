/**
 * 空间导航 composable（TV 模式专用）
 *
 * 行为：
 *  - 仅在 <html data-mode="tv"> 下生效（非 tv 模式时 disable）
 *  - 监听全局 keydown，把方向键转化为 [data-focusable="true"] 元素之间的几何最近邻焦点切换
 *  - Enter / Space 在非 button/a/input 元素聚焦时主动派发 click（让原生处理）
 *  - Escape / Backspace（KeyCode 4 已被 dpad.ts 映射为 Escape）触发 router.back()
 *  - 路由切换时：旧路由焦点元素的稳定 ID 写 sessionStorage；新路由进入后尝试恢复，失败则聚焦第一个 focusable
 *  - D-pad keyCode 兼容已由 dpad.ts 在 main.ts 装载的 installDpadBridge 处理（派发标准 KeyboardEvent）
 *
 * 暴露：focusFirst / focusElement / enable / disable
 *
 * 使用：在 App.vue setup 内调用一次 useSpatialNavigation()
 *
 * 参考：04-tv-addendum.md 第 3 节
 */

import { onScopeDispose, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useViewMode } from '@/composables/useViewMode'
import { normalizeDpadKey } from '@/utils/dpad'

const FOCUSABLE_SELECTOR = '[data-focusable="true"]:not([disabled]):not([aria-hidden="true"])'
const FOCUS_MEMORY_PREFIX = 'gf-tv-focus:'
const FOCUS_RESTORE_DELAY = 50

interface Rect {
  el: HTMLElement
  cx: number
  cy: number
  top: number
  left: number
  right: number
  bottom: number
}

interface Api {
  focusFirst: (container?: HTMLElement | null) => boolean
  focusElement: (el: HTMLElement | null) => boolean
  enable: () => void
  disable: () => void
}

let installed = false

/** 元素是否在视口内（中心点判定） */
function isVisible(el: HTMLElement): boolean {
  const r = el.getBoundingClientRect()
  if (r.width === 0 || r.height === 0) return false
  const style = window.getComputedStyle(el)
  if (style.visibility === 'hidden' || style.display === 'none') return false
  return true
}

/** 拿当前 DOM 中所有可见 focusable */
function getCandidates(container?: HTMLElement | null): HTMLElement[] {
  const root = container ?? document.body
  const list = Array.from(
    root.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTOR)
  )
  return list.filter(isVisible)
}

function rectOf(el: HTMLElement): Rect {
  const r = el.getBoundingClientRect()
  return {
    el,
    cx: r.left + r.width / 2,
    cy: r.top + r.height / 2,
    top: r.top,
    left: r.left,
    right: r.right,
    bottom: r.bottom
  }
}

/**
 * 在指定方向上寻找最近邻
 *
 * 评分公式：主轴距离 + 0.5 × 副轴偏移（让正前方优先）
 * 如果主轴上没有任何候选（同行/同列），fallback 选择全局欧氏最近的同方向元素
 */
function findNearest(
  current: HTMLElement,
  dir: 'up' | 'down' | 'left' | 'right'
): HTMLElement | null {
  const all = getCandidates()
  if (all.length === 0) return null
  const cur = rectOf(current)
  let best: { el: HTMLElement; score: number } | null = null

  for (const cand of all) {
    if (cand === current) continue
    const r = rectOf(cand)
    let primary = 0
    let secondary = 0
    let valid = false
    switch (dir) {
      case 'up':
        if (r.bottom <= cur.top + 1) {
          primary = cur.top - r.bottom
          secondary = Math.abs(r.cx - cur.cx)
          valid = true
        }
        break
      case 'down':
        if (r.top >= cur.bottom - 1) {
          primary = r.top - cur.bottom
          secondary = Math.abs(r.cx - cur.cx)
          valid = true
        }
        break
      case 'left':
        if (r.right <= cur.left + 1) {
          primary = cur.left - r.right
          secondary = Math.abs(r.cy - cur.cy)
          valid = true
        }
        break
      case 'right':
        if (r.left >= cur.right - 1) {
          primary = r.left - cur.right
          secondary = Math.abs(r.cy - cur.cy)
          valid = true
        }
        break
    }
    if (!valid) continue
    // 完全错开行列时（例：左侧但比当前高很多），加大惩罚
    const overlapPenalty = secondary > Math.max(cur.bottom - cur.top, cur.right - cur.left) ? 1.5 : 1
    const score = primary + secondary * 0.5 * overlapPenalty
    if (!best || score < best.score) {
      best = { el: cand, score }
    }
  }

  return best?.el ?? null
}

/** 把焦点移动到 el，并 scrollIntoView 居中 */
function focusAndScroll(el: HTMLElement | null): boolean {
  if (!el) return false
  try {
    el.focus({ preventScroll: true })
  } catch {
    el.focus()
  }
  el.scrollIntoView({ block: 'center', inline: 'center', behavior: 'smooth' })
  return true
}

/** 给元素生成稳定 ID（用于 sessionStorage 记忆） */
function stableIdOf(el: HTMLElement): string {
  if (el.id) return `#${el.id}`
  const name = el.getAttribute('name')
  if (name) return `name:${name}`
  // 优先 data-* 属性
  const tagAttrs: string[] = []
  for (const attr of ['data-key', 'data-id', 'data-href', 'href', 'aria-label']) {
    const v = el.getAttribute(attr)
    if (v) {
      tagAttrs.push(`${attr}=${v}`)
    }
  }
  if (tagAttrs.length) {
    return `${el.tagName.toLowerCase()}|${tagAttrs.join('|')}`
  }
  // 兜底用 DOM 路径 + index
  const candidates = getCandidates()
  const idx = candidates.indexOf(el)
  return `idx:${idx}`
}

function findElementByStableId(id: string): HTMLElement | null {
  if (id.startsWith('#')) {
    return document.querySelector<HTMLElement>(id)
  }
  if (id.startsWith('idx:')) {
    const idx = Number(id.slice(4))
    const list = getCandidates()
    return list[idx] ?? null
  }
  if (id.startsWith('name:')) {
    return document.querySelector<HTMLElement>(`[name="${id.slice(5)}"]`)
  }
  // tag|attr=value|... 形式
  const [tag, ...attrs] = id.split('|')
  if (!tag || attrs.length === 0) return null
  const sel = `${tag}${attrs
    .map((a) => {
      const eq = a.indexOf('=')
      if (eq < 0) return ''
      const k = a.slice(0, eq)
      const v = a.slice(eq + 1).replace(/"/g, '\\"')
      return `[${k}="${v}"]`
    })
    .join('')}`
  return document.querySelector<HTMLElement>(sel)
}

export function useSpatialNavigation(): Api {
  const { isTV } = useViewMode()
  const route = useRoute()
  const router = useRouter()

  let active = false
  let lastFocusPath = route.fullPath

  function isEditingTarget(t: EventTarget | null): boolean {
    const el = t as HTMLElement | null
    if (!el) return false
    const tag = el.tagName?.toLowerCase()
    if (tag === 'input' || tag === 'textarea' || tag === 'select') return true
    if (el.isContentEditable) return true
    return false
  }

  /** 当前 focus 落在哪个 focusable，未命中则返回首个 */
  function currentFocusable(): HTMLElement | null {
    const ae = document.activeElement as HTMLElement | null
    if (ae && ae.matches(FOCUSABLE_SELECTOR)) return ae
    if (ae && ae.closest) {
      const inside = ae.closest(FOCUSABLE_SELECTOR) as HTMLElement | null
      if (inside) return inside
    }
    return getCandidates()[0] ?? null
  }

  function move(dir: 'up' | 'down' | 'left' | 'right'): boolean {
    const cur = currentFocusable()
    if (!cur) {
      const first = getCandidates()[0]
      return focusAndScroll(first ?? null)
    }
    const next = findNearest(cur, dir)
    return focusAndScroll(next)
  }

  function handleKeydown(e: KeyboardEvent): void {
    if (!active) return
    if (isEditingTarget(e.target)) {
      // 输入框聚焦：仅 Escape 拦截做返回
      const k = normalizeDpadKey(e)
      if (k === 'Escape') {
        ;(e.target as HTMLElement).blur()
      }
      return
    }
    const key = normalizeDpadKey(e)

    switch (key) {
      case 'ArrowUp':
        if (move('up')) e.preventDefault()
        break
      case 'ArrowDown':
        if (move('down')) e.preventDefault()
        break
      case 'ArrowLeft':
        if (move('left')) e.preventDefault()
        break
      case 'ArrowRight':
        if (move('right')) e.preventDefault()
        break
      case 'Enter':
      case ' ':
      case 'Spacebar': {
        const el = currentFocusable()
        if (!el) return
        const tag = el.tagName?.toLowerCase()
        // 原生可点击元素（button / a）会自然处理 Enter，跳过
        if (tag === 'button' || tag === 'a' || tag === 'input') return
        e.preventDefault()
        el.click()
        break
      }
      case 'Escape':
      case 'Backspace': {
        // 输入控件已上面提前处理；这里只处理整页返回
        e.preventDefault()
        // history.length 兜底：回不去就回首页
        if (window.history.length > 1) {
          router.back()
        } else {
          void router.push('/index')
        }
        break
      }
      default:
        break
    }
  }

  /** 记忆当前路由焦点 */
  function rememberFocus(): void {
    try {
      const ae = document.activeElement as HTMLElement | null
      if (!ae || !ae.matches(FOCUSABLE_SELECTOR)) return
      const id = stableIdOf(ae)
      sessionStorage.setItem(FOCUS_MEMORY_PREFIX + lastFocusPath, id)
    } catch {
      /* ignore */
    }
  }

  /** 恢复焦点：先按记忆，否则聚焦首个 */
  function restoreFocus(path: string): void {
    let target: HTMLElement | null = null
    try {
      const id = sessionStorage.getItem(FOCUS_MEMORY_PREFIX + path)
      if (id) {
        target = findElementByStableId(id)
      }
    } catch {
      /* ignore */
    }
    if (!target) {
      target = getCandidates()[0] ?? null
    }
    if (target) {
      // 仅当 body 上没有有效焦点（默认 body）时才接管，避免覆盖组件内部 autofocus
      const ae = document.activeElement
      const noActive = !ae || ae === document.body || (ae as HTMLElement).tagName === 'BODY'
      if (noActive) {
        focusAndScroll(target)
      }
    }
  }

  function enable(): void {
    if (active) return
    active = true
    window.addEventListener('keydown', handleKeydown, true)
  }

  function disable(): void {
    if (!active) return
    active = false
    window.removeEventListener('keydown', handleKeydown, true)
  }

  function focusFirst(container?: HTMLElement | null): boolean {
    const list = getCandidates(container)
    return focusAndScroll(list[0] ?? null)
  }

  function focusElement(el: HTMLElement | null): boolean {
    return focusAndScroll(el)
  }

  // 路由切换：先记忆旧路径焦点，再延迟恢复新路径
  watch(
    () => route.fullPath,
    (next, prev) => {
      lastFocusPath = prev ?? next
      rememberFocus()
      lastFocusPath = next
      if (!active) return
      // 等组件渲染完成（页面切换 transition + 异步数据）
      window.setTimeout(() => {
        if (!active) return
        restoreFocus(next)
      }, FOCUS_RESTORE_DELAY)
    }
  )

  // beforeunload 也尝试记一次
  const onBeforeUnload = (): void => {
    rememberFocus()
  }
  if (typeof window !== 'undefined') {
    window.addEventListener('beforeunload', onBeforeUnload)
  }

  // 跟 viewMode 联动：进入 TV → enable，离开 → disable
  watch(
    isTV,
    (v) => {
      if (v) {
        enable()
        // 首次进入也尝试聚焦（路由可能已就位）
        window.setTimeout(() => {
          if (active) restoreFocus(route.fullPath)
        }, FOCUS_RESTORE_DELAY)
      } else {
        disable()
      }
    },
    { immediate: true }
  )

  onScopeDispose(() => {
    disable()
    if (typeof window !== 'undefined') {
      window.removeEventListener('beforeunload', onBeforeUnload)
    }
  })

  return { focusFirst, focusElement, enable, disable }
}

/** 防止重复 install 的简便包装（App.vue 内调用） */
export function installSpatialNavigationOnce(): Api | null {
  if (installed) return null
  installed = true
  return useSpatialNavigation()
}
