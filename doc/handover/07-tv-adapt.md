# 07 前端交接 — STORY-013 + STORY-014（TV 焦点系统 + 组件适配）

> 阶段：TV 模式空间导航与组件行为差异化
> 完成时间：2026-05-09
> 工作目录：`D:/Git/GoFilm/client-v2/`
> 前置文档：
> - `doc/handover/00-context.md`
> - `doc/handover/04-tv-addendum.md`（TV 设计 token + 行为基线）
> - `doc/handover/03-arch-handover.md`
> - `doc/handover/06-frontend.md` / `06-frontend-story-008-009.md` / `06-frontend-story-010.md`

---

## 1. 范围

- STORY-013：焦点系统 + 空间导航 composable + D-pad bridge + 全局焦点样式 + TV 路由守卫确认
- STORY-014：TV 模式各组件行为差异化（hover-only → focus 替代；尺寸放大；安全区缩进）

不在本任务范围：
- Capacitor Android 工程生成（`npx cap add android`）
- AndroidManifest banner / leanback 配置
- video.js TV 控件接管
- HistoryView 真实卡片渲染（视图本身仍是占位）

---

## 2. 落地文件清单

### 新增（1 个）
| 文件 | 用途 |
|---|---|
| `src/composables/useSpatialNavigation.ts` | 空间导航 composable，TV 模式下接管方向键 / Enter / Escape |

### 改动
| 文件 | 改动 |
|---|---|
| `src/utils/dpad.ts` | installDpadBridge 增强：capture 阶段 preventDefault；e.target 兜底 activeElement / window；keyCode 字段同步派发 |
| `src/App.vue` | setup 内调用 `installDpadBridge()` + `installSpatialNavigationOnce()` |
| `src/assets/styles/theme.css` | TV `[data-focusable="true"]` :focus + :focus-visible 双触发；禁用 TV 默认 `:focus-visible` 蓝边 |
| `src/components/film/FilmCard.vue` | TV 焦点环（cyan 4px + scale 1.06 + 阴影）；标题 / 浮层标题字号上调 |
| `src/components/film/FilmRow.vue` | TV 默认显示箭头 64px；卡片间距 +50%（24px / sp-6）；container 安全区缩进；focusin → inline:'center' scroll |
| `src/components/film/HeroCarousel.vue` | TV 高 75vh；信息区安全区 padding；CTA "立即播放"自动获焦；箭头加大；dot 加大 |
| `src/components/film/EpisodeTabs.vue` | TV 集数 chip 高 64px / 字号 base；source-tab 高 64px；focus 环；gap sp-4 |
| `src/components/film/FilmFilterBar.vue` | TV chip 高 44px / 字号 base；focus 环 |
| `src/components/base/BasePagination.vue` | TV chip 56×56；focus 环 |
| `src/components/base/BaseDialog.vue` | TV 关闭按钮 56×56；focus 环 |
| `src/components/layout/PublicHeader.vue` | TV 模式历史用 `BaseDialog`（800px 抽屉）替代 hover 浮层；nav-link / brand / icon-btn / search-input 全部 data-focusable + tabindex；TV 焦点环 |
| `src/components/layout/PublicLayout.vue` | TV 模式 `.container-page` 容器宽度 1600 居中 + tv-safe padding |

### 已存在不动
- `src/composables/useViewMode.ts`
- `src/router/guards.ts`（TV 禁止 `/manage` 守卫已在前期完成）
- `src/views/public/PlayView.vue`（D-pad 快捷键 + TV 安全区 padding 已在 STORY-010 完成）

---

## 3. 焦点系统数据流

```
keydown (D-pad keyCode 19-23 / Esc / Enter)
  │
  ├─ installDpadBridge (capture, App.vue 安装)
  │    若 e.key 已是标准 key → 跳过
  │    否则 preventDefault + 派发新事件 (key='ArrowUp/.../Enter/Escape')
  │
  └─ useSpatialNavigation handler (capture 阶段，TV 时启用)
       ├─ 输入框聚焦：仅 Escape 拦截做 blur
       ├─ ArrowXxx → findNearest(currentFocusable, dir) → focus + scrollIntoView center
       ├─ Enter / Space (非 button/a/input) → currentFocusable.click()
       └─ Escape / Backspace → router.back() | router.push('/index')

route.fullPath 变化
  ├─ rememberFocus(prev): sessionStorage.setItem('gf-tv-focus:'+prev, stableId)
  └─ setTimeout 50ms restoreFocus(next):
       sessionStorage.getItem → 找元素 → focus，否则 focusFirst()
```

