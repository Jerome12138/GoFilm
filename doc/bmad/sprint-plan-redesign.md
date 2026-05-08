# Sprint 计划 — GoFilm Vue3 重构

> Sprint：S1 (单一冲刺，跨度由实际节奏决定，目标完成全部用户端 + 管理端 + TV)
> Story 编号：STORY-001..STORY-018
> 总计点数：约 78 pt（1pt ≈ 0.5 工程日）

## 故事看板

### 阶段 A — 基础设施（先做，串行）
| ID | 标题 | 点 | 角色 | 依赖 | 验收要点 |
|---|---|---|---|---|---|
| STORY-001 | 项目脚手架与依赖（Vite5 + Vue3.5 + UnoCSS + Pinia + axios） | 5 | frontend | - | `pnpm dev` 起在 :3600；UnoCSS 输出；TS 严格通过；alias `@/` 生效 |
| STORY-002 | 设计 Token 落地（theme.css 含 mobile/desktop/tv 三套） | 3 | frontend | 001 | CSS 变量全部就位；切换 `data-mode` 视觉变化 |
| STORY-003 | 路由 + 布局壳（PublicLayout + ManageLayout + AuthLayout）| 4 | frontend | 002 | 三套 layout 切换正确；404 兜底；title 自动 |
| STORY-004 | 网络层（axios + 拦截器 + token + loading + 错误处理） | 4 | frontend | 001 | 401 跳登录；token 续期；silent 选项；BizError 抛出 |
| STORY-005 | API 函数模块化（types/film + types/manage + api/*） | 5 | frontend | 004 | 全部接口签名落地；types 完整 |

### 阶段 B — 用户端核心
| ID | 标题 | 点 | 角色 | 依赖 | 验收要点 |
|---|---|---|---|---|---|
| STORY-006 | base 组件（BaseImage / BaseSkeleton / BasePagination / BaseEmpty / BaseTag / BaseButton / BaseDialog） | 5 | frontend | 002 | 自动注册；mobile/desktop/tv 表现正确 |
| STORY-007 | 影视业务组件（HeroCarousel / FilmRow / FilmCard / FilmGrid / FilmFilterBar / EpisodeTabs / RelatedList） | 8 | frontend | 006 | 横滚滑动；hover/focus 行为；自适应列数 |
| STORY-008 | 首页 HomeView（轮播 + 多分类 row） | 4 | frontend | 005,007 | API `/index` 接通；移动 / pad / pc / TV 四档表现达标 |
| STORY-009 | 影片详情 FilmDetailView | 3 | frontend | 005,007 | API `/filmDetail` 接通；剧情展开收起；播放跳转 |
| STORY-010 | 播放页 PlayView（video.js + 播放历史 + D-pad 快捷键） | 5 | frontend | 005,007 | OK 暂停/播放、左右快进、上下音量；cookie filmHistory 兼容 |
| STORY-011 | 搜索 SearchView | 2 | frontend | 005,007 | 关键字搜索 + 分页 + query 同步 |
| STORY-012 | 分类首页 ClassifyView + 筛选 ClassifySearchView | 4 | frontend | 005,007 | 旧 query 100% 兼容；筛选 tag 切换 |

### 阶段 C — TV 适配
| ID | 标题 | 点 | 角色 | 依赖 | 验收要点 |
|---|---|---|---|---|---|
| STORY-013 | useViewMode + useSpatialNavigation + 全局 focus 样式 | 4 | frontend | 002 | 方向键导航顺畅；焦点环可见；UA / 参数切换正确 |
| STORY-014 | TV 模式组件适配（HeroCarousel + FilmRow + Player + 头部） | 4 | frontend | 013,007,010 | 上述组件在 TV mode 切换正确行为；安全区生效 |
| STORY-015 | Capacitor Android TV 打包 | 5 | frontend | 008,010,014 | `cap build` 出 APK；Android TV 模拟器启动 + 遥控器导航 OK |

### 阶段 D — 管理端
| ID | 标题 | 点 | 角色 | 依赖 | 验收要点 |
|---|---|---|---|---|---|
| STORY-016 | 登录 LoginView + ManageLayout（侧栏 + 头部 + 个人下拉 + 修改密码） | 4 | frontend | 003,004,005 | 登录后写 token；侧栏可折叠；改密成功 |
| STORY-017 | 管理端核心页（Dashboard / Collect / Cron / Film 4 页 / File 2 页 / SiteConfig） | 8 | frontend | 016 | 列表 CRUD 全通；上传带进度；分类树展示 |

### 阶段 E — 收口
| ID | 标题 | 点 | 角色 | 依赖 | 验收要点 |
|---|---|---|---|---|---|
| STORY-018 | 联调 + 构建 + 性能预算验证（首屏 < 200KB gzip）+ QA 冒烟 | 4 | frontend+qa | all | `pnpm build` 通过；bundle visualizer 检查；冒烟全过 |

## 执行节奏

- 阶段 A 全部串行，作为后续基础（不可越过）
- 阶段 B 内：006 → 007 串行，008/009/011/012 可并行（各自独立 view），010 单独串行（依赖 video.js 封装）
- 阶段 C：依赖阶段 B 部分完成（不要求全部）
- 阶段 D：可与 C 并行（资源允许时）
- 阶段 E：所有 story 完成后

## 每个 story 的 dev-full 流程
1. **dev**：vue3-frontend agent 编码
2. **review**：code-reviewer agent 审查
3. **fix**：dev 修 CRITICAL/HIGH
4. **qa**：手工冒烟（QA agent 在 STORY-018 一次性兜底）
5. **complete**：更新 sprint-status.yaml，递增 completed_points
