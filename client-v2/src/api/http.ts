import axios, {
  AxiosError,
  type AxiosInstance,
  type AxiosResponse,
  type InternalAxiosRequestConfig
} from 'axios'
import type { ApiResp } from '@/types/api'
import { BizError } from '@/types/api'
import { logger } from '@/utils/logger'
import { useUserStore } from '@/stores/user'
import { useUIStore } from '@/stores/ui'

/**
 * axios 扩展 config：silent 跳过全局 loading 与 toast
 * __pushed: 内部标记，记录该请求已 pushLoading，确保响应阶段必 pop
 */
declare module 'axios' {
  export interface AxiosRequestConfig {
    silent?: boolean
  }
  export interface InternalAxiosRequestConfig {
    silent?: boolean
    __pushed?: boolean
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

/** 暴露给路由守卫等模块的全局 toast 入口 */
export function toast(type: ToastType, msg: string): void {
  if (toastApi) {
    toastApi.push(type, msg)
  }
}

/** axios 实例 */
export const http: AxiosInstance = axios.create({
  baseURL: import.meta.env.VITE_API_BASE || '/api',
  timeout: 80_000,
  headers: { 'Content-Type': 'application/json' }
})

function safePop(config: InternalAxiosRequestConfig | undefined): void {
  if (!config?.__pushed) return
  try {
    useUIStore().popLoading()
  } catch {
    /* ignore — pinia 未就绪场景 */
  }
  config.__pushed = false
}

/** ===== 请求拦截器 ===== */
http.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    try {
      const userStore = useUserStore()
      if (userStore.token) {
        config.headers.set('Authorization', `Bearer ${userStore.token}`)
      }
      if (!config.silent) {
        useUIStore().pushLoading()
        config.__pushed = true
      }
    } catch (e) {
      logger.warn('http request interceptor pre-init', e)
    }

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
 */
http.interceptors.response.use(
  ((resp: AxiosResponse<ApiResp<unknown>>) => {
    safePop(resp.config as InternalAxiosRequestConfig)

    try {
      const newToken = resp.headers['new-token']
      if (typeof newToken === 'string' && newToken.length > 0) {
        useUserStore().setToken(newToken)
      }
    } catch (e) {
      logger.warn('new-token write failed', e)
    }

    const body = resp.data
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
    safePop(error.config as InternalAxiosRequestConfig | undefined)
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
      useUserStore().clearAuth()
      const { default: router } = await import('@/router')
      const cur = router.currentRoute.value
      if (cur.path !== '/login' && !silent) {
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
      toastError(msg ?? '权限不足，仅管理员可操作')
    }
    return
  }

  if (!silent) {
    toastError(msg ?? '服务器繁忙，请稍后再试')
  }
}
