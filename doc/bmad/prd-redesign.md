# PRD - GoFilm 前端重构（Redesign）

> 版本：v1.0  PRD-lite  
> 作者：产品经理  
> 范围：`client/` → `client-v2/` 全量重构  
> 原则：**同等功能 + 新外观/新交互**，API 与路由 query 完全兼容

---

## 1. 需求背景

### 1.1 业务目标
- **视觉现代化**：将旧站从 Element Plus 紫色拼接风格升级为 Netflix / Disney+ 暗色沉浸风格，提升停留时长与品牌感。
- **响应式自适应**：覆盖桌面（≥1440）/ 平板（768–1024）/ 移动端（≤768），同一套代码多端可用。
- **零后端改动**：后端契约（API 路径、参数、响应字段）100% 不变，重构仅限前端。
- **后台管理可用化**：旧后台 UI 老旧、留白多，重构后保持功能不丢失，但改用更清晰的暗色管理风格。

### 1.2 用户痛点（旧站）
- 首屏紫色渐变 banner 与影视主题违和；电影海报小而稀疏，缺少 Netflix 式横向滚动行。
- 详情页"立即播放 / 选集"分散，主操作不突出；移动端 title_mt 排版拥挤。
- 播放页在窄屏下控制条按钮过小、选集换源体验差。
- 搜索 / 筛选页全靠 `location.href` 整页跳转，体感慢，无 SPA 过渡。
- 后台首页空白，没有数据看板；列表交互依赖原生 `el-table`，移动端不可用。

### 1.3 预期收益
- 视觉与一线流媒体看齐，提升用户感知质量。
- SPA 内导航（router.push）替代整页跳转，关键流程交互延迟 < 200ms。
- 一套响应式代码同时服务 PC/Pad/Mobile，移动端首屏可用。
- 后台从"能用"升级到"好用"，新增数据看板基础。

---

## 2. 角色

| 角色 | 描述 | 入口 |
| --- | --- | --- |
| **访客**（用户端） | 浏览/搜索/筛选/观看影片，无登录态，观看历史以 cookie 存储 | `/`（重定向到 `/index`） |
| **站长**（管理端） | 管理采集站、定时任务、影片分类与上下架、文件、站点配置 | `/login` → `/manage/*` |

> 旧站未实现普通用户注册/登录（Login 注册按钮 disabled），本次重构**沿用现状**，不引入用户账户体系。

---

## 3. 用户故事

### 3.1 访客（用户端）

#### 首页 `/index`
- 作为访客，我希望进入首页就看到一个**沉浸式 Hero 大图轮播**（含影片名/标签/海报），以便我直观感受站点的内容定位。
- 作为访客，我希望看到按 Pid 分组的**横向滚动剧集行**（电影 / 剧集 / 动漫 / 综艺），每行右侧显示该分类的子分类入口与"更多"。
- 作为访客，我希望在大屏（lg 及以上）看到右侧"🔥热播榜"侧栏，移动端则降级隐藏。
- 作为访客，我希望卡片悬浮时有放大与渐变遮罩，能看到副标题和清晰的"点击播放"暗示。

#### 影片详情 `/filmDetail?link={id}`
- 作为访客，我希望详情页有**全屏背景模糊海报 + 前景信息卡**，以便快速决定是否观看。
- 作为访客，我希望"立即播放"按钮**主操作显著**（圆角大按钮 + 强调色），并能看到默认播放源。
- 作为访客，我希望剧集列表按"播放源 Tab + 集数网格"展示，集数过多时可滚动，**当前集**有高亮。
- 作为访客，我希望剧情简介**默认 2 行+展开按钮**（旧站逻辑保留）。
- 作为访客，我希望页面底部展示"相关推荐"列表，复用首页卡片样式。

