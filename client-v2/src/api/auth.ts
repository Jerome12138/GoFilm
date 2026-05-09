import { http } from './http'
import type {
  ChangePasswordPayload,
  LoginPayload,
  UserInfo
} from '@/types/user'

/** POST /api/login 登录（成功后响应头 new-token 由拦截器写入） */
export const login = (data: LoginPayload): Promise<void> =>
  http.post<unknown, void>('/login', data)

/** GET /api/logout 退出登录 */
export const logout = (): Promise<void> =>
  http.get<unknown, void>('/logout')

/** POST /api/changePassword 修改密码 */
export const changePassword = (data: ChangePasswordPayload): Promise<void> =>
  http.post<unknown, void>('/changePassword', data)

/** GET /api/manage/user/info 当前登录用户信息 */
export const getUserInfo = (): Promise<UserInfo> =>
  http.get<unknown, UserInfo>('/manage/user/info')
