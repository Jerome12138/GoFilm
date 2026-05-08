# TV 端 + Capacitor 增量补丁

> 本文件是对 PRD / 视觉规范 / 架构文档的增量补丁，说明电视端适配与 Android TV 打包细节。
> 适用于 `client-v2/` 单一代码库，TV 模式与 mobile/desktop 共享业务代码。

## 1. 三模式（mode）

| mode | 触发条件 | 入口 |
|---|---|---|
| `mobile` | 视口宽 < 768 且 hover 无 / pointer 粗 | 默认 |
| `desktop` | 视口宽 768–1919 且支持 hover / 鼠标 | 默认 |
| `tv` | UA 命中 `SmartTV / Tizen / WebOS / HbbTV / Hisense / MiTV / Android TV / AFT[A-Z]+`<br>OR `?mode=tv` 查询参数<br>OR `localStorage['gf-mode'] === 'tv'`<br>OR 视口 ≥ 1920 且 `(hover: none)` | 自动 + 手动切换 |

实现：`src/composables/useViewMode.ts` 返回 `mode: Ref<'mobile'|'desktop'|'tv'>`，根 `<html data-mode="tv">` 切换。
`useUIStore` 持久化用户手动选择到 `localStorage`。

## 2. TV 设计 Token 增量（基于 design-tokens.md）

`src/assets/styles/theme.css` 内 `[data-mode="tv"] :root` 段：

```css
[data-mode="tv"] :root {
  /* TV 安全区 */
  --gf-tv-safe: 48px;

  /* 字号放大（基础 +25%） */
  --gf-fs-xs: 0.875rem;   /* 14 */
  --gf-fs-sm: 1rem;       /* 16 */
  --gf-fs-base: 1.25rem;  /* 20 */
  --gf-fs-md: 1.5rem;     /* 24 */
  --gf-fs-lg: 1.75rem;    /* 28 */
  --gf-fs-xl: 2rem;       /* 32 */
  --gf-fs-2xl: 2.5rem;    /* 40 */
  --gf-fs-3xl: 3.25rem;   /* 52 */
  --gf-fs-hero: clamp(3.5rem, 5vw + 1rem, 6rem);

  /* 间距放大 */
  --gf-space-4: 24px;
  --gf-space-6: 32px;
  --gf-space-8: 48px;
  --gf-space-12: 72px;

  /* 容器：TV 走 1600，居中 */
  --gf-container-max-2xl: 1600px;

  /* 焦点环（TV 专用） */
  --gf-tv-focus-ring: 0 0 0 4px var(--gf-brand-cyan), 0 24px 60px rgba(0,0,0,0.75);
  --gf-tv-focus-scale: 1.06;
}
```

页面外层留白：`padding-block: var(--gf-tv-safe)`，左右 `padding-inline: var(--gf-tv-safe)`。

## 3. 焦点系统

### 3.1 全局 focus 样式
```css
[data-mode="tv"] :focus-visible,
[data-mode="tv"] [data-focus="true"] {
  outline: none;
  transform: scale(var(--gf-tv-focus-scale));
  box-shadow: var(--gf-tv-focus-ring);
  z-index: 5;
  transition: transform var(--gf-dur-fast) var(--gf-ease-spring),
              box-shadow var(--gf-dur-fast) var(--gf-ease-standard);
}
```

### 3.2 空间导航
- 引入：自研 `useSpatialNavigation()` composable（轻量，约 200 行），不引入 norigin-spatial-navigation（CommonJS 与 Vite 兼容差）
- 规则：方向键在 `[data-spatial-row]` / `[data-spatial-grid]` 容器内查询所有 `[data-focusable]` 元素，根据几何位置（`getBoundingClientRect`）选择最近邻
- 进入 Hero / Row 时自动聚焦第一项；切换路由时记忆上次焦点
- `Enter`/`Space` 触发 click，`Escape`/`Backspace`（KeyCode 4）走 `router.back()`

### 3.3 D-pad 桥接（Web 端事件映射）
`src/utils/dpad.ts`：
```ts
const KEY_MAP: Record<number, string> = {
  19: 'ArrowUp',   // KEYCODE_DPAD_UP
  20: 'ArrowDown',
  21: 'ArrowLeft',
  22: 'ArrowRight',
  23: 'Enter',     // KEYCODE_DPAD_CENTER
  66: 'Enter',     // KEYCODE_ENTER
  4:  'Escape',    // KEYCODE_BACK
  82: 'ContextMenu', // KEYCODE_MENU
}
```
监听 keydown，遇到 keyCode 命中则 `e.preventDefault()` 并派发标准 KeyboardEvent。

### 3.4 必须 focusable 的元素
所有 `<a>`, `<button>`, `<input>`, `FilmCard`, `EpisodeChip`, `FilmFilterTag`, `Pagination`, `BaseDialog 关闭按钮` 加 `data-focusable="true"`，并默认 `tabindex="0"`。

## 4. TV 模式下组件行为差异

