# GoFilm Vue3 重构 - 技术架构方案

> 本文档为 `client-v2/` 全新重构的技术蓝图。  
> 输出对象：前端开发、QA、Lead Dev。  
> 兼容契约：API 路径、query 参数、token 头、cookie history 与旧版完全一致。

---

## 1. 方案概述

`client-v2/` 采用 **Vue 3.5 + Vite 5 + TypeScript 5 + Pinia 2 + UnoCSS** 的现代前端组合，目标对齐 Netflix / Disney+ 暗色视觉。  
整体走 **SPA + 客户端路由**，沿用 `vue-router 4`，鉴权与请求层重写为 Pinia store + Axios 拦截器，但保留旧后端 `/api` 路径与响应结构 100% 兼容。  
样式从 Element Plus 全量依赖切换为 **UnoCSS 原子类 + 极少量自研基础组件**，仅在管理后台保留可选的 Element Plus 局部按需引入（fallback）。  
新版按"用户端 / 管理端"双布局拆分，所有视图懒加载，首屏只装首页骨架与 hero。

---

## 2. 技术选型（精确版本）

| 层面 | 选型 | 版本 | 理由 |
|---|---|---|---|
| 视图框架 | Vue | `3.5.13` | 稳定大版本，支持 `defineModel`、Suspense、`useTemplateRef` |
| 构建 | Vite | `5.4.x` | 与 Vue 3.5 兼容最佳，Rollup 4 |
| 语言 | TypeScript | `5.6.x` | 严格模式 + `verbatimModuleSyntax` |
| 路由 | vue-router | `4.4.x` | 保持兼容旧路径，支持 typed routes |
| 状态管理 | Pinia | `2.2.x` | 轻量 + 组合式 API |
| HTTP | axios | `1.7.x` | 拦截器、AbortController |
| 样式 | UnoCSS | `0.62.x` | 原子化、按需 |
| 图标 | UnoCSS preset-icons | `0.62.x` | iconify on-demand |
| 属性化 | UnoCSS preset-attributify | `0.62.x` | `<div text-sm text-white/70>` 写法 |
| 自动导入 | unplugin-auto-import | `0.18.x` | 自动 import vue/pinia 常用 API |
| 组件自动注册 | unplugin-vue-components | `0.27.x` | 仅用于 base 组件目录 |
| 视频播放 | video.js | `8.17.x` | 与旧版统一播放体验 |
| Vue 播放器封装 | @videojs-player/vue | `1.0.x` | 与旧版一致 |
| 工具库 | @vueuse/core | `11.x` | useResizeObserver / useStorage / useDebounceFn |
| 单元测试 | vitest | `2.1.x` | 与 vite 同生态 |
| Lint | eslint | `9.x`（flat config） | 配 `@vue/eslint-config-typescript` |
| 格式化 | prettier | `3.x` | 与 eslint 协作 |
| 包管理 | pnpm | `>=9` | workspace + 速度 |
| Node | node | `>=20.10` | Vite 5 要求 |

> **不引入** Element Plus 主全量。仅在管理后台日期/上传等成本高组件考虑按需 fallback。

---

## 3. 目录结构

