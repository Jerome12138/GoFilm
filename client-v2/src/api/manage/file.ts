import { http } from '../http'
import type { PhotoWallResp } from '@/types/manage'

/**
 * GET /api/manage/file/list 文件列表
 * 后端返回 `{ list, page: { total, current, pageSize, pageCount } }`
 */
export const list = (params: { current?: number }): Promise<PhotoWallResp> =>
  http.get<unknown, PhotoWallResp>('/manage/file/list', { params })

/** GET /api/manage/file/del 删除文件（id 后端 ParseUint） */
export const remove = (id: string | number): Promise<void> =>
  http.get<unknown, void>('/manage/file/del', { params: { id } })

/**
 * POST /api/manage/file/upload 单文件上传
 * 后端 `Success(link, ...)` 直接返回字符串 URL
 */
export const upload = (
  form: FormData,
  onProgress?: (progress: number) => void
): Promise<string> =>
  http.post<unknown, string>('/manage/file/upload', form, {
    headers: { 'Content-Type': 'multipart/form-data' },
    onUploadProgress: (e) => {
      if (onProgress && e.total) {
        onProgress(e.loaded / e.total)
      }
    }
  })
