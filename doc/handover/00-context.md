# 共享上下文（重构起点）

## 目标
对 `D:\Git\GoFilm\client/`（Vue3 影视站）进行现代化重构，参照 Netflix / Disney+ 暗色风格。
- 自适应：**桌面 / 平板 / 移动端 / 电视端（TV）** 四端形态
- API 与旧后端 **完全兼容**（base 路径 `/api`，开发期代理到 `127.0.0.1:3601`）
- 全量重构范围：用户端 + 管理端（管理端不需 TV 适配）
- 新项目落到 `D:\Git\GoFilm\client-v2/`，旧项目保留
- 已在仓库根 `git init`，工作分支 `feature/redesign`

## TV 端适配硬约束（**重要**）
- 目标分辨率：1920×1080（主）、3840×2160（兼容缩放）
- 交互模型：**遥控器 / 键盘方向键**为主，鼠标/触摸为辅，不能依赖 hover
- 焦点系统：所有可交互元素必须有清晰的 **focus 指示**（强对比描边 + 轻微 scale 1.06 + 阴影），focus 替代 hover 行为
- 空间导航：上下左右方向键在 Hero / Row / 卡片网格之间正确移动，OK/Enter 选中，Back/Esc 返回
- 安全区：内容距离屏幕边缘至少 48px（10-foot UI 安全区）
- 字号放大：TV 模式下基础字号 +25%（基准 18px，标题 ≥ 32px），行高 1.5 起
- 文字对比度：WCAG AAA（背景与文本 ≥ 7:1）
- 不出现 hover-only 信息（如悬浮卡片才显示标题，TV 必须默认显示）
- Hero 轮播自动播放间隔 ≥ 6 秒（TV 上人们看更慢）
- 横向滚动 Row 必须支持键盘左右键平滑滚动 + 边缘渐隐遮罩
- 视频播放器在 TV 模式下需支持遥控器快捷键（OK 暂停/播放，左右快进/快退 10s，上下音量）
- 检测电视端的策略：UA 嗅探（含 `SmartTV|Tizen|WebOS|HbbTV|Hisense|MiTV`）+ URL 参数 `?mode=tv` 强制 + 用户设置持久化（localStorage `gf-mode`）；或当宽度 ≥ 1920 且无 mouse hover media 支持时启用
- 三种模式：`mobile`（< 768）、`desktop`（768–1919）、`tv`（≥ 1920 或 UA 命中）；平板归 desktop

## TV 端打包：Capacitor 方案（已选定）
- 不重写为 Flutter / 原生，业务代码 100% 复用 Vue3
- 在 `client-v2/` 完成 Web TV 模式开发（焦点系统、空间导航、放大 token、安全区）
- 新增 `client-v2/android/` 由 Capacitor 生成（`@capacitor/core` + `@capacitor/android` + Android TV `<intent-filter>`）
- D-pad keycode 桥接：通过原生 plugin（或 capacitor-android-tv 社区插件）将 `KEYCODE_DPAD_UP=19/DOWN=20/LEFT=21/RIGHT=22/CENTER=23/BACK=4` 转 web 端 `keydown`（ArrowUp/Down/Left/Right/Enter/Escape），让同一套焦点系统直接接住
- AndroidManifest 加 `android.software.leanback`（require=false 兼容手机）+ `MAIN/LEANBACK_LAUNCHER` intent
- TV 启动 Splash + 1080p 画质资源（不放 4K 压缩素材）
- 应用图标：banner.png（TV 大图标 320×180）+ 普通 launcher 图标
- 优先使用 WebView 性能最优配置：`hardwareAccelerated=true`、`androidx.webkit.WebSettingsCompat`
- 视频播放：video.js 走原生 WebView 解码即可（HLS/MP4 都行）
- 包名：`com.gofilm.app`，TV 端入口可用 URL hash `#/tv` 或 `?mode=tv` 强制 TV 视图
- 后续如需 4K：评估 ExoPlayer 桥替换 video.js

