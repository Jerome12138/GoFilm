import { defineStore, storeToRefs } from 'pinia'
import { ref, toRaw, watch } from 'vue'
import { COOKIE_KEYS, getCookie, setCookie } from '@/utils/cookie'
import { logger } from '@/utils/logger'
import { useUserStore } from './user'

/**
 * 观看历史 store —— 登录态自动切换 远端 / 本地
 *
 * 设计:
 *  - **未登录**: 真实存储 = cookie + localStorage (兼容旧站 key `filmHistory`,
 *    对象映射 `{ [filmId]: HistoryRecord }`); 一切写入持久化到本地.
 *  - **登录**: 真实存储 = 后端 `/user/history` 接口. 本地 cookie/LS 不再被新写覆盖,
 *    保留登录前的"匿名快照", 退出登录后还原.
 *  - **登录瞬间**: 把本地所有记录 push 一遍到后端 (upsert 幂等), 然后 list() 拉云端
 *    并替换内存 map. 这样以前匿名看的东西不会丢.
 *  - **登出瞬间**: 重新从 cookie/LS 加载, 退回本地视图.
 *
 * 读路径 (PlayView/FilmDetailView 调 store.get(id)) 始终走内存 map, 性能足够;
 * 写路径在 remote 模式下 fire-and-forget POST, 不阻塞 UI.
 *
 * link 形如：`/play?id=xxx&source=yyy&episode=z&currentTime=NN`
 */

export interface HistoryRecord {
  /** 影片 ID (字符串形式, mid 的 toString) */
  id: string
  name: string
  /** 完整跳转链接 `/play?id=...&source=...&episode=...&currentTime=...` */
  link: string
  /** 集数显示名（来自 PlayEpisode.episode） */
  episode: string
  timeStamp: number
  /** 播放源 ID，便于浮层重新拼参数 */
  source?: string
  /** 集数索引（detail.list[i].linkList[idx]） */
  episodeIndex?: number
  /** 进度（秒），便于继续播放 */
  currentTime?: number
  /** 海报 */
  picture?: string
  /** 一级分类 (后端 pid, 仅 remote 模式下回写) */
  pid?: number
  /** 二级分类 (后端 cid) */
  cid?: number
  /** 视频总时长 (秒, 可选) */
  duration?: number
}

export type HistoryMap = Record<string, HistoryRecord>

const MAX_ITEMS = 100
const LS_KEY = COOKIE_KEYS.FILM_HISTORY
const SAFE_PLAY_LINK = /^\/play\?/

function safeParse(raw: string): HistoryMap {
  if (!raw) {
    return {}
  }
  try {
    const parsed = JSON.parse(raw) as unknown
    if (!parsed || typeof parsed !== 'object') {
      return {}
    }
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

function loadFromLocal(): HistoryMap {
  const fromCookie = safeParse(getCookie(COOKIE_KEYS.FILM_HISTORY))
  let fromLS: HistoryMap = {}
  if (typeof localStorage !== 'undefined') {
    try {
      fromLS = safeParse(localStorage.getItem(LS_KEY) ?? '')
    } catch {
      /* ignore */
    }
  }
  const merged: HistoryMap = { ...fromCookie }
  for (const [id, ls] of Object.entries(fromLS)) {
    const cookieRec = merged[id]
    if (!cookieRec || (ls.timeStamp ?? 0) > (cookieRec.timeStamp ?? 0)) {
      merged[id] = ls
    }
  }
  return merged
}

function persistLocal(map: HistoryMap): void {
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
      /* quota exceeded - ignore */
    }
  }
}

