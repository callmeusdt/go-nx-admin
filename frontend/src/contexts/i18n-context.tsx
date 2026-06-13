import React, { createContext, useContext, useState, useCallback } from 'react'

type Locale = 'zh-CN' | 'en-US'

const zhCN: Record<string, string> = {
  'login.title': 'NX Admin',
  'login.subtitle': '请输入账号密码登录',
  'login.username': '用户名',
  'login.password': '密码',
  'login.submit': '登录',
  'login.loading': '登录中...',
  'login.error': '验证码校验失败',
  'login.network_error': '网络错误',
  'dashboard.title': '仪表盘',
  'dashboard.users': '用户总数',
  'dashboard.online': '在线用户',
  'dashboard.roles': '角色数量',
  'dashboard.logins': '今日登录',
  'users.title': '用户管理',
  'roles.title': '角色管理',
  'menus.title': '菜单管理',
  'apis.title': 'API管理',
  'media.title': '媒体管理',
  'system_config.title': '系统配置',
  'audit_logs.title': '操作日志',
  'login_logs.title': '登录日志',
  'captcha.title': '安全验证',
  'captcha.hint': '拖动滑块使拼图对准缺口',
  'captcha.refresh': '换一张',
  'captcha.cancel': '取消',
  'captcha.confirm': '确认',
  'common.create': '新增',
  'common.edit': '编辑',
  'common.delete': '删除',
  'common.save': '保存',
  'common.cancel': '取消',
  'common.search': '搜索',
  'common.loading': '加载中...',
  'common.no_data': '暂无数据',
  'common.actions': '操作',
  'common.confirm_delete': '确认删除',
  'common.upload': '上传文件',
  'common.export': '导出',
  'sidebar.dashboard': '仪表盘',
  'sidebar.backend': '后台管理',
  'sidebar.system': '系统管理',
  'sidebar.users': '用户管理',
  'sidebar.roles': '角色管理',
  'sidebar.menus': '菜单管理',
  'sidebar.apis': 'API管理',
  'sidebar.login_logs': '登录日志',
  'sidebar.audit_logs': '操作日志',
  'sidebar.media': '媒体管理',
  'sidebar.system_config': '系统配置',
}

const enUS: Record<string, string> = {
  'login.title': 'NX Admin',
  'login.subtitle': 'Please sign in to continue',
  'login.username': 'Username',
  'login.password': 'Password',
  'login.submit': 'Sign In',
  'login.loading': 'Signing in...',
  'login.error': 'CAPTCHA verification failed',
  'login.network_error': 'Network error',
  'dashboard.title': 'Dashboard',
  'dashboard.users': 'Total Users',
  'dashboard.online': 'Online Users',
  'dashboard.roles': 'Total Roles',
  'dashboard.logins': 'Today Logins',
  'users.title': 'Users',
  'roles.title': 'Roles',
  'menus.title': 'Menus',
  'apis.title': 'APIs',
  'media.title': 'Media',
  'system_config.title': 'Settings',
  'audit_logs.title': 'Audit Logs',
  'login_logs.title': 'Login Logs',
  'captcha.title': 'Security Verification',
  'captcha.hint': 'Drag the slider to align the puzzle piece',
  'captcha.refresh': 'Refresh',
  'captcha.cancel': 'Cancel',
  'captcha.confirm': 'Confirm',
  'common.create': 'Create',
  'common.edit': 'Edit',
  'common.delete': 'Delete',
  'common.save': 'Save',
  'common.cancel': 'Cancel',
  'common.search': 'Search',
  'common.loading': 'Loading...',
  'common.no_data': 'No data',
  'common.actions': 'Actions',
  'common.confirm_delete': 'Confirm Delete',
  'common.upload': 'Upload',
  'common.export': 'Export',
  'sidebar.dashboard': 'Dashboard',
  'sidebar.backend': 'Backend',
  'sidebar.system': 'System',
  'sidebar.users': 'Users',
  'sidebar.roles': 'Roles',
  'sidebar.menus': 'Menus',
  'sidebar.apis': 'APIs',
  'sidebar.login_logs': 'Login Logs',
  'sidebar.audit_logs': 'Audit Logs',
  'sidebar.media': 'Media',
  'sidebar.system_config': 'Settings',
}

const I18nContext = createContext<{
  locale: Locale
  setLocale: (locale: Locale) => void
  t: (key: string) => string
} | null>(null)

export const useI18n = () => {
  const ctx = useContext(I18nContext)
  if (!ctx) throw new Error('useI18n must be used within I18nProvider')
  return ctx
}

function getDict(locale: Locale): Record<string, string> {
  return locale === 'zh-CN' ? zhCN : enUS
}

export const I18nProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [locale, setLocale] = useState<Locale>(() => {
    const saved = localStorage.getItem('locale')
    return (saved === 'en-US' ? 'en-US' : 'zh-CN') as Locale
  })

  const updateLocale = useCallback((next: Locale) => {
    localStorage.setItem('locale', next)
    setLocale(next)
  }, [])

  const t = useCallback((key: string): string => {
    return getDict(locale)[key] || key
  }, [locale])

  return <I18nContext.Provider value={{ locale, setLocale: updateLocale, t }}>{children}</I18nContext.Provider>
}
