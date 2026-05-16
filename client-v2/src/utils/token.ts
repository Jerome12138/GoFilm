/**
 * Token 持久化.
 *
 * 协议:
 *  - 服务端登录返回 body.token (LoginResult.token), 前端写入 localStorage
 *  - 请求时 http 拦截器以 `Authorization: Bearer <token>` 注入
 *  - 已废弃: 旧版 { key: 'auth-token', value }+`auth-token` 头方案
 */

const STORAGE_KEY = 'auth-token'

/** 写入 token; 传空串视为清除. */
export function setToken(value: string): void {
  try {
    if (value) {
      localStorage.setItem(STORAGE_KEY, value)
    } else {
      localStorage.removeItem(STORAGE_KEY)
    }
  } catch {
    /* 隐私模式 / 配额满, 忽略 */
  }
}

/** 读取 token; 不存在返回空串. */
export function getToken(): string {
  try {
    return localStorage.getItem(STORAGE_KEY) ?? ''
  } catch {
    return ''
  }
}

/** 清除 token. */
export function clearToken(): void {
  try {
    localStorage.removeItem(STORAGE_KEY)
  } catch {
    /* ignore */
  }
}
