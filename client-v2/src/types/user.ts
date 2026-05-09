/** 登录请求体（后端 json 标签 userName，Go 默认大小写不敏感） */
export interface LoginPayload {
  userName: string
  password: string
}

/** 后端用户信息（沿用旧站结构） */
export interface UserInfo {
  id?: number
  uid?: string
  userName?: string
  username?: string
  email?: string
  gender?: number
  nickname?: string
  nickName?: string
  avatar?: string
  status?: number
  role?: string
}

/** 修改密码 payload（后端读 params["password"] / params["newPassword"]） */
export interface ChangePasswordPayload {
  password: string
  newPassword: string
}
