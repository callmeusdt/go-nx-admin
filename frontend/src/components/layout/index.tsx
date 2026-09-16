import React, { useState, useEffect, createContext, useContext } from 'react'
import { Outlet, useLocation } from 'react-router-dom'
import { Sidebar } from './sidebar'
import { TabBar } from './tabs'
import { UserMenu } from './user-menu'
import { GlobalTools } from './global-tools'
import { LockScreen } from './lock-screen'
import { PanelLeftClose, PanelLeft, ChevronRight, Home } from 'lucide-react'
import { useI18n } from '../../contexts/i18n-context'

const builtinRouteLabels: Record<string, string> = {
  '/dashboard': '仪表盘',
  '/users': '用户管理',
  '/roles': '角色管理',
  '/menus': '菜单管理',
  '/audit-logs': '操作日志',
  '/apis': 'API管理',
  '/login-logs': '登录日志',
  '/media': '媒体管理',
  '/system-config': '系统配置',
}

export const RouteLabelsContext = createContext<Record<string, string>>({})

export const Layout: React.FC = () => {
  const { routeLabel, t } = useI18n()
  const [sidebarCollapsed, setSidebarCollapsed] = useState(() => window.matchMedia('(max-width: 767px)').matches)
  useEffect(() => {
    const viewport = window.matchMedia('(max-width: 767px)')
    const update = () => setSidebarCollapsed(viewport.matches)
    viewport.addEventListener('change', update)
    return () => viewport.removeEventListener('change', update)
  }, [])
  const location = useLocation()
  const extraLabels = useContext(RouteLabelsContext)
  const routeLabels = { ...builtinRouteLabels, ...extraLabels }

  const breadcrumbs = location.pathname
    .split('/')
    .filter(Boolean)
    .map((seg, i, arr) => {
      const path = '/' + arr.slice(0, i + 1).join('/')
      return { label: routeLabel(path, routeLabels[path] || seg), path }
    })

  return (
    <div className="flex h-screen bg-gray-100">
      <Sidebar collapsed={sidebarCollapsed} />
      <div className="flex-1 flex flex-col min-w-0">
        <header className="h-14 bg-white border-b border-gray-200 flex items-center gap-2 px-3 shrink-0">
          <button
            aria-label={t(sidebarCollapsed ? 'navigation.expand' : 'navigation.collapse')}
            aria-expanded={!sidebarCollapsed}
            onClick={() => setSidebarCollapsed(!sidebarCollapsed)}
            className="p-1.5 rounded hover:bg-gray-100 text-gray-400 hover:text-gray-600 transition-colors"
          >
            {sidebarCollapsed ? <PanelLeft className="w-5 h-5" /> : <PanelLeftClose className="w-5 h-5" />}
          </button>

          <div className="hidden sm:flex min-w-0 items-center gap-1 text-xs text-gray-400 border-l border-gray-200 pl-3 ml-0.5 h-full">
            <Home className="w-3 h-3" />
            {breadcrumbs.map((bc, i) => (
              <React.Fragment key={bc.path}>
                <ChevronRight className="w-3 h-3" />
                <span className={i === breadcrumbs.length - 1 ? 'text-gray-700 font-medium' : ''}>
                  {bc.label}
                </span>
              </React.Fragment>
            ))}
          </div>

          <div className="ml-auto flex items-center gap-2">
            <GlobalTools />
            <UserMenu />
          </div>
        </header>
        <TabBar />
        <main className="flex-1 overflow-auto p-3 sm:p-6">
          <Outlet />
        </main>
      </div>
      <LockScreen />
    </div>
  )
}
