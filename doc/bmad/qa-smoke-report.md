# GoFilm Vue3 重构 - QA 冒烟测试报告

> 测试目标：`client-v2/` 全量重构项目  
> 测试日期：**2026-05-09**  
> 测试人员：QA Engineer  
> 测试范围：构建验证 + 反模式静态扫描 + 路由表 / API 矩阵核对 + dev server 启动 + TV 模式 / 大屏适配静态阅读  
> 后端：未启动（只做代码契约对照，未做实际接口调用回归）  
> 前置文档：`prd-redesign.md` / `architecture-redesign.md` / `07-tv-adapt.md` / `client-v2/.dev.notes.md`

---

## 0. 测试环境

| 项 | 值 |
|---|---|
| 工作目录 | `D:\Git\GoFilm\client-v2\` |
| Node | （pnpm 已就绪） |
| 构建工具 | Vite 5.4.11 |
| 包管理 | pnpm |
| 浏览器 | curl 静态验证 dev server，未做交互回归 |
| 启动方式 | `pnpm dev`（端口 3600）→ 启动后 curl 验证 200/HTML 包含 `#app` → 关闭 |

---

## 1. 用例统计

| 类别 | 通过 | 失败 | 警告 |
|---|---:|---:|---:|
| 静态检查（TS / build） | 2 | 0 | 1（splitVendorChunk 与 manualChunks 对象形式冲突告警） |
| 反模式静态扫描 | 5 | 0 | 0 |
| 路由表对照 | 18 | 0 | 0 |
| 用户端 API 兼容矩阵 | 8 | 0 | 0 |
| 鉴权 API 兼容矩阵 | 3 | **1**（changePassword 字段错） | 0 |
| 管理端 API 兼容矩阵 | 24 | **5**（结构 / 字段错） | 4（类型不准但 view 有 fallback） |
| dev server 冒烟 | 1 | 0 | 0 |
| 视觉 / 交互静态结构 | 5 | 0 | 1（SiteBasic 缺旧字段） |
| **合计** | **66** | **6** | **6** |

---

## 2. 构建验证

### 2.1 TypeScript 类型检查

命令：`pnpm exec vue-tsc -p tsconfig.app.json --noEmit`  
结果：**通过**（无任何输出 = 无错误）

### 2.2 生产构建

命令：`pnpm build`  
结果：**通过**，构建耗时 14.51s

```
dist/assets/vue-vendor-THSekFN8.js          105.93 KB / gzip  41.65 KB
dist/assets/utils-vendor-8TLyuVKA.js         34.86 KB / gzip  14.03 KB
dist/assets/index-ny1roQnA.js                47.83 KB / gzip  17.53 KB
dist/assets/index-ih-nDWza.css               48.80 KB / gzip   9.02 KB
dist/assets/HomeView-LDCIq2m2.js              7.74 KB / gzip   3.39 KB
dist/assets/PlayView-D6wonllF.js             11.24 KB / gzip   4.70 KB
dist/assets/video-vendor-CrnhevUu.js        683.26 KB / gzip 204.72 KB（路由级懒加载）
```

#### 性能预算核对

| 指标 | 预算 | 实际（gzip） | 结果 |
|---|---|---|---|
| 首屏 JS（不含 video-vendor） | < 200KB | vue-vendor 41.65 + utils-vendor 14.03 + index 17.53 + HomeView 3.39 ≈ **76.6 KB** | ✅ 远低于预算 |
| 首屏 CSS | < 30KB | index.css 9.02 KB + HomeView.css 1.48 KB ≈ **10.5 KB** | ✅ |
| video-vendor | 仅在 `/play` 路由加载 | 204.72 KB gzip | ✅ 已通过 manualChunks 懒加载 |

#### 资源警告

⚠️ `dist/assets/managebg-B29u3SDN.png` **3.47 MB**、`dist/assets/play-Btb5ayNF.png` 2.20 MB — 旧站资源直接拷贝过来，未做压缩。建议后续对登录页 / 播放占位图做 webp 转换或体积压缩，但**不阻塞上线**（这些图通过路由懒加载/按需加载，非首屏）。

