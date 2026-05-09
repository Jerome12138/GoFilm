import { http } from '../http'
import type { UserInfo } from '@/types/user'

/** GET /api/manage/user/info 当前登录用户信息 */
export const info = (): Promise<UserInfo> =>
  http.get<unknown, UserInfo>('/manage/user/info')