```
client-v2/
├── index.html
├── vite.config.ts
├── uno.config.ts
├── tsconfig.json
├── tsconfig.app.json
├── tsconfig.node.json
├── package.json
├── .env.development
├── .env.production
├── public/
│   ├── favicon.ico
│   └── iconfont/                # 沿用旧站 iconfont 资产
└── src/
    ├── main.ts
    ├── App.vue
    ├── env.d.ts
    │
    ├── api/                     # 接口模块（按业务域）
    │   ├── index.ts             # 桶导出
    │   ├── http.ts              # axios 实例 + 拦截器
    │   ├── film.ts              # 用户端影片 API
    │   ├── auth.ts              # 登录 / 登出 / 改密
    │   └── manage/
    │       ├── index.ts
    │       ├── collect.ts       # 采集源
    │       ├── cron.ts          # 定时任务
    │       ├── film.ts          # 影片管理 / 分类树
    │       ├── file.ts          # 文件上传
    │       ├── system.ts        # 站点配置
    │       └── user.ts          # 用户信息
    │
    ├── assets/
    │   ├── styles/              # 仅放无法用 UnoCSS 表达的全局
    │   │   ├── reset.css
    │   │   ├── theme.css        # CSS 变量（color/spacing/--ui-scale）
    │   │   └── iconfont.css
    │   └── images/              # 404.png / play.png / managebg.png
    │
    ├── components/
    │   ├── base/                # 自研无业务的原子组件，自动注册
    │   │   ├── BaseButton.vue
    │   │   ├── BaseDialog.vue
    │   │   ├── BaseEmpty.vue
    │   │   ├── BaseImage.vue    # 懒加载 + 错误占位
    │   │   ├── BasePagination.vue
    │   │   ├── BaseSkeleton.vue
    │   │   └── BaseTag.vue
    │   ├── layout/              # 布局壳
    │   │   ├── PublicLayout.vue
    │   │   ├── PublicHeader.vue
    │   │   ├── PublicFooter.vue
    │   │   ├── ManageLayout.vue
    │   │   ├── ManageHeader.vue
    │   │   └── ManageSidebar.vue
    │   └── film/                # 影视业务组件
    │       ├── HeroCarousel.vue
    │       ├── FilmRow.vue      # 横向滚动剧集行
    │       ├── FilmCard.vue
    │       ├── FilmGrid.vue     # 自适应网格
    │       ├── FilmFilterBar.vue
    │       ├── EpisodeTabs.vue
    │       └── RelatedList.vue
    │
    ├── composables/             # 组合式
    │   ├── useBreakpoint.ts     # 响应式断点
    │   ├── useFilmHistory.ts    # cookie 观看历史（key: filmHistory）
    │   ├── usePagination.ts
    │   ├── useQuerySync.ts      # 路由 query <-> ref 同步
    │   ├── useLoading.ts        # 全局 loading service
    │   └── usePlayer.ts         # video.js 封装
    │
    ├── stores/
    │   ├── index.ts
    │   ├── user.ts              # useUserStore
    │   ├── site.ts              # useSiteStore
    │   ├── nav.ts               # useNavStore
    │   ├── history.ts           # useHistoryStore
    │   └── ui.ts                # useUIStore（侧边栏折叠 / 主题）
    │
    ├── router/
    │   ├── index.ts
    │   ├── routes.public.ts
    │   ├── routes.manage.ts
    │   └── guards.ts
    │
    ├── views/
    │   ├── public/
    │   │   ├── HomeView.vue
    │   │   ├── FilmDetailView.vue
    │   │   ├── PlayView.vue
    │   │   ├── SearchView.vue
    │   │   ├── ClassifyView.vue
    │   │   ├── ClassifySearchView.vue
    │   │   └── HistoryView.vue   # 新增（可选）
    │   ├── auth/
    │   │   └── LoginView.vue
    │   ├── manage/
    │   │   ├── DashboardView.vue
    │   │   ├── collect/
    │   │   │   └── CollectListView.vue
    │   │   ├── cron/
    │   │   │   └── CronListView.vue
    │   │   ├── film/
    │   │   │   ├── FilmListView.vue
    │   │   │   ├── FilmAddView.vue
    │   │   │   ├── FilmDetailView.vue
    │   │   │   └── FilmClassView.vue
    │   │   ├── file/
    │   │   │   ├── FileUploadView.vue
    │   │   │   └── FileGalleryView.vue
    │   │   └── system/
    │   │       └── SiteConfigView.vue
    │   └── error/
    │       └── NotFoundView.vue
    │
    ├── types/
    │   ├── api.ts               # ApiResp / Pagination
    │   ├── film.ts              # FilmListItem / FilmDetail / PlayInfo / Category
    │   ├── manage.ts            # CollectSource / CronTask / FilmClass / FileItem
    │   ├── user.ts              # UserInfo / LoginPayload
    │   └── env.d.ts
    │
    └── utils/
        ├── token.ts             # localStorage auth-token
        ├── cookie.ts            # 兼容旧 cookieUtil
        ├── format.ts            # 时间 / 文本格式化
        ├── url.ts               # 安全 query 拼装
        └── logger.ts
```

---

## 4. 路由设计

> 全部沿用旧路径与 query，确保旧站外链 / SEO 不失效。

### 4.1 路由表