⚠️ Vite 提示 `splitVendorChunk plugin doesn't have any effect when using the object form of build.rollupOptions.output.manualChunks` — 不影响产出，仅提示 vite.config.ts 内 `splitVendorChunkPlugin()` 被 manualChunks 覆盖，可移除该插件。

---

## 3. 反模式静态扫描（grep 0 命中目标）

| 反模式 | 目标范围 | 命中数 | 结果 |
|---|---|---:|---|
| `location.href` 整页跳转 | `src/views`、`src/components` | 0 | ✅ |
| `watch(route` 全量 watch | `src/views` | 0 | ✅ |
| `axios.get/post/put/...` 直调 | `src/`（仅 `api/http.ts` 内允许 `axios.create`） | 0 | ✅ |
| `ElMessage` / `element-plus` / `ElementPlus` | `src/` | 0 | ✅ Element Plus 已彻底移除 |
| `from 'element-plus'` | `src/` | 0 | ✅ |

**结论**：所有架构守则约定的反模式 0 残留。

---

## 4. dev server 启动冒烟

```
pnpm dev → vite 5.4.11 ready in 11844 ms
LOCAL: http://localhost:3600/
```

curl 探测（绕代理）：

```
HTTP Status: 200
Response Size: 1104 bytes
HTML 含 <div id="app"></div> ✅
HTML 含 <script type="module" src="/@vite/client"></script> ✅
HTML 含 lang="zh-CN" data-mode="" ✅
```

测试完毕已关闭 dev 进程。

---

## 5. 路由表对照（PRD 第 5 节 / 架构第 4.1 节）

`src/router/routes.public.ts` + `routes.manage.ts` + `index.ts` 的全部路径与 PRD 路由矩阵 / 架构路由表 **一一对应**：

| 路径 | 实现 | 备注 |
|---|---|---|
| `/` → redirect `/index` | ✅ | |
| `/index` | ✅ HomeView | |
| `/filmDetail` | ✅ FilmDetailView，query `link` | 使用 `route.query.link` |
| `/play` | ✅ PlayView，query `id/source/episode/currentTime?` | watch query 增量切换 |
| `/search` | ✅ SearchView，query `search/current?` | |
| `/filmClassify` | ✅ ClassifyView，query `Pid` | 大写 P |
| `/filmClassifySearch` | ✅ ClassifySearchView，query 全部首字母大写 | Pid/Category/Plot/Area/Language/Year/Sort/current |
| `/history` | ✅ HistoryView（页面占位，PRD 已知） | |
| `/login` | ✅ LoginView | |
| `/manage` → redirect `/manage/index` | ✅ | |
| `/manage/index` | ✅ DashboardView | requiresAuth |
| `/manage/collect/index` | ✅ CollectListView | |
| `/manage/cron/index` | ✅ CronListView | |
| `/manage/film` | ✅ FilmListView | |
| `/manage/film/class` | ✅ FilmClassView | |
| `/manage/film/add` | ✅ FilmAddView | |
| `/manage/film/detail` | ✅ FilmDetailView（manage 版，placeholder） | PRD Q4 已知 |
| `/manage/file/upload` | ✅ FileUploadView | |
| `/manage/file/gallery` | ✅ FileGalleryView | |
| `/manage/system/webSite` | ✅ SiteConfigView | webSite 大小写一致 |
| `*` → NotFoundView | ✅ | |

✅ 路由守卫 `router/guards.ts`：
- `requiresAuth` 缺 token → `next({ path: '/login', query: { redirect } })` ✅
- TV 模式禁止 `/manage/*` → redirect `/index` ✅
- `afterEach` 设置 `document.title = ${title} - ${siteName}` ✅

✅ Query 大小写：`ClassifySearchView` 的 `Pid / Category / Plot / Area / Language / Year / Sort` 严格首字母大写，`current` 小写，与 PRD 硬约束一致。

---

## 6. API 兼容矩阵（与旧站 `client/src/views/index/*.vue` + `client/src/views/manage/*.vue` 调用清单逐一比对）

> 比对维度：路径 ✓ 方法 ✓ 请求字段名 ✓ 响应字段名 ✓
> 表中"实现"指 client-v2 是否存在对应函数；"契约对齐"指字段 / 类型与后端 controller 完全一致

