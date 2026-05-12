# 前后端集成契约审计报告

> 生成日期: 2026-05-12
> 分支: feature/redesign (commit 81d4609)
> 类型: **静态契约审计** (本机 Go / Docker 未安装, 远端 MySQL/Redis 不可达, 无法做运行时 e2e)

## 0. 审计前提说明

本机环境 `go`、`docker` 缺失, 远端 192.168.20.10:3307 (MySQL) / 6379 (Redis) 探测超时, 无法启动 `server/main.go` 与真实数据通路, 因此无法跑 Playwright 关掉 mock 后的运行时 e2e。

替代方案: 把 `client-v2/src/api/*.ts` 中实际发出的请求与 `server/router/router.go` + 各 Controller 逐条核对 URL / method / 入参 / 出参 / 鉴权 / 错误码 5 维, 列出契约偏差。

如果后续要做运行时 e2e, 需要先满足以下任意一种:
- 本机装 Go + 本地起 MySQL/Redis (docker compose 或单机)
- 把 vite.config.ts 的 `proxy.target` 指向已部署的真实 server (前提是访问可达)

---

## 1. 契约矩阵 (核心 33 条)

图例: ✅ 完全一致 | ⚠️ 可工作但有偏差 | ❌ 不一致需修

### 1.1 公共 API (无需登录)

| # | 前端 (api/film.ts) | 后端 (router.go) | URL | Method | 入参 | 出参 | 状态 |
|---|---|---|---|---|---|---|---|
| 1 | `getIndex()` | `Index` | `/index` | GET | — | `IndexPageData {banner, content[]}` | ✅ |
| 2 | `getNavCategory()` | `CategoriesInfo` | `/navCategory` | GET | — | `NavCategory[]` | ✅ |
| 3 | `getSiteBasic()` | `SiteBasicConfig` | `/config/basic` | GET | — | `BasicConfig` | ⚠️ 字段名 `describe` ↔ `description` |
| 4 | `getFilmDetail(id)` | `FilmDetail` | `/filmDetail` | GET | `id: string→Atoi` | `{detail, relate}` | ✅ |
| 5 | `getPlayInfo(...)` | `FilmPlayInfo` | `/filmPlayInfo` | GET | `id, playFrom, episode` | `{detail, current, currentPlayFrom, currentEpisode, relate}` | ✅ |
| 6 | `getClassify(Pid)` | `FilmClassify` | `/filmClassify` | GET | `Pid` (大写首字母) | `{title, content:{news,top,recent}}` | ✅ |
| 7 | `searchClassify(...)` | `FilmTagSearch` | `/filmClassifySearch` | GET | `Pid/Category/Plot/Area/Language/Year/Sort/current` (全部大写首字母) | `{title, list, page, search, params}` | ✅ |
| 8 | `searchFilm({keyword, current})` | `SearchFilm` | `/searchFilm` | GET | `keyword, current` | 成功: `{list, page}`；**无结果: `code=-1, msg=暂无相关影片信息`** | ⚠️ "无结果"用 -1 误判为失败 toast |

### 1.2 用户登录 / 资料

| # | 前端 (api/auth.ts) | 后端 | URL | Method | 入参 | 出参 | 状态 |
|---|---|---|---|---|---|---|---|
| 9 | `login({userName, password})` | `Login` | `/user/login` (兼容 `/login`) | POST | json {userName, password} | **body: `{code:0, data:null, msg:"登录成功!!!"}`**, token 在响应头 `new-token` | ⚠️ 前端 ts 标注 `Promise<UserInfo>` 但实际是 null, store 已经容错 (登录后再拉 `/user/info`) |
| 10 | `logout()` | `Logout` | `/user/logout` | GET | header `auth-token` | `{code:0, data:null}` | ✅ |
| 11 | `changePassword({password, newPassword})` | `UserPasswordChange` | `/user/changePassword` | POST | json {password, newPassword} | `{code:0}` | ✅ |
| 12 | `getUserInfo()` | `UserInfo` | `/user/info` | GET | header `auth-token` | `UserInfoVo {id, userName, email, gender, nickName, avatar, status, role}` | ✅ |