#### 播放页 `/play?id=&source=&episode=&currentTime?`
- 作为访客，我希望播放器**铺满 16:9 容器**，桌面端宽屏占主区，移动端竖向占满宽度。
- 作为访客，我希望保留 video.js 内核的快捷键（空格/←/→/↑/↓）与倍速 0.5/1.0/1.5/2.0。
- 作为访客，我希望"自动连播 / 下一集"按钮明显，**下一集无可用时按钮置灰**。
- 作为访客，我希望换源/换集**仅刷新视频流不刷新整页**（保持 SPA 体验）。
- 作为访客，我希望关闭页面或换集时**自动写入 cookie 历史记录**（key `filmHistory`，结构同旧站）。

#### 搜索 `/search?search=&current?`
- 作为访客，我希望顶栏搜索框任意页面可用，回车或点击搜索按钮跳转 `/search?search=...`。
- 作为访客，我希望搜索结果以**横向卡片（海报+元信息+剧情节选+播放按钮）**展示，并支持分页。
- 作为访客，我希望搜索结果为空时看到 EmptyState 提示。

#### 分类首页 `/filmClassify?Pid=`
- 作为访客，我希望看到三个滚动行：**最新上映 / 排行榜 / 最近更新**，每行有"更多"链接到筛选页对应 Sort 参数。
- 作为访客，我希望页面顶部有"分类名 / 分类库"双 Tab，和筛选页互通。

#### 分类筛选 `/filmClassifySearch?Pid=&Category=&Plot=&Area=&Language=&Year=&Sort=&current=`
- 作为访客，我希望看到一组**多维筛选 Tag 行**（剧情/地区/语言/年份/排序），点击 Tag 立刻刷新结果。
- 作为访客，我希望筛选结果用网格卡片+分页器展示。
- 作为访客，我希望筛选切换时**URL query 同步更新**，便于分享/收藏（兼容旧站）。

### 3.2 站长（管理端）

#### 登录 `/login`
- 作为站长，我希望用账号密码登录，登录成功 token 存 localStorage，跳转到 `/manage/index`。
- 作为站长，我希望登录失败有清晰错误提示。
- 作为站长，我希望支持回车提交、密码可见性切换（保留旧站交互）。

#### 后台首页 `/manage/index`
- 作为站长，我希望进入后台看到**简易数据看板**（影片总数 / 采集站数 / 定时任务数 / 最近采集状态），替代旧站的空白页。
- 作为站长，我希望左侧为可折叠侧边栏，顶部为站点名+用户菜单（含修改密码、退出）。

#### 采集管理 `/manage/collect/index`
- 作为站长，我希望以表格列出所有采集站（名称 / 类型 / URI / 状态 / 权重 / 采集间隔）。
- 作为站长，我希望能添加 / 编辑 / 删除 / 启停 / 测试连通性 / 触发立即采集。
- 作为站长，我希望"启动采集"时弹窗选择采集范围（分类/全量/增量），调用 `/manage/spider/start`。

#### 定时任务 `/manage/cron/index`
- 作为站长，我希望以表格管理 cron 任务（名称 / 表达式 / 关联资源站 / 状态 / 上次运行）。
- 作为站长，我希望能添加 / 编辑 / 删除 / 启停。

#### 文件管理 `/manage/file/upload` `/manage/file/gallery`
- 作为站长，我希望能上传图片到服务器（`/manage/file/upload`），上传后即可在图库管理中看到。
- 作为站长，我希望图库支持网格预览、复制链接、删除。

#### 影片管理 `/manage/film` / `class` / `add` / `detail`
- 作为站长，我希望能搜索/分页查看所有影片（`/manage/film/search/list`）。
- 作为站长，我希望能管理影片分类树（`/manage/film/class/tree` 增删改）。
- 作为站长，我希望能手工添加一部影片（`/manage/film/add`）。
- 作为站长，我希望能查看单部影片详情（`film/detail` 路由保留，旧站 placeholder，本次可继续 placeholder 或简单实现）。