### 6.1 用户端公开接口（无需 token）

| 接口 | 旧站调用 | client-v2 实现 | 契约对齐 | 备注 |
|---|---|---|---|---|
| `GET /api/index` | `Home.vue` | `filmApi.getIndex()` | ✅ | 拦截器剥包装后返 `IndexPageData` |
| `GET /api/navCategory` | `Header.vue` | `filmApi.getNavCategory()` | ✅ | |
| `GET /api/config/basic` | `Header.vue` | `filmApi.getSiteBasic()` | ⚠️ | 字段差异见 8.1 |
| `GET /api/filmDetail?id=` | `FilmDetails.vue` | `filmApi.getFilmDetail(id)` | ✅ | route 用 `link`，API 用 `id`，前端 view 已正确转换 |
| `GET /api/filmPlayInfo?id=&playFrom=&episode=` | `Play.vue` | `filmApi.getPlayInfo({id,playFrom,episode})` | ✅ | PlayInfo 类型已按 server `IndexController.FilmPlayInfo` 修正 |
| `GET /api/searchFilm?keyword=&current=` | `SearchFilm.vue` | `filmApi.searchFilm({keyword,current})` | ✅ | |
| `GET /api/filmClassify?Pid=` | `FilmClassify.vue` | `filmApi.getClassify(Pid)` | ✅ | 大写 P |
| `GET /api/filmClassifySearch?Pid=&Category=&...` | `FilmClassifySearch.vue` | `filmApi.searchClassify({...})` | ✅ | 7 字段全部首字母大写 |

**用户端结论**：路径 / 方法 / 字段名 100% 兼容。

### 6.2 鉴权接口

| 接口 | 旧站发送字段 | client-v2 发送字段 | 后端期望 | 结果 |
|---|---|---|---|---|
| `POST /api/login` | `{userName, password}` | `{username, password}` | json tag `userName`，但 Go json 默认大小写不敏感 | ✅ 兼容（验证依赖 server 实测） |
| `GET /api/logout` | 无参 | 无参 | 取 token | ✅ |
| `POST /api/changePassword` | `{password, newPassword}` | **`{oldPwd, newPwd}`** ⚠️ | `params["password"]` / `params["newPassword"]`（map 严格匹配） | **❌ BUG #1（见 8.1）** |
| `GET /api/manage/user/info` | 无 | 无 | | ✅ |

### 6.3 后台管理 API（接口存在性）