### 1.3 后台用户管理 (admin only)

| # | 前端 | 后端 | URL | Method | 入参 | 出参 | 状态 |
|---|---|---|---|---|---|---|---|
| 13 | `createUser(...)` | `ManageUserCreate` | `/manage/user/create` | POST | `{userName, password, email?, nickName?, role?}` | `UserInfoVo` | ✅ |
| 14 | `listUsers({current, pageSize})` | `ManageUserList` | `/manage/user/list` | GET | query `current, pageSize` | `{list: UserInfoVo[], page: {pageSize, current, pageCount, total}}` | ✅ |
| 15 | `manage.user.info()` | `UserInfo` | `/manage/user/info` | GET | header `auth-token` + admin | `UserInfoVo` | ✅ |

### 1.4 后台仪表盘 / 系统配置

| # | 前端 | 后端 | URL | Method | 入参 | 出参 | 状态 |
|---|---|---|---|---|---|---|---|
| 16 | `manage.system.dashboard()` | `ManageIndex` | `/manage/index` | GET | — | **`{code:0, data:null, msg:"后台管理中心"}`** | ❌ 前端类型 `DashboardStat` 但后端返回 null. 后端未实现统计 |
| 17 | `manage.system.getBasic()` | `SiteBasicConfig` | `/manage/config/basic` | GET | — | `BasicConfig` | ⚠️ 同 #3 字段名 |
| 18 | `manage.system.updateBasic(data)` | `UpdateSiteBasic` | `/manage/config/basic/update` | POST | json `BasicConfig` | `{code:0}` | ⚠️ 同 #3 字段名 |

### 1.5 后台采集源

| # | 前端 | 后端 | URL | Method | 入参 | 出参 | 状态 |
|---|---|---|---|---|---|---|---|
| 19 | `collect.list()` | `FilmSourceList` | `/manage/collect/list` | GET | — | `FilmSource[]` | ✅ |
| 20 | `collect.options()` | `GetNormalFilmSource` | `/manage/collect/options` | GET | — | `[{id, name}]` | ✅ |
| 21 | `collect.find(id)` | `FindFilmSource` | `/manage/collect/find` | GET | `id` (string) | `FilmSource` | ✅ |
| 22 | `collect.remove(id)` | `FilmSourceDel` | `/manage/collect/del` | GET | `id` (string) | `{code:0}` | ✅ |
| 23 | `collect.add(data)` | `FilmSourceAdd` | `/manage/collect/add` | POST | `FilmSource` | `{code:0}` | ✅ |
| 24 | `collect.update(data)` | `FilmSourceUpdate` | `/manage/collect/update` | POST | `FilmSource` | `{code:0}` | ✅ |
| 25 | `collect.change(data)` | `FilmSourceChange` | `/manage/collect/change` | POST | `FilmSource (id+state+syncPictures)` | `{code:0}` | ✅ |
| 26 | `collect.test(data)` | `FilmSourceTest` | `/manage/collect/test` | POST | `FilmSource` | `{code:0, msg}` | ⚠️ 前端类型 `{ok, msg}` 但后端只返回 msg, 没有 `ok` 字段 |
| 27 | `collect.startSpider(data)` | `StarSpider` | `/manage/spider/start` | POST | `CollectParams {id, ids, time, batch}` | `{code:0}` | ✅ |
| 28 | `collect.spiderClassCover()` | `CoverFilmClass` | `/manage/spider/class/cover` | GET | — | **`{code:0, data:null, msg:"重置成功"}`** | ❌ 函数名暗示拉列表, 实际是 "重置/覆盖" 动作, 不返回数据 |

### 1.6 后台 Cron / 影片 / 文件

