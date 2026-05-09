import { defineStore } from 'pinia'
import { ref, toRaw } from 'vue'
import { COOKIE_KEYS, getCookie, setCookie } from '@/utils/cookie'

/**
 * 观看历史
 * - 沿用旧站 cookie key `filmHistory`
 * - 内存映射 list 提供给 UI 渲染
 * - 写入时 toRaw 避免 reactive proxy 序列化问题（旧站踩过）
 */
export interface HistoryItem {
  id: string
  name: string
  picture: string
  source: string
  episode: string
  currentTime: number
  updatedAt: number
}

const MAX_ITEMS = 100

function loadFromCookie(): HistoryItem[] {
  const raw = getCookie(COOKIE_KEYS.FILM_HISTORY)
  if (!raw) {
    return []
  }
  try {
    const parsed = JSON.parse(raw) as unknown
    if (Array.isArray(parsed)) {
      return parsed.filter((it): it is HistoryItem => typeof it === 'object' && it !== null)
    }
  } catch {
    // ignore
  }
  return []
}

function persist(items: HistoryItem[]): void {
  const plain = items.map((it) => ({ ...toRaw(it) }))
  setCookie(COOKIE_KEYS.FILM_HISTORY, JSON.stringify(plain), 30)
}

export const useHistoryStore = defineStore('history', () => {
  const list = ref<HistoryItem[]>(loadFromCookie())

  function record(item: HistoryItem): void {
    const idx = list.value.findIndex((x) => x.id === item.id)
    const next = { ...item, updatedAt: Date.now() }
    if (idx >= 0) {
      list.value.splice(idx, 1)
    }
    list.value.unshift(next)
    if (list.value.length > MAX_ITEMS) {
      list.value.length = MAX_ITEMS
    }
    persist(list.value)
  }

  function remove(id: string): void {
    list.value = list.value.filter((x) => x.id !== id)
    persist(list.value)
  }

  function clear(): void {
    list.value = []
    persist(list.value)
  }

  return { list, record, remove, clear }
})