## 旧项目技术栈
Vue 3.2 + Vite 4 + TypeScript 4 + Element Plus 2.4 + axios + video.js + @videojs-player/vue。
auto-import + components 自动注册 Element Plus。

## 目标技术栈
Vue 3.5 + Vite 5 + TypeScript 5 + Pinia + UnoCSS（或 Tailwind）+ vue-router 4 + axios + video.js
+ 自研轻量组件（保留必要时回退 Element Plus）。

## 必须保持不变的 API 接口
### 用户端
- `GET  /api/index` 首页（轮播 / 分类 / 热门）
- `GET  /api/navCategory` 顶级分类导航
- `GET  /api/config/basic` 站点信息（siteName/logo）
- `GET  /api/filmDetail?id=` 影片详情 + 相关
- `GET  /api/filmPlayInfo?id=&playFrom=&episode=` 播放信息
- `GET  /api/filmClassify?Pid=` 分类首页
- `GET  /api/filmClassifySearch?Pid=&Category=&Plot=&Area=&Language=&Year=&Sort=&current=` 筛选搜索
- `GET  /api/searchFilm?keyword=&current=` 关键字搜索

### 鉴权
- `POST /api/login` 登录（返回 token 走响应头 `new-token`）
- `GET  /api/logout` 退出
- `POST /api/changePassword`
- `GET  /api/manage/user/info`

### 后台管理
- `GET  /api/manage/index`
- `GET  /api/manage/config/basic`、`POST /api/manage/config/basic/update`
- `GET  /api/manage/collect/{list,options,find,del}`、`POST /api/manage/collect/{add,update,change,test}`
- `POST /api/manage/spider/start`、`GET /api/manage/spider/class/cover`
- `GET  /api/manage/cron/{list,find,del}`、`POST /api/manage/cron/{add,update,change}`
- `GET  /api/manage/film/{search/list,class/tree,class/find,class/del}`、`POST /api/manage/film/{class/update,add}`
- `GET  /api/manage/file/{list,del}`、`POST /api/manage/file/upload`

## 旧项目关键路由（必须维持 query 兼容）
| 路由 | 关键参数 | 用途 |
| --- | --- | --- |
| `/index` | - | 首页 |
| `/filmDetail` | `link={id}` | 影片详情 |
| `/play` | `id`,`source`,`episode`,`currentTime?` | 播放页（cookie 写入观看历史） |
| `/search` | `search`,`current?` | 搜索结果 |
| `/filmClassify` | `Pid` | 分类首页（最新/排行/最近更新） |
| `/filmClassifySearch` | `Pid`,`Category`,`Plot`,`Area`,`Language`,`Year`,`Sort`,`current` | 分类筛选 |
| `/login` | - | 后台登录 |
| `/manage/index` 等 | - | 后台 |

## 设计基调（Netflix / Disney+）
- 深色主背景（`#0b0b0f` / `#141518` 系），高对比卡片
- 主色一档：偏红或紫的强调色（旧站使用紫/品红渐变 `#9b49e7→#4ad1e5`，可作为继承）
- 大幅 Hero 轮播 + 横向滚动剧集行（Row）
- 卡片悬浮放大、渐变遮罩
- 移动优先，断点：sm 360 / md 768 / lg 1024 / xl 1440 / 2xl 1920
- 大屏（>=1440）走中央 1280–1500 容器，超大屏（>=1920）走 1600

## 既有可继承的内容
- iconfont 图标、play.png、404.png、managebg.png 等资产
- 历史观看记录用 cookie 存（key `filmHistory`），新版可保留
- 后台路由结构（侧边栏分组）
- video.js 仍作为播放器内核

## 交付目录
- `doc/bmad/`  PRD、架构、sprint plan、status
- `doc/design/` 视觉规范、mockup
- `doc/handover/` 各阶段交接说明
- `client-v2/`  新项目代码