#### 站点配置 `/manage/system/webSite`
- 作为站长，我希望能修改站点名称、Logo、SEO 信息（`/manage/config/basic` 读写）。

---

## 4. API 兼容矩阵（**硬约束：路径 / 参数名 / 响应字段不得改**）

### 4.1 用户端
| 接口 | 方法 | 调用方页面 | 用途 |
| --- | --- | --- | --- |
| `/api/index` | GET | 首页 `Home.vue` | 轮播 / 分类行 / 热门榜 |
| `/api/navCategory` | GET | 顶部 `Header.vue` | 顶级分类导航 |
| `/api/config/basic` | GET | 顶部 `Header.vue` | siteName / logo |
| `/api/filmDetail?id=` | GET | 详情 `FilmDetails.vue` | 影片详情 + 相关推荐 |
| `/api/filmPlayInfo?id=&playFrom=&episode=` | GET | 播放 `Play.vue` | 播放源 / 集数 / 当前 link |
| `/api/filmClassify?Pid=` | GET | 分类首页 `FilmClassify.vue` | 最新/排行/最近更新 |
| `/api/filmClassifySearch?Pid=&Category=&Plot=&Area=&Language=&Year=&Sort=&current=` | GET | 筛选 `FilmClassifySearch.vue` | 多维筛选 + 分页 |
| `/api/searchFilm?keyword=&current=` | GET | 搜索 `SearchFilm.vue` | 关键词搜索 |

### 4.2 鉴权
| 接口 | 方法 | 调用方 | 备注 |
| --- | --- | --- | --- |
| `/api/login` | POST | 登录页 | 响应头 `new-token` 写 localStorage |
| `/api/logout` | GET | 后台 Header | 清 token + 跳 `/login` |
| `/api/changePassword` | POST | 后台用户菜单 | 修改密码 |
| `/api/manage/user/info` | GET | 后台 Header | 当前登录用户信息 |

### 4.3 后台管理
| 接口 | 方法 | 调用方 |
| --- | --- | --- |
| `/api/manage/index` | GET | 后台首页 |
| `/api/manage/config/basic` | GET | 站点配置 / 侧边栏 |
| `/api/manage/config/basic/update` | POST | 站点配置 |
| `/api/manage/collect/list` | GET | 采集管理 |
| `/api/manage/collect/options` | GET | 采集表单选项 |
| `/api/manage/collect/find` | GET | 采集编辑 |
| `/api/manage/collect/del` | GET | 采集删除 |
| `/api/manage/collect/add` | POST | 采集新增 |
| `/api/manage/collect/update` | POST | 采集编辑 |
| `/api/manage/collect/change` | POST | 采集启停切换 |
| `/api/manage/collect/test` | POST | 采集站连通性测试 |
| `/api/manage/spider/start` | POST | 启动采集 |
| `/api/manage/spider/class/cover` | GET | 采集分类覆盖列表 |
| `/api/manage/cron/list` | GET | 定时任务 |
| `/api/manage/cron/find` | GET | 定时任务编辑 |
| `/api/manage/cron/del` | GET | 定时任务删除 |
| `/api/manage/cron/add` | POST | 定时任务新增 |
| `/api/manage/cron/update` | POST | 定时任务编辑 |
| `/api/manage/cron/change` | POST | 定时任务启停 |
| `/api/manage/film/search/list` | GET | 影片搜索 |
| `/api/manage/film/class/tree` | GET | 分类树 |
| `/api/manage/film/class/find` | GET | 分类详情 |
| `/api/manage/film/class/del` | GET | 分类删除 |
| `/api/manage/film/class/update` | POST | 分类更新 |
| `/api/manage/film/add` | POST | 影片添加 |
| `/api/manage/file/list` | GET | 图库 |
| `/api/manage/file/del` | GET | 图库删除 |
| `/api/manage/file/upload` | POST | 文件上传 |

---

