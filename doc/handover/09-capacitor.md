# 09 前端交接 — STORY-015（Capacitor 打包 Android TV APK）

> 阶段：Web TV 模式产物打包为 Android / Android TV APK
> 完成时间：2026-05-09
> 工作目录：`D:/Git/GoFilm/client-v2/`
> 前置文档：
> - `doc/handover/00-context.md`
> - `doc/handover/04-tv-addendum.md`（第 6 节 Capacitor 步骤）
> - `doc/handover/07-tv-adapt.md`（TV 焦点系统 / 组件适配，已具备）

---

## 1. 交付清单

### 新增依赖（client-v2/package.json）
- `@capacitor/core@^8.3.2`
- `@capacitor/android@^8.3.2`
- `@capacitor/splash-screen@^8.0.1`
- 开发依赖：`@capacitor/cli@^8.3.2`

### 新增 / 改动文件
| 路径 | 类型 | 说明 |
|---|---|---|
| `client-v2/capacitor.config.ts` | 新增 | Capacitor 主配置（appId、webDir、SplashScreen、allowMixedContent） |
| `client-v2/package.json` | 改动 | 新增 `cap:sync` / `cap:open` / `cap:build` 三个脚本 |
| `client-v2/src/composables/useViewMode.ts` | 改动 | 增加 `detectCapacitorAndroid()`，原生壳默认 TV 模式 |
| `client-v2/src/App.vue` | 改动 | 注册 `window.gfTvBack`：BACK 键交给 vue-router 处理 |
| `client-v2/android/` | 新增（cap add android 生成） | 完整 Android 工程 |
| `android/app/src/main/AndroidManifest.xml` | 改写 | leanback feature / banner / hardwareAccelerated / LEANBACK_LAUNCHER / cleartext / 网络权限 |
| `android/app/src/main/java/com/gofilm/app/MainActivity.java` | 改写 | dispatchKeyEvent：DPAD_CENTER → web Enter，BACK → 调 window.gfTvBack |
| `android/app/src/main/res/drawable-xhdpi/banner.png` | 新增（占位） | 复制自旧项目 managebg.png，运营时应替换 |
| `android/app/src/main/res/drawable/banner.png` | 新增（占位） | 兜底密度 |

### 是否成功生成 android/ 工程
**成功**。本机已安装 Capacitor CLI 所需 Node 23.3.0；执行 `npx cap add android` 全程无报错，
`cap sync android` 也已成功把 dist 推到 `android/app/src/main/assets/public/`。
完整的 Gradle 工程（含 `gradlew`、`build.gradle`、`settings.gradle`、`gradle.properties`）已就位。

---

## 2. 关键配置摘录

### 2.1 capacitor.config.ts
```ts
const config: CapacitorConfig = {
  appId: 'com.gofilm.app',
  appName: 'GoFilm',
  webDir: 'dist',
  server: { androidScheme: 'https' },
  android: {
    allowMixedContent: true,           // 视频源可能为 http
    backgroundColor: '#0b0b0fff'
  },
  plugins: {
    SplashScreen: {
      launchShowDuration: 800,
      backgroundColor: '#0b0b0f',
      androidScaleType: 'CENTER_CROP',
      showSpinner: false,
      splashFullScreen: true,
      splashImmersive: true
    }
  }
}
```

### 2.2 AndroidManifest 关键片段
```xml
<uses-feature android:name="android.software.leanback" android:required="false" />
<uses-feature android:name="android.hardware.touchscreen" android:required="false" />

<application
    android:banner="@drawable/banner"
    android:hardwareAccelerated="true"
    android:usesCleartextTraffic="true"
    ...>
  <activity android:launchMode="singleTask" android:exported="true" ...>
    <intent-filter>
      <action android:name="android.intent.action.MAIN" />
      <category android:name="android.intent.category.LAUNCHER" />
      <category android:name="android.intent.category.LEANBACK_LAUNCHER" />
    </intent-filter>
  </activity>
</application>

<uses-permission android:name="android.permission.INTERNET" />
<uses-permission android:name="android.permission.ACCESS_NETWORK_STATE" />
```

