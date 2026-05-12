# GoFilm

一个基于 Vue 3 和 Gin 实现的在线观影网站, 支持桌面端 / 移动端 / TV 端 / Android 原生包.

效果展示: <a href="https://m.mubai.link/" target="_blank">点击访问演示站点</a>

## 简介

**GoFilm** 由两部分组成:

- **前端 `client-v2/`** —— Vue 3.5 + Vite 5 + TypeScript 5.6 + Pinia 2 + UnoCSS 单页应用, 同一份代码服务用户端 (`/index` 起)、管理后台 (`/manage` 起) 和 TV 端 (UA 自动识别), 并通过 Capacitor 打包出 Android APK.
- **后端 `server/`** —— Gin + GORM + go-redis, 用 gocolly + robfig/cron 做公共影视资源采集与定时更新, JWT 鉴权, 普通用户 / 管理员双角色.

> 历史的 Vue 2 + ElementPlus 版本 (`client/`) 已被 `client-v2/` 全量取代, 已从仓库移除. 想看旧版可回溯到 `81d4609` 之前的提交.

## 功能概览

**用户端 (`/index` 起)**

- 首页轮播 + 多分类 Row + 热门
- 分类导航 / 分类筛选 / 关键字搜索
- 影片详情 (海报 / 评分 / 标签 / 多源选集 / 相关推荐)
- 播放页 (video.js + 自动续播 + 自动下一集 + 键盘 / D-pad 快捷键)
- **观看历史 / 我的收藏**: 未登录走 cookie+localStorage, 登录后无缝迁移到云端, 跨设备同步
- 普通用户登录 (后端 `/user/login`, 由管理员后台创建账号)
- 默认访问: `服务器IP:默认端口 [http://127.0.0.1:3600]`

**管理后台 (`/manage` 起, 需管理员 role)**

- 仪表盘 (影片数 / 采集源数 / 定时任务数)
- 采集源管理 / 影视采集任务 / 定时更新
- 影片管理 / 影片分类
- 文件管理 (单 / 多文件上传, 图片墙)
- 站点配置 (站名 / Logo / SEO / 维护提示)
- 管理后台访问需登录, 默认账号: `admin admin` (首次登录后请立即改密)
- 默认访问: `http://127.0.0.1:3600/manage`

**TV 端**

- UA 自动识别 (Tizen / WebOS / Android TV)
- 按 `data-mode='tv'` 切换更大字号 / 安全区 / 焦点描边 / D-pad 导航
- 详见 `doc/handover/07-tv-adapt.md`

**Android 原生包**

- `client-v2/` 集成了 Capacitor, `pnpm cap:build` 直出 APK
- 详见 `doc/handover/09-capacitor.md`

## 目录结构

- `client-v2/` Vue 3 客户端项目 — [README](./client-v2/README.md)
- `server/` Go 服务端接口项目 — [README](./server/README.md)
- `film/` 部署相关配置 (Docker Compose / Nginx) — [README](./film/README.md)
- `doc/` PRD / 架构 / 交付 / 测试报告归档

```text
GoFilm
├─ client-v2/                  # Vue 3 SPA (用户端 + 管理后台 + TV)
│  ├─ src/
│  │  ├─ api/                  # axios 封装 + 模块化接口
│  │  ├─ components/           # base/ film/ layout/ manage/
│  │  ├─ composables/          # useFilmHistory / usePlayer / useViewMode ...
│  │  ├─ mock/                 # VITE_USE_MOCK=1 时启用的离线 mock
│  │  ├─ router/               # routes.public + routes.manage + guards
│  │  ├─ stores/               # pinia: user/history/favorite/site/nav/ui
│  │  ├─ types/                # 全局 DTO 类型 (与后端契约对齐)
│  │  ├─ views/                # public/ manage/ auth/ error/
│  │  └─ main.ts
│  ├─ tests/                   # Playwright e2e + 视口适配矩阵
│  ├─ android/                 # Capacitor Android 工程
│  ├─ uno.config.ts
│  ├─ vite.config.ts
│  └─ package.json
├─ server/                     # Gin 后端
│  ├─ controller/              # API 入口
│  ├─ logic/                   # 业务逻辑
│  ├─ model/system/            # 数据模型 + 持久化
│  ├─ plugin/
│  │  ├─ db/                   # mysql + redis 客户端
│  │  ├─ middleware/           # JWT / RequireAdmin / CORS
│  │  └─ spider/               # 影视资源采集器
│  ├─ router/                  # gin 路由
│  ├─ config/                  # 端口 / 表名 / Redis key / DSN
│  └─ main.go
├─ film/                       # 部署
│  ├─ data/
│  │  ├─ nginx/                # html (上传 client-v2/dist) + nginx.conf
│  │  └─ redis/                # redis.conf
│  ├─ server/                  # 镜像构建用 server 快照 (可与根 server 同步)
│  ├─ docker-compose.yml
│  └─ Dockerfile
├─ doc/                        # 设计 / 架构 / 交付 / 测试报告
├─ LICENSE
└─ README.md
```