export const useHistoryStore = defineStore('history', () => {
  /** 内部存储以 map 形式 */
  const map = ref<HistoryMap>(loadFromLocal())
  const list = ref<HistoryRecord[]>([])
  /** 当前是否处于远端模式 (登录后) */
  const remoteMode = ref(false)
  /** 拉取远端的 loading (HistoryView 进度态可用) */
  const remoteLoading = ref(false)

  function refreshList(): void {
    list.value = Object.values(map.value).sort(
      (a, b) => (b.timeStamp ?? 0) - (a.timeStamp ?? 0)
    )
  }

  refreshList()

  /** 写入一条记录（同 id 合并）
   *  - 总是更新内存 map (供同会话内的 PlayView/FilmDetailView 即时读取)
   *  - 本地模式: 同时落 cookie/LS
   *  - 远端模式: fire-and-forget POST /user/history, 不写 cookie/LS
   */
  function record(item: Omit<HistoryRecord, 'timeStamp'> & { timeStamp?: number }): void {
    if (!item.id) {
      return
    }
    const safeLink =
      typeof item.link === 'string' && SAFE_PLAY_LINK.test(item.link) ? item.link : ''
    const next: HistoryRecord = {
      id: String(item.id),
      name: item.name,
      link: safeLink,
      episode: item.episode,
      picture: item.picture,
      source: item.source,
      episodeIndex: item.episodeIndex,
      currentTime: item.currentTime,
      pid: item.pid,
      cid: item.cid,
      duration: item.duration,
      timeStamp: item.timeStamp ?? Date.now()
    }
    map.value[next.id] = next
    if (!remoteMode.value) {
      persistLocal(map.value)
    } else {
      void pushRemote(next)
    }
    refreshList()
  }

  function get(id: string): HistoryRecord | undefined {
    return map.value[id]
  }

  /** 删除一条
   *  - 本地: 直接删 + persist
   *  - 远端: DELETE /user/history?mid=, 失败时仍把本地内存删除 (保持 UI 即时响应)
   */
  async function remove(id: string): Promise<void> {
    if (!map.value[id]) return
    delete map.value[id]
    if (remoteMode.value) {
      try {
        const { remove: removeRemote } = await import('@/api/history')
        await removeRemote({ mid: id })
      } catch (e) {
        logger.warn('history remove (remote) failed', e)
      }
    } else {
      persistLocal(map.value)
    }
    refreshList()
  }

  /** 清空 */
  async function clear(): Promise<void> {
    map.value = {}
    if (remoteMode.value) {
      try {
        const { clear: clearRemote } = await import('@/api/history')
        await clearRemote()
      } catch (e) {
        logger.warn('history clear (remote) failed', e)
      }
    } else {
      persistLocal(map.value)
    }
    refreshList()
  }

  /** fire-and-forget 写云端 */
  async function pushRemote(rec: HistoryRecord): Promise<void> {
    try {
      const mod = await import('@/api/history')
      await mod.upsert({
        mid: Number(rec.id) || 0,
        cid: rec.cid ?? 0,
        pid: rec.pid ?? 0,
        name: rec.name,
        picture: rec.picture ?? '',
        playFrom: rec.source ?? '',
        playFromName: '',
        episode: rec.episodeIndex ?? 0,
        episodeName: rec.episode ?? '',
        progress: Math.floor(rec.currentTime ?? 0),
        duration: Math.floor(rec.duration ?? 0)
      })
    } catch (e) {
      logger.warn('history upsert (remote) failed', e)
    }
  }

  /** 远端记录 → 本地记录映射 */
  function fromRemote(r: import('@/types/history').RemoteHistoryItem): HistoryRecord {
    const mid = String(r.mid)
    const params = new URLSearchParams()
    params.set('id', mid)
    if (r.playFrom) params.set('source', r.playFrom)
    params.set('episode', String(r.episode ?? 0))
    if ((r.progress ?? 0) > 0) params.set('currentTime', String(r.progress))
    return {
      id: mid,
      name: r.name,
      link: `/play?${params.toString()}`,
      episode: r.episodeName || String(r.episode ?? 0),
      picture: r.picture,
      source: r.playFrom,
      episodeIndex: r.episode,
      currentTime: r.progress,
      pid: r.pid,
      cid: r.cid,
      duration: r.duration,
      timeStamp: r.updatedAt ? new Date(r.updatedAt).getTime() : Date.now()
    }
  }

  /** 从云端全量拉取并替换内存 map (不写 cookie/LS) */
  async function syncFromRemote(): Promise<void> {
    remoteLoading.value = true
    try {
      const { list: listRemote } = await import('@/api/history')
      const resp = await listRemote({ current: 1, pageSize: MAX_ITEMS })
      const next: HistoryMap = {}
      for (const r of resp?.list ?? []) {
        const rec = fromRemote(r)
        next[rec.id] = rec
      }
      map.value = next
      refreshList()
    } catch (e) {
      logger.warn('history sync (remote) failed, keeping current map', e)
    } finally {
      remoteLoading.value = false
    }
  }

  /** 登录瞬间: 把本地 map 全量 upsert 到云端, 然后从云端拉新 list 替换内存 */
  async function mergeLocalIntoRemote(): Promise<void> {
    const locals = Object.values(map.value)
    if (locals.length === 0) {
      return
    }
    try {
      const { upsert } = await import('@/api/history')
      // 串行 upsert: 影片数一般不多, 串行降低后端瞬时压力且不需要并发控制
      for (const rec of locals) {
        const mid = Number(rec.id) || 0
        if (mid <= 0 || !rec.name) continue
        try {
          await upsert({
            mid,
            cid: rec.cid ?? 0,
            pid: rec.pid ?? 0,
            name: rec.name,
            picture: rec.picture ?? '',
            playFrom: rec.source ?? '',
            playFromName: '',
            episode: rec.episodeIndex ?? 0,
            episodeName: rec.episode ?? '',
            progress: Math.floor(rec.currentTime ?? 0),
            duration: Math.floor(rec.duration ?? 0)
          })
        } catch (e) {
          logger.warn('history merge (single) failed', rec.id, e)
        }
      }
    } catch (e) {
      logger.warn('history merge (remote) failed', e)
    }
  }

  /** 监听登录态切换, 自动驱动模式切换 */
  const userStore = useUserStore()
  const { isLoggedIn } = storeToRefs(userStore)
  watch(
    isLoggedIn,
    async (now, prev) => {
      // immediate 首次执行 prev === undefined: 不算"登录瞬间", 不触发 merge,
      // 只切到 remote 模式并拉云端. 仅当 prev 明确为 false → true 的过渡时
      // 才视为本次会话内"刚登录", 把本地匿名记录合并上去.
      const firstRun = prev === undefined
      if (now && !prev) {
        if (!firstRun) {
          await mergeLocalIntoRemote()
        }
        remoteMode.value = true
        await syncFromRemote()
      } else if (!now && prev) {
        // 登出: 退回本地 (cookie/LS 仍是登录前的快照)
        remoteMode.value = false
        map.value = loadFromLocal()
        refreshList()
      }
    },
    { immediate: true }
  )

  return {
    map,
    list,
    remoteMode,
    remoteLoading,
    record,
    get,
    remove,
    clear,
    /** HistoryView 下拉刷新等场景显式触发 */
    refresh: syncFromRemote
  }
})
