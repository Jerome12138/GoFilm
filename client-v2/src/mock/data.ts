/**
 * Mock 数据集 —— 仅供 dev 模式 + VITE_USE_MOCK=1 时启用
 *
 * 数据范围：
 *  - 6 部完整影片（电影 / 电视剧 / 动漫，含详情 + 多源多集）
 *  - 6 个一级分类 + 30+ 二级分类
 *  - 站点信息 / 管理端示例（用户、采集源、定时任务、文件库）
 *  - 视频源使用 Big Buck Bunny 公开 MP4（每个 episode 用 fragment 区分）
 *  - 海报使用 picsum.photos seed 模式，确保稳定与缓存友好
 */

import type {
  ClassifyData,
  ClassifySearchResp,
  FilmDetail,
  FilmDetailResp,
  FilmListItem,
  IndexPageData,
  NavCategory,
  PlayInfo,
  PlaySource,
  SearchFilmResp,
  ManageFilmSearchResp,
  ClassCoverItem
} from '@/types/film'
import type {
  CollectSource,
  CollectOption,
  CronTask,
  DashboardStat,
  FileItem,
  FilmClass,
  PhotoWallResp,
  SiteBasic
} from '@/types/manage'
import type { UserInfo } from '@/types/user'

/* ============================================================
 * 工具：稳定海报 / 演示视频
 * ============================================================ */

/**
 * 演示视频源 — 主要用 HLS m3u8 (公开工程测试流), 验证 hls.js + 广告过滤代理链路.
 * 这些是各 CDN 厂商/Apple 公开发布的 engineering test stream, 非版权影视内容.
 *
 * 备用源 (sourceIdx > 0) 也走 HLS, 让"切换播放源"能切到不同流验证 ABR.
 * MP4 池作为兜底备用 (如果某些环境 HLS 不通 — 但 video.js + vhs 现代浏览器全支持).
 */
const HLS_POOL = [
  // Mux 标准测试流 (Big Buck Bunny, 公开域动画短片, hls.js 官方示例都用它)
  'https://test-streams.mux.dev/x36xhzz/x36xhzz.m3u8',
  // Apple HLS 工程示例 (BipBop 16:9 多码率, 标准测试用)
  'https://devstreaming-cdn.apple.com/videos/streaming/examples/bipbop_16x9/bipbop_16x9_variant.m3u8',
  // Apple HLS 高级特性示例 (含字幕轨)
  'https://devstreaming-cdn.apple.com/videos/streaming/examples/adv_dv_atmos/main.m3u8',
  // Mux 单码率短测试流
  'https://test-streams.mux.dev/test_001/stream.m3u8',
  // Apple HLS Live 测试 (循环点播形式)
  'https://devstreaming-cdn.apple.com/videos/streaming/examples/img_bipbop_adv_example_ts/master.m3u8'
]

/**
 * 备用兜底 MP4 池 (Google Sample Video Bucket — 公开工程测试资源).
 * 仅当 HLS 集合不可用时回退. 通常不进入选择.
 */
const MP4_FALLBACK_POOL = [
  'https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/BigBuckBunny.mp4',
  'https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/ElephantsDream.mp4',
  'https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/Sintel.mp4',
  'https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/TearsOfSteel.mp4'
]

function pickVideo(filmId: number, sourceIdx: number, epIdx: number): string {
  // 不同片/集/源 轮询不同 HLS 流, 让"切源/切集"能切到不同内容直观验证
  const hash = (filmId * 7 + sourceIdx * 3 + epIdx) & 0x7fffffff
  // 80% 概率用 HLS (主验证路径); 20% 偶尔 MP4 用于兜底验证
  if (hash % 5 === 0) {
    return MP4_FALLBACK_POOL[hash % MP4_FALLBACK_POOL.length] ?? MP4_FALLBACK_POOL[0]!
  }
  return HLS_POOL[hash % HLS_POOL.length] ?? HLS_POOL[0]!
}

function poster(seed: string, w = 300, h = 450): string {
  return `https://picsum.photos/seed/${encodeURIComponent(seed)}/${w}/${h}`
}

function banner(seed: string): string {
  return `https://picsum.photos/seed/${encodeURIComponent(seed + '-banner')}/1280/720`
}

