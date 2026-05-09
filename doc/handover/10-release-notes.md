# GoFilm Vue3 重构 - Release Notes

> 发布版本：`feature/redesign` → 准备合入 `main`
> 发布日期：2026-05-09
> 范围：`client-v2/` 全量重构（用户端 + 管理端 + TV 容器）
> 后端：保持原 API 不变，零侵入

---

## 1. 总览

将旧版 `client/` (Vue2 + Webpack) 全量重构为现代化技术栈，并新增 TV 适配。所有故事 (STORY-001 ~ STORY-018, 78 SP) 已完成。

| 维度 | 旧版 client/ | 新版 client-v2/ |
|---|---|---|
| 框架 | Vue 2.6 | **Vue 3.5.13** |
| 构建 | Webpack 4 | **Vite 5.4** |
| 类型 | JS | **TypeScript 5.6** |
| 状态 | Vuex 3 | **Pinia 2.2** |
| 路由 | vue-router 3 | **vue-router 4.4** |
| 样式 | Sass | **UnoCSS 0.62 + CSS 变量** |
| 图标 | Element-UI | **本地 SVG 集合（BaseIcon）** |
| 播放器 | xgplayer | **video.js 8.17 + @videojs-player/vue** |
| 端形态 | 桌面 + 移动 | 桌面 + Pad + 移动 + **TV (D-pad)** |
| 安卓打包 | 无 | **Capacitor 6 → Android TV APK** |

---

## 2. 验收指标

| 指标 | 目标 | 实测 |
|---|---|---|
| 首屏 JS gzip | ≤ 200KB | **76 KB** ✅（vue-vendor 41.65 + index 17.69 + utils-vendor 14.03 + Home 3.40） |
| Home / Detail / Play 首次访问 | 路由级懒加载 | ✅ |
| 类型检查 | 0 error | ✅（vue-tsc --noEmit 静默通过） |
| 生产构建 | 0 error | ✅（23.36s） |
| API 路径 / 字段大小写 | 与旧端一致 | ✅（含 Pid/Cid/Plot/Area/Language/Year/Sort 与 `/filmDetail?link=`） |
| QA 报告 P0 | 全修复 | ✅（9 项已在 `44087fe` 修复） |

---

## 3. 用户端能力清单

| 模块 | 路由 | 状态 |
|---|---|---|
| 首页（轮播 + 多分类列表） | `/` | ✅ |
| 影片详情 | `/filmDetail?link=:id` | ✅ |
| 播放页（多源 / 多集 / 进度记忆） | `/play?id=&source=&episode=&currentTime=` | ✅ |
| 搜索（多 tab + 历史） | `/search?keyword=` | ✅ |
| 分类首页 | `/classify/:pid` | ✅ |
| 分类筛选页 | `/search?Pid=&Category=&Plot=&Area=&Language=&Year=&Sort=` | ✅ |
| 观看历史（卡片网格 + 单条移除 + 清空） | `/history` | ✅（本次新增） |

---

## 4. 管理端能力清单

| 模块 | 路由 | 状态 |
|---|---|---|
| 登录 / 鉴权 / 自动续期 | `/manage/login` | ✅ |
| 控制台 | `/manage/dashboard` | ✅ |
| 影片列表 / 编辑 / 详情 | `/manage/film*` | ✅ |
| 分类树 | `/manage/film/class` | ✅ |
| 采集源 CRUD + 启停 + 触发采集 | `/manage/collect` | ✅ |
| 定时任务 CRUD + 多源选择 | `/manage/cron` | ✅ |
| 文件上传（拖拽 + 进度） | `/manage/file/upload` | ✅ |
| 图墙 (PhotoWall) | `/manage/file/gallery` | ✅ |
| 站点配置 | `/manage/system/site` | ✅ |
| 修改密码（弹窗） | header → menu | ✅ |

---

## 5. TV 适配