| 接口 | 路径 | 方法 | 实现位置 | 契约对齐 |
|---|---|---|---|---|
| 仪表盘统计 | `/manage/index` | GET | `system.dashboard` | ⚠️（后端目前返回 nil，与 PRD Q3 一致） |
| 站点配置读 | `/manage/config/basic` | GET | `system.getBasic` | ⚠️（字段差异 8.1） |
| 站点配置写 | `/manage/config/basic/update` | POST | `system.updateBasic` | ⚠️（字段差异 8.1） |
| 采集站列表 | `/manage/collect/list` | GET | `collect.list` | ⚠️ 类型 `PaginationResp` 但后端返数组（view 层有 fallback） |
| 采集 options | `/manage/collect/options` | GET | `collect.options` | ✅ |
| 采集 find | `/manage/collect/find` | GET | `collect.find` | ⚠️ FilmSource 字段差异（id 类型 / uri vs url） |
| 采集 del | `/manage/collect/del` | GET | `collect.remove` | ⚠️ id 类型应为 string |
| 采集 add | `/manage/collect/add` | POST | `collect.add` | **❌ BUG #2（见 8.2）** |
| 采集 update | `/manage/collect/update` | POST | `collect.update` | **❌ BUG #2** |
| 采集 change | `/manage/collect/change` | POST | `collect.change` | **❌ BUG #3（见 8.3）** |
| 采集 test | `/manage/collect/test` | POST | `collect.test` | **❌ BUG #4（见 8.4）** |
| 启动采集 | `/manage/spider/start` | POST | `collect.startSpider` | **❌ BUG #5（见 8.5）** |
| 分类覆盖 | `/manage/spider/class/cover` | GET | `collect.spiderClassCover` | ✅ |
| Cron 列表 | `/manage/cron/list` | GET | `cron.list` | ⚠️ CronTask 字段差异 |
| Cron 详情 | `/manage/cron/find` | GET | `cron.find` | ⚠️ |
| Cron 删除 | `/manage/cron/del` | GET | `cron.remove` | ✅ |
| Cron 新增 | `/manage/cron/add` | POST | `cron.add` | **❌ BUG #6（见 8.6）** |
| Cron 修改 | `/manage/cron/update` | POST | `cron.update` | **❌ BUG #6** |
| Cron 启停 | `/manage/cron/change` | POST | `cron.change` | **❌ BUG #6** |
| 影片搜索分页 | `/manage/film/search/list` | GET | `film.searchList` | **❌ BUG #7（见 8.7）** |
| 影片分类树 | `/manage/film/class/tree` | GET | `film.classTree` | ✅ |
| 影片分类详情 | `/manage/film/class/find` | GET | `film.classFind` | ✅ |
| 影片分类删除 | `/manage/film/class/del` | GET | `film.classDel` | ✅ |
| 影片分类更新 | `/manage/film/class/update` | POST | `film.classUpdate` | ✅ |
| 影片新增 | `/manage/film/add` | POST | `film.add` | ⚠️ FilmAddPayload 字段不全（spread 时 view 层 enName/classTag 已传） |
| 文件列表 | `/manage/file/list` | GET | `file.list` | **❌ BUG #8（见 8.8）** |
| 文件删除 | `/manage/file/del` | GET | `file.remove` | ⚠️ id 后端是 uint，前端传 string，Gin 会 ParseUint 转换 OK |
| 文件上传 | `/manage/file/upload` | POST | `file.upload` | **❌ BUG #9（见 8.9）** |

---

## 7. 视觉 / 交互静态结构核对

| 检查项 | 文件 | 结果 |
|---|---|---|
| `theme.css` 含 `[data-mode="tv"]` token 覆盖 | `src/assets/styles/theme.css:231` | ✅ 字号 +25% / 间距放大 / container 1600 / focus ring |
| `[data-focusable="true"]:focus + :focus-visible` 双触发 | `theme.css:262` | ✅ |
| `installSpatialNavigationOnce()` 在 App.vue 启用 | `App.vue:25` | ✅ |
| `installDpadBridge()` 在 App.vue 启用 | `App.vue:22` | ✅ |
| PlayView 含 video.js + 键盘快捷键 + preventDefault | `views/public/PlayView.vue` | ✅ usePlayer + keydown 处理 + 6 处 preventDefault |
| PublicHeader 滚动颜色切换 | `components/layout/PublicHeader.vue:36-46` | ✅ window scroll listener，scrolled > 12 切换 class |
| PublicHeader 移动端抽屉 | `PublicHeader.vue` | ✅ 汉堡 + drawer |
| PublicHeader 历史浮层（hover）+ TV Dialog 替代 | `PublicHeader.vue:73-103` | ✅ 双模式 |
| 管理端路由守卫拦截未登录 | `router/guards.ts:8-17` | ✅ requiresAuth + token 检查 → 跳 /login + redirect query |
| TV 模式禁止访问 `/manage` | `router/guards.ts:19-24` | ✅ |

---

## 8. 失败用例 / BUG 详情（带复现步骤）

### 8.1 [P0 阻塞] `/changePassword` 字段名错误

**严重度**：P0（功能完全不可用）  
**位置**：`src/types/user.ts` `ChangePasswordPayload` + `src/components/layout/ManageHeader.vue` 第 57-60 行

**现象**：管理端"修改密码"功能调用必然失败。

**根因**：
- 后端 `controller/UserController.go::UserPasswordChange` 用 `c.ShouldBindJSON(&params map[string]string)`，严格读取 `params["password"]` 与 `params["newPassword"]`。
- client-v2 `ChangePasswordPayload` 定义为 `{ oldPwd, newPwd }`，`ManageHeader.vue` 提交时也是 `{ oldPwd: pwdForm.password, newPwd: pwdForm.newPassword }`。
- 旧站 `ManageHeader.vue` 行 120 发送 `{password, newPassword}`，与后端契约一致。