### 元素稳定 ID 规则（按优先级）
1. `#${el.id}`
2. `name:${name}` 或 `tag|attr=value|...`（data-key / data-id / data-href / href / aria-label）
3. `idx:${index}`（在所有可见 focusable 列表中的位置）

### data-focusable 标记位置
现已加 `data-focusable="true" tabindex="0"` 的元素（按 SFC）：

| SFC | 元素 |
|---|---|
| `BaseButton` | `<button>`（disabled/loading 时移除） |
| `BaseDialog` | 关闭按钮 |
| `BasePagination` | 页码 chip / prev / next |
| `FilmCard` | RouterLink 卡片 |
| `FilmRow` | "更多" 链接 / 左右箭头 |
| `HeroCarousel` | 左右箭头 / 圆点指示器 |
| `EpisodeTabs` | 播放源 tab / 集数 chip |
| `FilmFilterBar` | 筛选 chip |
| `PublicHeader` | brand / nav-link / 搜索 input / 图标按钮 / 历史项 |
| `PlayView` | 自动连播 / 下一集（BaseButton 自带）/ EpisodeTabs |

---

## 4. 关键决策

### 4.1 空间导航算法（自研，未引入 norigin-spatial-navigation）
- 评分：主轴距离 + 副轴偏移 × 0.5
- 完全错位（副轴偏移 > 元素自身尺寸）加 1.5 倍惩罚
- 候选必须严格在方向之外（`r.bottom <= cur.top` 等），不允许重叠
- 没有候选时不动焦点，不会自动绕回（防止"上"键时跳到底部干扰用户预期）

### 4.2 D-pad bridge 始终安装
非 TV 模式下因 keyCode 19/20/21/22/23 不会触发，install 安全。空间导航由 `watch(isTV)` 控制 enable/disable，不会在桌面拦截方向键。

### 4.3 焦点记忆按 fullPath 隔离
`sessionStorage` 而非 `localStorage`：会话级生命周期，关闭浏览器 / Capacitor 应用时自动清空，不污染下次启动。

### 4.4 :focus + :focus-visible 双覆盖
Chromium 对程序化派发的 keydown 不一定标记 keyboard 模态（trust 状态），仅写 `:focus-visible` 会丢样式。在 `[data-mode='tv']` 范围内同时写 `:focus` 与 `:focus-visible`，双保险。

### 4.5 hover-only 行为全部用 focus 替代
- FilmCard：TV 卡片标题 + 蒙版默认显示（mask opacity 0.65 / hover-info opacity 1）
- HeroCarousel：箭头默认显示（不靠 hover）
- FilmRow：箭头默认显示（不靠 hover）
- PublicHeader：历史浮层在 TV 下走 Dialog（不靠 hover），桌面保留 hover 浮层

### 4.6 容器居中 1600 而非 1920
`--gf-container-max-2xl: 1600px` 已在 04-tv-addendum 定义，PublicLayout 内 `[data-mode='tv'] .container-page { max-width: var(--gf-container-max-2xl) }` 让所有 view 自动居中，左右各预留约 160px 安全区。

---

## 5. 测试清单

### 5.1 编译
```
cd client-v2
pnpm exec vue-tsc -p tsconfig.app.json --noEmit   # ✓
pnpm build                                         # ✓ 14.66s
```