- `useViewMode` 三路触发：UA 检测（含 `tv`/`bravia`/`shield`/`crkey` 等） / URL `?mode=tv` / localStorage
- `[data-mode="tv"]` 主题 token 覆盖（更大字号 / 更亮焦点 / 关闭悬停）
- `useSpatialNavigation`：基于 DOM 几何位置的方向键焦点跳转
- `dpad.ts`：Android TV `KEYCODE_DPAD_*` → 浏览器 `ArrowKey` / `Enter` / `Escape` 桥接
- `data-focusable="true"` 标记可聚焦元素
- Capacitor 容器：`com.gofilm.app`，AndroidManifest 含 `LEANBACK_LAUNCHER` + `android.software.leanback`，`MainActivity.dispatchKeyEvent` 派发 D-pad

---

## 6. 本次（10-release）增量

1. **HistoryView 真实卡片** — `client-v2/src/views/public/HistoryView.vue`
   - 网格布局（自适应 150~180px）
   - 海报 + 集数标签 + 进度时间 + 单条移除按钮 + 清空按钮
   - 点击直接跳回 `record.link`（含 `currentTime`）
2. **vite 警告清理** — `vite.config.ts` 移除冗余 `splitVendorChunkPlugin()`（与 `manualChunks` 对象形式冲突）

---

## 7. 已知遗留 / 后续优化（非阻塞）

| 项 | 优先级 | 说明 |
|---|---|---|
| `video-vendor` 683KB（gzip 204KB） | 低 | video.js 自身体积，路由级懒加载已隔离 |
| `managebg.png` 背景图压缩 | 低 | 视觉资源优化 |
| Dashboard 数据真实化 | 中 | 当前仍为 placeholder，需后端聚合接口配合 |
| SiteBasic 字段全量对齐 | 低 | 仅暴露常用字段，缺 SEO/统计代码等次要字段 |
| Video.js D-pad 精细控制 | 低 | 当前仅基础前进/后退 |

---

## 8. 风险与回滚

- 旧 `client/` 目录保持原样未触碰，回滚 = 直接发布旧 `client/`
- `client-v2/` 与后端通过 `/api/*` 反代（`vite.config.ts` 中代理 `127.0.0.1:3601`），生产部署需在 nginx 同步规则
- Token 双轨：localStorage `auth-token` 持久化 + 响应头 `new-token` 续期，与旧端 cookie 模式不冲突（同源不同 storage）

---

## 9. Sprint 提交记录

| Commit | 说明 |
|---|---|
| `e64ad76` | chore: snapshot before vue3 redesign |
| `849feb4` | docs: PRD/UX/architecture/sprint-plan + TV addendum |
| `3cf6ce8` | feat(client-v2): foundation scaffolding (STORY-001..005) |
| `cf801a9` | feat(client-v2): base + film components (STORY-006/007) |
| `b7e041a` | feat(client-v2): user-facing pages (STORY-008..012) + layout polish |
| `0dcdc28` | feat(client-v2): TV mode + manage console (STORY-013..017) |
| `44087fe`+ | fix(client-v2): align manage API with backend (9 P0 from QA) |
| 本次 | feat(client-v2): HistoryView 真实卡片 + vite 配置清理 |

---

## 10. 验收建议

合入 `main` 前：

1. 后端启动 (`server/`)，前端 `cd client-v2 && pnpm dev`
2. 走通用户端核心路径：首页 → 详情 → 播放（记录历史）→ 历史页 → 搜索 → 分类筛选
3. 走通管理端：登录 → 影片列表 → 采集源新增 → 触发采集 → 文件上传 → 站点配置
4. TV 验证：`?mode=tv` 进入，键盘方向键测试焦点流转 + Enter 进入；如有 Android TV 真机，跑 `cd android && ./gradlew assembleDebug` 安装 APK
5. 合入策略：建议 squash merge 或保留 7 个语义化 commit