## 5. 路由 / Query 兼容矩阵（**硬约束：query 参数名不变**）

| 路由 | 关键参数 | 备注 |
| --- | --- | --- |
| `/` | - | 重定向到 `/index` |
| `/index` | - | 首页 |
| `/filmDetail` | `link={id}` | 注意是 `link` 不是 `id` |
| `/play` | `id`, `source`, `episode`, `currentTime?` | `currentTime` 用于断点续播 |
| `/search` | `search`, `current?` | `search` 不是 `keyword`（但 API 参数是 `keyword`） |
| `/filmClassify` | `Pid` | 大写 P |
| `/filmClassifySearch` | `Pid`, `Category`, `Plot`, `Area`, `Language`, `Year`, `Sort`, `current` | 全部首字母大写（除 current） |
| `/login` | - | 后台登录 |
| `/manage/index` | - | 后台首页 |
| `/manage/collect/index` | - | 采集管理 |
| `/manage/system/webSite` | - | 注意 webSite 大小写 |
| `/manage/cron/index` | - | 定时任务 |
| `/manage/file/upload` | - | 文件上传 |
| `/manage/file/gallery` | - | 图库 |
| `/manage/film` | - | 影片信息 |
| `/manage/film/class` | - | 影视分类 |
| `/manage/film/add` | - | 影片添加 |
| `/manage/film/detail` | - | 视频详情（旧站 placeholder） |
| `*` | - | 404 页 |

> 兼容性策略：旧外链（如其他站点引用的 `/filmDetail?link=xxx`）**必须仍能访问到对应内容**。新版可在内部使用 vue-router `params` 但 URL 必须保留 query 形式。

---

## 6. 视觉与交互参照（Netflix / Disney+）

### 6.1 关键视觉特征
- **暗色主背景**：`#0b0b0f` 主背景，`#141518` 卡片底色，分区无强分隔线，靠间距与微弱描边。
- **强调色**：保留旧站紫/品红渐变 `#9b49e7 → #4ad1e5`（继承品牌资产），主 CTA 用纯色（建议品红 `#e50914` 风格 或紫色 `#9b49e7`）。
- **Hero 区**：1 屏占据 60vh，左下角影片名+标签+CTA，右下角海报小卡，背景渐变遮罩 `linear-gradient(to right, #000 0%, transparent 60%)`。
- **横向滚动行（Row）**：海报 16:9 或 2:3，行高自适应，左右箭头按钮（桌面），移动端用原生触摸滚动+滚动条隐藏。
- **卡片 Hover**：`scale(1.05) + translateY(-4px)` + 显示渐变遮罩 + 副标题浮现，过渡 200ms ease。
- **字体**：标题 600/700，正文 400/500；建议 Inter / Noto Sans SC 系列。

### 6.2 关键交互特征
- 全站 SPA 导航（`router.push`），消除旧站 `location.href` 整页跳转。
- 路由切换时主区淡入 150ms。
- 顶栏滚动到一定阈值后从透明变实色（Netflix style）。
- 列表加载使用骨架屏，避免空白闪烁。
- 移动端：底部 tabbar 可选；首页/分类页采用横向触摸滑动；搜索框点击展开全屏覆盖层。

### 6.3 后台风格
- 暗色（与用户端一致）+ 更密的信息密度。
- 侧边栏分组：网站管理 / 采集管理 / 定时任务 / 影片管理 / 文件管理。
- 表格使用紫色描边 + 行 hover 高亮，保留 Element Plus 表格的功能但改主题。

---

## 7. 多语言需求

> 本次重构 **MVP 阶段不引入多语言**（保持中文文案与旧站一致），但代码层面预留 i18n 接入点：

- 文案集中放 `client-v2/src/locales/zh-CN.ts`，组件不硬编码字符串。
- 选用 `vue-i18n@9`，懒加载语言包。
- 不做 RTL 适配（业务无需求）。