| # | 前端 | 后端 | URL | Method | 入参 | 出参 | 状态 |
|---|---|---|---|---|---|---|---|
| 29 | `cron.list()` | `FilmCronTaskList` | `/manage/cron/list` | GET | — | `FilmCollectTask[]` | ✅ |
| 30 | `cron.add(data)` | `FilmCronAdd` | `/manage/cron/add` | POST | `FilmCronVo` | `{code:0}` | ✅ |
| 31 | `film.searchList(...)` | `FilmSearchPage` | `/manage/film/search/list` | GET | `name/pid/cid/plot/area/language/year/remarks/beginTime/endTime/current/pageSize` | `{params: {paging}, list, options}` | ✅ |
| 32 | `file.list(...)` | `PhotoWall` | `/manage/file/list` | GET | `current` | `{list, page}` | ✅ |
| 33 | `file.upload(form)` | `SingleUpload` | `/manage/file/upload` | POST | multipart `file` | `link: string` (data 直接是 URL 字符串) | ✅ |

---

## 2. 高优先级偏差 (P0 / P1)

### P0-1 ❌ `BasicConfig.describe` ↔ `SiteBasic.description` 字段名不一致

**影响**: 系统配置页 (`/manage/settings`) 读到的 `describe` 字段, 前端按 `description` 字段渲染, 永远显示空; 用户编辑 "网站描述" 后提交, 后端写入的也是空串。

**位置**:
- 后端 `server/model/system/Manage.go:15` → `Describe string \`json:"describe"\``
- 前端 `client-v2/src/types/manage.ts:6` → `description: string`
- 前端缺失: `state` (bool), `hint` (string)
- 前端多余: `filing`, `record`, `copyright`, `security`

**建议**:
1. 前端 `SiteBasic` 改用 `describe`, 增补 `state`/`hint`, 删掉无对应后端字段的 `filing/record/copyright/security`
2. 或: 后端 `BasicConfig` 增字段并保留 `describe` → `description` 序列化迁移

### P0-2 ❌ `/manage/index` 实际无统计数据

**影响**: 后台仪表盘 (`ManageDashboardView.vue`) 拿不到 filmCount/collectCount/cronCount/diskUsage, 真实环境会显示全 0 或骨架。

**位置**:
- 后端 `controller/ManageController.go:14` → `system.SuccessOnlyMsg("后台管理中心", c)` (data=nil)
- 前端 `api/manage/system.ts:5` → `Promise<DashboardStat>`

**建议**: 后端 `ManageIndex` 聚合 `len(IL.GetMovieList())` + 采集源数 + cron 数 + 磁盘使用, 返回 `DashboardStat` 结构; 或前端暂时直接调 `collect.list().length` + `cron.list().length` 拼仪表盘。

### P0-3 ❌ `/manage/spider/class/cover` 接口语义反了

**影响**: 前端 `manage.collect.spiderClassCover()` 期望返回封面列表, 调用后 `data===null`, 任何 `.map()` 都会崩。

**位置**:
- 后端 `SpiderController.go:82` `CoverFilmClass` 是 **重置覆盖动作**, 不返回数据
- 前端 `api/manage/collect.ts:46` 类型为 `ClassCoverItem[]`

**建议**: 二选一
- 若是动作: 前端把返回类型改 `void`, 调用点改 toast 后跳列表
- 若需列表: 后端新增 `GET /manage/spider/class/cover/list` 返回 `[]ClassCoverItem`, 现接口保留为 `POST` 动作

### P1-1 ⚠️ `/user/login` 返回 `data:null`, 前端类型标错

**影响**: 仅类型层不准, 运行时 store 已经做了 `try { fetchInfo() } catch` 兜底, 但下次接手的人看 `Promise<UserInfo>` 会被误导。

**建议**: `api/auth.ts:16` 把 `Promise<UserInfo>` 改为 `Promise<void>`, 显式注释 token 在 header / 用户信息要二次拉。

### P1-2 ⚠️ `/searchFilm` 无结果当失败处理

**影响**: 用户搜不到内容时, 全局 toast 弹出 "暂无相关影片信息" 红条, UX 不够柔和。

**位置**: `controller/IndexController.go:105` `if page.Total <= 0 { system.Failed(...) }`

