# 架构 → 前端开发交接说明

> 阅读顺序：先看 `doc/handover/00-context.md`，再看 `doc/bmad/architecture-redesign.md`，最后本文件。  
> 本文件是**落地速查表 + 编码守则 + 反模式清单**，目标让前端开发拿到任务后能直接定位文件。

---

## 1. 文件落点速查表

| 我要做… | 改 / 新建这些文件 |
|---|---|
| **首页（轮播 + 多分类 row）** | `src/views/public/HomeView.vue`<br>`src/components/film/HeroCarousel.vue`<br>`src/components/film/FilmRow.vue`<br>`src/components/film/FilmCard.vue`<br>`src/api/film.ts::getIndex` 已就绪 |
| **影片详情** | `src/views/public/FilmDetailView.vue`<br>`src/components/film/RelatedList.vue`<br>`src/api/film.ts::getFilmDetail`<br>`src/types/film.ts::FilmDetail` |
| **播放页** | `src/views/public/PlayView.vue`<br>`src/components/film/EpisodeTabs.vue`<br>`src/composables/usePlayer.ts`<br>`src/composables/useFilmHistory.ts`（cookie key `filmHistory`） |
| **搜索页** | `src/views/public/SearchView.vue`<br>`src/api/film.ts::searchFilm`<br>`src/composables/usePagination.ts` |
| **分类首页** | `src/views/public/ClassifyView.vue`<br>`src/api/film.ts::getClassify` |
| **分类筛选** | `src/views/public/ClassifySearchView.vue`<br>`src/components/film/FilmFilterBar.vue`<br>`src/composables/useQuerySync.ts` |
| **顶部 Header / Footer** | `src/components/layout/PublicHeader.vue` / `PublicFooter.vue`<br>读取 `useSiteStore` + `useNavStore` |
| **登录** | `src/views/auth/LoginView.vue`<br>`src/api/auth.ts::login`<br>`src/stores/user.ts::login` |
| **后台仪表盘** | `src/views/manage/DashboardView.vue`<br>`src/api/manage/system.ts::dashboard` |
| **后台采集源管理** | `src/views/manage/collect/CollectListView.vue`<br>`src/api/manage/collect.ts` |
| **后台定时任务** | `src/views/manage/cron/CronListView.vue`<br>`src/api/manage/cron.ts` |
| **后台影片管理** | `src/views/manage/film/FilmListView.vue` 等 4 个<br>`src/api/manage/film.ts` |
| **后台文件** | `src/views/manage/file/FileUploadView.vue`<br>`src/api/manage/file.ts::upload`（含 onProgress） |
| **后台站点配置** | `src/views/manage/system/SiteConfigView.vue`<br>`src/api/manage/system.ts::getBasic / updateBasic` |
| **新增一个全站 store** | `src/stores/xxx.ts`，并在 `src/stores/index.ts` 导出 |
| **新增 base 组件** | `src/components/base/BaseXxx.vue`（自动注册，无需 import） |
| **新增 composable** | `src/composables/useXxx.ts`，命名 `use` 开头，函数式返回对象 |
| **新增类型** | `src/types/xxx.ts`，避免在 view 内裸写 interface |
| **改 / 加路由** | `src/router/routes.public.ts` 或 `routes.manage.ts`，meta 必须填 `layout` |
| **错误码 / 全局错误处理** | `src/api/http.ts`，**不要在业务代码里 try-catch toast** |
| **全局 loading** | 已自动；如需局部 loading 用 `useLoading()` composable |
| **图片懒加载 / 占位** | 直接用 `<BaseImage :src="..." />`，**不要写 `<img>`** |
| **断点判断** | `const { isMobile, isDesktop } = useBreakpoint()`，**不要监听 resize** |
| **观看历史** | `useHistoryStore().record({ ... })`，cookie 已沿用旧 key |

---

## 2. 编码风格守则（5 条）

1. **`<script setup lang="ts">` 一律组合式**。组件 props/emits 用 `defineProps<T>()` / `defineEmits<{...}>()` 泛型形式，不写 runtime 声明。
2. **API 调用必经 `api/*.ts`，view 内禁止 `axios.get`**。view 拿到的就是已剥壳的 DTO，不要再访问 `.data.data`。
3. **样式优先 UnoCSS 原子类 + attributify**（`<div text-sm text-white/70 hover:bg-zinc-800>`）。需要复用的样式抽 `shortcuts`（`uno.config.ts`），少写 `<style scoped>`。
4. **路由 query 出入口集中在 `useQuerySync` / `router.push({ query })`**。组件渲染期间不要直接读写 `window.location.search`。
5. **类型先行**：先在 `types/` 写 DTO，再在 `api/` 写函数，最后在 view 用。`any` 必须注释原因。

---

## 3. 必须避免的反模式

