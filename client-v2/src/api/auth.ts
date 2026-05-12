import { http } from './http'
import type {
  ChangePasswordPayload,
  CreateUserPayload,
  LoginPayload,
  ManageUserListResp,
  UserInfo
} from '@/types/user'

/**
 * POST /api/user/login 登录
 * - 兼容旧路径 /login
 * - 后端实际响应: `{code:0, data:null, msg:"登录成功!!!"}`，token 由响应头 new-token 写入
 *   故返回类型为 void；登录后请显式 GET /user/info 拉取用户资料（含 role）
 * - 失败：拦截器抛 BizError
 */
export const login = (data: LoginPayload): Promise<void> =>
  http.post<unknown, void>('/user/login', data)

/** GET /api/user/logout 退出登录 */
export const logout = (): Promise<void> =>
  http.get<unknown, void>('/user/logout')

/** POST /api/user/changePassword 修改密码 */
export const changePassword = (data: ChangePasswordPayload): Promise<void> =>
  http.post<unknown, void>('/user/changePassword', data)

/**
 * GET /api/user/info 当前登录用户信息（普通用户 + 管理员都可调用）
 * 返回 UserInfo，含 role 字段
 */
export const getUserInfo = (): Promise<UserInfo> =>
  http.get<unknown, UserInfo>('/user/info')

/* ============== 管理员后台用户管理 ============== */

/** POST /api/manage/user/create 创建用户（仅管理员） */
export const createUser = (data: CreateUserPayload): Promise<UserInfo> =>
  http.post<unknown, UserInfo>('/manage/user/create', data)

/** GET /api/manage/user/list 用户列表（仅管理员） */
export const listUsers = (params?: {
  current?: number
  pageSize?: number
}): Promise<ManageUserListResp> =>
  http.get<unknown, ManageUserListResp>('/manage/user/list', { params })