### 5.2 浏览器手动（开发者工具）
```js
// 切到 TV 模式
localStorage.setItem('gf-mode','tv'); location.reload()
```
- [x] `<html data-mode="tv">`
- [x] 字号显著放大（基础 16 → 20px，标题 → 32px）
- [x] 1600 居中容器，左右安全区 48px
- [x] HeroCarousel 自动 6s 切换；CTA "立即播放" 默认获焦
- [x] 方向键在卡片间移动焦点，焦点元素 scale 1.06 + cyan 4px ring
- [x] Enter 在卡片上触发跳转详情页
- [x] Escape 返回上一页；history 为空时回 /index
- [x] 访问 `/manage/index` 自动 redirect 到 `/index`
- [x] 头部历史按钮：点击弹出 Dialog（不再是 hover 浮层）
- [x] FilmRow 焦点元素自动滚到行内居中

```js
// 退出 TV 模式
localStorage.removeItem('gf-mode'); location.reload()
```
- [x] 桌面交互完全不变（hover 行为恢复，字号还原）

### 5.3 Capacitor / Android TV（待 STORY-016 接入）
本期未产出 Android 工程，但已为 D-pad keyCode 桥接做好前端准备：
- Android WebView 默认会把 D-pad 当方向键派发，Spatial Navigation 直接接住
- 若需要细粒度（如 BACK 键不退 Activity 而走 router.back），见 04-tv-addendum 6.4 方案 B（Kotlin dispatchKeyEvent）

---

## 6. 给下游开发者的提示

1. **新增交互元素**（视图层）必须加 `data-focusable="true"` + `tabindex="0"`，否则 TV 上焦点跳不过去。
2. **TV 模式样式**仍写 unscoped `<style>` 块（Vue scoped 不支持 `:global()`），与之前 STORY 一致。
3. **新增页面**默认会被空间导航接管：进入路由后 50ms restoreFocus → focusFirst。如果某个页面要自定义初始焦点，可在 `onMounted` 内调用 `useSpatialNavigation().focusElement(myRef.value)`。
4. **键盘冲突**：useSpatialNavigation handler 在 capture 阶段安装。若新组件需要拦截方向键（如自定义滑块），用 `e.stopPropagation()` 或在 input/textarea 内即可，已默认放行编辑控件。
5. **Dialog 焦点穿透**：BaseDialog 打开时 dialog 元素 focus，但内部交互元素仍走全局空间导航。若发现焦点跑到 dialog 外，需要在 BaseDialog 内部添加 focus trap（暂未发现问题，留作 follow-up）。
6. **HistoryView 真实化**时，记得把卡片的 `<a>` 或按钮加 data-focusable，不需额外写 TV 样式（FilmCard 已自带）。
7. **测试 D-pad** 不需要真机：浏览器调用 `window.dispatchEvent(new KeyboardEvent('keydown', { keyCode: 19 }))` 即可模拟 D-pad UP。

---

## 7. 已知限制 / Follow-up（不阻塞 Capacitor）

| 序号 | 问题 | 建议处理 |
|---|---|---|
| 1 | HistoryView 视图未实装卡片列表 | 后续 ticket 实装时使用 FilmCard，焦点天然支持 |
| 2 | video.js 控件 TV 焦点接管粗糙 | 自定义控件层 / video.js plugin |
| 3 | 搜索 input 在 TV 上无虚拟键盘提示 | Capacitor Android TV 会弹原生 IME，无需前端处理 |
| 4 | HeroCarousel 自动 focus 时机依赖 setTimeout 250ms | 后续可改 watch banner.length>0 |
| 5 | TV 模式 BaseDialog 无显式 focus trap | 实测无穿透问题；如需可加 inert 兜底 |
| 6 | 空间导航不支持 wrap-around（边界回绕） | 当前设计就是不绕回；如需可加 fallback |
| 7 | TV 模式下 prefers-reduced-motion 不影响 focus 动画 | theme.css 已全局降级 transition-duration |

---

## 8. 验证产物

```
dist/assets/index-b4TqnVto.js         37.07 KB / gzip 13.69 KB
dist/assets/HomeView-Ckj3Ci-I.js       7.67 KB
dist/assets/PlayView-BYBa9d6-.js      11.18 KB
dist/assets/FilmDetailView-BGFeYK9f.js 6.39 KB
dist/assets/index-YLaLw_uM.css        39.43 KB
build time                            14.66s
```

新增 `useSpatialNavigation.ts` ~340 行；其他改动多为样式块追加，无显著 JS 体积影响。
