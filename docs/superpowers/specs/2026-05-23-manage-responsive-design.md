# Manage 后台响应式设计 spec

> 日期: 2026-05-23 · 状态: draft (rev2, 含 spec-reviewer 反馈修订)

## 背景

当前 \`ManageLayout.vue\` 模板硬编码 \`data-mode="desktop"\`，sidebar + main 横向 flex 在 mobile 完全不可用。
项目已有 \`useViewMode\` (mobile/desktop/tv) 和 \`PublicLayout\` 的响应式实现，**manage 区是唯一没接入响应式的部分**。

## 目标

让所有 \`/manage/*\` 路由在 mobile (\<768px) / tablet (768-1024) / desktop (\>=1024) 三档下都可用，
关键操作（登录、用户管理、影片管理、文件管理、采集、Cron、系统配置）触摸友好。

## 设计决定

### 断点

**沿用现有 uno.config.ts 定义，不改动：**

```ts
breakpoints: { sm: 360px, md: 768px, lg: 1024px, xl: 1440px, 2xl: 1920px }
```

响应式语义映射：
- \`md\` (768px) = mobile ↔ tablet 分界
- \`lg\` (1024px) = tablet ↔ desktop 分界

不引入新断点，避免破坏 public 区现有的 \`container-page\` (用了 2xl) 等组件。

### useViewMode 扩展

新增 tablet 档：

| 视口 (auto 计算时) | mode |
|---|---|
| < 768 | mobile |
| 768-1023 | tablet |
| >= 1024 | desktop |

TV 检测逻辑 (UA / URL / persisted) **完全不变**。

**优先级链 (无改动)**: \`persisted > URL > UA > auto\`，只是 auto 分支从二档变三档。

**persisted set 维持不变**: \`{tv, mobile, desktop}\`，**tablet 不入 persisted**。原因：用户主动 setMode 是为了"我想看 mobile 版" / "我想看桌面版"，没有"我想看 tablet 版"的诉求；setMode API 不破坏。

新增 computed: \`isTablet\`, \`isNarrow = isMobile.value || isTablet.value\`。

### 现有 isMobile 调用点审计

| 文件 | 调用 | tablet 时行为 | 决定 |
|---|---|---|---|
| \`views/public/SearchView.vue\` | 2 处, 控制"mobile 竖排单列" vs "网格" | 落入"非 mobile"分支，走网格 (768px 已够放多列) | **符合预期, 不改** |

无需迁移到 \`isNarrow\`。

### Sidebar 三变体 (按 mode)

| Mode | 形态 | 触发交互 |
|---|---|---|
| mobile | drawer (75% 宽 max 280px, 外侧遮罩) | 顶部汉堡按钮点开, 点遮罩/菜单项关闭 |
| tablet | icon-rail (60px 常驻, 只图标 + hover tooltip) | 直接点图标切页 |
| desktop | full (现有 220px 完整菜单) | 不变 |

### 表格 mobile (ManageTable)

新增 prop \`mobileVariant: card | collapse | scroll\` (默认 \`card\`)。

**card 模式渲染规则**：
- 渲染容器从 \`<table>\` 改为 \`<div class="gf-card-list">\`
- 每行 → 一张卡片，结构：
  - 标题行：取 \`cols\` 数组第一个 col 的 cell 值（如果调用方提供了 \`#cell-<key>\` slot，则用 slot 渲染结果）
  - meta 行：剩余 cols 的 cell 值，按 \`<label>: <value>\` 形式列出 (label = col.title)
  - actions 行：现有 \`#actions\` slot 不变
- 排序 / 选择框 (\`selectable\`) 在 card 模式 fallback：选择框移到卡片左上角；排序触发隐藏 (无表头)
- 若调用方传了 \`#mobile-card\` slot，则**完全覆盖**默认渲染，传入 \`{ row, index }\`

**collapse / scroll** 后续阶段实现，MVP 只做 card。

### 弹窗 / 表单 (ManageSheet)

**新组件，API 显式不猜：**

```ts
props: {
  modelValue: boolean
  title?: string
  // 显式选择, 不再 auto 数字段
  desktopMode?: modal       // 当前只支持 modal, 预留扩展
  mobileMode?: sheet | fullsheet  // 默认 sheet
}
slots: { default, footer? }
emits: [update:modelValue, close]
```

**实际形态**：
- desktop / tablet → 居中 modal (复用现有 modal 视觉)
- mobile + \`mobileMode=sheet\` → 底部 Sheet (max-height 75vh, grabber, 下拉关)
- mobile + \`mobileMode=fullsheet\` → 全屏 Sheet (顶部 ← 返回 / 保存 / title)

**选用规则 (调用方判断, 写进组件 JSDoc)**：
- 字段 ≤5 或纯确认型：\`mobileMode=sheet\`
- 字段 >5 或含富文本 / 多步：\`mobileMode=fullsheet\`

### 触摸目标

严格 **WCAG 44×44 pt** 下限。**仅在 \`isNarrow\` (mobile + tablet) 触发**。
- 按钮 \`min-height: 44px\`
- 表格行 / 卡片可点击区 ≥44px 高
- 图标按钮 hit area 44×44 (visual size 仍可 24×24, 用 padding 撑)
- desktop 行为不变 (避免破坏现有节奏)

### 实现策略 (混合)

| 类型 | 走哪条路 |
|---|---|
| 大行为/状态 (抽屉开关、表格↔卡片、sheet/modal) | JS (useViewMode) |
| 样式/排版 (padding、字号、栅格、按钮宽度) | UnoCSS 断点 (\`md:\`/\`lg:\` 前缀) |
| 能 CSS 解决就别加 JS | 默认规则 |

## 现有 modal 调用清点 (用于 P2 sizing)

实测命中模式 (\`<Teleport / <dialog / v-model.*(visible|show|open)\`):

| 文件 | modal 数 |
|---|---|
| CollectListView.vue | 1 |
| CronListView.vue | 1 |
| FilmClassView.vue | 1 |
| 其他 7 个 manage 页 | 0 |
| **合计** | **3** |

P2 工作量 = 3 个调用方迁移，**远比 spec 上一版估的 5-10 小**。

## 组件清单 (改动)

| 文件 | 改动 |
|---|---|
| \`src/composables/useViewMode.ts\` | 扩展 auto 分支为三档, 新增 isTablet/isNarrow |
| \`src/components/layout/ManageLayout.vue\` | 去硬编码 data-mode, 接入 useViewMode, 加 drawerOpen 状态 |
| \`src/components/layout/ManageHeader.vue\` | 加汉堡按钮 (仅 mobile 可见), emit toggle-drawer |
| \`src/components/layout/ManageSidebar.vue\` | 加 variant prop (drawer/icon-rail/full), 三套样式单组件复用菜单数据 |
| \`src/components/manage/ManageSheet.vue\` | **新增** (按上述显式 API) |
| \`src/components/manage/ManageTable.vue\` | 加 mobileVariant prop + card 渲染分支 |
| 3 个 manage 页 (Collect/Cron/FilmClass) | modal 调用迁移到 ManageSheet |
| 所有 manage 页 | 按需指定 ManageTable.mobileVariant (默认 card 不用动) |

## 迁移阶段

| 阶段 | 内容 | 独立 ship |
|---|---|---|
| P0 | useViewMode 扩展 + ManageLayout 接入 + ManageHeader 汉堡按钮 | yes (desktop 不变) |
| P1 | ManageSidebar 三变体 | yes (mobile/tablet 立刻可用) |
| P2 | ManageSheet + 迁移 3 个 modal | yes (所有弹窗 mobile 化) |
| P3 | ManageTable 加 mobileVariant=card 默认 | yes (所有表格 mobile 化) |
| P4 | 触摸目标、字号、间距零散调整 | yes |
| P5 | Playwright 3 视口 smoke 测试 | yes (测试目录) |

## 测试

- 自动 (新增): \`client-v2/tests/manage-responsive.spec.ts\`, 3 viewport (375x667 / 768x1024 / 1280x800), 10 路由 = 30 截屏, 人工 review。playwright config 已存在 (\`client-v2/playwright.config.ts\`)
- 手动: iOS Safari + Android Chrome 各一遍

## 风险

| 风险 | 缓解 |
|---|---|
| ManageTable 改造破坏现有 desktop | 默认行为 100% 兼容, card 路径仅在 isMobile 触发 |
| ManageSheet 迁移漏 props (z-index/自定义 footer) | 第一波先迁 1 个 (FilmClassView), 验证 API 后再迁剩 2 个 |
| 抽屉边缘滑动手势 | MVP 不做, 只点遮罩关; 后期视需要加 |

## YAGNI 排除

- 用户主动切 desktop/mobile 开关 (useViewMode 已有 localStorage)
- Sidebar 折叠状态持久化 (tablet/desktop 都固定)
- 暗色/亮色主题切换 (项目目前只有 dark)
- managebg.png CDN URL (改 CSS 渐变, 不依赖外部资源 — 已在 950951e 提交完成)
- ManageSheet 自动数字段定 sheet/fullsheet (改为调用方显式 mobileMode prop)
- ManageTable collapse / scroll mobileVariant 实现 (P0-P5 只做 card 默认; 未来需要再加)
