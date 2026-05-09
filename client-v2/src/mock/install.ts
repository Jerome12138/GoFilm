/**
 * Mock 适配器：把 axios 请求短路到本地数据
 *
 * 启用条件：`import.meta.env.DEV && import.meta.env.VITE_USE_MOCK`
 *
 * 实现：覆盖 axios 实例的 adapter，根据 url 路径派发到 handlers，
 * 返回构造的 AxiosResponse（响应拦截器仍会剥包装/写 token），
 * 下游业务无感知。
 */

import type {
  AxiosAdapter,
  AxiosResponse,
  InternalAxiosRequestConfig
} from 'axios'
import { http } from '@/api/http'
import { dispatch } from './handlers'

interface MockResult {
  /** 业务包装的 data，会被响应拦截器剥到 body.data */
  data: unknown
  status?: number
  statusText?: string
  headers?: Record<string, string>
  /** 模拟延迟（ms） */
  delay?: number
}

function buildResponse(
  config: InternalAxiosRequestConfig,
  result: MockResult
): AxiosResponse {
  return {
    data: {
      code: 0,
      msg: 'ok',
      success: true,
      data: result.data
    },
    status: result.status ?? 200,
    statusText: result.statusText ?? 'OK',
    headers: result.headers ?? {},
    config,
    request: {}
  }
}

const mockAdapter: AxiosAdapter = async (config: InternalAxiosRequestConfig) => {
  const url = (config.url ?? '').replace(/^\/+/, '/')
  const method = (config.method ?? 'get').toLowerCase()
  const data =
    typeof config.data === 'string'
      ? safeParse(config.data)
      : config.data ?? {}
  const params = config.params ?? {}

  const handlerResult = dispatch({ url, method, data, params })

  if (!handlerResult) {
    // 找不到匹配的 mock：返回 404 让业务侧错误处理
    return Promise.reject({
      response: {
        status: 404,
        statusText: 'Mock Not Found',
        data: { code: 404, msg: `mock 未匹配 ${method.toUpperCase()} ${url}` },
        config,
        headers: {}
      },
      config,
      request: {},
      message: 'Mock Not Found'
    })
  }

  const delay = handlerResult.delay ?? 60
  if (delay > 0) {
    await new Promise((r) => setTimeout(r, delay))
  }
  return buildResponse(config, handlerResult)
}

function safeParse(s: string): unknown {
  try {
    return JSON.parse(s)
  } catch {
    return s
  }
}

/** 在 main.ts 中按需调用 */
export function installMockAdapter(): void {
  http.defaults.adapter = mockAdapter
  // eslint-disable-next-line no-console
  console.info(
    '%c[mock]%c axios adapter installed (VITE_USE_MOCK=1)',
    'color:#9b49e7;font-weight:bold',
    'color:inherit'
  )
}
