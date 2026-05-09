import type { CapacitorConfig } from '@capacitor/cli'

/**
 * GoFilm Capacitor 配置（Android TV 打包）
 *
 * - 用户端 Vue3 SPA 编译产物 dist/ 作为 webDir
 * - allowMixedContent 打开：第三方采集源可能为 http
 * - 包名 com.gofilm.app（Android TV Leanback Launcher 启动）
 * - SplashScreen 短暂展示主背景色（#0b0b0f）后即进 WebView
 */
const config: CapacitorConfig = {
  appId: 'com.gofilm.app',
  appName: 'GoFilm',
  webDir: 'dist',
  server: {
    androidScheme: 'https'
  },
  android: {
    allowMixedContent: true,
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

export default config
