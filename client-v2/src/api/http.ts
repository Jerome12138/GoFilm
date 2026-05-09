import axios, {
  AxiosError,
  type AxiosInstance,
  type AxiosResponse,
  type InternalAxiosRequestConfig
} from 'axios'
import type { ApiResp } from '@/types/api'
import { BizError } from '@/types/api'
import { logger } from '@/utils/logger'

/**
 * axios 扩展 config：silent 跳过全局 loading 与 toast
 */
declare module 'axios' {
  export interface AxiosRequestConfig {
    silent?: boolean
  }
  export interface InternalAxiosRequestConfig {
    silent?: boolean
  }
}

/** 全局 toast 入口（由 BaseToastContainer 注册） */
type ToastType = 'success' | 'error' | 'info' | 'warning'
interface ToastApi {
  push: (type: ToastType, msg: string) => void
}

let toastApi: ToastApi | null = null

/** 在 BaseToastContainer 挂载时由其调用 */
export function registerToast(api: ToastApi): void {
  toastApi = api
}

function toastError(msg: string): void {
  if (toastApi) {
    toastApi.push('error', msg)
  } else if (import.meta.env.DEV) {
    logger.error('[toast]', msg)
  }
}

/** axios 实例 */
export const http: AxiosInstance = axios.create({
  baseURL: import.meta.env.VITE_API_BASE || '/api',
  timeout: 80_000,
  headers: { 'Content-Type': 'application/json' }
})

/** ===== 请求拦截器 ===== */
http.interceptors.request.use(
  async (config: InternalAxiosRequestConfig) => {
    // 注入 token —— 动态 import 防止循环依赖
    try {
      const [{ useUserStore }, { useUIStore }] = await Promise.all([
        import('@/stores/user'),
        import('@/stores/ui')
      ])
      const userStore = useUserStore()
      if (userStore.token) {
        config.headers.set('auth-token', userStore.token)
      }
      if (!config.silent) {
        useUIStore().pushLoading()
      }
    } catch (e) {
      logger.warn('http request interceptor pre-init', e)
    }

    // 跳过 undefined 参数
    if (config.params && typeof config.params === 'object') {
      const cleaned: Record<string, unknown> = {}
      for (const [k, v] of Object.entries(config.params)) {
        if (v !== undefined && v !== null && v !== '') {
          cleaned[k] = v
        }
      }
      config.params = cleaned
    }
    return config
  },
  (error) => Promise.reject(error)
)

/** ===== 响应拦截器 =====
 * 注：拦截器剥离 ApiResp 包装层，直接返回 body.data。
 * 业务侧使用 `http.get<unknown, ResultDTO>(...)` 模式 → axios 类型解析为 ResultDTO。
 * 这与 axios 默认的 AxiosResponse<T> 返回值形态不一致，需要 `as any` 旁路类型签名。
 */
http.interceptors.response.use(
  (async (resp: AxiosResponse<ApiResp<unknown>>) => {
    const { useUIStore } = await import('@/stores/ui')
    const { useUserStore } = await import('@/stores/user')
    const uiStore = useUIStore()
    const userStore = useUserStore()

    if (!resp.config.silent) {
      uiStore.popLoading()
    }

    // token 续期
    const newToken = resp.headers['new-token']
    if (typeof newToken === 'string' && newToken.length > 0) {
      userStore.setToken(newToken)
    }

    const body = resp.data
    // 没有标准包装结构（后端偶发 raw 响应）
    if (!body || typeof body !== 'object' || !('code' in body)) {
      return body as unknown
    }

    const ok =
      body.code === 0 ||
      body.code === '0' ||
      body.code === 200 ||
      body.code === '200' ||
      body.success === true
    if (!ok) {
      if (!resp.config.silent) {
        toastError(body.msg ?? '请求失败')
      }
      return Promise.reject(new BizError(body))
    }
    return body.data
  }) as unknown as (resp: AxiosResponse) => Promise<AxiosResponse>,
  async (error: AxiosError<ApiResp<unknown>>) => {
    try {
      const { useUIStore } = await import('@/stores/ui')
      if (!error.config?.silent) {
        useUIStore().popLoading()
      }
    } catch {
      // ignore
    }
    await handleHttpError(error)
    return Promise.reject(error)
  }
)

/** 全局 HTTP 错误处理 */
async function handleHttpError(error: AxiosError<ApiResp<unknown>>): Promise<void> {
  const silent = error.config?.silent
  const status = error.response?.status
  const msg = error.response?.data?.msg

  if (status === 401) {
    try {
      const { useUserStore } = await import('@/stores/user')
      const { default: router } = await import('@/router')
      useUserStore().setToken('')
      const cur = router.currentRoute.value
      if (cur.path !== '/login') {
        await router.replace({
          path: '/login',
          query: { redirect: cur.fullPath }
        })
      }
    } catch (e) {
      logger.warn('401 redirect failed', e)
    }
    if (!silent) {
      toastError(msg ?? '登录已过期，请重新登录')
    }
    return
  }

  if (status === 403) {
    if (!silent) {
      toastError('无访问权限')
    }
    return
  }

  if (!silent) {
    toastError(msg ?? '服务器繁忙，请稍后再试')
  }
}
