# PM → UX / Architect 交接文档

> 来源：`doc/bmad/prd-redesign.md` v1.0  
> 受众：UX 设计师、技术架构师  
> 目的：在进入设计与方案阶段前，明确**做什么、不能改什么、风险在哪**

---

## 1. 前置文档
- 共享上下文：`doc/handover/00-context.md`
- PRD 主文档：`doc/bmad/prd-redesign.md`

请 UX/Architect 先通读以上两份再开工。

---

## 2. 关键页面清单与优先级

### 2.1 用户端（P0 = 必须 MVP，P1 = 二期可推迟）

| 优先级 | 页面 | 路由 | 说明 |
| --- | --- | --- | --- |
| P0 | 首页 | `/index` | Hero + 横向滚动行，门面 |
| P0 | 影片详情 | `/filmDetail?link=` | 转化关键页 |
| P0 | 播放页 | `/play?id=&source=&episode=` | 核心体验 |
| P0 | 搜索结果 | `/search?search=` | 高频路径 |
| P0 | 分类筛选 | `/filmClassifySearch?Pid=...` | 内容发现核心 |
| P1 | 分类首页 | `/filmClassify?Pid=` | 可与首页样式复用 |
| P0 | 顶部导航/Footer | 全局 | Header 含搜索 + 历史记录下拉 |

### 2.2 后台管理

| 优先级 | 页面 | 路由 |
| --- | --- | --- |
| P0 | 登录 | `/login` |
| P0 | 后台框架（侧栏+顶栏） | `/manage/*` |
| P0 | 采集管理 | `/manage/collect/index` |
| P0 | 站点配置 | `/manage/system/webSite` |
| P1 | 后台首页看板 | `/manage/index` |
| P1 | 定时任务 | `/manage/cron/index` |
| P1 | 影片管理（搜索/分类/添加） | `/manage/film*` |
| P1 | 文件管理 | `/manage/file/*` |

> 排期建议：Sprint 1 完成 P0 全部用户端；Sprint 2 完成 P0 后台 + 部分 P1。

---

## 3. 必须保留的功能点

### 3.1 用户端
1. **首页轮播**：保留 PC 大幅 banner + 海报小卡组合形态（可换皮，不可砍）。
2. **首页"分组+热播榜"**：每个分类一行影片 + 右侧热播榜（桌面）。
3. **详情页选集 Tab + 集数网格**：换源 Tab + 集数列表（旧站 `play-tab-group + play-list-item` 结构）。
4. **详情页剧情简介展开/收起**：默认 2 行，长文本展开按钮。
5. **播放页快捷键**：空格/←/→/↑/↓（旧站已实现，必须保留）。
6. **播放页倍速**：0.5/1.0/1.5/2.0。
7. **播放页自动连播**：开关 + 下一集按钮，无下一集时禁用。
8. **播放页断点续播**：通过 query `currentTime` 传入，video.js ready 时跳转。
9. **观看历史 cookie**：key `filmHistory`，结构 `{ [filmId]: { name, link, episode, timeStamp } }`，**新站必须读写兼容**（老用户切换不丢数据）。
10. **顶栏历史记录下拉**：从 cookie 读，按 timeStamp 倒序，可清空。
11. **搜索栏全站可用**：Header 内置，回车提交。
12. **筛选 Tag 多维联动**：Pid/Category/Plot/Area/Language/Year/Sort，URL query 同步。

### 3.2 后台
1. **登录 → token 写 localStorage**：响应头 `new-token`，旧站 `utils/token.ts` 行为保留。
2. **路由守卫**：未登录访问 `/manage/*` 跳 `/login`。
3. **采集站表格的 8 个核心字段**：name / resultModel / collectType / uri / syncPictures / state / grade / interval。
4. **侧栏分组**：网站管理 / 采集管理 / 定时任务 / 影片管理 / 文件管理。

---

## 4. API / 路由兼容硬约束

### 4.1 不可改项
- 所有 API 路径（`/api/...`）保持不变，参数名大小写敏感，**禁止改名**（含 `Pid`、`Category`、`Plot` 等首字母大写参数）。
- 所有路由 query key 不变（如 `filmDetail` 用 `link` 而非 `id`，`search` 用 `search` 而非 `keyword`）。
- cookie key `filmHistory` 不变，value 结构不变。
- localStorage token key 沿用旧站（请架构师查 `utils/token.ts` 确认）。

### 4.2 可改项
- HTTP 客户端实现（旧站用 axios + 简易封装，新站可换为 axios 实例 + 拦截器或 ofetch）。
- 状态管理：旧站用 reactive，新站建议 Pinia。
- 组件库：旧站重度依赖 Element Plus，新站可用 UnoCSS/Tailwind + 自研组件，**保留 video.js 内核**。
- 路由实现：旧站部分跳转用 `location.href`（首页 a 标签 / 搜索 / 筛选），新站统一改 `router.push`，**但 URL 表达式必须一致**。

---

## 5. 风险点