/* ============================================================
 * 一级 / 二级分类
 * ============================================================ */

export const NAV_CATEGORIES: NavCategory[] = [
  {
    id: 1,
    pid: 0,
    name: '电影',
    show: true,
    children: [
      { id: 11, pid: 1, name: '动作', show: true },
      { id: 12, pid: 1, name: '科幻', show: true },
      { id: 13, pid: 1, name: '喜剧', show: true },
      { id: 14, pid: 1, name: '爱情', show: true },
      { id: 15, pid: 1, name: '剧情', show: true }
    ]
  },
  {
    id: 2,
    pid: 0,
    name: '电视剧',
    show: true,
    children: [
      { id: 21, pid: 2, name: '国产剧', show: true },
      { id: 22, pid: 2, name: '美剧', show: true },
      { id: 23, pid: 2, name: '韩剧', show: true },
      { id: 24, pid: 2, name: '日剧', show: true }
    ]
  },
  {
    id: 3,
    pid: 0,
    name: '综艺',
    show: true,
    children: [
      { id: 31, pid: 3, name: '真人秀', show: true },
      { id: 32, pid: 3, name: '脱口秀', show: true }
    ]
  },
  {
    id: 4,
    pid: 0,
    name: '动漫',
    show: true,
    children: [
      { id: 41, pid: 4, name: '日漫', show: true },
      { id: 42, pid: 4, name: '国漫', show: true },
      { id: 43, pid: 4, name: '剧场版', show: true }
    ]
  },
  {
    id: 5,
    pid: 0,
    name: '纪录片',
    show: true,
    children: [
      { id: 51, pid: 5, name: '自然', show: true },
      { id: 52, pid: 5, name: '历史', show: true }
    ]
  },
  {
    id: 6,
    pid: 0,
    name: '短剧',
    show: true,
    children: [{ id: 61, pid: 6, name: '都市', show: true }]
  }
]

/* ============================================================
 * 影片库
 * ============================================================ */

interface MockFilmInput {
  id: number
  name: string
  enName?: string
  pid: number
  cid: number
  cName: string
  area: string
  year: string
  director: string
  actor: string
  remarks: string
  classTag: string
  content: string
  dbScore: string
  releaseDate: string
  /** 每个源的集数；电影通常 [1]，电视剧 [12, 12] */
  episodeShape: number[]
  /** 海报 / banner 种子（默认用 name） */
  seed?: string
}

function makeSource(
  filmId: number,
  sourceIdx: number,
  sourceName: string,
  episodeCount: number
): PlaySource {
  const linkList = Array.from({ length: episodeCount }, (_, i) => {
    const epIdx = i + 1
    const label = episodeCount === 1 ? '正片' : `第 ${epIdx} 集`
    const base = pickVideo(filmId, sourceIdx, epIdx)
    return {
      episode: label,
      // 每集 link 必须唯一（用于 EpisodeTabs key 与高亮判定）
      link: `${base}#film=${filmId}&src=${sourceIdx}&ep=${epIdx}`
    }
  })
  return {
    id: `s${filmId}-${sourceIdx}`,
    name: sourceName,
    linkList
  }
}

function makeFilmDetail(input: MockFilmInput): FilmDetail {
  const seed = input.seed ?? input.name
  const list: PlaySource[] = input.episodeShape.map((count, idx) => {
    const sourceName = idx === 0 ? '主线' : `备用源 ${idx + 1}`
    return makeSource(input.id, idx, sourceName, count)
  })
  return {
    id: input.id,
    name: input.name,
    picture: poster(seed),
    remarks: input.remarks,
    year: input.year,
    area: input.area,
    cName: input.cName,
    cid: input.cid,
    pid: input.pid,
    descriptor: {
      cName: input.cName,
      classTag: input.classTag,
      director: input.director,
      actor: input.actor,
      releaseDate: input.releaseDate,
      area: input.area,
      year: input.year,
      remarks: input.remarks,
      content: input.content,
      dbScore: input.dbScore
    },
    list
  }
}

