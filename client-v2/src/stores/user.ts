import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import type {
  ChangePasswordPayload,
  LoginPayload,
  UserInfo
} from '@/types/user'
import { clearToken, getToken, setToken } from '@/utils/token'

/**
 * 用户态 store：token、个人信息、登录/登出/改密
 *
 * 关键约定（参见 doc/handover/user-api-frontend-guide.md）：
 *  - 登录走 POST /user/login，token 由响应拦截器从 new-token 头写入
 *  - 登录后立即拉 GET /user/info 获取 role（0 普通 / 1 管理员）
 *  - isAdmin 决定后台入口与路由守卫
 *  - 401 由 http 拦截器自行清 token + redirect
 */
export const useUserStore = defineStore('user', () => {
  const token = ref<string>(getToken()?.value ?? '')
  const info = ref<UserInfo | null>(null)

  const isLoggedIn = computed(() => token.value.length > 0)

  /** 是否管理员（role==1） */
  const isAdmin = computed(() => Number(info.value?.role ?? 0) === 1)

  /** 显示用昵称：nickName > userName > username > 默认 */
  const displayName = computed(() => {
    const i = info.value
    return i?.nickName || i?.userName || i?.username || '用户'
  })

  function setTokenValue(t: string): void {
    token.value = t
    if (t) {
      setToken(t)
    } else {
      clearToken()
    }
  }

  /** 登录：兼容传 username 或 userName 字段；成功后自动拉 user info */
  async function login(
    payload: LoginPayload | { username: string; password: string }
  ): Promise<UserInfo> {
    const { login: doLogin } = await import('@/api/auth')
    const userName =
      'userName' in payload ? payload.userName : (payload as { username: string }).username
    const data = await doLogin({ userName, password: payload.password })
    // doLogin 可能直接返回 UserInfo（新接口），但旧路径 /login 兼容时返回 void。
    // 无论如何 token 已由 new-token 头写入，再拉一次 info 拿到准确 role。
    let me: UserInfo | null = data ?? null
    try {
      me = await fetchInfo()
    } catch {
      // 极端情况下 /user/info 失败，退化为 doLogin 返回值（可能没有 role）
    }
    return me ?? data ?? ({} as UserInfo)
  }

  async function fetchInfo(): Promise<UserInfo> {
    const { getUserInfo } = await import('@/api/auth')
    const res = await getUserInfo()
    info.value = res
    return res
  }

  async function changePassword(payload: ChangePasswordPayload): Promise<void> {
    const { changePassword: doChange } = await import('@/api/auth')
    await doChange(payload)
  }

  async function logout(): Promise<void> {
    try {
      const { logout: doLogout } = await import('@/api/auth')
      await doLogout()
    } catch {
      // 无论后端是否成功，前端都清 token
    } finally {
      info.value = null
      setTokenValue('')
    }
  }

  /** http 拦截器在 401 时调用，避免循环依赖 */
  function clearAuth(): void {
    info.value = null
    setTokenValue('')
  }

  return {
    token,
    info,
    isLoggedIn,
    isAdmin,
    displayName,
    login,
    fetchInfo,
    changePassword,
    setToken: setTokenValue,
    logout,
    clearAuth
  }
})