---

## 8. 大屏交互规范（本项目主要面向 PC/移动 Web，不针对教育触控大屏）

> 注：本规范条目源于通用规范模板，本项目实际目标是 PC/Pad/Mobile Web，故重新定义如下：

- **桌面（≥1440）**：中央 1280–1500 容器，超大屏 1600；卡片每行 6–7 个。
- **平板（768–1024）**：每行 4 个卡片，简化筛选 Tag。
- **移动（≤768）**：每行 2 个卡片，搜索栏置顶，分类导航横向可滚。
- **触控目标**：移动端按钮 ≥ 40px，避免误触。
- **字号下限**：移动端正文 ≥ 13px，桌面端 ≥ 14px；标题对比度 ≥ 4.5:1。

---

## 9. 验收标准（DoD）

### 9.1 功能完整性
- [ ] 用户端 6 个页面 + 后台 11 个页面 100% 落地
- [ ] 首页 hero 轮播、横向滚动行、热播榜（桌面）正常加载
- [ ] 详情页"立即播放""相关推荐""选集换源"全部可用
- [ ] 播放页 video.js 集成，快捷键、倍速、自动连播、cookie 历史记录均正常
- [ ] 搜索 / 筛选 / 分类首页 query 参数与旧站 100% 一致
- [ ] 后台登录鉴权 + 路由守卫（未登录跳 `/login`）
- [ ] 后台采集管理：CRUD + 启停 + 测试 + 触发采集 全部可用
- [ ] 后台定时任务、文件、影片、站点配置 全部 CRUD 完整

### 9.2 兼容性
- [ ] 所有 API 路径与请求参数与旧站完全一致（diff 0 处后端契约改动）
- [ ] 所有路由 query 参数名与旧站完全一致（直接复制旧外链可访问）
- [ ] cookie key `filmHistory` 数据结构兼容（旧用户访问新站不丢历史）

### 9.3 视觉与交互
- [ ] 视觉走查通过（设计师 + PM 一轮）
- [ ] 桌面 / 平板 / 移动三档断点验证，无错位
- [ ] SPA 路由切换无整页刷新
- [ ] Lighthouse Performance ≥ 80（桌面） / ≥ 60（移动）

### 9.4 工程质量
- [ ] TypeScript 严格模式，无 `any` 滥用
- [ ] ESLint + Prettier 通过
- [ ] 关键页面（首页/详情/播放）有手动测试用例文档

---

## 10. 待确认问题

| # | 问题 | 优先级 | 建议方案 |
| --- | --- | --- | --- |
| Q1 | Hero 轮播数据从 `/api/index` 取还是从首条影片取？旧站用了**硬编码 4 条**模拟数据 | 高 | 建议从 `/api/index` 取榜单前 N 条作为轮播来源；硬编码作为兜底 |
| Q2 | 视频源是否存在跨域？旧站设了 `crossorigin="anonymous"` | 高 | 让架构师 + 后端联合验证；如有 CORS 问题考虑后端代理 |
| Q3 | 后台首页看板数据是否有现成接口？`/api/manage/index` 旧站只 toast 一条 msg，未渲染 | 中 | 先做 placeholder 卡片，看板数据待后端补充接口 |
| Q4 | `/manage/film/detail` 旧站为 placeholder（Temp.vue），是否本次实现 | 中 | 建议 MVP 阶段保留 placeholder，二期实现 |
| Q5 | 是否引入 Pinia？旧站全用 reactive | 低 | 建议引入，集中管理 user / site / nav 三处全局态 |
| Q6 | 旧版"修改密码"入口在哪？源码有 API 但 UI 未找到 | 低 | 后台用户菜单新增入口 |

---

## 11. 原型文件

> 本次重构以**视觉 mockup**（设计师交付）为主，**不出 HTML 线框**。如需 PM 侧线框示意，可在 `doc/design/` 下补充。