const FILM_INPUTS: MockFilmInput[] = [
  {
    id: 101,
    name: '流浪地球 III',
    enName: 'The Wandering Earth III',
    pid: 1,
    cid: 12,
    cName: '科幻',
    area: '中国大陆',
    year: '2027',
    director: '郭帆',
    actor: '吴京,刘德华,李雪健,王智',
    remarks: 'IMAX 公映',
    classTag: '科幻,灾难,冒险',
    content:
      '太阳系危机进入终章，行星发动机全球部署完毕。人类联合政府决定启动"流浪计划"最后一阶段——脱离太阳系。然而月球的剧变与外星文明的接触让一切再生变数……',
    dbScore: '8.6',
    releaseDate: '2027-02-08',
    episodeShape: [1, 1]
  },
  {
    id: 102,
    name: '沙丘：第二部',
    enName: 'Dune: Part Two',
    pid: 1,
    cid: 12,
    cName: '科幻',
    area: '美国',
    year: '2024',
    director: '丹尼斯·维伦纽瓦',
    actor: '提莫西·查拉梅,赞达亚,丽贝卡·弗格森',
    remarks: 'HD 中字',
    classTag: '科幻,冒险,剧情',
    content:
      '保罗·厄崔迪与契妮和弗瑞曼人结盟。在踏上复仇之路、击退毁灭其家族的阴谋者的同时，他必须在他生命所爱与已知宇宙的命运之间做出选择。',
    dbScore: '8.4',
    releaseDate: '2024-03-08',
    episodeShape: [1, 1]
  },
  {
    id: 103,
    name: '蜘蛛侠：纵横宇宙',
    enName: 'Spider-Man: Across the Spider-Verse',
    pid: 1,
    cid: 13,
    cName: '动画',
    area: '美国',
    year: '2023',
    director: '华金·多斯·桑托斯',
    actor: '沙梅克·摩尔,海莉·斯坦菲尔德',
    remarks: 'BD 4K',
    classTag: '动画,动作,冒险',
    content: '迈尔斯·莫拉莱斯穿越多元宇宙，与一群守护者形成的蜘蛛侠"小队"展开冒险，但蜘蛛侠们却因如何面对新威胁而争执不休。',
    dbScore: '9.0',
    releaseDate: '2023-06-02',
    episodeShape: [1, 1]
  },
  {
    id: 201,
    name: '三体 第二季',
    enName: 'Three-Body Season 2',
    pid: 2,
    cid: 21,
    cName: '剧情',
    area: '中国大陆',
    year: '2025',
    director: '杨磊',
    actor: '张鲁一,于和伟,陈瑾,王子文',
    remarks: '更新至 12 集',
    classTag: '科幻,悬疑,剧情',
    content:
      '面壁计划全面启动，罗辑、希恩斯、雷迪亚兹、泰勒分别提出对抗三体的方案。叶文洁与三体世界的对话越发深入。地球面临前所未有的"黑暗森林"威胁。',
    dbScore: '8.7',
    releaseDate: '2025-01-15',
    episodeShape: [12, 12]
  },
  {
    id: 202,
    name: '黑镜 第六季',
    enName: 'Black Mirror Season 6',
    pid: 2,
    cid: 22,
    cName: '科幻',
    area: '英国',
    year: '2023',
    director: '查理·布鲁克',
    actor: '安妮·墨菲,奥卡菲娜,迈克尔·塞拉',
    remarks: '5 集全',
    classTag: '科幻,悬疑,剧情',
    content: '奈飞《黑镜》重启回归，本季 5 集独立故事继续以荒诞、反讽视角探讨技术与人性的灰色边界。',
    dbScore: '7.6',
    releaseDate: '2023-06-15',
    episodeShape: [5, 5]
  },
  {
    id: 401,
    name: '咒术回战 第二季',
    enName: 'Jujutsu Kaisen Season 2',
    pid: 4,
    cid: 41,
    cName: '热血',
    area: '日本',
    year: '2023',
    director: '朴性厚',
    actor: '榎木淳弥,内田雄马,瀨戶麻沙美',
    remarks: '更新至 23 集',
    classTag: '动作,热血,奇幻',
    content:
      '"涩谷事变"开始。咒术界至上等级的咒霊突袭涩谷，五条悟被封印于狱门疆。虎杖悠仁、伏黒惠、釘崎野薔薇与众咒术师在断绝中迎击两面宿傩。',
    dbScore: '9.2',
    releaseDate: '2023-07-06',
    episodeShape: [23, 23]
  }
]