| 路径 | 组件 | layout | requiresAuth | 备注 |
|---|---|---|---|---|
| `/` | redirect → `/index` | - | false | 兼容根访问 |
| `/index` | `HomeView` | public | false | 首页（轮播 + 多分类 row） |
| `/filmDetail` | `FilmDetailView` | public | false | query: `link` |
| `/play` | `PlayView` | public | false | query: `id`, `source`, `episode`, `currentTime?` |
| `/search` | `SearchView` | public | false | query: `search`, `current?` |
| `/filmClassify` | `ClassifyView` | public | false | query: `Pid` |
| `/filmClassifySearch` | `ClassifySearchView` | public | false | query: `Pid`,`Category`,`Plot`,`Area`,`Language`,`Year`,`Sort`,`current` |
| `/history` | `HistoryView` | public | false | 新增（可选）观看历史 |
| `/login` | `LoginView` | auth | false | 登录后写 token |
| `/manage` | redirect → `/manage/index` | manage | true | |
| `/manage/index` | `DashboardView` | manage | true | |
| `/manage/collect/index` | `CollectListView` | manage | true | |
| `/manage/cron/index` | `CronListView` | manage | true | |
| `/manage/film` | `FilmListView` | manage | true | |
| `/manage/film/class` | `FilmClassView` | manage | true | |
| `/manage/film/add` | `FilmAddView` | manage | true | |
| `/manage/film/detail` | `FilmDetailView`（manage 版） | manage | true | |
| `/manage/file/upload` | `FileUploadView` | manage | true | |
| `/manage/file/gallery` | `FileGalleryView` | manage | true | |
| `/manage/system/webSite` | `SiteConfigView` | manage | true | |
| `/:pathMatch(.*)*` | `NotFoundView` | minimal | false | 404 |

### 4.2 meta 类型

```ts
declare module 'vue-router' {
  interface RouteMeta {
    /** 布局壳：public / manage / auth / minimal */
    layout?: 'public' | 'manage' | 'auth' | 'minimal'
    /** 是否需要登录态（仅 manage 段） */
    requiresAuth?: boolean
    /** 浏览器标题（与站点名拼接） */
    title?: string
    /** keepAlive 缓存 */
    keepAlive?: boolean
  }
}
```

### 4.3 守卫（`router/guards.ts`）

- `beforeEach`: 仅当 `to.meta.requiresAuth` 且 token 缺失 → `next({ path:'/login', query:{ redirect: to.fullPath }})`
- `afterEach`: 设置 `document.title = ${meta.title} - ${siteName}`
- 登录成功后从 `route.query.redirect` 还原跳转，保持 query 完整

### 4.4 懒加载约定

```ts
const HomeView = () => import('@/views/public/HomeView.vue')
```

所有 view 必须懒加载；layout 直接静态 import（首包必须）。

---

## 5. 状态管理（Pinia）

> 命名采用组合式风格 `defineStore(id, () => { ... })`。

### 5.1 `useUserStore` (`stores/user.ts`)

```ts
interface UserInfo {
  uid: string
  username: string
  nickname?: string
  avatar?: string
  role?: string
}

export const useUserStore = defineStore('user', () => {
  const token = ref<string>(getToken()?.value ?? '')
  const info  = ref<UserInfo | null>(null)

  const isLoggedIn = computed(() => token.value.length > 0)

  async function login(payload: LoginPayload): Promise<void> { /* 调 auth.login */ }
  async function fetchInfo(): Promise<UserInfo> { /* 调 manage/user/info */ }
  async function changePassword(p: { oldPwd: string; newPwd: string }): Promise<void> {}
  function setToken(t: string): void {}
  function logout(): Promise<void> { /* 调 auth.logout 后清 token */ }
  return { token, info, isLoggedIn, login, fetchInfo, changePassword, setToken, logout }
})
```

### 5.2 `useSiteStore` (`stores/site.ts`)

```ts
interface SiteBasic {
  siteName: string
  logo: string
  keyword: string
  description: string
  filing: string
  domain?: string
}

export const useSiteStore = defineStore('site', () => {
  const basic = ref<SiteBasic | null>(null)
  const loaded = ref(false)
  async function ensureLoaded(): Promise<void> { /* 仅首次拉 /config/basic */ }
  return { basic, loaded, ensureLoaded }
})
```

### 5.3 `useNavStore` (`stores/nav.ts`)

