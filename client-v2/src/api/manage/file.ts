import { http } from '../http'
import type { PaginationResp } from '@/types/api'
import type { FileItem } from '@/types/manage'

/** GET /api/manage/file/list 文件列表（分页） */
export const list = (params: {
  current?: number
  size?: number
  type?: string
}): Promise<PaginationResp<FileItem>> =>
  http.get<unknown, PaginationResp<FileItem>>('/manage/file/list', { params })

/** GET /api/manage/file/del 删除文件 */
export const remove = (id: string): Promise<void> =>
  http.get<unknown, void>('/manage/file/del', { params: { id } })

/**
 * POST /api/manage/file/upload 文件上传
 * @param onProgress 上传进度回调（0-1）
 */
export const upload = (
  form: FormData,
  onProgress?: (progress: number) => void
): Promise<FileItem> =>
  http.post<unknown, FileItem>('/manage/file/upload', form, {
    headers: { 'Content-Type': 'multipart/form-data' },
    onUploadProgress: (e) => {
      if (onProgress && e.total) {
        onProgress(e.loaded / e.total)
      }
    }
  })
