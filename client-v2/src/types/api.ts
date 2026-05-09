/**
 * 通用 API 响应包装
 *
 * 后端历史上有两种成功标记：
 *  - code === 0
 *  - code === '200'
 * 响应拦截器会做双兼容并直接返回 data。
 */
export interface ApiResp<T> {
  code: number | string
  msg?: string
  success?: boolean
  data: T
}

/** 通用分页响应 */
export interface PaginationResp<T> {
  list: T[]
  total: number
  current: number
  size: number
  pages?: number
}

/** 拦截器抛出的业务错误 */
export class BizError extends Error {
  public readonly code: number | string
  public readonly data?: unknown
  constructor(payload: { code: number | string; msg?: string; data?: unknown } | undefined) {
    super(payload?.msg ?? '请求失败')
    this.name = 'BizError'
    this.code = payload?.code ?? -1
    this.data = payload?.data
  }
}