```ts
interface NavCategory {
  id: number
  pid: number
  name: string
  show: boolean
  children?: NavCategory[]
}

export const useNavStore = defineStore('nav', () => {
  const list = ref<NavCategory[]>([])
  const loaded = ref(false)
  async function ensureLoaded(): Promise<void> {}
  const findById = (id: number): NavCategory | undefined => { /* ... */ }
  return { list, loaded, ensureLoaded, findById }
})
```

### 5.4 `useHistoryStore` (`stores/history.ts`)

> 沿用旧版 cookie key `filmHistory`，结构兼容；新增内存映射加快渲染。

```ts
interface HistoryItem {
  id: string
  name: string
  picture: string
  source: string
  episode: string
  currentTime: number
  updatedAt: number
}

export const useHistoryStore = defineStore('history', () => {
  const list = ref<HistoryItem[]>(loadFromCookie())
  function record(item: HistoryItem): void { /* upsert 后写 cookie */ }
  function remove(id: string): void {}
  function clear(): void {}
  return { list, record, remove, clear }
})
```

### 5.5 `useUIStore` (`stores/ui.ts`)

```ts
export const useUIStore = defineStore('ui', () => {
  const sidebarCollapsed = ref(false)
  const theme = ref<'dark' | 'light'>('dark')   // 默认 dark
  const loading = ref(false)
  const loadingCount = ref(0)
  function pushLoading(): void {}
  function popLoading(): void {}
  function toggleSidebar(): void {}
  return { sidebarCollapsed, theme, loading, loadingCount, pushLoading, popLoading, toggleSidebar }
})
```

---

## 6. 网络层

### 6.1 axios 实例（`api/http.ts`）

```ts
const http = axios.create({
  baseURL: '/api',
  timeout: 80_000,
  headers: { 'Content-Type': 'application/json' }
})
```

### 6.2 请求拦截器

- 从 `useUserStore().token` 注入请求头：`config.headers['auth-token'] = token`
- `useUIStore().pushLoading()` 进度条 +1
- 统一 `params` 序列化（数组 `repeat`，跳过 `undefined`）
- 支持 `config.silent = true` 跳过全局 loading 与错误提示
- 注入 `AbortController`，并在 view `onBeforeUnmount` 时取消（通过 `useAbortable` composable）

### 6.3 响应拦截器

```ts
http.interceptors.response.use(
  (resp) => {
    useUIStore().popLoading()
    // token 续期：响应头含 new-token 则写回 store
    const newToken = resp.headers['new-token']
    if (newToken) useUserStore().setToken(newToken)

    const body = resp.data as ApiResp<unknown>
    // 业务码：旧后端约定 code === 0 / "200" 为成功（保留双兼容）
    const ok = body && (body.code === 0 || body.code === '200' || body.success === true)
    if (!ok) {
      if (!resp.config.silent) toast.error(body?.msg ?? '请求失败')
      return Promise.reject(new BizError(body))
    }
    return body.data
  },
  (err) => {
    useUIStore().popLoading()
    handleHttpError(err)              // 见下
    return Promise.reject(err)
  }
)
```

### 6.4 错误码映射

| 场景 | 处理 |
|---|---|
| `401` | 清 token → `router.replace({ path:'/login', query:{ redirect: currentFullPath }})` + toast 后端 msg |
| `403` | toast `无访问权限` |
| `5xx` / 网络错误 | toast `服务器繁忙，请稍后再试` |
| `BizError` | toast `body.msg`（已在响应拦截器处理） |

### 6.5 加载动画 service

- 由 `useUIStore.loadingCount` 驱动顶部进度条（自研 `BaseTopProgress` 或 nprogress）
- `loadingCount === 0` 才隐藏，避免并发请求闪烁
- 提供 `useLoading()` composable 给业务局部 loading（不与全局耦合）

---

## 7. API 模块化

> 所有函数都强类型，**直接返回 `data`**（响应拦截器已剥壳）。  
> 路径与 query 名 100% 兼容旧实现。

### 7.1 桶导出（`api/index.ts`）

```ts
export * from './http'
export * as filmApi  from './film'
export * as authApi  from './auth'
export * as manageApi from './manage'
```

### 7.2 用户端（`api/film.ts`）— 示例签名

