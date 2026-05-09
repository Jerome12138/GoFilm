import { computed, onScopeDispose, ref, watch, type ComputedRef, type Ref } from 'vue'

/**
 * 三档视图模式：mobile / desktop / tv
 *
 * 触发 TV 模式优先级（高 → 低）：
 *  1. localStorage['gf-mode'] = 'tv'
 *  2. URL ?mode=tv
 *  3. UA 命中 SmartTV / Tizen / WebOS / HbbTV / Hisense / MiTV / Android TV / AFT[A-Z]+
 *  4. 视口 ≥ 1920 且 (hover: none)
 *
 * 否则按视口宽度：< 768 → mobile，>= 768 → desktop
 *
 * 写入 <html data-mode="...">，监听 resize 自动切换 mobile/desktop。
 * 用户手动 setMode 后写 localStorage 持久化。
 */

export type ViewMode = 'mobile' | 'desktop' | 'tv'

const STORAGE_KEY = 'gf-mode'
const TV_UA_REGEX =
  /SmartTV|Tizen|WebOS|HbbTV|Hisense|MiTV|Android TV|AFT[A-Z]+|GoogleTV|AppleTV|BRAVIA|VIDAA/i

let installed = false
const mode = ref<ViewMode>('desktop')

function readUrlMode(): ViewMode | null {
  if (typeof window === 'undefined') {
    return null
  }
  const sp = new URLSearchParams(window.location.search)
  const v = sp.get('mode')
  if (v === 'tv' || v === 'mobile' || v === 'desktop') {
    return v
  }
  return null
}

function readPersistedMode(): ViewMode | null {
  try {
    const v = localStorage.getItem(STORAGE_KEY)
    if (v === 'tv' || v === 'mobile' || v === 'desktop') {
      return v
    }
  } catch {
    // ignore
  }
  return null
}

function detectTV(): boolean {
  if (typeof window === 'undefined') {
    return false
  }
  if (TV_UA_REGEX.test(navigator.userAgent)) {
    return true
  }
  const w = window.innerWidth
  if (w >= 1920) {
    const noHover = window.matchMedia?.('(hover: none)').matches ?? false
    if (noHover) {
      return true
    }
  }
  return false
}

function detectMode(): ViewMode {
  // 优先级：persisted > URL > auto
  const persisted = readPersistedMode()
  if (persisted) {
    return persisted
  }
  const urlMode = readUrlMode()
  if (urlMode) {
    return urlMode
  }
  if (detectTV()) {
    return 'tv'
  }
  if (typeof window === 'undefined') {
    return 'desktop'
  }
  return window.innerWidth < 768 ? 'mobile' : 'desktop'
}

function applyMode(value: ViewMode): void {
  if (typeof document === 'undefined') {
    return
  }
  document.documentElement.setAttribute('data-mode', value)
}

function install(): void {
  if (installed) {
    return
  }
  installed = true
  mode.value = detectMode()
  applyMode(mode.value)

  if (typeof window === 'undefined') {
    return
  }

  const onResize = (): void => {
    const persisted = readPersistedMode()
    if (persisted) {
      // 用户手动选择优先，不被 resize 覆盖
      return
    }
    if (mode.value === 'tv' && detectTV()) {
      return
    }
    const w = window.innerWidth
    const next: ViewMode =
      detectTV() ? 'tv' : w < 768 ? 'mobile' : 'desktop'
    if (mode.value !== next) {
      mode.value = next
      applyMode(next)
    }
  }
  window.addEventListener('resize', onResize)

  // 监听 storage 事件，多 tab 同步
  const onStorage = (e: StorageEvent): void => {
    if (e.key === STORAGE_KEY) {
      mode.value = detectMode()
      applyMode(mode.value)
    }
  }
  window.addEventListener('storage', onStorage)

  onScopeDispose(() => {
    window.removeEventListener('resize', onResize)
    window.removeEventListener('storage', onStorage)
  })
}

function setMode(value: ViewMode | null): void {
  try {
    if (value === null) {
      localStorage.removeItem(STORAGE_KEY)
    } else {
      localStorage.setItem(STORAGE_KEY, value)
    }
  } catch {
    // ignore
  }
  mode.value = value ?? detectMode()
  applyMode(mode.value)
}

export function useViewMode(): {
  mode: Ref<ViewMode>
  setMode: (value: ViewMode | null) => void
  isMobile: ComputedRef<boolean>
  isDesktop: ComputedRef<boolean>
  isTV: ComputedRef<boolean>
} {
  install()

  // 保险起见，watch mode 时再次同步 DOM（避免 SSR / 早期 install 失败）
  watch(mode, applyMode, { immediate: true })

  return {
    mode,
    setMode,
    isMobile: computed(() => mode.value === 'mobile'),
    isDesktop: computed(() => mode.value === 'desktop'),
    isTV: computed(() => mode.value === 'tv')
  }
}
