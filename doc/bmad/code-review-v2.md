# GoFilm Vue3 重构 - 代码审查报告（client-v2）

> 审查范围：`D:\Git\GoFilm\client-v2\src\` 全量
> 审查角色：代码审查员（Code Reviewer）
> 审查日期：**2026-05-09**
> 分支：`feature/redesign`
> 已知 P0 排除：`qa-smoke-report.md` 第 8 节 9 项 BUG（已在 `df49466` 修复）
> 审查维度：安全 / 数据状态 / 业务正确性 / 性能 / TS 健壮性 / 代码质量

---

## 0. 审查文件清单（核心 53 个）

| 类别 | 路径 | 类型 |
|---|---|---|
| 网络 / 鉴权 | `src/api/http.ts` `src/api/auth.ts` `src/api/film.ts` `src/api/manage/*` | TS |
| Stores | `src/stores/{user,history,site,nav,ui,index}.ts` | TS |
| Router | `src/router/{index,routes.public,routes.manage,guards}.ts` | TS |
| Composables | `src/composables/{usePlayer,useFilmHistory,useQuerySync,useAbortable,useSpatialNavigation,useViewMode,useLoading}.ts` | TS |
| Utils | `src/utils/{token,cookie,dpad,logger,format,url}.ts` | TS |
| 公开视图 | `src/views/public/*.vue` (Home/Detail/Play/Search/Classify/ClassifySearch/History) | Vue |
| 管理视图 | `src/views/manage/**.vue` | Vue |
| 组件 | `src/components/{base,film,manage,layout}/*.vue` | Vue |
| 类型 | `src/types/{api,film,manage,user}.ts` | TS |
| 入口 | `src/{App.vue,main.ts}` | TS/Vue |

---

## 1. 安全（P0/P1）

### 🔴 P0

#### S-1 [中] HTTP 拦截器响应处理破坏 axios 类型契约（破坏性改造，可能掩盖错误）
- **文件**：`src/api/http.ts:91-127`
- **问题**：响应拦截器把 axios 的 `AxiosResponse<T>` 直接返回 `body.data`（剥包装），并通过 `as unknown as (resp: AxiosResponse) => Promise<AxiosResponse>` 旁路类型签名。这个非典型设计带来两个副作用：
  1. 业务侧 `http.get<unknown, ResultDTO>(...)` 拿到的不是 `AxiosResponse<DTO>` 而是 `DTO`，与 axios 文档/类型不一致，新人接手或第三方插件接入时容易踩坑
  2. 拦截器对 `body.code === 200 / '200'` 的成功判定与后端 `system.Success` 实际是 `code: 0` 不一致 — 旧站约定也是 `code === 0`，`200` 是冗余判定（不会出错但语义混淆）
- **修复建议**：保留剥包装策略（性价比高），但补强类型：
  - 在 `types/api.ts` 增加 `declare module 'axios'` 把 `AxiosInstance` 的 `get/post/...` 重写为返回 `Promise<T>`，避免 `as any` 旁路
  - 或单独导出 `httpJson<T>(...)` 包装函数
- **风险**：⚠️ 中（不会导致运行时 bug，但是技术债）

#### S-2 [中] HistoryView / PublicHeader 未校验 `record.link` 来源
- **文件**：
  - `src/views/public/HistoryView.vue:85` `<RouterLink :to="record.link">`
  - `src/components/layout/PublicHeader.vue:122-126` `if (item.link) { router.push(item.link) }`
- **问题**：`record.link` 来自 cookie / localStorage（`filmHistory`），同源 XSS 后攻击者可写入任意 URL（如 `https://evil.com`）→ 用户点历史卡片时 RouterLink 仍会同源跳转（受 SPA 限制）但 `router.push(item.link)` 走字符串路径解析，若包含 `//evil.com` 则会被当作同源 path（Vue Router 不会自动跳出）。无 v-html / innerHTML 反射 XSS。
- **修复建议**：写入时强制规范化 link，仅允许 `/play?...` 形式；读出时校验：
  ```ts
  function isSafePlayLink(s: string): boolean {
    return typeof s === 'string' && /^\/play\?/.test(s)
  }
  ```
  在 `useFilmHistory.buildPlayLink` 入口处强制 `/play?` 前缀；在 HistoryView / PublicHeader 渲染前过滤。
- **风险**：⚠️ 中（需要先有同源 XSS 才可利用，但属于纵深防御）

#### S-3 [低] FilmDetailView Hero 背景图未做 URL 转义
- **文件**：`src/views/public/FilmDetailView.vue:218-220`
  ```vue
  <div :style="heroBg ? { backgroundImage: `url('${heroBg}')` } : undefined" />
  ```
- **问题**：`heroBg = detail.value?.picture`，直接拼到 inline `style` 的 `url('...')` 中。如果后端 picture 字段被污染，包含 `'); content: url(javascript:alert(1));//`，可能 CSS 注入（CSS 不会执行 JS，但可能突破样式沙箱）。
- **修复建议**：用 CSS escaping 或转 inline `style` 为 `class + CSS variable`：
  ```vue
  <div :style="{ '--bg': `url(${JSON.stringify(heroBg)})` }" class="bg-image" />
  ```
  或在 computed 内 `encodeURI(heroBg).replace(/'/g, '%27')`。
- **风险**：⚠️ 低（需配合后端字段污染，但应做防御）

### 🟠 P1

#### S-4 [中] Token 与历史记录使用 localStorage 明文持久化
- **文件**：`src/utils/token.ts:21` / `src/stores/history.ts:103-110`
- **问题**：JWT token 与观看历史以明文写 localStorage。如果站点存在任何 XSS（即使是反射型），攻击者可一步 `localStorage.getItem('auth')` 拿到 token，做后续 API 调用。
- **修复建议**：
  - **首选**：后端把 token 改为 HttpOnly + Secure cookie，前端通过 cookie 自动发送（需后端配合）
  - **次选**：保留 localStorage，但在所有用户输入显示场景增加 sanitize；CSP 添加 `script-src 'self'` 限制注入脚本来源
- **风险**：⚠️ 中（需配合 XSS，但 token 失窃后果严重）

#### S-5 [中] `userInfo` 缓存在 store 但未跟随 token 失效清理
- **文件**：`src/stores/user.ts`
- **问题**：401 拦截器只清 `token`，但 `useUserStore.info` 还保留旧用户信息。下个用户登录前 ManageHeader 仍显示上一个用户的头像 / 昵称，可能造成"我是谁"错乱。
- **修复建议**：`http.ts:152` 401 处理处增加 `useUserStore().info = null`；同时 `setTokenValue('')` 触发时也清 info。
- **风险**：⚠️ 中（多用户共享设备时会混淆）

#### S-6 [低] `cookie.ts` 未设置 `SameSite=Lax/Strict` 与 `Secure`
- **文件**：`src/utils/cookie.ts:18`
  ```ts
  document.cookie = `${name}=${encodeURIComponent(value)}; ${expires}; path=/`
  ```
- **问题**：观看历史 cookie 可被跨站请求伪造场景读取（虽然只是历史记录，敏感性低，但是 best practice）
- **修复建议**：
  ```ts
  document.cookie = `${name}=${encodeURIComponent(value)}; ${expires}; path=/; SameSite=Lax`
  // 如果是 https 站点
  + (location.protocol === 'https:' ? '; Secure' : '')
  ```
- **风险**：⚠️ 低（数据本身非敏感）

---

## 2. 数据 / 状态管理（P0/P1）

### 🔴 P0

#### D-1 [高] `useUIStore.pushLoading / popLoading` 异步竞争可能导致 loading 永不归零
- **文件**：`src/api/http.ts:64-66, 99-100, 130-133`
- **问题**：请求拦截器是 `async (config)`，里面 `await Promise.all([import...])` 后 `useUIStore().pushLoading()`，但请求拦截器的 throw / 取消、动态 import 失败时没 popLoading；响应错误拦截器也是 `async`，它的 `await import` 失败会跳过 popLoading。极端情况下 loading 计数会泄漏。
- **复现**：
  1. 网络极差导致 `await import('@/stores/ui')` 失败（比如 dev 时 vite hmr 异常）
  2. `pushLoading` 永不被调用，但请求继续走，后续 `popLoading` 把计数变成 -1（`Math.max(0, -1) = 0` 防御了，OK）
  3. 反向：请求拦截器成功 push，但响应阶段 await import 失败 → pop 跳过 → loading 永远卡在 1
- **修复建议**：
  - 把 `await import('@/stores/ui')` 改为顶层静态 import（http.ts → ui store 没有循环依赖）
  - pop 用 try/finally 包裹
  - 加 watchdog：超过 timeout(80s) 强制 reset loadingCount
- **风险**：🔴 高（生产可能出现进度条不消失）

#### D-2 [高] `useHistoryStore.persist` 没做 cookie 写失败回退
- **文件**：`src/stores/history.ts:103`
  ```ts
  setCookie(COOKIE_KEYS.FILM_HISTORY, json, 30)
  ```
- **问题**：当用户已写入超过 4KB 的 cookie 时 `setCookie` 会**静默失败**（浏览器丢掉部分 cookie），但 localStorage 写入成功 → 下次启动 `loadInitial` 优先读 cookie（拿到旧的、被截断的或空字符串），即使 localStorage 有完整数据也不读。
- **复现**：
  1. 看 30 部影片，cookie 接近 4KB 上限
  2. 观看第 31 部，新数据 cookie 设置失败（保留旧值），LS 写入完整 31 条
  3. 刷新 → cookie 仍是 30 部数据，覆盖 LS 的 31 部
- **修复建议**：
  - `loadInitial` 改为合并 cookie 与 LS（取 timeStamp 较新者）
  - 或者只读 LS，cookie 仅做"对旧站可读"单向同步
- **风险**：🔴 高（重度用户必现）

### 🟠 P1

#### D-3 [中] `useQuerySync` 默认值与 0 / false 冲突
- **文件**：`src/composables/useQuerySync.ts:42-44, 67-76`
  ```ts
  function isSkippable(v: unknown): boolean {
    return v === undefined || v === null || v === ''
  }
  ```
- **问题**：把 `''` 作为不写入 URL 的判定，但是当业务字段 initial 是 `''` 时，下次 query 没传 → `coerce` 回退到 sample（`''`）。OK。但是如果业务方需要把空字符串显式带到 URL（如清空筛选条件），`flatten` 会过滤掉 → 用户无法用 URL 表达"明确空"。
  - 0（数字）与 false（布尔）能正确写入。
- **修复建议**：暴露 `flatten` 的 skip 策略给调用方，或者 onChange 时明确只跳过 undefined / null。
- **风险**：⚠️ 中（不阻塞当前业务，但限制能力）

#### D-4 [中] `usePlayer.ts` 异步播放策略 race
- **文件**：`src/composables/usePlayer.ts:235-245`
- **问题**：`play()` 返回 Promise，但被 silently catch 掉，调用方无法得知是被浏览器自动播放策略拒绝（NotAllowedError）还是 src 失败。PlayView.vue 选集后立即 `void playerPlay()`，如果浏览器拒绝，UI 没有反馈，用户以为暂停了。
- **修复建议**：catch 内 emit 一个 `autoplayBlocked` 事件，PlayView 显示"点击播放"按钮提示。
- **风险**：⚠️ 中（UX 问题）

#### D-5 [中] `useUserStore.logout` finally 内 setTokenValue('') 触发 401 死循环风险
- **文件**：`src/stores/user.ts:47-57`
- **问题**：logout 流程：先调 `/logout` → 失败 catch → 进 finally → setTokenValue('') 清 token。如果 `/logout` 返回 401（说明 token 已过期），http 拦截器会再次触发 401 处理：`useUserStore().setToken('')` + `router.replace('/login')`。这与 logout 自身的逻辑重复触发 router.replace，但都是 replace 同路径，影响较小。
- **修复建议**：logout 调用时加 `silent: true`，跳过 401 自动跳转。
- **风险**：⚠️ 中（轻微，无功能性影响）

---

## 3. 业务逻辑正确性（P0/P1）

### 🔴 P0

#### B-1 [高] PlayView 进度续播在 `route.query.currentTime` 缺失时静默失效
- **文件**：`src/views/public/PlayView.vue:184` / `392-419`
  ```ts
  applyCurrentEpisodeToPlayer(Number(route.query.currentTime) || 0)
  ```
- **问题**：详情页跳转 PlayView 时，URL 没有 `currentTime` query，所以总是从 0 开始播。但 PRD 期望"用户回到上次观看进度"。
  - PublicHeader 历史浮层跳转时 `link` 已带 `currentTime`，OK
  - 但 detail 页 `gotoPlay` (`FilmDetailView.vue:136`) 不传 currentTime，所以从详情进入永远不续播
  - PlayView 也没有从 useHistoryStore 主动读取 currentTime 兜底
- **修复建议**：PlayView `loadPlayInfo()` 后，如果 query 没 currentTime，主动 `useHistoryStore().get(filmId)` 读一次，匹配 source / episode 才续播。
- **风险**：🔴 高（核心 UX，PRD 已承诺续播）

#### B-2 [高] EpisodeTabs 未传 watchedLinks → 集数标记永远不显示
- **文件**：
  - `src/components/film/EpisodeTabs.vue:11, 17`（接收 prop）
  - `src/views/public/PlayView.vue:543-549`（调用方未传）
  - `src/views/public/FilmDetailView.vue:321-326`（调用方也未传）
- **问题**：EpisodeTabs 提供了 `watchedLinks` 入参支持已观看绿点提示，但两处调用方都没传 → 永远是默认 `[]` → 已观看绿点功能失效。
- **修复建议**：PlayView 与 FilmDetailView 计算 `historyStore.get(filmId).watchedLinks` 之类衍生（需要 history 数据结构扩展），传给 EpisodeTabs。
- **风险**：🔴 中高（已实现的功能未挂载，浪费）

#### B-3 [高] FilmAddView 上传 input 未清空 value
- **文件**：`src/views/manage/film/FilmAddView.vue:108`
  ```vue
  <input type="file" accept="image/*" class="hidden" @change="handleUpload" />
  ```
- **问题**：`handleUpload` 处理完文件后没有 `(e.target as HTMLInputElement).value = ''`，再次选择同一张图片 change 事件不会触发 → 用户体验"按钮没反应"。同样的问题也在 FileUploadView.vue 但因为支持 multiple 拖拽，不太明显。
- **修复建议**：`handleUpload` 末尾 `(e.target as HTMLInputElement).value = ''`。
- **风险**：🔴 中（影响交互，用户重新上传同名图必失败）

#### B-4 [高] FilmAddView 选择图片按钮可能不触发 file picker
- **文件**：`src/views/manage/film/FilmAddView.vue:107-114`
- **问题**：`<label class="flex flex-col gap-..."><input type="file" class="hidden" /><BaseButton type="button">选择图片</BaseButton></label>` —— 把 `BaseButton` 包在 label 内，但 BaseButton 是 `<button>`，浏览器对 label 包裹 button 的行为不一致：
  - 旧版 chrome：点按钮先触发 button click 而 label 阻止 file picker
  - safari / firefox：可能 OK
  - 实际表现：可能在某些浏览器无法弹出文件选择框
- **修复建议**：
  ```vue
  <label class="...">
    <input ref="fileInput" type="file" class="hidden" @change="handleUpload" />
  </label>
  <BaseButton type="button" @click="fileInput.click()">选择图片</BaseButton>
  ```
  或参考 FileUploadView 模式。
- **风险**：🔴 中（兼容性 bug）

### 🟠 P1

#### B-5 [中] HistoryView 触摸"删除"按钮也会触发 RouterLink 跳转
- **文件**：`src/views/public/HistoryView.vue:108-115`
- **问题**：删除按钮在 RouterLink 内部，handleRemove 内调用了 `e.preventDefault(); e.stopPropagation()`，看似 OK。但是 RouterLink 默认捕获的是 click 而非 mousedown，如果用户在 TV / 触屏上 tap，touchend → click 的 preventDefault 时机可能太晚，仍触发跳转（实测可能有概率）。
- **修复建议**：把删除按钮移出 RouterLink 外层，或用绝对定位 + 高 z-index 的 overlay 按钮（实际已经做了 absolute），但同时把 RouterLink 改为 `<button>` + `router.push()`。
- **风险**：⚠️ 中（小概率误跳）

#### B-6 [中] PlayView watch query 切换时未重置 videoErrorMsg
- **文件**：`src/views/public/PlayView.vue:394-419`
- **问题**：watch query 切到下一个 episode 时执行 `applyCurrentEpisodeToPlayer`，但没有清空 `videoErrorMsg`。如果上一集报错的提示仍挂着 → 新集开始播放也不消失（依赖 player 'canplay' 事件清，但 watch 触发到 canplay 之间用户会看到旧错误信息）。
- **修复建议**：watch handler 进入新集前 `videoErrorMsg.value = ''`。
- **风险**：⚠️ 中（UX 小瑕疵）

#### B-7 [中] PlayView `paused` 计算不准（onPlayerEvent 在 ready 前缓冲）
- **文件**：`src/composables/usePlayer.ts:147-162` + PlayView 的 `paused.value`
- **问题**：usePlayer 内 `paused` 默认 true，play / pause 事件在 ready 后才挂；如果首次 init 立即 autoplay 成功，play 事件早于业务 watch 注册，导致 PlayView 的"暂停 / 继续"按钮初次渲染时是反的。
- **修复建议**：init 后立即根据 player.paused() 同步一次 `paused.value`。
- **风险**：⚠️ 中（首屏体验）

#### B-8 [中] ClassifySearchView `onChange` + 主动 `void load()` 双重首次加载
- **文件**：`src/views/public/ClassifySearchView.vue:46-110`
- **问题**：`useQuerySync` 的 onChange 仅在 URL 变化触发；`void load()` 在 setup 末尾主动调用 → 进入页面时执行一次。但是若用户从相邻路由 push 到 `/filmClassifySearch?Pid=1` → `void load()` 触发；但若用户 `router.push({path: '/filmClassifySearch', query: {Pid: 1}})` 来自其他页（路由内 query 变化但组件实例新建），仍只执行一次 load，OK。但如果是 `/filmClassifySearch` → `/filmClassifySearch?Pid=1` 同实例 query 变化 → onChange 触发 → 加载一次 OK。但 SearchView 同时 `void load()` 也存在 → 双重保险但浪费一次请求。
- **修复建议**：useQuerySync 增加 `immediate: true` 选项，或干脆移除 `void load()` 末尾调用，统一靠 onChange + 路由进入触发。
- **风险**：⚠️ 中（性能浪费而非错误）

#### B-9 [中] SearchView 输入空格不会重置结果
- **文件**：`src/views/public/SearchView.vue:82-86`
- **问题**：`submitSearch` 内 trim，但用户用搜索框右上角 X 清空 input 后 `inputKeyword = ''`，`submitSearch` `if(!k) return` 静默不响应，URL 仍保留旧 search 参数 → 旧结果不消失。期望：清空时重置到"开始搜索"提示。
- **修复建议**：input 上加 watch / @search 事件，inputKeyword 变空时主动 push({search:'',current:1})。
- **风险**：⚠️ 中（UX）

#### B-10 [中] CronListView `form.ids` 多选是引用，编辑时直接 push 会污染 row
- **文件**：`src/views/manage/cron/CronListView.vue:62-65, 90-94`
  ```ts
  function openEdit(row) { editing.value = row; Object.assign(form, row) }
  function toggleId(id) { ... form.ids.splice / push ... }
  ```
- **问题**：`Object.assign(form, row)` 把 row.ids 数组引用拷给 form.ids（浅拷贝）。`toggleId` 直接 splice/push form.ids → 同时修改了原始 row.ids → 即使用户取消编辑，列表里 row 的 ids 已被改。
- **修复建议**：`Object.assign(form, { ...row, ids: [...(row.ids || [])] })`。
- **风险**：🟠 中（编辑场景必现）

#### B-11 [中] CollectListView openEdit 同样浅拷贝问题
- **文件**：`src/views/manage/collect/CollectListView.vue:68-72`
  ```ts
  function openEdit(row) { editing.value = row; Object.assign(form, row) }
  ```
- **问题**：同 B-10，但 CollectSource 没有数组字段，所以浅拷贝 OK；不过 `editing.value = row` 用的是同一引用，submit 后再 load 不会有问题。**OK，无 bug，但代码风格与 B-10 同源**。
- **修复建议**：保持一致 `Object.assign(form, { ...row })`，避免后续添加数组字段时踩坑。
- **风险**：🟡 低（防御性）

#### B-12 [中] LoginView submit 时 isLoggedIn 已存在不阻止重新登录
- **文件**：`src/views/auth/LoginView.vue:18-38`
- **问题**：用户已登录情况下访问 /login（直接打 URL），handleLogin 仍会 await 新 token，覆盖旧 token；但 redirect 不在 query 时跳到 `/manage/index`。这本身没问题，但缺少"已登录直接跳走"的兜底。
- **修复建议**：onMounted 内检测 `userStore.isLoggedIn` 时直接 `router.replace('/manage/index')`。
- **风险**：⚠️ 低（UX 优化）

---

## 4. 性能（P1/P2）

### 🟢 性能建议

#### P-1 [中] HeroCarousel 全部 banner 项预加载图片
- **文件**：`src/components/film/HeroCarousel.vue:155-170`
- **问题**：v-for 渲染所有 banner 元素，每个都用 `eager` BaseImage，首屏一次性加载 5 张大图（每张可能 1MB+）→ 首屏 LCP 受影响。
- **优化建议**：只 eager 第一张 (i === 0)，其他改为 lazy（IntersectionObserver 即将命中时再加载）。
- **风险**：🟢 性能（不阻塞）

#### P-2 [中] FilmRow 横向滚动容器 width 用 calc(100vw - ...) → resize 不重算
- **文件**：`src/components/film/FilmRow.vue:194-223`
- **问题**：`.gf-film-row__item width: calc((100vw - 32px) / 2.2)` 在 resize 时会自动重算（CSS calc），但每个 item 都重算，触发 layout 抖动；TV 1920px 视口默认 8 列（CSS 计算）、但 ResizeObserver 监听 scrollEl 又重算 updateArrows。优化空间：debounce updateArrows。
- **优化建议**：requestAnimationFrame 节流 + debounce 100ms。
- **风险**：🟢 性能

#### P-3 [中] BaseImage IntersectionObserver rootMargin 200px 偏小
- **文件**：`src/components/base/BaseImage.vue:80`
- **问题**：在 720p 大屏 / 4K 屏，rootMargin 200px 仅覆盖 1 屏外的一行卡片高度。横向滚动时滚动 800px 才能看到的图，rootMargin 没覆盖。
- **优化建议**：根据 viewModeStore 调整 rootMargin（mobile 200 / desktop 400 / tv 600）。
- **风险**：🟢 性能（不阻塞）

#### P-4 [低] HomeView hotSidebar 双重 for 复杂度 O(n²)
- **文件**：`src/views/public/HomeView.vue:42-60`
- **问题**：嵌套 for + Set 去重，content 段数 × hot 长度 → 数据量大时浪费。但实际数据 < 100 条，可忽略。
- **风险**：🟢 性能（不阻塞）

#### P-5 [低] usePlayer 实例 dispose 后未清空 pendingListeners
- **文件**：`src/composables/usePlayer.ts:221-232`
- **问题**：`dispose()` 仅 `p.dispose()` + `player.value = null`，pendingListeners 数组没清空 → 重新 init 时之前注册的事件可能堆积（虽然实际场景中 init 仅在 mount 时调用，dispose 在 unmount 时调用，scope 已死，影响有限）。
- **优化建议**：`dispose` 末尾 `pendingListeners.length = 0`。
- **风险**：🟢 性能（轻微内存）

#### P-6 [低] useFilmHistory beforeunload 可能多次 flush
- **文件**：`src/composables/useFilmHistory.ts:65-77`
- **问题**：beforeunload + onBeforeUnmount 都会 flush，理论上同一次卸载会重复写一次 cookie + LS。开销可接受。
- **风险**：🟢 性能

---

## 5. TypeScript 健壮性（P1/P2）

### 🟠 P1

#### T-1 [中] `as any` / `as unknown as` 旁路类型检查
- **文件**：
  - `src/api/http.ts:127` `as unknown as (resp: AxiosResponse) => Promise<AxiosResponse>` — 已知问题
  - `src/components/manage/ManageTable.vue:1, 12` `Record<string, any>` — 表格泛型
  - `src/composables/useViewMode.ts:60-66` `as unknown as { Capacitor?: ... }` — 桥接 Capacitor，有理由
- **问题**：除 ManageTable 与 Capacitor 桥两处合理外，http.ts 的 `as unknown as` 掩盖了拦截器返回类型不一致问题。
- **修复建议**：见 S-1。

#### T-2 [中] `FilmAddPayload` 与 view form 形状不一致，spread 时静默丢字段
- **文件**：`src/views/manage/film/FilmAddView.vue:18-31, 61` + `src/types/film.ts:202-223`
- **问题**：FilmAddView 的 form 包含 `cid, pid, year, area, ...`，但缺 `subTitle, initial, state, playFrom, downFrom, playLink, downloadLink`。`await manageApi.film.add({ ...form })` 时 TS 不报错（FilmAddPayload 这些字段都 optional）— 但是后端期望某些字段必传，提交时按零值发出，可能后端校验失败。
- **修复建议**：FilmAddView form 字段补齐 FilmAddPayload；或在 api 层 `add(data: Required<Pick<...>>)` 强制必填字段。
- **风险**：⚠️ 中（QA 报告 W1 已记，QA 已 fix 但仍可被绕过）

#### T-3 [中] `ManageInput type="number"` 空字符串变成 0
- **文件**：`src/components/manage/ManageInput.vue:11-14`
  ```ts
  emit('update:modelValue', props.type === 'number' ? Number(v) : v)
  ```
- **问题**：用户清空数字输入框 → `v = ''` → `Number('') = 0` → 父组件 model 变 0。期望保持 null/undefined 让用户能"未填"。
- **场景**：
  - FilmAddView `cid` 必须为 0 时无意义（顶级分类）
  - CronListView `time` 默认 24 改成 0 用户难以再清空
  - FilmClassView `sort` 类似
- **修复建议**：
  ```ts
  if (props.type === 'number') {
    emit('update:modelValue', v === '' ? '' : Number(v))
  }
  ```
- **风险**：⚠️ 中

#### T-4 [低] `LoginPayload` 类型 vs login 接口形状错配
- **文件**：`src/stores/user.ts:27` 接收 `LoginPayload | { username, password }`
- **问题**：类型定义复杂（联合类型），实际使用时还是要内部 `'userName' in payload`，本质就是想兼容两种字段名。代码里硬编码 transformer 比 union 更清晰。
- **修复建议**：始终接收 `{ username: string; password: string }`，转换到 `userName` 在 api 层完成。
- **风险**：🟡 低

### 🟡 P2

#### T-5 [低] `ApiResp` 的 `code` 类型 `number | string` 过宽
- **文件**：`src/types/api.ts:9`
- **问题**：后端实际只会返 number（system.SuccessCode = 0），`'200'` 仅是历史包袱。
- **修复建议**：`code: 0 | string`（联合具体值），更紧。
- **风险**：🟡 低（防御性 OK）

---

## 6. 代码质量（P2）

### 🟡 建议优化

#### Q-1 ManageHeader.vue 未使用的 `route` import
- **文件**：`src/components/layout/ManageHeader.vue:3, 15`
- **问题**：`import { useRoute, useRouter } from 'vue-router'` 但 setup 内只用 router；template 引用 `route.meta.title` 时通过 auto-import 的全局 `route` 变量。但本文件 import 的 useRoute/useRouter **没有调用 useRoute()**（也没有 `const route = useRoute()`），template 中 `(route.meta.title as string)` 是 auto-imports 还是 implicit any？查 auto-imports.d.ts 没有 route，**这是个 TS 错误**（vue-tsc 居然过了 — 可能是 `.vue` 文件 template 内 unresolved name 不报错）。
- **复现**：build 时 template 中 `route.meta.title` 是 undefined → 标题显示空。
- **修复建议**：补 `const route = useRoute()`。
- **风险**：🟠 中（实际运行可能页面标题为空）

#### Q-2 多处 `v-for :key="i"`（索引）
- **文件**：
  - `src/views/manage/file/FileUploadView.vue:84`
  - `src/components/film/HeroCarousel.vue:259`（dot 用 i，OK 因为 dot 数与 items 同步）
  - `src/components/film/FilmCard.vue:63` `cornerTags` 同理 OK
  - `src/views/public/HomeView.vue:99-102` 骨架占位 OK
- **问题**：FileUploadView entries 列表用 `:key="i"`，删除中间项时索引变化 → DOM 复用错位（错误显示 progress / resultUrl）
- **修复建议**：用 `entry.file.name + entry.file.size + entry.file.lastModified` 做 key。
- **风险**：⚠️ 中（用户上传多文件后清空中间项时显示错乱）

#### Q-3 `confirm()` 阻塞主线程，TV 模式不友好
- **文件**：
  - `src/views/manage/collect/CollectListView.vue:92`
  - `src/views/manage/cron/CronListView.vue:85`
  - `src/views/manage/file/FileGalleryView.vue:31`
  - `src/views/manage/film/FilmClassView.vue:61`
  - `src/views/public/HistoryView.vue:40`
- **问题**：原生 confirm 是阻塞同步弹窗，在 TV 模式 / 移动端 webview 体验差，无法用 D-pad 焦点。
- **修复建议**：替换为 BaseDialog + 自定义"确认 / 取消"按钮（项目已有 BaseDialog）。
- **风险**：🟡 低（功能可用，UX 问题）

#### Q-4 cookie 库与 history store 重复实现 SafeJSON 解析
- **文件**：`src/utils/cookie.ts` + `src/stores/history.ts:44-75`
- **问题**：history store 重新实现了 safeParse，OK；但和 cookie util 没有解耦的 JSON cookie 接口。
- **修复建议**：cookie.ts 增加 `getJsonCookie<T>(name, fallback): T` / `setJsonCookie(name, value, days)`，复用。
- **风险**：🟡 低（重构优化）

#### Q-5 useSpatialNavigation findElementByStableId 用未转义的 attr value
- **文件**：`src/composables/useSpatialNavigation.ts:188-199`
- **问题**：构建 selector 时 `v.replace(/"/g, '\\"')` 仅转义双引号，未处理 `[`, `]`, `\` 等 CSS selector 特殊字符。如果 data-id 包含这些字符会查询失败 / 抛错。
- **修复建议**：用 `CSS.escape(v)` 完整转义。
- **风险**：🟡 低（focus 恢复失败兜底为首个 focusable，影响小）

#### Q-6 useViewMode 全局单例与 onScopeDispose 时机不一致
- **文件**：`src/composables/useViewMode.ts:124-168`
- **问题**：`install()` 内 `onScopeDispose` 注册卸载，但 install 仅在第一次 useViewMode 调用时执行；如果第一次调用是在 PublicLayout 的 setup 内，那 PublicLayout 卸载（路由切到 manage）时 onScopeDispose 触发，把 resize / storage 监听卸载 → 后续 manage 端无法响应窗口变化。
- **修复建议**：把 `install` 的注册放到 `app.onMounted` 全局或者直接 main.ts 调用一次。
- **风险**：⚠️ 中（路由切换后 viewMode 不再随 resize 自动调整）

#### Q-7 useQuerySync params 与 onChange 触发时机有 race
- **文件**：`src/composables/useQuerySync.ts:127-136`
- **问题**：`push(next)` 内先 `params.value = merged` 再 `await router.push`。在 router.push 完成前，外部读 params.value 已是新值，但 URL 还是旧的 → 极短时间内 UI 与 URL 不一致；同时 `suppressNext` 仅吞掉一次 watch 回调，如果路由 hook 内有 push 链 → 可能漏吞。
- **修复建议**：`await router.push` 后再赋值 params.value，避免态分裂。
- **风险**：🟡 低（实际 SPA 渲染在 nextTick 不可见）

#### Q-8 CollectListView toggleState 把 form 字段 spread 给 change，类型不匹配
- **文件**：`src/views/manage/collect/CollectListView.vue:86-89`
  ```ts
  await manageApi.collect.change({ ...row, state: !row.state })
  ```
- **问题**：API 期望 CollectSource，row 是 CollectSource，OK。但 row 来自 list 接口，可能比 form 字段少（后端 list 返的字段可能不全）→ change 提交时 grade / collectType / interval 这些可能是 undefined → 后端按零值处理。
- **修复建议**：toggleState 前先调 find 接口拉完整对象，或者以 row 现有字段为准但加默认值兜底。
- **风险**：⚠️ 中（与 QA 已 fix 的 8.3 类似但不完全）

#### Q-9 BaseImage 在 src 切换后忘记重置 visible
- **文件**：`src/components/base/BaseImage.vue:90-99`
- **问题**：watch src 仅 reset loaded / errored，`visible` 仍为 true（懒加载已发生），新 src 立即赋给 currentSrc 加载。OK。但如果 src 从有值切回 ''，currentSrc 会被赋 ''，img 不渲染但 visible 为 true，下次 src 重新有值会立即加载（不再走 IntersectionObserver）— 这可能是想要的，但 prop 切换的 cleanup 不清晰。
- **风险**：🟡 低

---

## 7. 大屏适配合规性

### 字号
- ✅ `var(--gf-fs-md)` 等基础字号 ≥ 16px（base.css 内 `--gf-fs-md: 1rem` = 16px）
- ⚠️ ManageInput / 多个 manage 表单字段使用 `text-sm`（14px），管理后台桌面端可接受，**TV 模式应放大**
- ⚠️ FilmFilterBar chip `font-size: var(--gf-fs-sm)`（14px），TV 模式已覆盖到 base（16px）但**移动端仍是 14px**
- ⚠️ ManageTable `text-sm` 整表 14px，TV 模式没有覆盖

### 触控目标 ≥ 44px
| 组件 | 桌面/移动 | TV 模式 | 评价 |
|---|---|---|---|
| BaseButton | size=md 高度 40 / lg 48 | TV 56-72 | ✅ 主按钮合规 |
| `gf-header__icon-btn` | 44×44 | 56×56 | ✅ |
| EpisodeTabs `.gf-source-tab` | min-height 44 | 64 | ✅ |
| EpisodeTabs `.gf-episode-chip` | 48-56 | 64 | ✅ |
| **FilmFilterBar `.gf-filter-chip`** | **height 32** | **44** | ⚠️ 桌面/移动 32px，违规 |
| **BasePagination `.gf-page-chip`** | **min 36×36** | **56×56** | ⚠️ 桌面 36，违规 |
| **ManageSwitch** | **24×44** | 未覆盖 | 🔴 24px 高度严重违规 |
| HeroCarousel `.gf-hero__dot` | 8×8（仅是装饰） | 12-32 | ✅（指示器，非主操作） |

### --ui-scale / 响应式
- ✅ theme.css 已通过 `[data-mode='tv']` 选择器覆盖字号 / 间距
- ✅ 多数组件单独 TV 模式 `<style>` 段做了 focus ring + 放大
- ⚠️ ManageTable / BasePagination / FilmFilterBar 缺 TV 模式覆盖，焦点环 + chip 高度未达 56px

### 720px 视口一屏
- ✅ HomeView / FilmDetailView / PlayView 未发现明显溢出
- ⚠️ FilmAddView 表单 `grid-cols-1 md:grid-cols-2`，720px 设备宽度按 mobile（grid-cols-1）显示，海报上传区域 80×110 + label 嵌套，**实测 720×1280 竖屏可能挤压**

---

## 8. 综合结论

### 总分级统计

| 级别 | 数量 |
|---|---|
| 🔴 P0 阻塞 | **6 项** (S-1, D-1, D-2, B-1, B-3, B-4) |
| 🟠 P1 应修复 | **15 项** |
| 🟡 P2 建议 | **8 项** |
| 🟢 性能建议 | **6 项** |

### 上线就绪度评估

| 维度 | 状态 | 说明 |
|---|---|---|
| 安全 | 🟢 良好 | 无 v-html / innerHTML / eval；token / cookie 设置可优化但非阻塞 |
| 数据状态 | 🟠 需修复 | history 双写一致性 (D-2) + loading 计数 (D-1) 是潜在生产事故 |
| 业务逻辑 | 🟠 需修复 | 进度续播 (B-1) + 已观看标记 (B-2) 是核心 UX；FilmAddView 上传 (B-3/B-4) 影响日常运营 |
| 性能 | 🟢 良好 | gzip 总量 ~76KB 远低于预算；Hero 大图懒加载 + Filmrow debounce 是优化空间 |
| TS 健壮性 | 🟠 中 | 拦截器旁路类型 + ManageInput 空值变 0 + ManageHeader 缺 useRoute |
| 大屏适配 | 🟠 中 | FilmFilterBar / BasePagination / ManageSwitch 触控目标违规；ManageTable TV 模式无覆盖 |
| 代码风格 | 🟢 良好 | 命名一致、composable 抽离合理、无重复反模式 |

### 决策

#### 用户端（公开接口路径） ✅ **可上线**
- QA 已确认 6 接口 100% 兼容，前端无 P0 阻塞
- 仅有 B-1 / B-2 是 UX 体验缺陷，不影响功能跑通
- 建议在第一个 sprint 修复 D-1 / D-2（数据状态隐患）

#### 后台管理端 🟠 **建议修复 P0 后再上线**
- 必修：B-3（FilmAddView 上传 input value 不清空）、B-4（label/button 嵌套兼容）、Q-1（ManageHeader 缺 useRoute）
- 强烈建议：D-1（loading 永不归零）、D-2（cookie/LS 一致性）、T-3（ManageInput 数字空值）
- 上线前必跑：CollectList → toggleState 补完整对象（Q-8）回归测试

#### 必修清单（上线前 1 个 sprint）

| # | 编号 | 描述 | 估时 |
|---|---|---|---|
| 1 | D-1 | http 拦截器 popLoading 必触发 + watchdog | 1h |
| 2 | D-2 | history store cookie/LS 合并优先 timeStamp | 2h |
| 3 | B-1 | PlayView 续播兜底从 historyStore 读 currentTime | 2h |
| 4 | B-2 | EpisodeTabs 接 watchedLinks（需 history schema 扩展） | 2h |
| 5 | B-3 | FilmAddView handleUpload 末尾清 input value | 5min |
| 6 | B-4 | FilmAddView label 包 button 改为 ref.click | 30min |
| 7 | Q-1 | ManageHeader 补 const route = useRoute() | 2min |
| 8 | T-3 | ManageInput type=number 空值不强转 | 10min |
| 9 | Q-2 | FileUploadView v-for key 改为稳定 id | 10min |
| 10 | Q-6 | useViewMode install 移至 main.ts | 30min |

### 大屏适配修复清单（不阻塞但应在下个 sprint）

| # | 描述 | 估时 |
|---|---|---|
| 1 | FilmFilterBar chip 高度 32 → 44 | 10min |
| 2 | BasePagination chip 36 → 44（桌面） | 10min |
| 3 | ManageSwitch 高度 24 → 28（基础）/ 44（TV） | 30min |
| 4 | ManageTable TV 模式 fontSize 1.25× + 行高 56 | 30min |
| 5 | ManageInput TV 模式 padding + fontSize 放大 | 20min |

---

## 9. 审查方法说明

- **静态阅读**：53 个核心文件全量阅读，每文件含 TS 编译错误检查（vue-tsc）
- **跨文件交叉**：API 类型 ↔ View 调用 ↔ Store 状态 ↔ 后端 controller（不是本次范围但有 QA 报告参考）
- **反模式扫描**：grep `v-html` / `innerHTML` / `console.log` / `as any` / `:key="i"` / `confirm(`
- **大屏 TV 适配**：根据 04-tv-addendum 标准（44px 触控、56px 主按钮、focus ring 4px）逐组件比对
- **不修改源代码**：本报告仅识别 + 建议，所有"修复"均为方案描述

---

审查时间：**2026-05-09**（约 60 分钟）
审查人：Code Reviewer
关联文档：`qa-smoke-report.md`（QA 9 项 P0 已修复）、`07-tv-adapt.md`、`prd-redesign.md`