```ts
import { http } from './http'
import type { IndexPageData, NavCategory, FilmDetailResp, PlayInfo,
  ClassifyData, ClassifySearchParams, PaginationResp, FilmListItem } from '@/types/film'

/** GET /api/index 首页（轮播/分类/热门聚合） */
export const getIndex = () =>
  http.get<unknown, IndexPageData>('/index')

/** GET /api/navCategory 顶级分类导航 */
export const getNavCategory = () =>
  http.get<unknown, NavCategory[]>('/navCategory')

/** GET /api/config/basic 站点基本信息 */
export const getSiteBasic = () =>
  http.get<unknown, SiteBasic>('/config/basic')

/** GET /api/filmDetail 影片详情（含相关推荐） */
export const getFilmDetail = (id: string) =>
  http.get<unknown, FilmDetailResp>('/filmDetail', { params: { id } })

/** GET /api/filmPlayInfo 播放信息 */
export const getPlayInfo = (params: { id: string; playFrom: string; episode: string }) =>
  http.get<unknown, PlayInfo>('/filmPlayInfo', { params })

/** GET /api/filmClassify 分类首页 */
export const getClassify = (Pid: number) =>
  http.get<unknown, ClassifyData>('/filmClassify', { params: { Pid } })

/** GET /api/filmClassifySearch 分类筛选 */
export const searchClassify = (params: ClassifySearchParams) =>
  http.get<unknown, PaginationResp<FilmListItem>>('/filmClassifySearch', { params })

/** GET /api/searchFilm 关键字搜索 */
export const searchFilm = (params: { keyword: string; current?: number }) =>
  http.get<unknown, PaginationResp<FilmListItem>>('/searchFilm', { params })
```

### 7.3 鉴权（`api/auth.ts`）

```ts
import type { LoginPayload, UserInfo } from '@/types/user'

export const login          = (data: LoginPayload) => http.post<unknown, void>('/login', data)
export const logout         = () => http.get<unknown, void>('/logout')
export const changePassword = (data: { oldPwd: string; newPwd: string }) =>
  http.post<unknown, void>('/changePassword', data)
export const getUserInfo    = () => http.get<unknown, UserInfo>('/manage/user/info')
```

### 7.4 后台 - 采集（`api/manage/collect.ts`）

```ts
export const list    = (params?: { current?: number; size?: number; keyword?: string }) =>
  http.get<unknown, PaginationResp<CollectSource>>('/manage/collect/list', { params })
export const options = () => http.get<unknown, CollectOption[]>('/manage/collect/options')
export const find    = (id: number) => http.get<unknown, CollectSource>('/manage/collect/find', { params: { id } })
export const remove  = (id: number) => http.get<unknown, void>('/manage/collect/del', { params: { id } })
export const add     = (data: CollectSource) => http.post<unknown, void>('/manage/collect/add', data)
export const update  = (data: CollectSource) => http.post<unknown, void>('/manage/collect/update', data)
export const change  = (data: { id: number; status: boolean }) => http.post<unknown, void>('/manage/collect/change', data)
export const test    = (data: { url: string }) => http.post<unknown, { ok: boolean; msg: string }>('/manage/collect/test', data)

// 爬虫
export const startSpider     = (data: { sourceId: number; mode: string }) =>
  http.post<unknown, void>('/manage/spider/start', data)
export const spiderClassCover = () =>
  http.get<unknown, ClassCoverItem[]>('/manage/spider/class/cover')
```

### 7.5 后台 - 定时（`api/manage/cron.ts`）

```ts
export const list   = () => http.get<unknown, CronTask[]>('/manage/cron/list')
export const find   = (id: number) => http.get<unknown, CronTask>('/manage/cron/find', { params: { id } })
export const remove = (id: number) => http.get<unknown, void>('/manage/cron/del', { params: { id } })
export const add    = (data: CronTask) => http.post<unknown, void>('/manage/cron/add', data)
export const update = (data: CronTask) => http.post<unknown, void>('/manage/cron/update', data)
export const change = (data: { id: number; status: boolean }) => http.post<unknown, void>('/manage/cron/change', data)
```

### 7.6 后台 - 影片（`api/manage/film.ts`）

