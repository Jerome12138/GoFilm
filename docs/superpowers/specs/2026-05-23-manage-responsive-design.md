# Manage 后台响应式设计 spec

> 日期: 2026-05-23 · 状态: draft

## 背景

当前 `ManageLayout.vue` 模板硬编码 `data-mode="desktop"`，sidebar + main 横向 flex 在 mobile 完全不可用。
项目已有 `useViewMode` (mobile/desktop/tv) 和 `PublicLayout` 的响应式实现，**manage 区是唯一没接入响应式的部分**。

## 目标

让所有 `/manage/*` 路由在 mobile (\<768px) / tablet (768-1024) / desktop (\>=1024) 三档下都可用，
关键操作（登录、用户管理、影片管理、文件管理、采集、Cron、系统配置）触摸友好。

## 设计决定

### 断点

```ts
// uno.config.ts breakpoints (覆盖默认)
{ sm: 640px, md: 768px, lg: 1024px, xl: 1280px }
```

useViewMode 扩展为四态：

| 视口 | mode |
|---|---|
| < 768 | mobile |
| 768-1023 | tablet |
| >= 1024 | desktop |
| (TV 触发条件命中) | tv |

新增 computed: `isNarrow = isMobile || isTablet`。**现有调用 \`mode === mobile\` 不破坏**（tablet 不归 mobile）。

### Sidebar 三变体

| Mode | 形态 | 触发交互 |
|---|---|---|
| mobile | drawer (75% 宽 max 280px, 外侧遮罩) | 顶部汉堡按钮点开, 点遮罩/菜单项关闭 |
| tablet | icon-rail (60px 常驻, 只图标 + hover tooltip) | 直接点图标切页 |
| desktop | full (现有 220px 完整菜单) | 不变 |

### 表格 mobile

ManageTable 默认 `mobileVariant: card`。
- **card**: 每行 → 卡片，第一列做标题、其他做 meta、actions slot 不变。智能 fallback 无需逐页配置
- **collapse**: 高密度页面 (如 Cron 任务列表) 覆盖，只显示 1-2 主列 + 点击展开
- **scroll**: 边缘场景兜底（确实不想改造的页面）

### 弹窗 / 表单

**新组件 ManageSheet.vue** 替换所有自写 modal：

```ts
props: {
  modelValue: boolean
  title?: string
  variant?: auto | modal | sheet | fullsheet
}
```

variant=auto 时：
- desktop → modal (居中弹窗)
- mobile + 字段 <=5 → sheet (底部 75% 升起)
- mobile + 字段 >5 → fullsheet (全屏 + 顶部 ← 返回/保存)

### 触摸目标

严格 **WCAG 44x44 pt** 下限。在 mobile mode 下：
- 所有按钮 min-height 44px (--gf-space-11)
- 表格行可点击区域 \>=44px 高
- 图标按钮 hit area 44x44 (visual size 可保持 24x24)

### 实现策略 (混合)

| 类型 | 走哪条路 |
|---|---|
| 大行为/状态 (抽屉开关、表格↔卡片、sheet/modal) | JS (useViewMode) |
| 样式/排版 (padding、字号、栅格、按钮宽度) | UnoCSS 断点 (md:/lg:) |
| 能 CSS 解决就别加 JS | 默认规则 |

## 组件清单 (改动)

| 文件 | 改动 |
|---|---|
| \`src/composables/useViewMode.ts\` | 扩展 tablet 档, 新增 isTablet/isNarrow |
| \`src/components/layout/ManageLayout.vue\` | 去硬编码 data-mode, 接入 useViewMode, 加 drawerOpen 状态 |
| \`src/components/layout/ManageHeader.vue\` | 加汉堡按钮 (仅 mobile 可见), emit toggle-drawer |
| \`src/components/layout/ManageSidebar.vue\` | 加 variant prop (drawer/icon-rail/full), 三套样式单组件复用菜单数据 |
| \`src/components/manage/ManageSheet.vue\` | **新增**, 封装 modal/sheet/fullsheet 三态 |
| \`src/components/manage/ManageTable.vue\` | 加 mobileVariant prop + 卡片渲染分支 |
| 各 manage 页面 (10 个) | 弹窗调用迁移到 ManageSheet, 表格按需指定 mobileVariant |
| \`uno.config.ts\` | 显式 breakpoints |

## 迁移阶段

| 阶段 | 内容 | 独立 ship |
|---|---|---|
| P0 | useViewMode 扩展 + ManageLayout 接入 + 汉堡按钮 | yes (desktop 行为不变) |
| P1 | ManageSidebar 三变体 | yes (mobile/tablet 立刻可用菜单) |
| P2 | ManageSheet + 迁移 5-10 个 modal 调用方 | yes (所有弹窗 mobile 化) |
| P3 | ManageTable mobileVariant + 智能 fallback | yes (所有表格 mobile 化) |
| P4 | 触摸目标、字号、间距零散调整 | yes |
| P5 | Playwright 3 视口 smoke 测试 | (测试目录) |

## 测试

- 自动: \`tests/manage-responsive.spec.ts\`, 3 viewport (375x667 / 768x1024 / 1280x800), 10 路由 = 30 截屏, 人工 review
- 手动: iOS Safari + Android Chrome 各跑一遍

## 风险

| 风险 | 缓解 |
|---|---|
| ManageTable 改造破坏现有 desktop | 默认行为 100% 兼容, mobile 路径仅在 isMobile 触发 |
| ManageSheet 替换可能漏 props | 第一波只迁 3 个简单弹窗, 验证 API 后再批量 |
| 抽屉边缘滑动手势 | MVP 不做, 只点遮罩关; 后期视需要加 |

## YAGNI 排除

- 用户主动切 desktop/mobile 开关 (useViewMode 已有 localStorage)
- Sidebar 折叠状态持久化 (tablet/desktop 都固定)
- 暗色/亮色主题切换 (项目目前只有 dark)
- managebg.png CDN URL (改为 CSS 渐变, 不依赖外部资源)