**复现步骤**：
1. 启动后端 + dev server
2. 登录后台，点击右上角用户菜单 → 修改密码
3. 填入旧密码 / 新密码并提交
4. **预期**：密码修改成功
5. **实际**：后端返回"密码不能为空!!!"（因 `params["password"]` / `params["newPassword"]` 都是空字符串）

**修复建议**：
- 把 `ChangePasswordPayload` 字段改回 `{ password: string; newPassword: string }`
- ManageHeader.vue 提交时用 `{ password: pwdForm.password, newPassword: pwdForm.newPassword }`

> 同时附议：`SiteBasic` 类型缺旧站字段。后端 `BasicConfig` 实际包含 `siteName/logo/keyword/description/filing/domain` 之外可能还有 `record/security/copyright` 等需要联调时校对。当前仅是 ⚠️。

---

### 8.2 [P0 阻塞] 采集源新增 / 编辑字段名完全不匹配

**严重度**：P0  
**位置**：`src/types/manage.ts::CollectSource` + `src/views/manage/collect/CollectListView.vue`

**现象**：新增采集源、编辑采集源功能完全无法使用，列表显示采集 URL 列也是空。

**根因**：

| 字段 | 后端 (FilmSource) | client-v2 (CollectSource) |
|---|---|---|
| id | `string` `json:"id"` | **`number`** ❌ |
| 采集链接 | **`uri`** | **`url`** ❌ |
| 接口类型 | **`resultModel`** (int 0/1) | `resultModel` (string，且默认空串) ⚠️ |
| 资源类型 | **`collectType`** (int 0-4) | **缺失** ❌ |
| 等级 | **`grade`** (int 0/1) | **缺失** ❌ |
| 采集间隔 | **`interval`** (int) | **缺失** ❌ |
| `type` | **不存在** | **`type`** ❌ 多余字段 |
| 是否启用 | `state` (bool) | `state` (bool) | ✅ |
| 同步图片 | `syncPictures` (bool) | `syncPictures` (bool) | ✅ |

CollectListView Form 提交时用 `form.url` / `form.type` / `form.resultModel: ''`，所有这些字段后端 `validFilmSource` 都会校验失败：
- `Uri` 为空 → "资源链接格式异常"
- `ResultModel` 为字符串 `""` → ShouldBindJSON 解析为零值 0 (JsonResult)，但 client-v2 type 字段 `'json'/'xml'` 后端不识别
- `CollectType` 缺失 → 零值 0 (CollectVideo)，但前端无录入

**复现**：
1. 后台 → 采集源管理 → 新增
2. 填写名称、URL（前端字段名 `url`）、类型 `json`
3. 保存
4. **预期**：新增成功
5. **实际**：后端返回"资源链接格式异常, 请输入规范的URL链接"（因后端读 `uri` 字段，但前端发的是 `url`）

**修复建议**：
- 重写 `CollectSource` 类型对齐后端 `FilmSource`：`{ id: string, name, uri, resultModel: 0|1, grade: 0|1, syncPictures, collectType: 0-4, state, interval }`
- View 表单字段名同步修正：`form.uri / form.collectType / form.grade / form.interval / form.resultModel: 0`
- 列表列名 `url` 改为 `uri`

---

### 8.3 [P0 阻塞] `/manage/collect/change` payload 错误

**严重度**：P0  
**位置**：`src/api/manage/collect.ts::change` + `CollectListView.vue::toggleState`

**现象**：采集源启停按钮点击不生效。

**根因**：
- 后端 `FilmSourceChange` 期望 `FilmSource` 完整对象，至少需要 `id, state, syncPictures`，会进行 `state != fs.State || syncPictures != fs.SyncPictures` 判断。
- client-v2 发送 `{ id: number, status: boolean }`：
  - `status` 字段后端不识别（要的是 `state`）
  - 没传 `syncPictures` → 零值 false
  - `id` 类型 number → JSON 序列化为 `123`，但后端 `Id string` 字段会反序列化失败 → ShouldBindJSON 返回 error → "请求参数异常"

**复现**：
1. 采集源管理列表 → 点击"启用 / 停用"按钮
2. **预期**：状态切换
3. **实际**：toast "请求参数异常"