```ts
export const searchList = (params: ManageFilmSearchParams) =>
  http.get<unknown, PaginationResp<FilmListItem>>('/manage/film/search/list', { params })
export const classTree  = () => http.get<unknown, FilmClass[]>('/manage/film/class/tree')
export const classFind  = (id: number) => http.get<unknown, FilmClass>('/manage/film/class/find', { params: { id } })
export const classDel   = (id: number) => http.get<unknown, void>('/manage/film/class/del', { params: { id } })
export const classUpdate = (data: FilmClass) => http.post<unknown, void>('/manage/film/class/update', data)
export const add        = (data: FilmAddPayload) => http.post<unknown, void>('/manage/film/add', data)
```

### 7.7 后台 - 文件（`api/manage/file.ts`）

```ts
export const list   = (params: { current?: number; size?: number; type?: string }) =>
  http.get<unknown, PaginationResp<FileItem>>('/manage/file/list', { params })
export const remove = (id: string) => http.get<unknown, void>('/manage/file/del', { params: { id } })
export const upload = (form: FormData, onProgress?: (p: number) => void) =>
  http.post<unknown, FileItem>('/manage/file/upload', form, {
    headers: { 'Content-Type': 'multipart/form-data' },
    onUploadProgress: e => onProgress?.(e.total ? e.loaded / e.total : 0)
  })
```

### 7.8 后台 - 系统 / 用户（`api/manage/system.ts`、`user.ts`）

```ts
// system.ts
export const dashboard = () => http.get<unknown, DashboardStat>('/manage/index')
export const getBasic  = () => http.get<unknown, SiteBasic>('/manage/config/basic')
export const updateBasic = (data: SiteBasic) =>
  http.post<unknown, void>('/manage/config/basic/update', data)

// user.ts
export const info = () => http.get<unknown, UserInfo>('/manage/user/info')
```

---

## 8. 类型定义（核心 DTO）

> 字段命名严格对照后端返回。可空字段统一可选。

```ts
// types/api.ts ----------------------------------------------------------
export interface ApiResp<T> {
  code: number | string          // 后端有 0 与 "200" 两种历史情况
  msg?: string
  success?: boolean
  data: T
}

export interface PaginationResp<T> {
  list: T[]
  total: number
  current: number
  size: number
  pages?: number
}

// types/film.ts ---------------------------------------------------------
export interface NavCategory {
  id: number
  pid: number
  name: string
  show: boolean
  children?: NavCategory[]
}

export interface FilmListItem {
  id: string | number
  mid?: string                   // 旧站 hot list 用 mid
  name: string
  picture: string
  remarks?: string
  year?: string
  area?: string
  cName?: string                 // 分类名
  cid?: number
  pid?: number
}

export interface FilmDescriptor {
  cName: string
  classTag?: string              // 逗号分隔
  director?: string
  actor?: string
  releaseDate?: string
  area?: string
  year?: string
  remarks?: string
  content?: string
  dbScore?: string | number
}

export interface PlaySource {
  id: string                     // sourceId
  name: string                   // 播放源名
  linkList: PlayEpisode[]
}

export interface PlayEpisode {
  episode: string                // 第 1 集
  link: string                   // 唯一标识 / m3u8
}

export interface FilmDetail extends FilmListItem {
  descriptor: FilmDescriptor
  list: PlaySource[]
}

export interface FilmDetailResp {
  detail: FilmDetail
  relate: FilmListItem[]
}

export interface PlayInfo {
  src: string                    // m3u8 / mp4
  type?: string                  // application/x-mpegURL
  episode: string
  link: string
  prev?: { episode: string; link: string } | null
  next?: { episode: string; link: string } | null
}

export interface ClassifyData {
  pid: number
  category: NavCategory
  newest: FilmListItem[]
  ranking: FilmListItem[]
  recent: FilmListItem[]
}

export interface ClassifySearchParams {
  Pid: number
  Category?: number | ''
  Plot?: string
  Area?: string
  Language?: string
  Year?: string
  Sort?: string
  current?: number
}

export interface IndexPageData {
  banner: FilmListItem[]
  content: Array<{
    nav: NavCategory
    movies: FilmListItem[]
    hot: FilmListItem[]
  }>
}

// types/user.ts ---------------------------------------------------------
export interface LoginPayload { username: string; password: string }
export interface UserInfo {
  uid: string
  username: string
  nickname?: string
  avatar?: string
  role?: string
}

// types/manage.ts -------------------------------------------------------
export interface SiteBasic {
  siteName: string
  logo: string
  keyword: string
  description: string
  filing: string
  domain?: string
}

export interface CollectSource {
  id: number
  name: string
  url: string
  type: string                   // json / xml
  resultModel: string
  state: boolean
  syncPictures?: boolean
}

export interface CronTask {
  id: number
  name: string
  cron: string
  jobType: string
  state: boolean
  remark?: string
}

export interface FilmClass {
  id: number
  pid: number
  name: string
  ename?: string
  show: boolean
  sort?: number
  children?: FilmClass[]
}

export interface FileItem {
  id: string
  name: string
  url: string
  size: number
  type: string
  createdAt: number
}

export interface DashboardStat {
  filmCount: number
  collectCount: number
  cronCount: number
  diskUsage?: { used: number; total: number }
}
```