const FILMS: FilmDetail[] = FILM_INPUTS.map(makeFilmDetail)

function toListItem(f: FilmDetail): FilmListItem {
  return {
    id: f.id,
    name: f.name,
    picture: f.picture,
    remarks: f.remarks,
    year: f.year,
    area: f.area,
    cName: f.cName,
    cid: f.cid,
    pid: f.pid,
    dbScore: f.descriptor?.dbScore
  }
}

const FILM_LIST: FilmListItem[] = FILMS.map(toListItem)

/* ============================================================
 * 首页聚合
 * ============================================================ */

function pickByPid(pid: number): FilmListItem[] {
  return FILM_LIST.filter((m) => m.pid === pid)
}

export const INDEX_DATA: IndexPageData = {
  banner: FILMS.slice(0, 4).map((f) => ({
    ...toListItem(f),
    // 首页 banner 用 16:9 大图（HeroCarousel 直接读 picture 当背景）
    picture: banner(String(f.id))
  })),
  content: NAV_CATEGORIES.slice(0, 4).map((nav) => {
    const all = pickByPid(nav.id)
    // 每个 row 至少 6 个：用本分类条目，不足时把全库其它影片随机补齐
    const padded = all.length >= 6 ? all : padTo(all, 6)
    return {
      nav,
      movies: padded,
      hot: padded.slice(0, 12)
    }
  })
}

function padTo(arr: FilmListItem[], n: number): FilmListItem[] {
  const out = [...arr]
  let i = 0
  while (out.length < n && FILM_LIST.length > 0) {
    const src = FILM_LIST[i % FILM_LIST.length] as FilmListItem
    out.push({ ...src, id: `${src.id}-pad-${out.length}`, mid: String(src.id) })
    i++
  }
  return out
}

/* ============================================================
 * 详情 / 播放
 * ============================================================ */

function getFilmById(id: string | number): FilmDetail | null {
  const num = Number(id)
  return FILMS.find((f) => f.id === num) ?? null
}

export function buildFilmDetailResp(id: string | number): FilmDetailResp | null {
  const f = getFilmById(id)
  if (!f) return null
  // relate：同 pid 其它影片 + 全库补齐
  const sameCat = FILM_LIST.filter(
    (m) => m.pid === f.pid && Number(m.id) !== Number(f.id)
  )
  const others = FILM_LIST.filter((m) => Number(m.id) !== Number(f.id))
  const relate = padTo([...sameCat, ...others].slice(0, 8), 8)
  return { detail: f, relate }
}

export function buildPlayInfoResp(
  id: string | number,
  playFrom: string,
  episodeStr: string
): PlayInfo | null {
  const f = getFilmById(id)
  if (!f) return null
  const sourceId = playFrom || f.list[0]?.id || ''
  const source = f.list.find((s) => s.id === sourceId) ?? f.list[0]
  if (!source) return null
  const epIdx = Math.max(
    0,
    Math.min(source.linkList.length - 1, Number(episodeStr) || 0)
  )
  const current = source.linkList[epIdx] ?? source.linkList[0]
  if (!current) return null
  const sameCat = FILM_LIST.filter(
    (m) => m.pid === f.pid && Number(m.id) !== Number(f.id)
  )
  return {
    detail: f,
    current,
    currentPlayFrom: source.id,
    currentEpisode: epIdx,
    relate: padTo(sameCat, 8)
  }
}

/* ============================================================
 * 搜索 / 分类筛选
 * ============================================================ */

export function buildSearchResp(keyword: string, current: number): SearchFilmResp {
  const k = keyword.trim().toLowerCase()
  const list = k
    ? FILM_LIST.filter(
        (f) =>
          f.name.toLowerCase().includes(k) ||
          (f.cName ?? '').toLowerCase().includes(k) ||
          (f.area ?? '').toLowerCase().includes(k)
      )
    : []
  return {
    list,
    page: {
      current: current || 1,
      pageSize: 10,
      total: list.length,
      pageCount: Math.max(1, Math.ceil(list.length / 10))
    }
  }
}

