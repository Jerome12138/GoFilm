# UX 交接文档（设计 → 前端）

> 给 `vue3-frontend` 的交接说明。前置文档：`doc/handover/00-context.md`、`doc/design/design-tokens.md`、`doc/design/components-spec.md`、`doc/design/page-mockups.md`。

---

## 1. 视觉基调（一句话）

Netflix / Disney+ 风格暗色影视站：黑底（`#0b0b0f`）+ Netflix 红主 CTA（`#E50914`）+ 紫品红渐变（`#9b49e7→#4ad1e5`，继承旧站，用于 Logo / 后台 / 登录强调）。

---

## 2. 设计 Token 一览表

### 颜色

| 类别 | Token | 值 |
|---|---|---|
| 背景底 | `--gf-bg-base` | `#0b0b0f` |
| 卡片底 | `--gf-bg-surface` | `#141518` |
| 浮起卡 | `--gf-bg-elevated` | `#1c1d22` |
| 玻璃 | `--gf-bg-glass` | `rgba(20,21,24,0.55)` + blur(18px) |
| 主文 | `--gf-text-primary` | `#FFFFFF` |
| 次文 | `--gf-text-secondary` | `rgba(255,255,255,0.78)` |
| 弱文 | `--gf-text-muted` | `rgba(255,255,255,0.55)` |
| 主品牌 | `--gf-brand-primary` | `#E50914`（Netflix 红） |
| 渐变 | `--gf-brand-gradient` | `linear-gradient(135deg, #9b49e7 0%, #4ad1e5 100%)` |
| 链接 | `--gf-text-link` | `#4ad1e5` |
| 成功 / 警告 / 错误 | `#22c55e` / `#f59e0b` / `#ef4444` | |

### 字体

| Token | 值 |
|---|---|
| 字体栈 | Inter + PingFang SC + Microsoft YaHei + system-ui |
| `--gf-fs-base` | 16px（用户端正文最低值） |
| `--gf-fs-md` | 18px（卡片标题 / 表单） |
| `--gf-fs-lg` | 20px（区块标题 / 按钮） |
| `--gf-fs-xl` | 24px（页面副标题） |
| `--gf-fs-2xl` | 30px（详情页主标题） |
| `--gf-fs-hero` | `clamp(2.5rem, 4vw + 1rem, 4.5rem)`（Hero 标题） |
| 行高 | `--gf-lh-tight` 1.15 / `--gf-lh-snug` 1.3 / `--gf-lh-normal` 1.5 / `--gf-lh-relaxed` 1.7 |

### 间距（4-pt）

`4 / 8 / 12 / 16 / 20 / 24 / 32 / 40 / 48 / 64`，对应 `--gf-space-1 ~ --gf-space-16`。

### 圆角

`sm 4 / md 8 / lg 12 / xl 20 / 2xl 28 / full 9999`。

### 阴影

`sm` 输入框 / `md` 默认卡片 / `lg` 浮起 / `xl` 弹层 / `hover` 卡片 hover 提升 / `brand-glow` 主 CTA 光晕 / `purple-glow` 渐变光晕 / `focus-ring` 表单焦点。

### 断点

`sm 360 / md 768 / lg 1024 / xl 1440 / 2xl 1920`。
容器最大宽：`xl: 1280` / `2xl: 1600`，居中显示。

### 动画

- 时长：`fast 150ms / base 250ms / slow 400ms / page 500ms`
- 缓动：`standard / out / in / spring（卡片 hover）/ linear`
- 必须实现：页面切换淡入、卡片 hover 1.08 放大、AppHeader 滚动后实色

---

## 3. 五条必须遵守的视觉守则

### 守则 1：不允许写死颜色 / 字号 / 间距

任何 hex / rgba / px 数值都必须取自 CSS 变量（或对应 UnoCSS / Tailwind preset）。
错误：`color: #fff; padding: 16px;`
正确：`color: var(--gf-text-primary); padding: var(--gf-space-4);`

