/**
 * URL → mock 数据派发
 *
 * 路由约定（与 vite proxy `/api` 去前缀后的形式一致）：
 *   /index, /navCategory, /config/basic
 *   /filmDetail?id=
 *   /filmPlayInfo?id=&playFrom=&episode=
 *   /searchFilm?keyword=&current=
 *   /filmClassify?Pid=
 *   /filmClassifySearch?Pid=...
 *   /login, /logout, /changePassword
 *   /manage/index, /manage/user/info, /manage/config/basic*
 *   /manage/collect/*, /manage/cron/*, /manage/spider/*
 *   /manage/film/*, /manage/file/*
 */

import {
  ADMIN_USER,
  buildClassifyData,
  buildClassifySearchResp,
  buildFilmDetailResp,
  buildManageFilmSearchResp,
  buildPhotoWallResp,
  buildPlayInfoResp,
  buildSearchResp,
  CLASS_COVERS,
  COLLECT_OPTIONS,
  COLLECT_SOURCES,
  CRON_TASKS,
  DASHBOARD_STAT,
  FILE_ITEMS,
  FILM_CLASSES,
  INDEX_DATA,
  NAV_CATEGORIES,
  SITE_BASIC
} from './data'

interface DispatchArgs {
  url: string
  method: string
  data: unknown
  params: Record<string, unknown>
}

interface MockResult {
  data: unknown
  status?: number
  delay?: number
  headers?: Record<string, string>
}

/** 收纳一个简易的可变数据池（add/update/del 的实时 mock 反馈） */
const MUTABLE_COLLECT = [...COLLECT_SOURCES]
const MUTABLE_CRON = [...CRON_TASKS]
const MUTABLE_FILE_LAST_ID = { value: 9999 }

function pickStr(p: Record<string, unknown>, k: string, def = ''): string {
  const v = p[k]
  return typeof v === 'string' ? v : v != null ? String(v) : def
}
function pickNum(p: Record<string, unknown>, k: string, def = 0): number {
  const v = p[k]
  if (typeof v === 'number') return v
  if (typeof v === 'string') {
    const n = Number(v)
    return Number.isFinite(n) ? n : def
  }
  return def
}