export function buildClassifyData(pid: number | string): ClassifyData | null {
  const pidNum = Number(pid)
  const nav = NAV_CATEGORIES.find((n) => n.id === pidNum)
  if (!nav) return null
  const inCat = pickByPid(pidNum)
  const list = inCat.length >= 8 ? inCat : padTo(inCat, 8)
  return {
    title: { id: nav.id, pid: 0, name: nav.name, show: nav.show },
    content: {
      news: list,
      top: [...list].reverse(),
      recent: list
    }
  }
}

const SORT_OPTIONS = ['热度', '最新', '评分']
const PLOT_BY_PID: Record<number, string[]> = {
  1: ['全部', '动作', '科幻', '喜剧', '爱情', '剧情', '动画'],
  2: ['全部', '都市', '古装', '悬疑', '历史', '科幻'],
  3: ['全部', '真人秀', '脱口秀', '音乐'],
  4: ['全部', '热血', '冒险', '奇幻', '日常'],
  5: ['全部', '自然', '历史', '人文'],
  6: ['全部', '都市', '甜宠']
}
const AREAS = ['全部', '中国大陆', '美国', '英国', '日本', '韩国']
const LANGUAGES = ['全部', '国语', '英语', '日语', '粤语']
const YEARS = ['全部', '2027', '2025', '2024', '2023', '2022']

export function buildClassifySearchResp(
  pid: string,
  current: number,
  filters: {
    Category: string
    Plot: string
    Area: string
    Language: string
    Year: string
    Sort: string
  }
): ClassifySearchResp {
  const pidNum = Number(pid)
  const nav = NAV_CATEGORIES.find((n) => n.id === pidNum)
  let list = pickByPid(pidNum)
  if (filters.Plot && filters.Plot !== '全部') {
    list = list.filter((m) => (m.cName ?? '').includes(filters.Plot))
  }
  if (filters.Area && filters.Area !== '全部') {
    list = list.filter((m) => m.area === filters.Area)
  }
  if (filters.Year && filters.Year !== '全部') {
    list = list.filter((m) => m.year === filters.Year)
  }
  // 不足 12 条时填充以便展示
  if (list.length < 12) {
    list = padTo(list, 12)
  }

  return {
    title: nav
      ? { id: nav.id, pid: 0, name: nav.name, show: nav.show }
      : { id: 0, pid: 0, name: '全部' },
    list,
    page: {
      current: current || 1,
      pageSize: 12,
      total: list.length,
      pageCount: Math.max(1, Math.ceil(list.length / 12))
    },
    search: {
      sortList: ['Plot', 'Area', 'Language', 'Year', 'Sort'],
      titles: {
        Plot: '类型',
        Area: '地区',
        Language: '语言',
        Year: '年份',
        Sort: '排序'
      },
      tags: {
        Plot: (PLOT_BY_PID[pidNum] ?? PLOT_BY_PID[1] ?? ['全部']).map((v) => ({
          Name: v,
          Value: v
        })),
        Area: AREAS.map((v) => ({ Name: v, Value: v })),
        Language: LANGUAGES.map((v) => ({ Name: v, Value: v })),
        Year: YEARS.map((v) => ({ Name: v, Value: v })),
        Sort: SORT_OPTIONS.map((v) => ({ Name: v, Value: v }))
      }
    },
    params: {
      Pid: pid,
      Category: filters.Category,
      Plot: filters.Plot,
      Area: filters.Area,
      Language: filters.Language,
      Year: filters.Year,
      Sort: filters.Sort
    }
  }
}

/* ============================================================
 * 站点配置 / 用户
 * ============================================================ */

export const SITE_BASIC: SiteBasic = {
  siteName: 'GoFilm 影视',
  logo: '',
  keyword: '电影,电视剧,动漫,综艺,在线观看',
  describe: '现代化影视聚合演示站点（Mock Demo）',
  domain: 'gofilm.local',
  state: true,
  hint: ''
}

export const ADMIN_USER: UserInfo = {
  id: 1,
  uid: 'mock-admin',
  userName: 'admin',
  username: 'admin',
  nickname: '演示管理员',
  nickName: '演示管理员',
  email: 'admin@gofilm.local',
  avatar: 'https://i.pravatar.cc/120?img=12',
  status: 0,
  role: 1
}

