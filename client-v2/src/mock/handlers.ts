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

/* ============================================================
 * Mock 用户中心数据：用 localStorage 按 mock_user_name 分账户持久化
 * key 形如 __mock_history__alice / __mock_favorite__admin
 * ============================================================ */

type MockUserKind = 'history' | 'favorite'

function mockStorageKey(kind: MockUserKind): string {
  let user = 'anon'
  try {
    if (typeof localStorage !== 'undefined') {
      user = localStorage.getItem('__mock_user_name') ?? 'anon'
    }
  } catch {
    /* ignore */
  }
  return `__mock_${kind}__${user}`
}

function mockReadList(kind: MockUserKind): Array<Record<string, unknown>> {
  if (typeof localStorage === 'undefined') return []
  try {
    const raw = localStorage.getItem(mockStorageKey(kind))
    if (!raw) return []
    const parsed = JSON.parse(raw) as unknown
    return Array.isArray(parsed) ? (parsed as Array<Record<string, unknown>>) : []
  } catch {
    return []
  }
}

function mockWriteList(kind: MockUserKind, list: Array<Record<string, unknown>>): void {
  if (typeof localStorage === 'undefined') return
  try {
    localStorage.setItem(mockStorageKey(kind), JSON.stringify(list))
  } catch {
    /* ignore */
  }
}

function mockUpsert(kind: MockUserKind, body: Record<string, unknown>): void {
  const mid = Number(body.mid)
  if (!Number.isFinite(mid) || mid <= 0) return
  const list = mockReadList(kind)
  const now = new Date().toISOString()
  const idx = list.findIndex((it) => Number(it.mid) === mid)
  const next: Record<string, unknown> = {
    ...body,
    mid,
    updatedAt: now
  }
  if (idx >= 0) {
    next.id = list[idx]!.id ?? Date.now()
    next.createdAt = list[idx]!.createdAt ?? now
    list[idx] = next
  } else {
    next.id = Date.now()
    next.createdAt = now
    list.unshift(next)
  }
  // 按 updatedAt 倒序
  list.sort((a, b) => String(b.updatedAt).localeCompare(String(a.updatedAt)))
  mockWriteList(kind, list)
}

function mockRemove(kind: MockUserKind, key: string | number | undefined): void {
  if (key == null) return
  const list = mockReadList(kind)
  const target = String(key)
  const filtered = list.filter(
    (it) => String(it.mid) !== target && String(it.id) !== target
  )
  if (filtered.length !== list.length) {
    mockWriteList(kind, filtered)
  }
}

