import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import type { LoginPayload, UserInfo, ChangePasswordPayload } from '@/types/user'
import { clearToken, getToken, setToken } from '@/utils/token'

/**
 * 用户态 store：token、个人信息、登录/登出/改密
 * - token 持久化兼容旧 client（localStorage key=auth）
 * - 不直接调 axios，所有请求走 api/auth.ts
 */
export const useUserStore = defineStore('user', () => {
  const token = ref<string>(getToken()?.value ?? '')
  const info = ref<UserInfo | null>(null)

  const isLoggedIn = computed(() => token.value.length > 0)

  function setTokenValue(t: string): void {
    token.value = t
    if (t) {
      setToken(t)
    } else {
      clearToken()
    }
  }

  async function login(payload: LoginPayload): Promise<void> {
    const { login: doLogin } = await import('@/api/auth')
    await doLogin(payload)
    // token 由响应拦截器从 new-token 头写入
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

  return {
    token,
    info,
    isLoggedIn,
    login,
    fetchInfo,
    changePassword,
    setToken: setTokenValue,
    logout
  }
})