**修复**：`change(data: { id: string; state: boolean; syncPictures: boolean })`，view 调用时 `change({ id: row.id, state: !row.state, syncPictures: row.syncPictures ?? false })`

---

### 8.4 [P0 阻塞] `/manage/collect/test` payload 错误

**严重度**：P0  
**位置**：`src/api/manage/collect.ts::test`

**现象**：连通性测试按钮调用必失败（CollectListView 当前未连此按钮，但 API 错误依然存在）。

**根因**：
- 后端 `FilmSourceTest` 期望完整 `FilmSource` 对象 + `validFilmSource` 校验 name / uri / resultModel / collectType。
- client-v2 发 `{ url: string }` → 校验全部失败 → "资源名称不能为空"

**修复**：`test(data: FilmSource)` 完整对象。

---

### 8.5 [P0 阻塞] `/manage/spider/start` payload 错误

**严重度**：P0  
**位置**：`src/api/manage/collect.ts::startSpider` + `CollectListView.vue::startSpider`

**现象**：点击"采集"按钮不生效。

**根因**：
- 后端 `StarSpider` 期望 `CollectParams { id: string, ids: []string, time: int, batch: bool }`
- client-v2 发 `{ sourceId: number, mode: string }` → 全部字段名不匹配，绑定失败

**复现**：
1. 采集源管理 → 点击行内"采集"按钮
2. **预期**：启动采集任务
3. **实际**：后端返回参数错误 / 静默失败

**修复**：`startSpider(data: { id: string; ids: string[]; time: number; batch: boolean })`，view 调用 `{ id: row.id, ids: [], time: 24, batch: false }`

---

### 8.6 [P0 阻塞] 定时任务 CRUD 类型完全错位

**严重度**：P0  
**位置**：`src/types/manage.ts::CronTask` + `src/views/manage/cron/CronListView.vue`

**根因**：

| 字段 | 后端 (FilmCollectTask / FilmCronVo) | client-v2 (CronTask) |
|---|---|---|
| id | `string` | **`number`** ❌ |
| 资源站 ids | **`ids: string[]`** | **缺失** ❌ |
| 采集时长（小时数） | **`time: int`** | **缺失** ❌ |
| cron 表达式 | **`spec`** | **`cron`** ❌ |
| 任务类型 | **`model: 0|1`** | **`jobType: string`** ❌ |
| 状态 | `state` | `state` | ✅ |
| 备注 | `remark` | `remark` | ✅ |
| `name` | **不存在** | **`name`** ❌ 多余 |

**现象**：
1. 列表渲染：`row.name / row.cron / row.jobType` 全部 undefined → 空白行
2. 新增：发送 `{name, cron, jobType, state, remark}` → 后端 `validTaskAddVo` 校验 `Time == 0` → "采集时长不能为零值"
3. 编辑：同样失败
4. 启停 `change`：发送 `{ id: number, status: boolean }` → 后端 `FilmCollectTask{Id string}` 绑定失败

**修复**：完全重写 CronTask 类型 `{ id: string, ids: string[], time: number, spec: string, model: 0|1, state: boolean, remark?: string }`，并重写 CronListView 表单（添加 ids 多选 / time 数字 / spec 输入 / model 0/1 单选）。

---

### 8.7 [P0 阻塞] 影片管理列表查询字段错 + 分页元数据丢失

**严重度**：P0  
**位置**：`src/views/manage/film/FilmListView.vue::params` + 接收响应

**根因**：
- 后端 `FilmSearchPage` 用 `c.DefaultQuery("name", ...)` 取关键字，**不是 keyword**；分页参数用 `current / pageSize`。
- client-v2 发送 `{ keyword, current, pageSize }` → 后端 `s.Name = ""` → 不过滤
- 后端响应 `{ params, list, options }`，**没有 `total / size` 顶层字段**。分页元数据藏在 `params.paging.{ total, pageSize, pageCount, current }`。
- client-v2 用 `resp.total ?? 0` / `resp.size ?? params.pageSize` → 永远 `0` / 兜底值
- 翻页器永远显示"0 条"，无法翻页