---

## 9. 构建与部署

### 9.1 `vite.config.ts` 关键配置

```ts
import { defineConfig, splitVendorChunkPlugin } from 'vite'
import vue from '@vitejs/plugin-vue'
import UnoCSS from 'unocss/vite'
import AutoImport from 'unplugin-auto-import/vite'
import Components from 'unplugin-vue-components/vite'
import { fileURLToPath, URL } from 'node:url'

export default defineConfig({
  resolve: {
    alias: { '@': fileURLToPath(new URL('./src', import.meta.url)) }
  },
  server: {
    host: '0.0.0.0',
    port: 3600,
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:3601',
        changeOrigin: true,
        rewrite: p => p.replace(/^\/api/, '')
      }
    }
  },
  plugins: [
    UnoCSS(),
    vue(),
    AutoImport({
      imports: ['vue', 'vue-router', 'pinia', '@vueuse/core'],
      dts: 'src/types/auto-imports.d.ts'
    }),
    Components({
      dirs: ['src/components/base'],
      dts: 'src/types/components.d.ts'
    }),
    splitVendorChunkPlugin()
  ],
  build: {
    target: 'es2020',
    cssCodeSplit: true,
    sourcemap: false,
    rollupOptions: {
      output: {
        manualChunks: {
          'vue-vendor':   ['vue', 'vue-router', 'pinia'],
          'video-vendor': ['video.js', '@videojs-player/vue'],
          'utils-vendor': ['axios', '@vueuse/core']
        }
      }
    },
    terserOptions: { compress: { drop_console: true, drop_debugger: true } }
  }
})
```

### 9.2 产物与 Nginx

- 输出目录保持默认 `dist/`，与现有 `Dockerfile` / Nginx 集成路径一致（不需要改部署脚本）
- 生产环境 `nginx.conf` 中：
  - `location /` → `try_files $uri $uri/ /index.html`（SPA 兜底）
  - `location /api/` → `proxy_pass http://goapi-upstream/`，剥 `/api`
  - 静态资源 `cache-control: public, max-age=31536000, immutable`（对 `assets/*-[hash].*`）
  - `index.html` `cache-control: no-cache`
- iconfont 仍走 `public/iconfont/` 静态目录

### 9.3 环境变量

```
.env.development  VITE_API_BASE=/api  VITE_APP_TITLE=GoFilm-Dev
.env.production   VITE_API_BASE=/api  VITE_APP_TITLE=GoFilm
```

---

## 10. 响应式策略

### 10.1 断点 Token（`uno.config.ts` breakpoints）

| Token | min-width | 设计目标 |
|---|---|---|
| `sm` | 360px | 手机 |
| `md` | 768px | 平板 |
| `lg` | 1024px | 小桌面 |
| `xl` | 1440px | 桌面 |
| `2xl` | 1920px | 大屏 |

### 10.2 容器策略

- `< lg`：100% 流式
- `lg–xl`：max-width 1200，padding 24
- `xl–2xl`：max-width 1440
- `>= 2xl`：max-width 1600，居中

### 10.3 `useBreakpoint` composable

```ts
export function useBreakpoint(): {
  current: ComputedRef<'sm'|'md'|'lg'|'xl'|'2xl'>
  isMobile: ComputedRef<boolean>      // < md
  isTablet: ComputedRef<boolean>      // md–lg
  isDesktop: ComputedRef<boolean>     // >= lg
  match: (q: 'sm'|'md'|'lg'|'xl'|'2xl') => ComputedRef<boolean>
}
```

