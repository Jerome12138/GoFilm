/**
 * Token 持久化 —— 兼容旧 client 实现
 * localStorage key: `auth`
 * 结构: { key: 'auth-token', value: '<token>' }
 *
 * 注意：旧站会把 token 直接当 header key 注入（headers[key] = value）。
 * 新站固定 key 为 'auth-token'，保持双向兼容。
 */

const STORAGE_KEY = 'auth'
const DEFAULT_HEADER_KEY = 'auth-token'

export interface AuthTokenStruct {
  key: string
  value: string
}

/** 写入 token */
export function setToken(value: string, headerKey: string = DEFAULT_HEADER_KEY): void {
  const auth: AuthTokenStruct = { key: headerKey, value }
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(auth))
  } catch {
    // 隐私模式 / 配额满，忽略
  }
}

/** 读取 token；不存在返回 null */
export function getToken(): AuthTokenStruct | null {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) {
      return null
    }
    const parsed = JSON.parse(raw) as Partial<AuthTokenStruct>
    if (typeof parsed.key === 'string' && typeof parsed.value === 'string') {
      return { key: parsed.key, value: parsed.value }
    }
    return null
  } catch {
    return null
  }
}

/** 清除 token */
export function clearToken(): void {
  try {
    localStorage.removeItem(STORAGE_KEY)
  } catch {
    // ignore
  }
}