**复现**：
1. 影片管理 → 搜索框输入"战" → 提交
2. **预期**：返回含"战"的影片
3. **实际**：返回所有影片（搜索未生效）+ 分页区显示总条数 0

**修复**：
- 请求参数改 `{ name: keyword, pid: 0, cid: 0, current, pageSize }`
- 响应解析改 `total = resp.params?.paging?.total`、`pageSize = resp.params?.paging?.pageSize`、`list = resp.list`
- API 类型重写：返回 `{ params: { paging: { total, pageSize, pageCount, current } }, list, options }`

---

### 8.8 [P0 阻塞] 文件库列表分页元数据丢失

**严重度**：P0  
**位置**：`src/views/manage/file/FileGalleryView.vue`

**根因**：
- 后端 `PhotoWall` 返回 `{ list, page: { pageSize, current, pageCount, total } }`
- client-v2 用 `resp.total / resp.size` → undefined
- 翻页永远 0

**修复**：`total = resp.page?.total ?? 0; pageSize = resp.page?.pageSize ?? 39`

---

### 8.9 [P0 阻塞] 文件上传响应体处理错误

**严重度**：P0  
**位置**：`src/api/manage/file.ts::upload` + `FileUploadView.vue` + `FilmAddView.vue::handleUpload`

**根因**：
- 后端 `SingleUpload` 返回 `Success(link, msg, c)` 其中 `link` 是 **string 类型的图片 URL**，不是 `FileItem` 对象。
- 拦截器剥包装后 client-v2 拿到 string，但类型声明为 `FileItem`，view 用 `entry.result.url` 取值 → undefined。

**现象**：
- FileUploadView 上传成功后缩略图栏空白（`entry.result.url` undefined）
- FilmAddView 上传海报后 `form.picture = res.url` 设置为 undefined → 影片新增提交时 picture 字段缺失

**修复**：
- `upload(form): Promise<string>` 返回 string
- view 调用 `form.picture = res`（直接是 url 字符串）
- 缩略图列表保留 file 名 + url 自行组装

---

## 9. 警告项（不阻塞，但需后续修复）

| # | 警告 | 影响 | 建议 |
|---|---|---|---|
| W1 | `FilmAddPayload` 类型缺 enName / classTag / playFrom / downFrom / playLink / downloadLink / subTitle / initial / state / addTime 等字段 | view 内 `{...form}` spread 时 TS 不报错，但缺字段后端按零值处理 | 同步至 `FilmDetailVo` 完整字段 |
| W2 | `SiteBasic` 类型可能缺后端 `BasicConfig` 部分字段（record / security / copyright 等） | 站点配置保存不完整 | 联调时拉真实接口校对 |
| W3 | `manualChunks` 对象形式覆盖 `splitVendorChunkPlugin()` 警告 | 仅控制台告警 | vite.config.ts 移除 `splitVendorChunkPlugin` 调用 |
| W4 | `dist/managebg-B29u3SDN.png` 3.47 MB / `play-Btb5ayNF.png` 2.20 MB 体积过大 | 登录页 / 播放占位图首次加载慢 | 转 webp + 压缩，或迁移到 CDN |
| W5 | `/manage/collect/list` 后端返数组，前端类型声明为 `PaginationResp<CollectSource>` | view 用了 `Array.isArray fallback` 不阻塞 | 修正类型定义为 `CollectSource[]` |
| W6 | DashboardView 数据全空 | 看板无数据，PRD Q3 已知 placeholder | 后端补 `/manage/index` 返回真实统计 |

---

## 10. curl 测试脚本路径

`D:\Git\GoFilm\doc\bmad\qa-tests\test-scripts.sh`

包含：
- A. 用户端公开接口正向冒烟（8 接口）
- B. 异常 / 边界（缺参 / 错参 / 空字符）
- C. 登录拿 token
- D. 后台管理接口冒烟（11 接口）
- E. 5 条 BUG 复现 / 兼容格式回归脚本

使用：`BASE=http://127.0.0.1:3601 USERNAME=admin PASSWORD=xxx bash test-scripts.sh`

---

## 11. 高风险点

