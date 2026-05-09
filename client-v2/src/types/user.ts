/** 登录请求体 */
export interface LoginPayload {
  username: string
  password: string
}

/** 后端用户信息 */
export interface UserInfo {
  uid: string
  username: string
  nickname?: string
  avatar?: string
  role?: string
}

/** 修改密码 payload */
export interface ChangePasswordPayload {
  oldPwd: string
  newPwd: string
}