### 2.3 MainActivity D-pad 桥
- DPAD_CENTER / ENTER → `window.dispatchEvent(new KeyboardEvent('keydown',{key:'Enter',keyCode:13,...}))`
- BACK → 异步调 `window.gfTvBack()`：返回 true 表示 vue-router 已 back，原生不退出；返回 false 走默认 onBackPressed。
- 方向键（UP/DOWN/LEFT/RIGHT）保留默认行为：Android WebView 自动派发 ArrowXxx，前端 useSpatialNavigation 直接接住。

### 2.4 useViewMode TV 触发优先级（更新后）
1. `localStorage['gf-mode'] = 'tv'`（用户手动）
2. URL `?mode=tv`
3. **`window.Capacitor.getPlatform() === 'android'`**（原生壳）
4. UA 命中 `SmartTV / Tizen / WebOS / HbbTV / Hisense / MiTV / Android TV / AFT[A-Z]+ / GoogleTV / AppleTV / BRAVIA / VIDAA`
5. 视口 ≥ 1920 且 `(hover: none)`

第 3 项是本期新增，确保 Capacitor APK 在普通 Android 平板 / 手机上启动也按 TV 模式渲染（用户可通过设置覆盖）。

### 2.5 window.gfTvBack（前端注册）
在 `App.vue` 的 setup 中注册：
```ts
window.gfTvBack = () => {
  if (window.history.length > 1) {
    router.back()
    return true
  }
  return false
}
```

---

## 3. 前置条件（构建 APK 必备）