> 以下来自旧项目踩过的坑 + Vue3 常见陷阱。**违反任意一条，code review 直接打回**。

### 3.1 不要 `watch(route, ...)` 来响应 query 变化
旧站到处是 `watch(() => route.query, ...)` 触发请求，会出现重复请求 / 死循环。  
**正确做法**：用 `useQuerySync` composable，把 query 双向绑定到 ref，再 `watchDebounced` 这些 ref 触发请求。

### 3.2 不要 `location.href = '/filmDetail?link=...'`
旧站点为了"刷新页面"用裸跳转，新站全部走 `router.push({ path: '/filmDetail', query: { link: id }})`。  
SPA 内不要触发整页刷新；详情→播放也要 `router.push`，不要 `<a href>`。

### 3.3 不要在组件 template 里手拼 query 字符串
错误：`:href="\`/filmClassifySearch?Pid=${pid}&Category=${cid}\`"`  
正确：`<router-link :to="{ path:'/filmClassifySearch', query:{ Pid: pid, Category: cid }}">` 或封装 `<FilmCategoryLink>`。

### 3.4 不要在 view 内做全局 toast / loading 包裹
错误：每个请求 `try { load.start(); await api(); ElMessage.success(...) }`  
正确：响应拦截器统一处理；业务只关心成功后的赋值。需要静默用 `http.get(url, { silent: true })`。

### 3.5 不要绕过 store 直接读 localStorage / cookie
错误：组件里 `JSON.parse(localStorage.getItem('auth'))`  
正确：`const { token, isLoggedIn } = storeToRefs(useUserStore())`；store 内部才碰存储。

### 3.6 不要给 reactive 对象重新赋值
错误：`data = reactive({}); data = { ... new ... }` —— 丢响应性  
正确：用 `ref()` 整体替换，或 `Object.assign(data, newObj)`。

### 3.7 不要在 `onMounted` 里同步多个串行请求
错误：`await getNav(); await getSite(); await getIndex();` 串行 3 个 RTT  
正确：`await Promise.all([navStore.ensureLoaded(), siteStore.ensureLoaded(), getIndex()])`。  
更好：nav/site 移到路由 `beforeEach` 或 App.vue setup 阶段一次性预热。

### 3.8 不要在 manage 后台用 Element Plus 全量
错误：`import ElementPlus from 'element-plus'; app.use(ElementPlus)`  
正确：仅按需 `import { ElDatePicker } from 'element-plus'`，并独立 import css；优先用 base 组件。

### 3.9 不要把响应式数据 `JSON.stringify` 后存 cookie
旧站观看历史这么写过，导致 reactive proxy 序列化偶尔报错。  
正确：先 `toRaw()` 或用普通对象再写 cookie；`useHistoryStore` 已封装。

### 3.10 不要忽略 AbortController
路由切换时，未结束的列表请求可能写到下一个页面。**列表/搜索请求必须带 `signal`**，或用封装 `useAbortable(api)` composable。

### 3.11 不要硬编码颜色 / 间距
错误：`background: #0b0b0f`  
正确：`bg-bg-base`（UnoCSS shortcut，对应 `--color-bg-base` 变量），便于换主题。Token 见 `assets/styles/theme.css`。

### 3.12 不要在 video.js 实例上直接 `v-model`
播放器封装在 `usePlayer` composable，组件只调 `playerRef.value.play()` 之类的方法；事件用 `@ready @ended @timeupdate`。

---

## 4. 开发节奏建议

1. **首版先写骨架**：`main.ts` → `router` → `App.vue` + 两个 layout → `Home`（先 mock 数据）→ 通走起来。
2. **再接接口**：把 `api/` + `types/` 落地，逐 view 替换 mock。
3. **最后做样式打磨**：UnoCSS shortcuts 抽取、动画、骨架屏。
4. **管理后台与用户端并行**：两块互不阻塞，但优先保证用户端首页 / 详情 / 播放。

---

## 5. 联调清单（启动前确认）

- [ ] 后端运行在 `127.0.0.1:3601`
- [ ] `pnpm install` 成功，`pnpm dev` 起在 `:3600`
- [ ] `/api/index` 能正常代理返回 JSON
- [ ] `localStorage.auth` 结构 `{ key:'auth-token', value:'xxx' }` 与旧站一致
- [ ] `iconfont/` 已复制进 `public/`
- [ ] `vite.config.ts` proxy 与旧站一致

完成以上 → 开始 ticket。

---

## 6. 文档索引

- 共享上下文：`doc/handover/00-context.md`
- PRD（产品）：`doc/bmad/prd-redesign.md`
- 视觉规范：`doc/design/visual-spec.md`
- 架构主文档：`doc/bmad/architecture-redesign.md`
- 接口签名详表：见架构主文档第 7 章
- DTO 类型定义：见架构主文档第 8 章