/* ============================================================
 * 管理端
 * ============================================================ */

export const DASHBOARD_STAT: DashboardStat = {
  filmCount: 1284,
  collectCount: 6,
  cronCount: 3
}

export const COLLECT_SOURCES: CollectSource[] = [
  {
    id: 'cs-001',
    name: '飞速影视',
    uri: 'https://api.feisuzy.com/inc/api.php',
    resultModel: 0,
    grade: 0,
    syncPictures: true,
    collectType: 0,
    state: true,
    interval: 0
  },
  {
    id: 'cs-002',
    name: '苹果资源',
    uri: 'https://www.apiapi.cc/api.php/provide/vod/at/json',
    resultModel: 0,
    grade: 0,
    syncPictures: true,
    collectType: 0,
    state: true,
    interval: 0
  },
  {
    id: 'cs-003',
    name: '量子资源',
    uri: 'https://lzapi.com/api.php/provide/vod/at/xml',
    resultModel: 1,
    grade: 1,
    syncPictures: false,
    collectType: 0,
    state: false,
    interval: 0
  }
]

export const COLLECT_OPTIONS: CollectOption[] = COLLECT_SOURCES.map((c) => ({
  id: c.id,
  name: c.name
}))

export const CRON_TASKS: CronTask[] = [
  {
    id: 'cron-001',
    ids: ['cs-001', 'cs-002'],
    time: 24,
    spec: '0 0 3 * * *',
    model: 0,
    state: true,
    remark: '每日 3 点全量采集主站'
  },
  {
    id: 'cron-002',
    ids: [],
    time: 6,
    spec: '0 0 */6 * * *',
    model: 1,
    state: true,
    remark: '6 小时增量采集'
  },
  {
    id: 'cron-003',
    ids: ['cs-003'],
    time: 1,
    spec: '0 30 1 * * *',
    model: 0,
    state: false,
    remark: '附属源测试任务'
  }
]

export const FILM_CLASSES: FilmClass[] = NAV_CATEGORIES.map((n, idx) => ({
  id: n.id,
  pid: 0,
  name: n.name,
  ename: n.name,
  show: n.show,
  sort: idx,
  children: (n.children ?? []).map((c, ci) => ({
    id: c.id,
    pid: c.pid,
    name: c.name,
    ename: c.name,
    show: c.show,
    sort: ci
  }))
}))

export const FILE_ITEMS: FileItem[] = Array.from({ length: 18 }, (_, i) => ({
  id: 1000 + i,
  name: `poster-${i + 1}.webp`,
  url: poster(`gallery-${i}`, 240, 320),
  size: 80 * 1024 + i * 1024,
  type: 'image/webp',
  createdAt: Date.now() - i * 3600_000
}))

export function buildPhotoWallResp(current: number): PhotoWallResp {
  const pageSize = 12
  const start = (Math.max(1, current) - 1) * pageSize
  const list = FILE_ITEMS.slice(start, start + pageSize)
  return {
    list,
    page: {
      current: current || 1,
      pageSize,
      total: FILE_ITEMS.length,
      pageCount: Math.ceil(FILE_ITEMS.length / pageSize)
    }
  }
}

export function buildManageFilmSearchResp(
  current: number,
  pageSize: number,
  name: string,
  pid?: number
): ManageFilmSearchResp {
  const ps = pageSize || 10
  let list = FILM_LIST
  if (name) {
    const k = name.toLowerCase()
    list = list.filter((m) => m.name.toLowerCase().includes(k))
  }
  if (pid) {
    list = list.filter((m) => m.pid === pid)
  }
  const start = (Math.max(1, current) - 1) * ps
  const slice = list.slice(start, start + ps)
  return {
    list: slice,
    options: {},
    params: {
      paging: {
        current: current || 1,
        pageSize: ps,
        total: list.length,
        pageCount: Math.max(1, Math.ceil(list.length / ps))
      }
    }
  }
}

export const CLASS_COVERS: ClassCoverItem[] = NAV_CATEGORIES.map((n) => ({
  id: n.id,
  name: n.name,
  cover: poster(`cover-${n.id}`, 600, 360)
}))