### 软件
| 工具 | 版本 | 安装方式 |
|---|---|---|
| JDK | 17（必须）| 推荐 [Adoptium Temurin 17](https://adoptium.net/) |
| Android Studio | Hedgehog 2023.1.1 或更新 | https://developer.android.com/studio |
| Android SDK | API 34（compile） + API 21（min） | Android Studio > SDK Manager |
| Android SDK Build-Tools | 34.0.0 | 同上 |
| Gradle | 8.x（项目已携带 wrapper，无需手装） | 自动 |
| Node | ≥ 20.10 | 已具备 |
| pnpm | ≥ 9 | 已具备 |

### 环境变量
- `JAVA_HOME` 指向 JDK 17
- `ANDROID_SDK_ROOT` 或 `ANDROID_HOME` 指向 Android SDK 目录（Android Studio 装好会自动设）
- 把 `%ANDROID_SDK_ROOT%/platform-tools` 加入 PATH（用 adb 安装）

### TV 模拟器
Android Studio > Device Manager > Create Device：
- 类别选 **TV**
- 设备型号：Android TV (1080p)
- System image：API Level **30+**（Android 11 TV）

---

## 4. 标准构建流程

### 4.1 首次构建（开发版 APK，调试签名）
```powershell
cd D:\Git\GoFilm\client-v2

# 1. 同步 web 资源到 android 工程
pnpm cap:sync

# 2. 在 Android Studio 打开
pnpm cap:open

# 3. 在 Android Studio 内：
#    - 选择 TV 模拟器
#    - 点击 Run 'app'（绿色三角）
```

或命令行直接构建 debug APK：
```powershell
cd D:\Git\GoFilm\client-v2\android
.\gradlew.bat assembleDebug
# 产物：android/app/build/outputs/apk/debug/app-debug.apk
```

通过 adb 安装到 TV 设备：
```powershell
adb connect <TV_IP>:5555
adb install -r android\app\build\outputs\apk\debug\app-debug.apk
```

### 4.2 增量开发：改前端 → 重新打包
```powershell
cd D:\Git\GoFilm\client-v2
pnpm cap:build      # = vite build && cap sync && cap copy
# 然后在 Android Studio Run；或命令行 .\gradlew.bat assembleDebug
```

`cap copy` 已包含在 sync 里，但保留 build 脚本以便分步排查。

---

## 5. 签名 APK（发布版）

### 5.1 生成 keystore（一次性）
```powershell
cd D:\Git\GoFilm\client-v2\android\app

keytool -genkey -v -keystore gofilm-release.keystore -alias gofilm -keyalg RSA -keysize 2048 -validity 10000
# 按提示填入密码、组织信息
# 产物：android/app/gofilm-release.keystore
```

**重要**：keystore 文件 + 两个密码必须妥善保管。**丢失后无法再发布同一应用的更新版本**。

### 5.2 配置 signingConfig
编辑 `android/app/build.gradle`，在 `android { ... }` 块内追加：
```gradle
android {
    // ... existing fields ...

    signingConfigs {
        release {
            storeFile file('gofilm-release.keystore')
            storePassword System.getenv('GOFILM_KEYSTORE_PWD') ?: 'YOUR_STORE_PWD'
            keyAlias 'gofilm'
            keyPassword System.getenv('GOFILM_KEY_PWD') ?: 'YOUR_KEY_PWD'
        }
    }

    buildTypes {
        release {
            signingConfig signingConfigs.release
            minifyEnabled false   // WebView 应用，关掉 R8 避免误删 capacitor 反射类
            shrinkResources false
            proguardFiles getDefaultProguardFile('proguard-android-optimize.txt'), 'proguard-rules.pro'
        }
    }
}
```

**安全建议**：把密码放进环境变量或 `~/.gradle/gradle.properties`，不要提交到 git。
在 `.gitignore` 中已自动忽略 `*.keystore`，但仍建议二次确认。

### 5.3 构建 release APK
```powershell
cd D:\Git\GoFilm\client-v2\android
.\gradlew.bat assembleRelease
# 产物：android/app/build/outputs/apk/release/app-release.apk
```

或构建 AAB（Google Play 上架格式）：
```powershell
.\gradlew.bat bundleRelease
# 产物：android/app/build/outputs/bundle/release/app-release.aab
```

### 5.4 验证签名
```powershell
cd D:\Git\GoFilm\client-v2\android
.\gradlew.bat signingReport
# 或
keytool -list -v -keystore app\gofilm-release.keystore
```

---

## 6. 替换 banner 图标

### 6.1 规格
- Android TV 大图标 banner.png：**320 × 180** PNG，TV-Safe Area 内文字 / logo 居中
- 普通 launcher 图标（手机 / 平板）：mipmap-* 已有 ic_launcher，运营时可单独替换

### 6.2 替换路径
```
android/app/src/main/res/drawable-xhdpi/banner.png      ← 主用
android/app/src/main/res/drawable/banner.png             ← 兜底
```

当前是占位（用旧项目 `client/src/assets/image/managebg.png` 复制），尺寸不严格符合 320×180，**正式发布前必须替换**。
推荐设计稿：暗色背景 + GoFilm 品牌色（紫品红渐变 `#9b49e7→#4ad1e5`）+ 居中文字 LOGO。

---

## 7. 已知限制 / 风险

| 序号 | 项 | 说明 |
|---|---|---|
| 1 | iOS / tvOS | **不打包**（需求未要求；Capacitor iOS 需 macOS + Xcode）|
| 2 | 4K 视频 | 不专门压制 4K 资源，全部 1080p 等比上采样；如需 4K 评估 ExoPlayer 桥 |
| 3 | 视频源混合内容 | `usesCleartextTraffic="true"` + `allowMixedContent: true` 已开启，第三方 http 源可播 |
| 4 | banner 图占位 | 当前是 managebg.png，**发布前必须替换** |
| 5 | 签名 keystore | 当前未生成，需运维侧执行 6.1 步骤 |
| 6 | minifyEnabled | release 关闭 R8（WebView 应用通常不必开启；开启会触发 capacitor / splash-screen 反射误删风险），需要瘦身时再单独评估 keep 规则 |
| 7 | TV 模拟器音视频 | Android TV 模拟器不带硬解，HLS 解码可能卡顿，**真机验证为准** |
| 8 | 后台 /manage 路由 | 已在 Vue-Router 守卫中：TV 模式自动跳 /index，不在 APK 中暴露管理界面 |

---

## 8. 验证产物

### 8.1 前端构建
```
pnpm exec vue-tsc -p tsconfig.app.json --noEmit   # ✓
pnpm build                                         # ✓ 16.27s
```

### 8.2 cap sync
```
√ Copying web assets from dist to android\app\src\main\assets\public in 124ms
√ Creating capacitor.config.json in android\app\src\main\assets in 3ms
√ Updating Android plugins in 42ms
[info] Found 1 Capacitor plugin for android:
       @capacitor/splash-screen@8.0.1
[info] Sync finished in 0.746s
```

### 8.3 Android 工程结构
```
android/
├── app/
│   ├── build.gradle
│   ├── capacitor.build.gradle
│   ├── proguard-rules.pro
│   └── src/main/
│       ├── AndroidManifest.xml             ← 已加 leanback / banner / hwAccel
│       ├── assets/public/                  ← cap sync 推入的 dist 镜像
│       ├── java/com/gofilm/app/MainActivity.java   ← D-pad 桥
│       └── res/
│           ├── drawable-xhdpi/banner.png   ← 占位 banner
│           ├── drawable/banner.png         ← 兜底密度
│           └── mipmap-*/ic_launcher.png    ← Capacitor 默认图标
├── build.gradle
├── settings.gradle
├── gradle.properties
├── gradlew / gradlew.bat
└── variables.gradle
```

---

## 9. 给下游 / 运维的 checklist

发布前：
- [ ] 替换 `banner.png` 为 320×180 品牌图
- [ ] 生成 release keystore，配置 `signingConfigs.release`
- [ ] 在 TV 真机（Android TV 设备 / 小米盒子 / Chromecast with Google TV）跑一轮焦点 + 视频回归
- [ ] 检查视频源混合内容播放（http HLS）
- [ ] 检查 BACK 键路由回退；首页按 BACK 退出 Activity
- [ ] 上架 Google Play 或侧加载 APK 到 TV

发版本：
- [ ] 改 `android/app/build.gradle` 的 `versionCode` (+1) / `versionName`（语义版本）
- [ ] `pnpm cap:build`
- [ ] `gradlew bundleRelease` 出 AAB

---

## 10. 文档自检（与任务清单对应）

| 任务 | 状态 |
|---|---|
| 1. 安装 capacitor 依赖 | ✓ |
| 2. capacitor.config.ts 配置 | ✓（按 04-tv-addendum 6.2） |
| 3. 初始化 android/ 工程 | ✓（`cap add android` 成功） |
| 4. AndroidManifest leanback / banner / 网络权限 | ✓（按 04-tv-addendum 6.3） |
| 5. MainActivity D-pad 桥 | ✓（dispatchKeyEvent + window.gfTvBack） |
| 6. banner 占位 | ✓（drawable-xhdpi 与 drawable 各一份） |
| 7. package.json `cap:sync` / `cap:open` / `cap:build` | ✓ |
| 8. useViewMode 检测 Capacitor 平台自动 TV | ✓ |
| 9. 交付文档 | ✓（本文件） |
| 10. `pnpm build` 不被破坏 | ✓（16.27s） |

下一阶段（STORY-016 之后，如要做）：
- 接入 capacitor-android-tv 社区插件做更精细的遥控器事件
- video.js 控件层接管 TV 焦点（自定义控件层或 plugin）
- 4K 资源链路（仅当业务需要）
