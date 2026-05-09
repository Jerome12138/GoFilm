# GoFilm Vue3 重构 - QA 全量静态回归报告（V2）

> 测试目标：`client-v2/`（Vue 3.5 + Vite 5.4 + TS 5.6 + Pinia + UnoCSS）
> 测试日期：**2026-05-09**
> 测试人员：QA Engineer
> 测试范围：构建验证 + 反模式扫描 + 路由/API 契约对照 + HistoryView 专项 + TV/无障碍核对
> 后端：未启动（仅静态契约对照）
> 测试基线：commit `df49466`（修复 9 项 P0）+ HistoryView follow-up + vite 配置清理
> 上一轮报告：`doc/bmad/qa-smoke-report.md`

---

## 0. 执行摘要

| 维度 | 上一轮（df49466 前） | 本轮 | 趋势 |
|---|---:|---:|---|
| 构建（TS + Vite） | 通过（含 splitVendor 警告） | **通过（0 警告，构建 40.0s）** | ✅ 改善 |
| 首屏 gzip JS | 76.6 KB | **76.9 KB** | 持平 |
| P0 BUG | 9 | **1**（HistoryView 进度字段丢失） | ✅ 大幅改善 |
| P1 BUG | 0 | 2 | ⚠️ 新引入 |
| P2 BUG | 6 | 5 | ✅ |
| 反模式残留 | 0 | **0** | ✅ |
| 路由 / API 契约 | 9 处不匹配 | **0 处不匹配** | ✅ 全部修复 |

---

## 1. 测试环境