底层用 `window.matchMedia` + `useEventListener`，不绑 resize。

### 10.4 移动 vs PC 布局切换

- **首页** Hero：移动 → 单图轮播 200px 高；PC → 全宽渐变背景 + 海报 240px
- **FilmRow**：移动 → 横向滑动，6 张可见 1.5；PC → 网格 12 张/行
- **管理后台**：移动 → 抽屉侧栏；PC → 固定侧栏 240px
- 骨架屏（`BaseSkeleton`）按断点输出不同形状（卡片数量随断点）

### 10.5 字体与缩放

- `html { font-size: clamp(14px, 1.05vw, 18px) }`
- 业务用 rem，避免硬编码 px（间距 token: `--space-1..8`）

---

## 11. 性能预算

| 指标 | 目标 |
|---|---|
| 首屏 JS（gzip） | < 200 KB（包括 vue-vendor + 首页 view） |
| 首屏 CSS（gzip） | < 30 KB |
| LCP | < 2.5s（4G 模拟） |
| TTI | < 3.5s |
| Hero 大图 | webp + `loading="eager"` + `fetchpriority="high"` |
| 非首屏图片 | `BaseImage` 懒加载（IntersectionObserver） |
| 视频 chunk | 路由 `/play` 才加载 video.js（manualChunks 分块） |
| Pinia store | 用前导入，不在 main.ts 一次注入全部 |

监控手段：`pnpm build` 后 `vite-bundle-visualizer` 检查；CI 增加 size-limit 检查。

---

## 12. 测试策略

| 类型 | 工具 | 覆盖 |
|---|---|---|
| 单元测试 | vitest + @vue/test-utils | `utils/*`、`composables/*`、关键 store action |
| 组件测试 | vitest + jsdom | `BaseImage`、`BasePagination`、`FilmCard`（属性传递、事件） |
| 接口契约 | vitest + msw（可选） | mock 旧后端响应，校验类型/字段 |
| 手工冒烟 | 文档化 checklist | 见 `doc/handover/qa-smoke.md`（QA 输出） |

> 不做 e2e。手工冒烟覆盖：登录、首页、详情、播放、搜索、筛选、后台增删改各一例。

---

## 13. 风险点

1. **后端 code 字段历史不一致**（`0` vs `"200"`）：响应拦截器已做双兼容。仍建议联调时收敛后端。
2. **token 头 key 不固定**：旧版从 localStorage 取 `{key, value}` 结构。新版仍兼容该结构，但默认 key 固定为 `auth-token`。
3. **cookie 写入观看历史**：移动端隐私模式 / Safari ITP 可能丢；保留 localStorage fallback。
4. **iconfont CDN**：旧站资产沿用，需复制 `public/iconfont/` 进新项目。
5. **Element Plus 完全去除后**：日期选择 / 复杂表单需自研或临时按需引入；首版优先用 `<input type="date">`。
6. **`/api/index` 数据量较大**：首屏需骨架屏 + 渐进式渲染，避免阻塞 LCP。
7. **管理后台权限模型**：现有后端只有 token 鉴权，无细粒度角色；新版 store 预留 `role` 字段，但暂不做菜单级权限。

---

## 14. 关键决策记录

1. **去 Element Plus 主依赖** — 用 UnoCSS + 自研 base 组件实现 Netflix 风暗色 UI，包体减少 ~120KB gzip。Element Plus 仅在管理后台必要时按需引入。
2. **API 全量 TS 函数化** — 替代旧 `ApiGet/ApiPost` 字符串拼接，所有接口出 `function(params): Promise<DTO>`，编译期强约束 query/响应字段，IDE 直接跳转。
3. **Pinia + composable 双层** — Store 只存全局共享态（user/site/nav/history/ui），列表 / 表单仍走 view 局部 + composable，避免巨型 store。
4. **路由完全兼容旧路径与 query** — 不改 URL，保 SEO 与外链；新增功能（如 `/history`）走新路径。
5. **首屏只 import 首页 view** — layout + 首页同包，其余路由全部 `() => import()`，配合 `splitVendorChunkPlugin` + `manualChunks` 控制首屏 < 200KB。

---

## 方案确认时间：2026-05-08