export function dispatch(args: DispatchArgs): MockResult | null {
  const { url, method, params, data } = args
  const m = method.toLowerCase()

  // ============== 用户端公开接口 ==============
  if (m === 'get' && url === '/index') {
    return { data: INDEX_DATA, delay: 120 }
  }
  if (m === 'get' && url === '/navCategory') {
    return { data: NAV_CATEGORIES }
  }
  if (m === 'get' && url === '/config/basic') {
    return { data: SITE_BASIC }
  }
  if (m === 'get' && url === '/filmDetail') {
    const id = pickStr(params, 'id') || pickStr(params, 'link')
    const resp = buildFilmDetailResp(id)
    return resp ? { data: resp } : { data: null, status: 200 }
  }
  if (m === 'get' && url === '/filmPlayInfo') {
    const id = pickStr(params, 'id')
    const playFrom = pickStr(params, 'playFrom')
    const episode = pickStr(params, 'episode', '0')
    const resp = buildPlayInfoResp(id, playFrom, episode)
    return resp ? { data: resp } : { data: null }
  }
  if (m === 'get' && url === '/searchFilm') {
    const keyword = pickStr(params, 'keyword')
    const current = pickNum(params, 'current', 1)
    return { data: buildSearchResp(keyword, current) }
  }
  if (m === 'get' && url === '/filmClassify') {
    const pid = pickStr(params, 'Pid')
    const resp = buildClassifyData(pid)
    return resp ? { data: resp } : { data: null }
  }
  if (m === 'get' && url === '/filmClassifySearch') {
    const pid = pickStr(params, 'Pid', '1')
    const current = pickNum(params, 'current', 1)
    return {
      data: buildClassifySearchResp(pid, current, {
        Category: pickStr(params, 'Category'),
        Plot: pickStr(params, 'Plot'),
        Area: pickStr(params, 'Area'),
        Language: pickStr(params, 'Language'),
        Year: pickStr(params, 'Year'),
        Sort: pickStr(params, 'Sort')
      })
    }
  }

  // ============== 鉴权 ==============
  if (m === 'post' && url === '/login') {
    return {
      data: ADMIN_USER,
      headers: { 'new-token': 'mock-token-' + Date.now() }
    }
  }
  if (m === 'get' && url === '/logout') {
    return { data: null }
  }
  if (m === 'post' && url === '/changePassword') {
    const body = (data ?? {}) as Record<string, unknown>
    if (!body.password || !body.newPassword) {
      return {
        data: { code: 400, msg: '参数缺失' },
        status: 200
      }
    }
    return { data: null }
  }
  if (m === 'get' && url === '/manage/user/info') {
    return { data: ADMIN_USER }
  }

  // ============== 仪表盘 / 站点配置 ==============
  if (m === 'get' && (url === '/manage/index' || url === '/manage/dashboard')) {
    return { data: DASHBOARD_STAT }
  }
  if (m === 'get' && url === '/manage/config/basic') {
    return { data: SITE_BASIC }
  }
  if (m === 'post' && url === '/manage/config/basic/update') {
    return { data: null }
  }
  if (m === 'get' && url === '/manage/config/basic/reset') {
    return { data: SITE_BASIC }
  }

  // ============== 采集源 ==============
  if (m === 'get' && url === '/manage/collect/list') {
    return { data: MUTABLE_COLLECT }
  }
  if (m === 'get' && url === '/manage/collect/options') {
    return { data: COLLECT_OPTIONS }
  }
  if (m === 'get' && url === '/manage/collect/find') {
    const id = pickStr(params, 'id')
    return { data: MUTABLE_COLLECT.find((c) => c.id === id) ?? null }
  }
  if (m === 'get' && url === '/manage/collect/del') {
    const id = pickStr(params, 'id')
    const idx = MUTABLE_COLLECT.findIndex((c) => c.id === id)
    if (idx >= 0) MUTABLE_COLLECT.splice(idx, 1)
    return { data: null }
  }
  if (m === 'post' && url === '/manage/collect/add') {
    const body = (data ?? {}) as Partial<typeof MUTABLE_COLLECT[number]>
    MUTABLE_COLLECT.push({
      ...(body as typeof MUTABLE_COLLECT[number]),
      id: 'cs-' + Math.random().toString(36).slice(2, 8)
    })
    return { data: null }
  }
  if (m === 'post' && (url === '/manage/collect/update' || url === '/manage/collect/change')) {
    const body = (data ?? {}) as Partial<typeof MUTABLE_COLLECT[number]>
    const idx = MUTABLE_COLLECT.findIndex((c) => c.id === body.id)
    if (idx >= 0) {
      const current = MUTABLE_COLLECT[idx] as typeof MUTABLE_COLLECT[number]
      MUTABLE_COLLECT[idx] = { ...current, ...body } as typeof MUTABLE_COLLECT[number]
    }
    return { data: null }
  }
  if (m === 'post' && url === '/manage/collect/test') {
    return { data: { ok: true, msg: '采集源连通性 OK（mock）' }, delay: 600 }
  }
  if (m === 'post' && url === '/manage/spider/start') {
    return { data: { msg: '已启动采集任务（mock）' }, delay: 400 }
  }
  if (m === 'get' && url === '/manage/spider/zero') {
    return { data: null }
  }
  if (m === 'get' && url === '/manage/spider/class/cover') {
    return { data: CLASS_COVERS }
  }

  // ============== 定时任务 ==============
  if (m === 'get' && url === '/manage/cron/list') {
    return { data: MUTABLE_CRON }
  }
  if (m === 'get' && url === '/manage/cron/find') {
    const id = pickStr(params, 'id')
    return { data: MUTABLE_CRON.find((c) => c.id === id) ?? null }
  }
  if (m === 'get' && url === '/manage/cron/del') {
    const id = pickStr(params, 'id')
    const idx = MUTABLE_CRON.findIndex((c) => c.id === id)
    if (idx >= 0) MUTABLE_CRON.splice(idx, 1)
    return { data: null }
  }
  if (m === 'post' && url === '/manage/cron/add') {
    const body = (data ?? {}) as Partial<typeof MUTABLE_CRON[number]>
    MUTABLE_CRON.push({
      ...(body as typeof MUTABLE_CRON[number]),
      id: 'cron-' + Math.random().toString(36).slice(2, 8)
    })
    return { data: null }
  }
  if (m === 'post' && (url === '/manage/cron/update' || url === '/manage/cron/change')) {
    const body = (data ?? {}) as Partial<typeof MUTABLE_CRON[number]>
    const idx = MUTABLE_CRON.findIndex((c) => c.id === body.id)
    if (idx >= 0) {
      const current = MUTABLE_CRON[idx] as typeof MUTABLE_CRON[number]
      MUTABLE_CRON[idx] = { ...current, ...body } as typeof MUTABLE_CRON[number]
    }
    return { data: null }
  }

  // ============== 影片管理 ==============
  if (m === 'get' && url === '/manage/film/search/list') {
    const current = pickNum(params, 'current', 1)
    const pageSize = pickNum(params, 'pageSize', 10)
    const name = pickStr(params, 'name')
    const pid = pickNum(params, 'pid', 0)
    return {
      data: buildManageFilmSearchResp(current, pageSize, name, pid || undefined)
    }
  }
  if (m === 'get' && url === '/manage/film/class/tree') {
    return { data: FILM_CLASSES }
  }
  if (m === 'get' && url === '/manage/film/class/find') {
    const id = pickNum(params, 'id', 0)
    const found =
      FILM_CLASSES.find((c) => c.id === id) ??
      FILM_CLASSES.flatMap((c) => c.children ?? []).find((c) => c.id === id) ??
      null
    return { data: found }
  }
  if (m === 'get' && url === '/manage/film/class/del') {
    return { data: null }
  }
  if (m === 'post' && url === '/manage/film/class/update') {
    return { data: null }
  }
  if (m === 'post' && url === '/manage/film/add') {
    return { data: null, delay: 300 }
  }

  // ============== 文件库 ==============
  if (m === 'get' && url === '/manage/file/list') {
    return { data: buildPhotoWallResp(pickNum(params, 'current', 1)) }
  }
  if (m === 'get' && url === '/manage/file/del') {
    const id = pickStr(params, 'id')
    const idx = FILE_ITEMS.findIndex((f) => String(f.id) === id)
    if (idx >= 0) FILE_ITEMS.splice(idx, 1)
    return { data: null }
  }
  if (m === 'post' && url === '/manage/file/upload') {
    const id = ++MUTABLE_FILE_LAST_ID.value
    const url2 = `https://picsum.photos/seed/upload-${id}/240/320`
    FILE_ITEMS.unshift({
      id,
      name: `upload-${id}.webp`,
      url: url2,
      size: 50 * 1024,
      type: 'image/webp',
      createdAt: Date.now()
    })
    // 后端约定 upload 返回 string url
    return { data: url2, delay: 500 }
  }

  return null
}
