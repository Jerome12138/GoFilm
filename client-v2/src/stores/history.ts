import { defineStore } from 'pinia'
import { ref, toRaw } from 'vue'
import { COOKIE_KEYS, getCookie, setCookie } from '@/utils/cookie'

/**
 * 观看历史 store
 *
 * 沿用旧站 cookie key `filmHistory`，使用旧站的 **对象映射** 结构：
 *   { [filmId]: { name, link, episode, timeStamp, picture? } }
 *
 * link 形如：`/play?id=xxx&source=yyy&episode=z&currentTime=NN`
 *
 * 仍然提供 list() 返回按 timeStamp 倒序排序的数组，便于 HistoryView 渲染。
 *
 * 写入时 toRaw 避免 reactive proxy 序列化问题（旧站踩过）。
 *
 * 同时双写 localStorage（key 同名），减少 cookie 4KB 限制带来的丢失风险。
 */

export interface HistoryRecord {
  /** 影片 ID */
  id: string
  name: string
  /** 完整跳转链接 `/play?id=...&source=...&episode=...&currentTime=...` */
  link: string
  /** 集数显示名（来自 PlayEpisode.episode） */
  episode: string
  timeStamp: number
  /** 播放源 ID，便于浮层重新拼参数（新站新增字段，旧站读取时忽略） */
  source?: string
  /** 集数索引（detail.list[i].linkList[idx]），便于跳转重建 query */
  episodeIndex?: number
  /** 进度（秒），便于继续播放 */
  currentTime?: number
  /** 海报（新站新增字段） */
  picture?: string
}

export type HistoryMap = Record<string, HistoryRecord>

const MAX_ITEMS = 100
const LS_KEY = COOKIE_KEYS.FILM_HISTORY

function safeParse(raw: string): HistoryMap {
  if (!raw) {
    return {}
  }
  try {
    const parsed = JSON.parse(raw) as unknown
    if (!parsed || typeof parsed !== 'object') {
      return {}
    }
    // 兼容历史 array 形式（早期 STORY-006 阶段曾用 array）
    if (Array.isArray(parsed)) {
      const out: HistoryMap = {}
      for (const it of parsed) {
        if (it && typeof it === 'object' && typeof (it as HistoryRecord).id === 'string') {
          const r = it as HistoryRecord
          out[r.id] = r
        }
      }
      return out
    }
    const map = parsed as Record<string, unknown>
    const out: HistoryMap = {}
    for (const [k, v] of Object.entries(map)) {
      if (v && typeof v === 'object') {
        out[k] = v as HistoryRecord
      }
    }
    return out
  } catch {
    return {}
  }
}

function loadInitial(): HistoryMap {
  // 优先 cookie（与旧站打通），失败回退 localStorage
  const fromCookie = safeParse(getCookie(COOKIE_KEYS.FILM_HISTORY))
  if (Object.keys(fromCookie).length > 0) {
    return fromCookie
  }
  if (typeof localStorage !== 'undefined') {
    try {
      return safeParse(localStorage.getItem(LS_KEY) ?? '')
    } catch {
      return {}
    }
  }
  return {}
}

function persist(map: HistoryMap): void {
  // 截断防止 cookie 超长（按 timeStamp 倒序保留 MAX_ITEMS）
  const items = Object.values(map)
    .sort((a, b) => (b.timeStamp ?? 0) - (a.timeStamp ?? 0))
    .slice(0, MAX_ITEMS)
  const trimmed: HistoryMap = {}
  for (const it of items) {
    trimmed[it.id] = { ...toRaw(it) }
  }
  const json = JSON.stringify(trimmed)
  setCookie(COOKIE_KEYS.FILM_HISTORY, json, 30)
  if (typeof localStorage !== 'undefined') {
    try {
      localStorage.setItem(LS_KEY, json)
    } catch {
      // quota exceeded — ignore
    }
  }
}

export const useHistoryStore = defineStore('history', () => {
  /** 内部存储以 map 形式（与 cookie 一致） */
  const map = ref<HistoryMap>(loadInitial())

  /**
   * 派生数组：按 timeStamp 倒序，HistoryView 用
   * 注：旧站点字段不含 picture / id 但 detail 接口有 — record() 写入时尽量补齐
   */
  const list = ref<HistoryRecord[]>([])

  function refreshList(): void {
    list.value = Object.values(map.value).sort(
      (a, b) => (b.timeStamp ?? 0) - (a.timeStamp ?? 0)
    )
  }

  refreshList()

  /** 写入一条记录（同 id 合并） */
  function record(item: Omit<HistoryRecord, 'timeStamp'> & { timeStamp?: number }): void {
    if (!item.id) {
      return
    }
    const next: HistoryRecord = {
      id: String(item.id),
      name: item.name,
      link: item.link,
      episode: item.episode,
      picture: item.picture,
      timeStamp: item.timeStamp ?? Date.now()
    }
    map.value[next.id] = next
    persist(map.value)
    refreshList()
  }

  function get(id: string): HistoryRecord | undefined {
    return map.value[id]
  }

  function remove(id: string): void {
    if (map.value[id]) {
      delete map.value[id]
      persist(map.value)
      refreshList()
    }
  }

  function clear(): void {
    map.value = {}
    persist(map.value)
    refreshList()
  }

  return { map, list, record, get, remove, clear }
})