### R1 视频源跨域 [高]
- **现象**：旧站 `<video-player crossorigin="anonymous" />`，依赖第三方采集源 CDN。
- **风险**：新版若启用 SPA 多次切换 src，可能触发更严格的 CORS preflight；部分源站不返回 CORS header 会导致播放失败。
- **应对**：
  - Architect 阶段验证视频源是否带 CORS（拿一条 `.m3u8` 直接 fetch 测试）。
  - 若不支持 CORS，方案 A：保留与旧站一致的 `<video>` 直链（仅播放、不读取像素数据）；方案 B：后端代理 m3u8 + ts 切片（成本高）。

### R2 cookie 历史记录的同源限制 [中]
- **现象**：cookie 写在当前域，新站若部署到不同域名/子域，老用户历史无法读取。
- **应对**：确认部署域名一致；若域名变化，做一次性数据迁移（首屏检测旧 cookie → 写入 localStorage → 清旧 cookie）。
- 同时建议 Architect 评估**改 localStorage**（容量更大、API 更友好），但需保留 cookie 兼容期。

### R3 路由 query 大小写陷阱 [高]
- 旧站用 `Pid`、`Category` 等大写开头 query，TypeScript 强类型化时易踩坑。
- **应对**：Architect 在 router 层做一层 query 标准化映射，业务代码内部用小驼峰，**仅在 URL 与 API 层保留原大小写**。

### R4 旧站 `location.href` 整页跳转语义 [中]
- 旧站搜索/筛选页用 `location.href` 触发整页刷新，等于"用 URL 当状态机"。
- 新站改 `router.push` 后需手动监听 query 变化重新拉数据，注意 `watch(route, ...)` 与 `onMounted` 的去重。

### R5 Hero 轮播数据来源 [中]
- 旧站硬编码 4 条 banner 数据（樱花庄/Re:0/五等分/我青春），与后台无关。
- 新站不应保留硬编码。**待 PO 确认**：是否复用 `/api/index` 中某个榜单作为轮播源？默认建议用首页热门榜前 5 条。

### R6 后台首页看板缺接口 [低]
- `/api/manage/index` 现仅返回 `msg`，没有数据看板字段。
- **应对**：MVP 阶段后台首页放欢迎卡片 + 快捷入口；看板等后端补接口后二期实现。

### R7 移动端播放器全屏与控制条 [中]
- video.js 默认控制条在小屏触控不友好。
- **应对**：UX 评估是否需要自定义移动端控制条（最小 P1）。

### R8 Element Plus 是否完全移除 [低]
- 后台表格、表单、对话框旧站重度依赖 Element Plus。
- **应对**：建议**用户端完全移除**，**后台保留 Element Plus**（但改暗色主题），降低重写工作量。Architect 决策。

---

## 6. 给 UX 的具体输入

请输出至少以下视觉资产（落到 `doc/design/`）：
1. **设计 Token**：色板（背景/卡片/强调/文字 4 套）、字体阶梯（H1–H4 / body / caption）、间距阶梯、圆角阶梯。
2. **首页 mockup**：桌面（1440） + 移动（375）两版。
3. **详情页 mockup**：桌面 + 移动。
4. **播放页 mockup**：桌面 + 移动（重点：选集换源的触控体验）。
5. **搜索/筛选页 mockup**：桌面 + 移动。
6. **后台 mockup**：登录页 + 采集管理表格页 + 站点配置表单页。
7. **组件库**：FilmCard（多尺寸） / Row（横向滚动） / Tag / Button（主/次/危险） / Pagination / EmptyState / Skeleton。

约束：
- 暗色基调；继承旧站紫色渐变品牌资产作为强调色之一。
- 卡片悬浮态、active 态、loading 骨架态都要画。
- 暗色下确保 WCAG AA 对比度。

---

## 7. 给 Architect 的具体输入

请在 `doc/bmad/architecture-redesign.md` 输出至少以下内容：
1. 项目脚手架（Vite 5 + Vue 3.5 + TS 5 + Pinia + UnoCSS/Tailwind + vue-router）。
2. 目录结构（views / components / composables / stores / api / locales / styles）。
3. **API client**：axios 实例 + 请求/响应拦截器（含 token 注入、`new-token` 头识别、错误统一处理）。
4. **API 类型定义**：基于 `doc/handover/00-context.md` 接口清单生成 TS interface。
5. **路由表**：保持 query 兼容；用 `meta` 标记是否需要登录。
6. **路由守卫**：未登录跳 `/login` 的逻辑迁移。
7. **状态管理切分**：user / site / nav / history（cookie 抽象层）。
8. **视频播放层封装**：基于 video.js + `@videojs-player/vue` 包装组件，统一 src 切换、快捷键、断点续播逻辑。
9. **响应式策略**：UnoCSS preset 断点（sm/md/lg/xl/2xl）与容器策略。
10. **打包与部署**：与旧站 dev 代理（`127.0.0.1:3601`）一致；产物输出到 `client-v2/dist/`。
11. 风险 R1–R8 的应对方案敲定。

---

## 8. 与 PM 沟通的开放问题

请把 PRD §10 的 Q1–Q6 在设计 / 架构方案中明确选型，回写到本文档第 5 节风险表对应项。

如有歧义，回到 `doc/handover/01-pm-handover.md` 找 PM 确认；不要私自变更 API 路径或 query 参数。