| 项 | 值 |
|---|---|
| 工作目录 | `D:\Git\GoFilm\client-v2\` |
| Node / pnpm | 已就绪 |
| Vite | 5.4.11 |
| Vue | 3.5.x |
| TypeScript | 5.6.x（vue-tsc） |
| 后端 | 未启动（静态对照 `D:\Git\GoFilm\server\controller\*.go`） |
| 旧站参照 | `D:\Git\GoFilm\client\src\router\router.ts` + `client\src\views\` |

---

## 2. 用例统计

| 类别 | 通过 | 失败 | 警告 |
|---|---:|---:|---:|
| 构建验证（vue-tsc + vite build） | 2 | 0 | 0 |
| 反模式静态扫描 | 6 | 0 | 0 |
| 路由表对照（与旧站 + 架构表） | 19 | 0 | 0 |
| 用户端 API 契约（8 接口） | 8 | 0 | 0 |
| 鉴权 API 契约（4 接口） | 4 | 0 | 0 |
| 管理端 API 契约（27 接口） | 27 | 0 | 0 |
| HistoryView 专项 | 6 | **1** | 1 |
| TV / 无障碍 | 5 | 0 | 1 |
| 边界 / 风险点 | 4 | 0 | 2 |
| **合计** | **81** | **1** | **4** |

---

## 3. 构建验证

### 3.1 TypeScript 类型检查

```
$ pnpm exec vue-tsc -p tsconfig.app.json --noEmit
（0 输出 → 0 error）
```

✅ **通过**

### 3.2 生产构建

```
$ pnpm build
✓ 326 modules transformed
✓ built in 40.00s
```

✅ **通过**，无 splitVendorChunk 警告（vite.config.ts 已清理冗余插件）

| 产物 | size | gzip |
|---|---:|---:|
| `vue-vendor` | 105.93 KB | **41.65 KB** |
| `utils-vendor` | 34.86 KB | **14.03 KB** |
| `index` (entry) | 48.25 KB | **17.69 KB** |
| `HomeView` | 7.74 KB | 3.40 KB |
| `index.css` | 50.18 KB | 9.26 KB |
| `video-vendor`（仅 /play 路由懒加载） | 683.26 KB | 204.72 KB |

#### 性能预算

| 指标 | 预算 | 实际（gzip） | 结果 |
|---|---|---|---|
| 首屏 JS（vue + utils + index + HomeView） | < 200 KB | **76.77 KB** | ✅ |
| 首屏 CSS | < 30 KB | 9.26 KB + 1.48 KB ≈ **10.74 KB** | ✅ |
| `video-vendor` 懒加载 | 仅 /play | 204.72 KB gzip | ✅ |

#### 资源警告（仍存在但不阻塞）

⚠️ `dist/assets/managebg-B29u3SDN.png` **3.47 MB**、`dist/assets/play-Btb5ayNF.png` **2.20 MB**
- 来源于旧站直接拷贝，非首屏（登录页 / 播放占位图）
- 建议后续转 webp 或 CDN，**不阻塞上线**

⚠️ `video-vendor` chunk 683 KB（>500 KB 阈值）
- 为视频播放器必需依赖，已通过 manualChunks 隔离 + 路由懒加载，仅在 /play 加载
- 不影响首屏，不阻塞

### 3.3 vite.config.ts 清理对比

| 项 | 上一轮 | 本轮 | 结果 |
|---|---|---|---|
| `splitVendorChunkPlugin()` 插件残留 | ❌ 与 manualChunks 冲突 | ✅ 已移除 | 修复 W3 |
| `manualChunks` 形式 | 对象形式 | 对象形式 | OK |
| `esbuild.drop` console/debugger | ✅ | ✅ | OK |
| dev proxy `/api → 127.0.0.1:3601` | ✅ | ✅ | OK |

---

## 4. 反模式静态扫描（grep 0 命中目标）

| 反模式 | 目标范围 | 命中 | 结果 |
|---|---|---:|---|
| `location.href` 整页跳转 | `src/views`、`src/components` | 0 | ✅ |
| `axios.get/post/put/...` 直调 | `src/`（仅 http.ts 内 axios.create） | 0 | ✅ |
| `ElMessage` / `element-plus` import | `src/` | 0 | ✅ |
| `console.log/warn/error` 业务代码 | `src/`（仅 `utils/logger.ts` 封装内允许） | 0 业务 + 4 logger 封装 | ✅ |
| `@ts-ignore` / `@ts-expect-error` | `src/` 业务代码 | 0 业务 + 2 自动生成 dts | ✅ |
| `: any` / `as any` 显式 | `src/` 业务代码 | 0（仅 `http.ts:89` 注释提及） | ✅ |
| `TODO` / `FIXME` / `XXX` | `src/` | 0 | ✅ |

**结论**：反模式 0 残留，相比上一轮无回归。

`@ts-nocheck` 仅出现在 `types/auto-imports.d.ts` 与 `types/components.d.ts`（unplugin 自动生成，无需修改）。

---

## 5. 路由表对照

`routes.public.ts` + `routes.manage.ts` + `index.ts` 全部路由：

| 路径 | 旧站对应 | client-v2 实现 | 守卫 | Query 大小写 |
|---|---|---|---|---|
| `/` → `/index` | `/` → `/index` | ✅ redirect | — | — |
| `/index` | `Home.vue` | ✅ HomeView | — | — |
| `/filmDetail` | `FilmDetails.vue` | ✅ FilmDetailView | — | `link` |
| `/play` | `Play.vue` | ✅ PlayView (懒加载) | — | `id/source/episode/currentTime?` |
| `/search` | `SearchFilm.vue` | ✅ SearchView | — | `search/current?` |
| `/filmClassify` | `FilmClassify.vue` | ✅ ClassifyView | — | `Pid` (大写) |
| `/filmClassifySearch` | `FilmClassifySearch.vue` | ✅ ClassifySearchView | — | `Pid/Category/Plot/Area/Language/Year/Sort/current` |
| `/history` | （旧站无独立页） | ✅ **HistoryView（本轮新实现）** | — | — |
| `/login` | `Login.vue` | ✅ LoginView | — | `redirect?` |
| `/manage` → `/manage/index` | `/manage` → `/manage/index` | ✅ redirect | requiresAuth | — |
| `/manage/index` | `Index.vue` | ✅ DashboardView | requiresAuth | — |
| `/manage/collect/index` | `CollectManage.vue` | ✅ CollectListView | requiresAuth | — |
| `/manage/cron/index` | `CronManage.vue` | ✅ CronListView | requiresAuth | — |
| `/manage/film` | `Film.vue` | ✅ FilmListView | requiresAuth | — |
| `/manage/film/class` | `FilmClass.vue` | ✅ FilmClassView | requiresAuth | — |
| `/manage/film/add` | `FilmAdd.vue` | ✅ FilmAddView | requiresAuth | — |
| `/manage/film/detail` | `Temp.vue`（旧站亦占位） | ✅ FilmDetailView (placeholder) | requiresAuth | — |
| `/manage/file/upload` | `FileUpload.vue` | ✅ FileUploadView | requiresAuth | — |
| `/manage/file/gallery` | `Temp.vue` | ✅ FileGalleryView | requiresAuth | — |
| `/manage/system/webSite` | `SiteConfig.vue` | ✅ SiteConfigView | requiresAuth | webSite 大小写一致 |
| `*` | `Error404.vue` | ✅ NotFoundView | — | — |

**守卫验证**（`router/guards.ts`）：
- ✅ `requiresAuth` 缺 token → `next({ path: '/login', query: { redirect } })`
- ✅ TV 模式禁止 `/manage/*` → `next({ path: '/index' })`
- ✅ `afterEach` 设置 `document.title = ${title} - ${siteName}`

**结论**：路由表 19/19 与旧站完全对齐，query 大小写严格遵守后端硬约束。

---

## 6. API 契约对照（与上一轮 9 项 P0 比对）

### 6.1 用户端公开接口（无需 token）

| 接口 | 路径 | 实现 | 字段对齐 | 上轮状态 → 本轮 |
|---|---|---|---|---|
| 首页聚合 | `GET /index` | `filmApi.getIndex()` | ✅ | OK → OK |
| 顶级分类 | `GET /navCategory` | `filmApi.getNavCategory()` | ✅ | OK → OK |
| 站点基础 | `GET /config/basic` | `filmApi.getSiteBasic()` | ✅ | ⚠️ → ✅ |
| 影片详情 | `GET /filmDetail?id=` | `filmApi.getFilmDetail(id)` | ✅ | OK → OK |
| 播放信息 | `GET /filmPlayInfo?id=&playFrom=&episode=` | `filmApi.getPlayInfo(...)` | ✅ | OK → OK |
| 关键字搜索 | `GET /searchFilm?keyword=&current=` | `filmApi.searchFilm(...)` | ✅ | OK → OK |
| 分类首页 | `GET /filmClassify?Pid=` | `filmApi.getClassify(Pid)` | ✅（大写 P） | OK → OK |
| 分类筛选 | `GET /filmClassifySearch?Pid=&Category=...` | `filmApi.searchClassify(...)` | ✅（7 字段全大写） | OK → OK |

✅ **8/8 通过**

### 6.2 鉴权接口

| 接口 | 路径 | 字段 | 后端期望 | 结果 |
|---|---|---|---|---|
| 登录 | `POST /login` | `{userName, password}` | `system.User{UserName,Password}` (Go json tag) | ✅ |
| 退出 | `GET /logout` | — | 取 token | ✅ |
| 修改密码 | `POST /changePassword` | **`{password, newPassword}`** ✅ | `params["password"]` / `params["newPassword"]` | ✅ **修复**（上轮 BUG #1） |
| 用户信息 | `GET /manage/user/info` | — | — | ✅ |

文件：`src/types/user.ts:23-26` 已改为 `{ password, newPassword }`，`ManageHeader.vue:57-60` 提交时也是这两个字段，与后端 `controller/UserController.go:63` 严格匹配。

✅ **4/4 通过**

### 6.3 后台管理 API（与后端 controller 字段交叉对照）

#### 站点配置

| 接口 | 字段对齐 | 状态 |
|---|---|---|
| `GET /manage/index`（仪表盘） | nil 兜底（后端 placeholder） | ✅ |
| `GET /manage/config/basic` | SiteBasic 七字段对齐 BasicConfig | ✅ |
| `POST /manage/config/basic/update` | 同上 + Domain/SiteName 必填校验 | ✅ |

#### 采集源（上轮 BUG #2 / #3 / #4 / #5）

| 字段 | 后端 (`system.FilmSource`) | client-v2 (`CollectSource`) | 上轮 → 本轮 |
|---|---|---|---|
| `id` | `string` | `string` ✅ | ❌ → ✅ |
| `name` | `string` | `string` ✅ | OK |
| `uri` | `string` | `uri` ✅（不再是 `url`） | ❌ → ✅ |
| `resultModel` | `int (0/1)` | `0 \| 1` ✅ | ❌ → ✅ |
| `grade` | `int (0/1)` | `0 \| 1` ✅ | ❌ 缺失 → ✅ |
| `syncPictures` | `bool` | `bool` ✅ | OK |
| `collectType` | `int (0-4)` | `0\|1\|2\|3\|4` ✅ | ❌ 缺失 → ✅ |
| `state` | `bool` | `bool` ✅ | OK |
| `interval` | `int` | `number` ✅ | ❌ 缺失 → ✅ |

`CollectListView.vue` 表单：name / uri / resultModel / grade / collectType / interval / syncPictures / state 全部录入字段就位（行 14-30 `form` 与列名完全对齐）。

| 接口 | 上轮 → 本轮 |
|---|---|
| `GET /manage/collect/list` | OK → OK |
| `GET /manage/collect/options` | OK → OK |
| `GET /manage/collect/find?id=` | ❌ id 类型 → ✅ string |
| `GET /manage/collect/del?id=` | ❌ → ✅ string |
| `POST /manage/collect/add` | ❌ → ✅ 完整 FilmSource |
| `POST /manage/collect/update` | ❌ → ✅ |
| `POST /manage/collect/change` | ❌ → ✅（CollectListView.vue:87 发送 `{...row, state: !row.state}`，包含 syncPictures 字段） |
| `POST /manage/collect/test` | ❌ → ✅ 完整 FilmSource |
| `POST /manage/spider/start` | ❌ → ✅（`{id, ids: [], time: 24, batch: false}` 严格 CollectParams） |
| `GET /manage/spider/class/cover` | OK → OK |

#### 定时任务（上轮 BUG #6）

| 字段 | 后端 (`FilmCollectTask` / `FilmCronVo`) | client-v2 (`CronTask`) | 上轮 → 本轮 |
|---|---|---|---|
| `id` | `string` | `string` ✅ | ❌ number → ✅ |
| `ids` | `[]string` | `string[]` ✅ | ❌ 缺失 → ✅ |
| `time` | `int` (小时) | `number` ✅ | ❌ 缺失 → ✅ |
| `spec` | `string` | `string` ✅ | ❌ `cron` → ✅ |
| `model` | `int (0/1)` | `0\|1` ✅ | ❌ `jobType:string` → ✅ |
| `state` | `bool` | `bool` ✅ | OK |
| `remark` | `string` | `string?` ✅ | OK |

`CronListView.vue` 表单（行 21-29）含 ids 多选 / time 数值 / spec 输入 / model 0/1 单选，与 `validTaskAddVo` 校验项一一匹配（`Time !=0`、`spec` 校验、`Model==1` 时 ids 必填）。

#### 影片管理（上轮 BUG #7）

| 维度 | 上轮 → 本轮 |
|---|---|
| 请求字段 | ❌ `keyword` → ✅ `name`（`ManageFilmSearchParams.name`） |
| 响应分页 | ❌ `resp.total/resp.size` → ✅ `resp.params.paging.total / pageSize` |
| 列表 | `resp.list` ✅ |

`FilmListView.vue:39-43` 解析逻辑与后端 `FilmController.go:82-86` 严格匹配。

#### 文件库（上轮 BUG #8 / #9）

| 接口 | 上轮 → 本轮 |
|---|---|
| `GET /manage/file/list` | ❌ resp.total → ✅ resp.page.total |
| `GET /manage/file/del?id=` | ✅ string|number ParseUint |
| `POST /manage/file/upload` | ❌ FileItem → ✅ string URL |

`FileGalleryView.vue:23-24` `total = resp.page?.total ?? 0`、`pageSize = resp.page?.pageSize ?? 39` 完全对齐 `FileController.go:108`。

`FileUploadView.vue:24` `entry.resultUrl = await manageApi.file.upload(fd, ...)` 直接接收 string URL，与 `SingleUpload` 返回的 `link string` 一致。

`FilmAddView.vue:52` `form.picture = res`（直接 string）也已修正。

#### 影片新增

| 字段 | 后端 (`FilmDetailVo`) | client-v2 (`FilmAddPayload`) | 状态 |
|---|---|---|---|
| name / pid / cid / area / year / director / actor / picture / classTag / content | ✅ 全部就位 | ✅ | ✅ |
| enName / subTitle / initial / state / playFrom / downFrom / playLink / downloadLink / list | ✅ 类型已声明 | ⚠️ `FilmAddView.vue:19-31` form 仅录入了 11 个核心字段，其余按零值发送 | OK（后端容忍） |

✅ **管理端 27/27 通过**（上一轮 5 个 ❌ + 4 个 ⚠️ 全部修复）

---

## 7. HistoryView 专项

### 7.1 实现回顾

- 文件：`src/views/public/HistoryView.vue`（172 行）+ `src/stores/history.ts`（169 行）
- store 结构：cookie + localStorage 双写，map 形式存储，按 timeStamp 倒序输出 list
- 列表渲染：`grid-cols-[repeat(auto-fill,minmax(180px,1fr))]`，海报 + 集数 tag + 进度条 + 移除 + 清空
- TV 模式：`data-focusable="true"` + focus-visible 描边
- 跳转：`<RouterLink :to="record.link">`，link 为 `/play?id=...&source=...&episode=...&currentTime=...`

### 7.2 用例

| 用例 | 结果 | 备注 |
|---|---|---|
| 空态渲染（list.length === 0） | ✅ | `BaseEmpty` "还没有观看记录" |
| 计数显示 | ✅ | `共 N 条记录`（行 58） |
| 海报兜底（无 picture） | ✅ | `BaseImage :src="record.picture \|\| ''"` ratio=2/3 |
| 集数 tag 渲染 | ✅ | `v-if="record.episode"` brand variant |
| 移除单条（`historyStore.remove(id)`） | ✅ | 行 45-49 e.preventDefault + e.stopPropagation 防止冒泡到 RouterLink |
| 清空全部（confirm + `historyStore.clear()`） | ✅ | 行 38-43 |
| 跳转 link 直接送 router | ✅ | `:to="record.link"` 直接字符串 path?query |
| TV 焦点态 | ✅ | gf-history-card focus-visible 内部 div 加 focus ring（149-157） |
| **进度显示 `formatProgress(record.currentTime)`** | **❌ 见 8.1** | **永远 undefined → 不渲染进度** |

### 7.3 与 PublicHeader 历史浮层一致性

PublicHeader.vue:103-112 `historyTop` 取 list 前 8 条，使用 `it.source ?? ''`，跳转走 `goHistoryItem`：若 `item.link` 存在则直接 `router.push(link)`，否则用 `id/source/episode` 拼装。

✅ 浮层在多数情况下能正常跳转（依赖 link 已含完整 query），但若 store 没存 source，则浮层 fallback 路径会拿到空 source。当前所有写入路径都通过 PlayView 的 `useFilmHistory.collect()` 走 store.record，**source 字段被 record 内部丢弃**（见 BUG 8.1）。

---

## 8. 缺陷列表

### 8.1 [P0] HistoryView 进度 / source / episodeIndex 字段被 store.record() 丢弃

**严重度**：P0（功能缺陷，不阻塞核心跳转，但本轮新增功能未达预期）
**位置**：
- `src/stores/history.ts` 行 132-147 `record()` 函数
- `src/views/public/HistoryView.vue` 行 118-122 `formatProgress(record.currentTime)`
- `src/composables/useFilmHistory.ts` 行 50-58 `flush() → store.record(snap)`

**现象**：
1. PlayView 通过 `useFilmHistory({ collect: () => ({...source, episodeIndex, currentTime, picture, ...}) })` 收集快照，包含 `source / episodeIndex / currentTime / picture` 等字段（PlayView.vue:140-149 已正确传入）。
2. **但 `useHistoryStore.record()` 内部仅取 `{ id, name, link, episode, picture, timeStamp }`（history.ts:136-143），把 `source / episodeIndex / currentTime` 静默丢弃。**
3. HistoryView 模板第 118-122 行 `formatProgress(record.currentTime)` → 因 `record.currentTime` 永远 undefined，进度小标永不渲染。
4. PublicHeader 历史浮层 `historyTop` 仍能工作（因 `record.link` 已是含 currentTime 的完整 query 字符串），但浮层 fallback 跳转分支 `goHistoryItem` 行 127-130 使用 `item.source` 时会拿到空字符串。

**根因**：`HistoryRecord` 接口（history.ts:20-37）虽然声明了 `source / episodeIndex / currentTime` 是可选字段，但 `record()` 函数体内的 `next` 对象 literal 没有把这些字段从入参 `item` 拷贝过来。

**复现步骤**：
1. 启动后端 + dev server，进入任一影片 `/filmDetail?link=...`
2. 点击播放，观看 30 秒，关闭播放页（触发 onBeforeUnmount → flush → record）
3. 进入 `/history`
4. **预期**：海报右下角显示 "0:30" 进度
5. **实际**：进度小标完全不渲染（`formatProgress(undefined)` 返回空串）

**验证**（控制台）：
```
JSON.parse(document.cookie.split('filmHistory=')[1])
// → { "<id>": { id, name, link, episode, picture, timeStamp } }
//    缺少 source / episodeIndex / currentTime
```

**修复建议**（一处改动）：

`src/stores/history.ts` 行 136-143 `record()` 函数中的 `next` 对象增加：
```ts
const next: HistoryRecord = {
  id: String(item.id),
  name: item.name,
  link: item.link,
  episode: item.episode,
  picture: item.picture,
  source: item.source,             // 新增
  episodeIndex: item.episodeIndex, // 新增
  currentTime: item.currentTime,   // 新增
  timeStamp: item.timeStamp ?? Date.now()
}
```

由于 `link` 字段已编码 currentTime，跳转能力本身不受影响；本修复主要恢复 HistoryView 进度展示 + 浮层 source fallback。

---

### 8.2 [P1] HistoryView 使用原生 `confirm()` 阻塞对话框，TV 模式不可用

**严重度**：P1
**位置**：`src/views/public/HistoryView.vue:40` 与 `FileGalleryView.vue:31` / `CollectListView.vue:92` / `CronListView.vue:85`

**现象**：清空 / 删除按钮使用浏览器原生 `confirm("...")`：
1. TV 浏览器（Tizen / WebOS / Capacitor WebView）部分实现不弹原生对话框，导致点击清空无任何反馈
2. 焦点系统（useSpatialNavigation）不能管理原生 confirm 弹窗内的"确定/取消"按钮
3. UI 风格与项目自有 `BaseDialog` 不统一

**修复建议**：换成自有 `BaseDialog` 组件 + 二段式确认。HistoryView 已有 `BaseDialog` 引用（其他文件用过），只需新增一个 confirmOpen 状态。

---

### 8.3 [P1] ManageHeader.vue 隐式依赖 unplugin-auto-import

**严重度**：P1（构建仍能过，但代码风格不一致）
**位置**：`src/components/layout/ManageHeader.vue` 行 2 与 行 69

**现象**：第 2 行 `import { reactive, ref } from 'vue'` 没有导入 `computed`，但第 69 行 `const avatar = computed(...)` 使用了 `computed`。
- 构建能过是因为 vite.config.ts 启用了 `unplugin-auto-import`（imports: ['vue', 'vue-router', 'pinia', '@vueuse/core']），自动注入 computed
- 但项目内**其他所有 Vue 文件都显式 import**（grep 验证），唯独此处隐式依赖，是风格不一致问题

**修复建议**：第 2 行改为 `import { computed, reactive, ref } from 'vue'`，避免维护者疑惑。

或者去除 vite.config.ts 中的 AutoImport 配置，强制全部显式 import；但需要清理 auto-imports.d.ts。

---

### 8.4 [P2] FilmAddView 表单字段不全（仍是 W1 警告）

**严重度**：P2
**位置**：`src/views/manage/film/FilmAddView.vue:19-31` form 对象

**现象**：FilmAddPayload 类型已声明 enName / subTitle / initial / state / playFrom / downFrom / playLink / downloadLink / list / remarks 等字段，但 form 仅录入 11 个核心字段。提交时其他字段按零值发送，对于"补录影片"场景一般够用，但**无法手工录入播放源**（list 字段为空 → 影片无法播放）。

**修复建议**：扩展 form，至少补录 `playFrom` + `playLink`（旧站 FilmAdd.vue 实现了播放源动态添加）；或在 UI 上明确告知"此页仅录元数据，播放源需通过采集源同步"。

---

### 8.5 [P2] FilmAddPayload 缺响应类型校验

**严重度**：P2
**位置**：`src/api/manage/film.ts:35-36` `add(data: FilmAddPayload): Promise<void>`

**现象**：后端 `FilmAdd` 返回 `SuccessOnlyMsg`（仅 msg，无 data），拦截器剥包装后返回 undefined / null。当前类型 `Promise<void>` OK，但若后续后端返回 ID 等字段则需调整。属于前瞻性风险点，不阻塞。

---

### 8.6 [P2] 资源体积优化（W4）

旧站 PNG 直接拷贝：
- `dist/assets/managebg-B29u3SDN.png` 3.47 MB
- `dist/assets/play-Btb5ayNF.png` 2.20 MB

非首屏，**不阻塞**，但建议后续转 webp（可省 70%+）或迁移 CDN。

---

### 8.7 [P2] DashboardView 数据空（W6）

后端 `/manage/index` 当前只返回 placeholder msg，前端 DashboardStat 有兜底。属于 PRD Q3 已知问题，**不阻塞**。

---

### 8.8 [P2] SiteBasic 字段不全（W2）

`SiteBasic` 已包含 siteName / logo / keyword / description / filing / domain / record? / copyright? / security?，但后端 `BasicConfig` 实际字段需联调时校对。当前可选字段标注为 `?`，**不阻塞**。

---

## 9. 与上一轮 QA 报告的差异

### 9.1 已修复（9 项 P0 + 1 项 W）

| BUG | 上一轮 | 本轮 | 验证位置 |
|---|---|---|---|
| #1 changePassword 字段错 | ❌ `{oldPwd, newPwd}` | ✅ `{password, newPassword}` | `types/user.ts:23-26` + `ManageHeader.vue:57-60` |
| #2 CollectSource 字段错 | ❌ `url/type/缺 grade/collectType/interval` | ✅ 完整 FilmSource 9 字段 | `types/manage.ts:24-44` + `CollectListView.vue:20-30` |
| #3 collect/change payload 错 | ❌ `{id:number, status}` | ✅ 完整 row spread + state 翻转 | `CollectListView.vue:87` |
| #4 collect/test payload 错 | ❌ `{url}` | ✅ 完整 FilmSource | `api/manage/collect.ts:37-38` |
| #5 spider/start payload 错 | ❌ `{sourceId, mode}` | ✅ `{id, ids:[], time:24, batch:false}` | `CollectListView.vue:97-104` |
| #6 CronTask 字段错 | ❌ `cron/jobType` 等 | ✅ `id:string, ids, time, spec, model:0\|1` | `types/manage.ts:65-80` + `CronListView.vue:21-29` |
| #7 film/search/list 字段错 | ❌ `keyword` + 顶层 total | ✅ `name` + `params.paging.total` | `FilmListView.vue:20-43` |
| #8 file/list 分页错 | ❌ resp.total | ✅ `resp.page.total` | `FileGalleryView.vue:23-24` |
| #9 file/upload 响应错 | ❌ FileItem 类型 | ✅ string URL | `api/manage/file.ts:22-30` + `FileUploadView.vue:24` + `FilmAddView.vue:52` |
| W3 splitVendorChunk 警告 | ⚠️ 与 manualChunks 冲突 | ✅ 已移除 | `vite.config.ts` |

### 9.2 遗留（不阻塞）

| 项 | 说明 |
|---|---|
| W1 FilmAddPayload 部分字段表单未录入 | 见 8.4 |
| W2 SiteBasic 字段联调验证 | 见 8.8 |
| W4 png 资源体积 | 见 8.6 |
| W6 DashboardView 真实数据 | 见 8.7 |
| F1-F8 其他 follow-up（旧站 history record、video.js TV D-pad 等） | 见上一轮第 12 节 |

### 9.3 新引入

| 项 | 说明 |
|---|---|
| **8.1 HistoryView record() 字段丢失（P0）** | follow-up 实现时 store.record 没把 source / episodeIndex / currentTime 写入持久化 |
| 8.2 confirm 阻塞对话框（P1） | HistoryView 新增的清空 / 移除使用了原生 confirm，TV 模式不可用 |
| 8.3 ManageHeader 隐式 auto-import（P1） | 风格不一致，先前未发现 |

---

## 10. TV / 无障碍核对

| 检查项 | 结果 | 文件 |
|---|---|---|
| `[data-focusable="true"]` 覆盖率 | 26 个文件 / 58 处 | 见反模式表 |
| `useSpatialNavigation` 实现完整 | ✅ 几何最近邻 + 焦点记忆 + sessionStorage | `composables/useSpatialNavigation.ts` |
| `installDpadBridge` Android keyCode 兼容 | ✅ 19/20/21/22/23/4 完整映射 | `utils/dpad.ts` |
| Esc/Backspace → router.back | ✅ history.length 兜底回首页 | `useSpatialNavigation.ts:277-288` |
| TV `[data-mode="tv"]` token 覆盖（字号 +25%、间距、container 1600） | ✅ | `assets/styles/theme.css:231` |
| HistoryView TV 焦点态 | ✅ gf-history-card focus-visible 描边 | `HistoryView.vue:153-157` |
| `aria-label` 覆盖 | 38 处 / 16 文件，主要在 base + layout | OK |
| HistoryView confirm 在 TV 不可用 | ⚠️ 见 8.2 | — |

---

## 11. 边界 / 风险点

### 11.1 已覆盖

1. ✅ http 拦截器对非标准包装（无 code 字段）做 fallback：`http.ts:110-112`
2. ✅ 401 拦截 → 清 token + redirect 到 /login
3. ✅ history.ts safeParse 兼容历史 array 形式
4. ✅ history.ts MAX_ITEMS=100 截断防 cookie 4KB 超限

### 11.2 风险

1. ⚠️ 后台管理端**没有契约测试**（PRD 12 节计划的 vitest + msw 仍未落地）。本轮虽然手工核对了 27 个接口字段，未来后端调整字段时 CI 不能自动捕获。
2. ⚠️ 所有 BUG 都靠人肉静态阅读发现（包括 8.1 这种"类型声明对，但运行时字段被丢"的问题，TS 类型检查无法识别）。建议补 contract 测试用例：mock 后端响应 → 调用 store.record(完整快照) → 断言 cookie 内字段齐全。

---

## 12. curl 测试脚本

路径：`D:\Git\GoFilm\doc\bmad\qa-tests\test-scripts.sh`（继承上一轮，本轮未变更）

包含：
- A. 用户端公开接口正向冒烟（8 接口）
- B. 异常 / 边界（缺参 / 错参 / 空字符）
- C. 登录拿 token
- D. 后台管理接口冒烟（11 接口）
- E. BUG 复现 / 兼容格式回归脚本（上一轮 9 项 P0 已全部修复，本轮可作为回归套件直接复跑）

使用：
```bash
BASE=http://127.0.0.1:3601 USERNAME=admin PASSWORD=xxx bash doc/bmad/qa-tests/test-scripts.sh
```

---

## 13. 上线建议

### 13.1 用户端（公开接口 + HistoryView）

⚠️ **建议修复 8.1 后再发**

- 用户端 7 个核心页面（首页 / 详情 / 播放 / 搜索 / 分类 / 筛选 / 历史）API 路径与字段 100% 兼容旧站 ✅
- 性能预算超额完成（首屏 76.77 KB gzip vs 200 KB 预算） ✅
- TS 严格模式 + 反模式 0 残留 + 路由 query 大小写合规 ✅
- TV 模式焦点系统、D-pad 桥接、空间导航全部就位 ✅
- ❌ HistoryView 进度展示因 store.record 字段丢失而无法显示

**8.1 修复成本极低（一处改动 3 行代码）**，强烈建议在合并前修复。如果业务方不在意进度小标，也可作为已知问题先发，下版本修复。

### 13.2 后台管理端

✅ **可上线**

- 上一轮 9 项 P0 BUG **全部修复并验证**（字段命名、类型、响应解析、响应格式全部对齐后端 controller）
- 27 个管理 API 契约 100% 通过
- 留有 8.2 P1（confirm 弹窗 TV 不可用）+ 8.3 P1（隐式 auto-import）+ 4 个 P2（不阻塞）

### 13.3 综合结论

**核心结论：相比上一轮 9 项 P0，本轮仅遗留 1 项 P0（8.1）+ 2 项 P1（8.2 / 8.3）。**

| 决策 | 推荐 |
|---|---|
| 修复 8.1 后用户端 + 管理端一起上线 | ✅ **首选**（修复成本约 5 分钟） |
| 不修复 8.1 直接上线，下版本修复 | ⚠️ 可接受（仅影响进度显示，不影响跳转） |
| 推迟至 P1 也清完再上线 | ❌ 过度（8.2 / 8.3 不影响核心功能） |

---

## 14. 测试用时

约 60 分钟（含 build 40s + 大量代码静态阅读 + 后端 controller 字段交叉对照 + 与上一轮报告比对）。

测试时间：2026-05-09