**建议**: 改成 `system.Success(gin.H{"list": []FilmListItem{}, "page": page}, "暂无相关影片信息", c)`, 前端正常拿空数组, 由 SearchView 自己渲染"无结果"占位。

### P1-3 ⚠️ `/manage/collect/test` 返回结构与前端定义不一致

**影响**: 测试采集源成功时, 前端读 `res.ok` 永远 `undefined`, 失败时拦截器抛 BizError, 两种状态走不同分支但 UI 都拿不到完整字段。

**位置**:
- 后端 `MissionController.go: FilmSourceTest` → `system.SuccessOnlyMsg("测试成功!!!", c)` (data=nil)
- 前端 `api/manage/collect.ts:38` 类型 `{ok: boolean, msg: string}`

**建议**: 前端类型改 `Promise<void>` (或 `Promise<{msg: string}>`); 调用点按 try/catch 判定成功/失败。

### P1-4 ⚠️ 用户观看历史 / 收藏 — 后端接口齐, 前端 0 调用

**影响**: 后端 `/user/history`、`/user/favorite` 一套 CRUD 已交付 (commit 00a4d3a), 但 `client-v2/src/api/` 下没有对应函数, 也没有 UI 入口 (PublicHeader 的"观看历史"菜单项点了会 404 路由)。

**核对**:
- 后端: `POST/GET/DELETE /user/history`, `DELETE /user/history/clear`, `POST/DELETE/GET /user/favorite`, `GET /user/favorite/check`
- 前端 api 层: **无** (仅 `mock/handlers.ts` 有占位)

**建议**: 立即新增 `client-v2/src/api/history.ts` + `client-v2/src/api/favorite.ts` 并补 `HistoryView.vue` / `FavoritesView.vue` 与 `PlayView.vue` 内的进度上报埋点。

### P1-5 ⚠️ CORS 头未暴露 `new-token`, 未允许 `auth-token`

**影响**: 仅当前后端跨域部署 (前端 CDN + 后端独立域名) 才会暴露:
- 浏览器拒绝读取 `new-token` 头 → token 刷新失效, 一段时间后 401
- 浏览器预检拒绝带 `auth-token` 自定义头 → 所有需登录接口失败

**位置**: `server/plugin/middleware/Cors.go:20-22`
```go
c.Header("Access-Control-Allow-Headers", "Authorization, Content-Length, X-CSRF-Token, Token,session, Content-Type")
c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Content-Type")
```

**建议**:
```go
c.Header("Access-Control-Allow-Headers", "Authorization, Content-Length, X-CSRF-Token, Token, session, Content-Type, auth-token")
c.Header("Access-Control-Expose-Headers", "Content-Length, new-token")
```
当前开发用 Vite 同源反代未触发, 不影响 dev, 但发布到生产前必修。

---

## 3. 中低优先级 (P2 / P3)

| 编号 | 描述 | 位置 |
|---|---|---|
| P2-1 | 401 响应体 `code === SUCCESS (0)`, 仅靠 HTTP status 才能识别失败, 拦截器侥幸正确 | `middleware/HandleJwt.go:23` |
| P2-2 | 旧路由 `/login`/`/logout`/`/changePassword` 仍挂在 root, 与 `/user/*` 同时存在; 前端只用 `/user/*`, 旧的应在下个发布周期移除以收敛攻击面 | `router.go:28-30` |
| P2-3 | 默认密码 `admin/admin` (`User.go:75-80`) 首次部署后未强制改密 | `model/system/User.go:73` |
| P2-4 | MySQL DSN 与 Redis 密码硬编码到源码, 应迁移到环境变量 / 配置文件 | `config/DataConfig.go:92, 103` |
| P3-1 | `UserInfo` 类型保留兼容字段 `uid/username/nickname` (旧站小写), 后端从未返回, 可清理 | `client-v2/src/types/user.ts:13-22` |
| P3-2 | 错误码全 `-1`, 业务无法按错误码分类处理 (如"密码错误" vs "用户不存在"), 当前依赖 msg 文本匹配 | `model/system/Response.go:15` |

---

## 4. 跑过 mock 但真实接口未跑过的功能清单