如发现 Token 不够用 → 不要私加魔法数字，先在 `design-tokens.md` 提议补充。

---

### 守则 2：移动优先 + 全断点验证

每个组件 / 页面必须在以下五档逐一过：
- 360（小屏手机）
- 768（平板竖屏）
- 1024（小屏桌面）
- 1440（标准桌面）
- 1920（大屏 / 4K 缩放）

不允许出现"桌面好看，移动错位"的情况。媒体查询统一使用 mobile-first（`min-width`）。
组件规范文档已给出每个组件在 4 档下的列数 / 高度 / 行为，**直接照抄实现**。

---

### 守则 3：触控目标最小 44×44px

所有可点击元素（按钮 / 图标 / Tab / 集数格 / 分页 chip）的命中区不得小于 44×44px。
视觉尺寸可以小（如 36px chip），但必须通过外层 padding 把命中区扩大到 44px。
不要依赖 `:hover` 实现关键交互（移动端没有 hover），所有 hover 必须有 `:active` 或 `:focus-visible` 等价反馈。

---

### 守则 4：暗色主题对比度与可读性

- 正文 `--gf-text-primary` 与 `--gf-bg-base` 对比度 ≥ 15:1（已满足 WCAG AAA）
- 次文 `--gf-text-secondary` ≥ 7:1（AAA），`--gf-text-muted` ≥ 4.5:1（AA）
- 不得在背景图上直接放白字（必须叠加蒙版 `--gf-mask-hero-bottom` / `--gf-bg-overlay`）
- 不得用纯灰边框分隔卡片，使用阴影 `--gf-shadow-md` 区分
- Focus ring 必须可见（`--gf-shadow-focus-ring`），勿全局 `outline: none`

---

### 守则 5：Netflix 红 vs 紫品红渐变的使用边界

避免红与紫渐变互相打架。规则：

| 场景 | 用 Netflix 红 `--gf-brand-primary` | 用紫品红渐变 `--gf-brand-gradient` |
|---|---|---|
| 用户端"立即播放" CTA | ✓ | ✗ |
| 详情页主操作 | ✓ | ✗ |
| Hero CTA | ✓ | ✗ |
| Logo 文字 | ✗ | ✓ |
| AppHeader 导航 active 下划线 | ✗ | ✓ |
| 集数格 active | ✗ | ✓ |
| 后台 Sidebar active | ✗ | ✓ |
| 后台数据强调 / 进度条 | ✗ | ✓ |
| 登录页 CTA | ✗ | ✓ |
| 分页 active chip | ✗ | ✓ |

简记：**用户端核心播放动作 = 红，品牌识别 / 后台 / 登录 = 渐变**。同一屏幕禁止两种强调色同时出现在主操作位。

---

## 4. 实现路径建议

1. 在 `client-v2/src/styles/tokens.css` 中粘贴 `design-tokens.md` 第 9 节"完整 CSS 变量清单"
2. UnoCSS / Tailwind preset 配置中将颜色 / 字号 / 间距 / 圆角 / 阴影映射到 CSS 变量
3. 在 `App.vue` 上设置 `body { background: var(--gf-bg-base); color: var(--gf-text-primary); font-family: var(--gf-font-sans); }`
4. 优先实现底层组件（Button / Input / FilmCard / FilmRow），再组装页面
5. 全局 `prefers-reduced-motion` 媒体查询：>200ms 动画降级为 80ms，禁用 scale 放大

---

## 5. 资产复用

继承自旧项目（来自 `00-context.md`）：
- iconfont 图标库（保留）
- play.png / 404.png（empty / loading 配图可继续用）
- managebg.png（登录页背景兜底）

---

## 6. 设计开发协作

- 实现中如发现规范缺失或矛盾，**先在本文档评论 / 提 issue，不要自行裁决**
- 新组件需求请补充到 `components-spec.md` 后再实现
- 像素级还原以 `components-spec.md` 中的尺寸表为准；页面整体结构以 `page-mockups.md` 为准