function mockClearList(kind: MockUserKind): void {
  mockWriteList(kind, [])
}

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

  // ============== 鉴权（新接口 + 旧路径兼容）==============
  // 简易 mock：admin → role=1，其它非空用户名 → role=0
  if (m === 'post' && (url === '/user/login' || url === '/login')) {
    const body = (data ?? {}) as { userName?: string; password?: string }
    const userName = (body.userName ?? '').trim()
    const password = body.password ?? ''
    if (!userName || !password) {
      return {
        data: { code: -1, msg: '用户名或密码不能为空' },
        status: 200
      }
    }
    const isAdmin = userName === 'admin'
    try {
      if (typeof localStorage !== 'undefined') {
        localStorage.setItem('__mock_role', isAdmin ? '1' : '0')
        localStorage.setItem('__mock_user_name', userName)
      }
    } catch {
      /* ignore */
    }
    // 新协议: body 直接返回 LoginResult{userName, token, expires, role}.
    // (旧实现把 user 对象当 body, token 走 new-token 头, 与后端契约不一致)
    const expires = Math.floor(Date.now() / 1000) + 10 * 24 * 3600
    return {
      data: {
        userName,
        token: 'mock-token-' + Date.now(),
        expires,
        role: isAdmin ? 1 : 0
      }
    }
  }
  if (m === 'get' && (url === '/user/logout' || url === '/logout')) {
    try {
      if (typeof localStorage !== 'undefined') {
        localStorage.removeItem('__mock_role')
        localStorage.removeItem('__mock_user_name')
      }
    } catch {
      /* ignore */
    }
    return { data: null }
  }
  if (m === 'post' && (url === '/user/changePassword' || url === '/changePassword')) {
    const body = (data ?? {}) as Record<string, unknown>
    if (!body.password || !body.newPassword) {
      return { data: { code: -1, msg: '原密码与新密码不能为空' }, status: 200 }
    }
    return { data: null }
  }
  if (m === 'get' && (url === '/user/info' || url === '/manage/user/info')) {
    // 通过 localStorage['__mock_role'] 选择 admin/普通用户；__mock_user_name 决定昵称
    let mockRole = 1
    let mockUserName = 'admin'
    try {
      if (typeof localStorage !== 'undefined') {
        const r = localStorage.getItem('__mock_role')
        if (r === '0') mockRole = 0
        const n = localStorage.getItem('__mock_user_name')
        if (n) mockUserName = n
      }
    } catch {
      /* ignore */
    }
    if (mockRole === 0) {
      return {
        data: {
          id: 10000 + (mockUserName.length % 100),
          uid: 'mock-' + mockUserName,
          userName: mockUserName,
          username: mockUserName,
          nickName: mockUserName,
          email: `${mockUserName}@example.com`,
          gender: 0,
          avatar: '',
          status: 0,
          role: 0
        }
      }
    }
    return { data: ADMIN_USER }
  }

  // ============== 用户中心（mock LS 持久化版） ==============
  // 用 localStorage 兜底, 不同 mock_role 各存一份, 实现"云端归属于账号"的语义.
  // 这样登录后能看到自己之前 POST 上去的内容, 退出再登录也保留, 接近真实后端体验.
  if (m === 'get' && url === '/user/history') {
    const list = mockReadList('history')
    return {
      data: {
        list,
        page: { pageSize: 20, current: 1, pageCount: 1, total: list.length }
      }
    }
  }
  if (m === 'post' && url === '/user/history') {
    mockUpsert('history', (data ?? {}) as Record<string, unknown>)
    return { data: null }
  }
  if (m === 'delete' && url === '/user/history/clear') {
    mockClearList('history')
    return { data: null }
  }
  if (m === 'delete' && url === '/user/history') {
    const id = (params.id ?? params.mid) as string | number | undefined
    mockRemove('history', id)
    return { data: null }
  }
  if (m === 'get' && url === '/user/favorite') {
    const list = mockReadList('favorite')
    return {
      data: {
        list,
        page: { pageSize: 20, current: 1, pageCount: 1, total: list.length }
      }
    }
  }
  if (m === 'get' && url === '/user/favorite/check') {
    const mid = String(params.mid ?? '')
    const list = mockReadList('favorite')
    return { data: { favorited: list.some((it) => String(it.mid) === mid) } }
  }
  if (m === 'post' && url === '/user/favorite') {
    mockUpsert('favorite', (data ?? {}) as Record<string, unknown>)
    return { data: null }
  }
  if (m === 'delete' && url === '/user/favorite') {
    mockRemove('favorite', params.mid as string | number | undefined)
    return { data: null }
  }

  // ============== 后台用户管理 ==============
  if (m === 'post' && url === '/manage/user/create') {
    const body = (data ?? {}) as Record<string, unknown>
    return {
      data: {
        id: Date.now() % 100000,
        uid: 'mock-' + (body.userName ?? 'new'),
        userName: body.userName,
        nickName: body.nickName ?? body.userName,
        email: body.email,
        role: body.role ?? 0,
        status: 0,
        gender: 0,
        avatar: ''
      }
    }
  }
  if (m === 'get' && url === '/manage/user/list') {
    const list = [
      { id: 10000, userName: 'admin', nickName: '演示管理员', email: 'admin@gofilm.local', role: 1, status: 0 },
      { id: 10001, userName: 'alice', nickName: 'Alice', email: 'alice@example.com', role: 0, status: 0 },
      { id: 10002, userName: 'bob', nickName: 'Bob', email: '', role: 0, status: 0 }
    ]
    return {
      data: {
        list,
        page: {
          pageSize: pickNum(params, 'pageSize', 20),
          current: pickNum(params, 'current', 1),
          pageCount: 1,
          total: list.length
        }
      }
    }
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