(即 mock 替代后看着工作, 切到真实后端可能挂)

| 功能 | mock 状态 | 真实后端状态 | 风险点 |
|---|---|---|---|
| 登录 + 角色路由 | ✅ | ✅ (契约一致) | 登录后跳转链路 OK |
| 后台仪表盘统计 | ✅ (mock 编了) | ❌ 后端返 null | UI 永远 0 (P0-2) |
| 观看历史/收藏 | ✅ (mock handlers 有) | 后端 ✅ 但前端无 api/UI | 整个功能不可用 (P1-4) |
| 站点基本配置编辑 | ✅ | ⚠️ describe ↔ description 错位 | 描述字段编辑不生效 (P0-1) |
| 采集源测试 | ✅ | ⚠️ 返回类型不匹配 | 成功提示能弹但 res.ok 未 undefined (P1-3) |
| 爬虫分类封面 | ✅ (mock 拼了数组) | ❌ 实际是动作 | 调 spiderClassCover().map() 会崩 (P0-3) |
| 关键字搜索 | ✅ 空结果返回空数组 | ⚠️ 空结果当失败 toast 红条 | UX 退化 (P1-2) |

---

## 5. 上线建议

### 5.1 必须修 (P0, 阻塞发布)
- [ ] **P0-1** 统一 `describe` 字段名 (前端改 `description→describe` 最简单)
- [ ] **P0-2** 后端实现仪表盘聚合 或 前端拼装 (任选一)
- [ ] **P0-3** `/manage/spider/class/cover` 接口语义对齐

### 5.2 强烈建议 (P1, 上线前)
- [ ] **P1-4** 把观看历史 + 收藏接到前端, 否则 commit 00a4d3a 价值未兑现
- [ ] **P1-5** CORS 头补 `auth-token` + 暴露 `new-token` (跨域部署场景)
- [ ] **P1-1** 前端 `login()` 返回类型修为 `void`
- [ ] **P1-2** `/searchFilm` 无结果改 Success 空数组
- [ ] **P1-3** `collect.test` 类型对齐

### 5.3 可下版本 (P2/P3)
- 移除旧 `/login` 路由, 加强配置外置, 默认密码强制改密

### 5.4 上线前必跑的运行时验证 (本机环境不具备时下移到 CI 或部署环境)

1. **依赖通路**: `nc -zv 192.168.20.10 3307` + `redis-cli -h 192.168.20.10 ping`
2. **启动 server**: `cd server && go run main.go`, 看启动日志 "AutoMigrate"、"SpiderInit" 无 panic
3. **建账号**: 首部署 admin/admin 登录 → 改密 → 用 admin 调 `POST /manage/user/create` 建普通用户
4. **关 mock 切真后端**: `VITE_USE_MOCK=0` (注释掉 `.env.development` 第 4 行), `pnpm dev`
5. **跑现有 Playwright e2e**:
   - 普通用户登录 → `/index` → 看到首页数据
   - 普通用户硬闯 `/manage` → 守卫拦回 `/index` + toast "权限不足"
   - admin 登录 → 直跳 `/manage/index`
   - 详情 → 播放 → 进度上报应触发 (待 P1-4 接好后)
6. **手测 P0/P1 修复**:
   - 站点设置 → 改"描述" → 刷新看是否保留
   - 后台 → 仪表盘 → 看到非 0 统计
   - 搜索 "xxxxx不存在的词" → 看到空结果占位而非红 toast

---

## 6. 结论

**前后端协议覆盖率: 33 条接口中 24 条完全一致, 5 条带可工作的小偏差 (P1/P2), 3 条不一致 (P0)。**

**当前分支不能直接发布**, 修完 3 条 P0 + 接好观看历史/收藏前端 (P1-4) + CORS (P1-5) 后, 才能在跨域生产环境正常工作。

如本机能拉起 Go 环境 + 本地起 MySQL/Redis (推荐 docker compose), 我可以接下来跑一遍真实联调的 Playwright e2e 把这份审计变成运行时验证报告。