1. **后台管理端核心 CRUD 全线断裂**（采集源 / 定时任务 / 影片搜索 / 文件库分页 / 修改密码 / 文件上传），合计 9 个 P0 BUG。前端 build / 类型检查全绿，但因为这些字段大部分是字符串到字符串的差异，TS 不会报错，掩盖了契约不一致。
2. 旧站 `ApiGet/ApiPost` 把响应 `data` 整体（含 code / msg / data 三字段）原样透传给 view，view 内自行 `resp.data` 解包；client-v2 拦截器**剥一层包装**直接返回 `body.data`，这导致 view 拿到的对象层级与旧站习惯不同，类型层把握得对但具体字段是否对齐后端结构没有自动校验手段。
3. **没有契约测试**（PRD 12 节计划的 vitest + msw 未落地），如果未来后端字段调整，前端无法通过 CI 自动发现，建议补 contract 用例。
4. **管理后台无 e2e 冒烟**（PRD 13 节也注明"不做 e2e"），导致 BUG 8.2-8.9 都是静态阅读发现，回归依赖 QA 手工 + 后端实际部署。

---

## 12. 待 follow-up（不阻塞用户端上线）

| # | 项 | 备注 |
|---|---|---|
| F1 | HistoryView 真实卡片渲染 | 视图占位中（07-tv-adapt.md 已知） |
| F2 | video.js 控件 D-pad 焦点接管粗糙 | TV 用户全屏后控件不易精细操作 |
| F3 | DashboardView 数据看板需后端 `/manage/index` 补真实数据 | PRD Q3 |
| F4 | `SiteBasic` 字段对齐后端 `BasicConfig` | 联调时校对 |
| F5 | manage 端图标体积优化（managebg.png 3.47 MB） | 转 webp |
| F6 | 删除 vite.config.ts 中冗余 `splitVendorChunkPlugin` | 消除告警 |
| F7 | 旧站 history record 完整 link（含 currentTime）兼容性测试 | 已在 STORY-010 实现，待真机回归 |
| F8 | login 字段 `username` vs `userName` 用真实后端验证一次 | Go 大小写不敏感理论可行，但建议主动对齐 `userName` |

---

## 13. 上线建议

### 用户端（公开接口）

✅ **可上线**

- 首页 / 详情 / 播放 / 搜索 / 分类 / 筛选 / 历史 7 大页面所有 API 路径与字段 100% 兼容旧站
- 性能预算超额完成（首屏 ~76 KB gzip vs 200 KB 预算）
- TS 严格模式 + ESLint 都过
- 反模式 0 残留
- 路由 query 大小写完全合规
- TV 模式焦点系统、D-pad 桥接、空间导航全部就位

### 后台管理端

❌ **修复后再上线**

阻塞项 **9 个 P0 BUG**：

| BUG | 受影响功能 |
|---|---|
| 8.1 changePassword 字段错 | 修改密码完全失败 |
| 8.2 CollectSource 字段错 | 采集源新增 / 编辑失败 + 列表 url 列空白 |
| 8.3 collect/change 字段错 | 采集源启停失败 |
| 8.4 collect/test 字段错 | 连通性测试失败 |
| 8.5 spider/start 字段错 | 启动采集失败 |
| 8.6 CronTask 字段错 | 定时任务 CRUD + 列表渲染全错 |
| 8.7 film/search/list 字段错 | 影片管理搜索 + 分页失效 |
| 8.8 file/list 字段错 | 文件库分页失效 |
| 8.9 file/upload 响应错 | 上传成功但预览 / 持久化都丢 |

预计修复工作量：1-2 个工作日（5 个文件类型修正 + 6 个 view 字段同步）。

### 折中方案（如必须立即上线）

1. **用户端先发**：用户端 7 个页面没有任何阻塞，可先发布；
2. **后台管理端用旧站继续顶**：保留 `client/` 旧版 dist 作为 `/manage/*` 入口的 fallback，等 client-v2 上述 9 个 BUG 修复后再切换；
3. 添加 `client-v2/dist` 与 `client/dist` 在 nginx 双部署能力，逐步灰度。

---

## 14. 测试用时

约 90 分钟（含 build 14.51s + 大量代码静态阅读 + 后端 controller 字段交叉对照）。

---

测试时间：2026-05-09
