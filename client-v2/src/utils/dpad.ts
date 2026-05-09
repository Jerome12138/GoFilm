/**
 * D-pad keycode → 标准 KeyboardEvent.key 映射
 * 来源：04-tv-addendum.md 3.3
 *
 * Android WebView / Smart TV 浏览器在按下遥控器时，会通过
 * keydown 派发数字 keyCode（无 key 字段或 key 不一致），需做归一化。
 */

export const DPAD_KEY_MAP: Readonly<Record<number, string>> = Object.freeze({
  19: 'ArrowUp', // KEYCODE_DPAD_UP
  20: 'ArrowDown', // KEYCODE_DPAD_DOWN
  21: 'ArrowLeft', // KEYCODE_DPAD_LEFT
  22: 'ArrowRight', // KEYCODE_DPAD_RIGHT
  23: 'Enter', // KEYCODE_DPAD_CENTER
  66: 'Enter', // KEYCODE_ENTER
  4: 'Escape', // KEYCODE_BACK
  82: 'ContextMenu', // KEYCODE_MENU
  85: 'MediaPlayPause',
  87: 'MediaTrackNext',
  88: 'MediaTrackPrevious',
  89: 'MediaRewind',
  90: 'MediaFastForward'
})

/** 把 KeyboardEvent 归一化为标准 key（D-pad keyCode 优先） */
export function normalizeDpadKey(e: KeyboardEvent): string {
  const mapped = DPAD_KEY_MAP[e.keyCode]
  return mapped ?? e.key
}

/**
 * 安装全局 D-pad 监听：把 keyCode 19/20/21/22/23 转换成标准 key
 *
 * 设计要点：
 *  - capture 阶段拦截，preventDefault 阻止原生 D-pad 默认行为（焦点漂移、滚动）
 *  - 派发一个克隆事件（标准 key 字段），冒泡阶段被 useSpatialNavigation 接住
 *  - dispatch 目标优先 e.target，否则 document.activeElement（避免 e.target 为 null）
 *  - 检测 e.key 已是标准 key 时直接放行，避免无限循环
 *
 * 兼容：Capacitor Android WebView / Smart TV 浏览器（Tizen / WebOS）
 */
export function installDpadBridge(opts: { preventBack?: boolean } = {}): () => void {
  if (typeof window === 'undefined') {
    return () => {
      /* noop */
    }
  }
  const handler = (e: KeyboardEvent): void => {
    const key = DPAD_KEY_MAP[e.keyCode]
    if (!key) {
      return
    }
    // 已有标准 key 的事件无需再派发，避免循环
    if (e.key === key) {
      return
    }
    if (opts.preventBack && key === 'Escape') {
      e.preventDefault()
    }
    // 阻止原生 D-pad 行为（系统级焦点漂移、滚动）
    e.preventDefault()

    // 派发标准化事件，让焦点系统统一接住
    const next = new KeyboardEvent('keydown', {
      key,
      keyCode: e.keyCode,
      bubbles: true,
      cancelable: true
    })
    const target =
      (e.target as EventTarget | null) ||
      (document.activeElement as EventTarget | null) ||
      window
    target.dispatchEvent(next)
  }
  window.addEventListener('keydown', handler, true)
  return () => window.removeEventListener('keydown', handler, true)
}
