export interface User {
  id: number
  username: string
  status: number
  role_id: number
  role: Role
  mfa_enabled?: boolean
  created_at: string
  updated_at?: string
  is_online?: boolean
  online_ip?: string
  online_user_agent?: string
  online_login_at?: string
  online_session_count?: number
  online_sessions?: OnlineSession[]
}

export interface OnlineSession {
  id: number
  ip: string
  user_agent: string
  login_at: string
  updated_at: string
  is_current: boolean
  can_kick: boolean
}

export interface IPWhitelist {
  id: number
  user_id: number
  ip: string
  remark: string
  enabled: boolean
  created_at: string
  updated_at: string
}

export interface Role {
  id: number
  name: string
  slug: string
  is_system: boolean
}

export interface Menu {
  id: number
  parent_id: number | null
  path: string
  permission: string
  name: string
  icon: string
  sort: number
  children?: Menu[]
}

export interface AuditLog {
  id: number
  operator: string
  title: string
  module: string
  path: string
  method: string
  ip: string
  status: number
  duration: number
  error: string
  created_at: string
}

export interface ApiItem {
  id: number
  group: string
  name: string
  path: string
  method: string
  description: string
  created_at: string
}

export interface LoginLog {
  id: number
  username: string
  ip: string
  geo: string
  user_agent: string
  status: string
  message: string
  duration: number
  created_at: string
}

export interface OnlineUser {
  id: number
  user_id: number
  username: string
  role: string
  ip: string
  user_agent: string
  login_at: string
  updated_at: string
}

export interface Media {
  id: number
  original_name: string
  file_name: string
  path: string
  url: string
  ext: string
  mime_type: string
  size: number
  type: 'image' | 'file' | 'video'
  created_at: string
}

export interface SystemConfig {
  id: number
  key: string
  value: string
  group: string
  type: 'string' | 'text' | 'bool' | 'number'
  label: string
  description: string
  sort: number
}

export interface DashboardStats {
  total_users: number
  online_users: number
  total_roles: number
  today_logins: number
  today_api_calls: number
}

export interface CaptchaData {
  captcha_enabled: boolean
  captcha_id: string
  master_img: string
  tile_img: string
  tile_width: number
  tile_height: number
  tile_y: number
}