## 快速开始 (本地开发)

```bash
# 0. 前置: Node 20+, pnpm 9+, Go 1.21+, MySQL 8, Redis 6+

# 1. 后端
cd server
# 改 config/DataConfig.go 里的 MysqlDsn 与 RedisAddr 为你的本地实例
go run main.go              # 默认 :3601

# 2. 前端
cd ../client-v2
pnpm install
pnpm dev                    # 默认 :3600, 自动反代 /api -> :3601

# 3. (可选) 不想起后端先体验 UI
# .env.development 设 VITE_USE_MOCK=1, 走前端内置 mock 适配器
```

首次启动会自动建 `users` 等表, 并把 `admin/admin` 升级为管理员账号. 登录后请立刻改密.

## 生产部署

走 `film/` 目录下的 Docker Compose, 步骤详见 [film/README.md](./film/README.md). 关键步骤:

```bash
# 1. client-v2 打包静态资源
cd client-v2 && pnpm build       # 产物在 client-v2/dist

# 2. 复制 dist 到 nginx html
cp -r client-v2/dist/* film/data/nginx/html/

# 3. 同步最新 server 代码到 film/server/, 启动 docker compose
cp -r server/* film/server/
cd film && docker compose build && docker compose up -d
```

## 测试

```bash
# 前端类型 + e2e
cd client-v2
pnpm type-check                  # vue-tsc --noEmit
pnpm test                        # vitest
pnpm exec playwright test        # 6 视口适配矩阵 (chromium)

# 后端单元 + 集成测试 (78 个)
cd server
go test ./...
```

## 起源

从正式接触编程语言到第一次动手敲代码, 当时有动手做一些东西的想法, 也正是在那时喜欢追番迷二次元, 曾想过做一个自己的动漫站.

但因为知识面匮乏, 总是在进行到某一步时就会遇到一些盲区, 从最开始的静态页面到后面的伪数据, 也实现过一些当时能做到的部分.

后面慢慢学习的过程中也渐渐遗忘了这个想法, 但因为一些偶然的因素, 想要做一个自己的开源项目, 于是就从零开始慢慢实现并完善了这个影视站的各个部分. 期间也一点点修改颠覆了一些最开始的思路, 但目前主体功能基本完善, 后续也会定期进行一些 bug 修复和新功能的更新.

如有发现 Bug 或者有好的建议, 可以进行反馈, 欢迎各位大佬来指点一二.

## 更新迭代计划

- 历史记录与收藏已经接入云端 (登录态), 跨设备同步; 安卓壳后续会加 push / 离线缓存
- TV 端会继续完善焦点管理, 增加遥控器手势支持
- 后端考虑把硬编码 DSN 迁到环境变量 / 配置文件

## JetBrains 开源证书

感谢 JetBrains 提供的免费开源许可, GoLand 和 WebStorm 为编程开发带来了良好的体验.

<a href="https://www.jetbrains.com/?from=GoFilm" target="_blank"><img src="https://resources.jetbrains.com/storage/products/company/brand/logos/jb_beam.svg" alt="JetBrains Logo (Main) logo."></a>
