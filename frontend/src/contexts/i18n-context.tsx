import { mediaConfigMessages } from './messages/media-config'
import { apisMessages } from './messages/apis'
import { logsMessages } from './messages/logs'
import { usersMessages } from './messages/users'
import { menusMessages } from './messages/menus'
import React, { createContext, useContext, useState, useCallback, useEffect } from 'react'
import { commonMessages } from './messages/common'
import { authMessages } from './messages/auth'
import { layoutMessages } from './messages/layout'
import { accountMessages } from './messages/account'
import { widgetsMessages } from './messages/widgets'
import { rolesMessages } from './messages/roles'
import { permissionMessages } from './messages/permissions'

export interface Language { locale: string; label: string; flag?: string }
export interface I18nOptions {
  languages?: Language[]
  messages?: Record<string, Record<string, string>>
  defaultLocale?: string
}
const defaultLanguages: Language[] = [
  { locale: 'zh-CN', label: '简体中文', flag: '🇨🇳' },
  { locale: 'en-US', label: 'English', flag: '🇺🇸' },
]

const builtinMessages: Record<string, Record<string, string>> = Object.fromEntries(
  Object.keys(commonMessages).map(locale => [locale, {
    ...commonMessages[locale], ...authMessages[locale], ...layoutMessages[locale],
    ...accountMessages[locale], ...menusMessages[locale], ...usersMessages[locale],
    ...logsMessages[locale], ...widgetsMessages[locale], ...rolesMessages[locale], ...permissionMessages[locale], ...mediaConfigMessages[locale], ...apisMessages[locale],
  }]),
)

const routeKeys: Record<string, string> = {
  '/dashboard': 'dashboard.title', '/users': 'users.title', '/roles': 'roles.title',
  '/menus': 'menus.title', '/apis': 'apis.title', '/audit-logs': 'audit_logs.title',
  '/login-logs': 'login_logs.title', '/media': 'media.title', '/system-config': 'system_config.title',
}

const I18nContext = createContext<{
  locale: string
  languages: Language[]
  setLocale: (locale: string) => void
  t: (key: string) => string
  routeLabel: (path: string, fallback: string) => string
} | null>(null)

export const useI18n = () => {
  const ctx = useContext(I18nContext)
  if (!ctx) throw new Error('useI18n must be used within I18nProvider')
  return ctx
}

function getDict(locale: string): Record<string, string> {
  return builtinMessages[locale] ?? builtinMessages['en-US']
}

export function builtinMessage(locale: string, key: string): string {
  return getDict(locale)[key] ?? key
}

export const I18nProvider: React.FC<{ children: React.ReactNode; options?: I18nOptions }> = ({ children, options }) => {
  const languages = options?.languages?.length ? options.languages : defaultLanguages
  const [locale, setLocale] = useState<string>(() => {
    const saved = localStorage.getItem('locale')
    return languages.find(item => item.locale === saved)?.locale
      ?? languages.find(item => item.locale === options?.defaultLocale)?.locale
      ?? languages[0].locale
  })

  const updateLocale = useCallback((next: string) => {
    if (!languages.some(item => item.locale === next)) return
    localStorage.setItem('locale', next)
    setLocale(next)
  }, [languages])

  const t = useCallback((key: string): string => {
    return options?.messages?.[locale]?.[key] ?? getDict(locale)[key] ?? key
  }, [locale, options?.messages])
  const routeLabel = useCallback((path: string, fallback: string) => {
    const override = options?.messages?.[locale]?.[`routes.${path}`]
    if (override !== undefined) return override
    const key = routeKeys[path] ?? (fallback === '后台管理' ? 'sidebar.backend' : fallback === '系统管理' ? 'sidebar.system' : undefined)
    return key ? t(key) : fallback
  }, [locale, options?.messages, t])
  useEffect(() => { document.documentElement.lang = locale }, [locale])

  return <I18nContext.Provider value={{ locale, languages, setLocale: updateLocale, t, routeLabel }}>{children}</I18nContext.Provider>
}