| 组件 | desktop/mobile | tv 行为 |
|---|---|---|
| `FilmCard` | hover 放大 + 显示 overlay | 默认显示标题与剧情简短信息，focus 替代 hover；卡片宽放大至 220–260px |
| `FilmRow` | 拖拽 / 按钮滚动 | 方向键左右滚，超出视口时自动 `scrollIntoView({inline:'center', behavior:'smooth'})` |
| `HeroCarousel` | 自动 3s 切换 + 按钮 | 自动 6s 切换，遥控器 OK 进入详情，方向键左右切，自动播放可暂停 |
| `PublicHeader` | 透明 → 滚动后实色 | 默认置顶大尺寸（高度 96），导航项放大 |
| `SearchBar` | 文字输入 | TV 上展开虚拟键盘提示（保留原生输入），点击后弹出搜索抽屉 |
| `Player` | 鼠标控件 | OK 暂停/播放，左右快进/快退 10s，上下音量，菜单键打开选集面板 |
| `Pagination` | 鼠标点击 | 方向键 + Enter，焦点环 |
| 后台 `/manage/*` | 全功能 | TV 模式不进入，需 desktop。强制路由守卫：`if (mode==='tv' && to.path.startsWith('/manage')) → /index` |

## 5. 三套布局规则（断点 × mode）

`src/components/layout/PublicLayout.vue` 内：
- `mode === 'tv'`：固定 1920 设计稿基线，安全区 padding 48，导航高度 96
- `mode === 'desktop'`：弹性容器 1280–1600，导航高度 64
- `mode === 'mobile'`：100% 流式，导航高度 48 + 底栏 56

## 6. Capacitor 打包步骤

### 6.1 依赖
```
pnpm add @capacitor/core @capacitor/android
pnpm add -D @capacitor/cli
```

### 6.2 配置（`capacitor.config.ts`）
```ts
import type { CapacitorConfig } from '@capacitor/cli'
const config: CapacitorConfig = {
  appId: 'com.gofilm.app',
  appName: 'GoFilm',
  webDir: 'dist',
  server: { androidScheme: 'https' },
  android: {
    allowMixedContent: true,         // 视频源可能 http
    backgroundColor: '#0b0b0fff'
  },
  plugins: {
    SplashScreen: {
      launchShowDuration: 800,
      backgroundColor: '#0b0b0f',
      androidScaleType: 'CENTER_CROP'
    }
  }
}
export default config
```

### 6.3 Android TV 适配
执行 `npx cap add android` 生成 `android/` 工程，编辑：
- `android/app/src/main/AndroidManifest.xml` 加：
  ```xml
  <uses-feature android:name="android.software.leanback" android:required="false" />
  <uses-feature android:name="android.hardware.touchscreen" android:required="false" />
  <application
    android:banner="@drawable/banner"
    android:isGame="false"
    android:hardwareAccelerated="true">
    <activity ...>
      <intent-filter>
        <action android:name="android.intent.action.MAIN" />
        <category android:name="android.intent.category.LAUNCHER" />
        <category android:name="android.intent.category.LEANBACK_LAUNCHER" />
      </intent-filter>
    </activity>
  </application>
  ```
- 提供 banner 图标：`android/app/src/main/res/drawable-xhdpi/banner.png`（320×180）
- `android:targetSdkVersion 34`，最低 `21`

### 6.4 D-pad keycode → web keyboard 桥
方案 A（最简）：什么都不做，Android WebView 默认会把 D-pad 当方向键派发给 web，焦点系统直接接住。仅需关闭原生焦点拦截。

方案 B（精细）：自定义 `MainActivity`，重写 `dispatchKeyEvent`：
```kotlin
override fun dispatchKeyEvent(event: KeyEvent): Boolean {
  if (event.action == KeyEvent.ACTION_DOWN) {
    val js = when (event.keyCode) {
      KeyEvent.KEYCODE_DPAD_CENTER, KeyEvent.KEYCODE_ENTER ->
        "window.dispatchEvent(new KeyboardEvent('keydown',{key:'Enter',keyCode:13,bubbles:true}))"
      KeyEvent.KEYCODE_BACK ->
        "if(window.gfTvBack&&window.gfTvBack())true; else null"
      else -> null
    }
    if (js != null) {
      bridge.webView.evaluateJavascript(js, null)
      return true
    }
  }
  return super.dispatchKeyEvent(event)
}
```

### 6.5 构建
```
pnpm build           # 出 dist/
npx cap sync android
npx cap open android # 在 Android Studio 编译 / 签名 APK
```

### 6.6 启动 URL 强制 TV 模式
`AndroidManifest.xml` 启动 Activity 加 `android:launchMode="singleTask"` 并通过 `onCreate` 注入 hash：
```kotlin
bridge.webView.loadUrl("file:///android_asset/public/index.html#/?mode=tv")
```
或在前端 `useViewMode` 中检测 `window.Capacitor?.platform === 'android'` 自动判定 TV。

## 7. 不做的事
- 不打 iOS / tvOS（用户未要求）
- 不集成 ExoPlayer（首版 video.js 走 WebView 即可）
- 不做 4K 资源（全部 1080p 等比缩放）
- 不做语音搜索 / 遥控器麦克风
