import React, { useState, useEffect } from 'react'
import { KeepAliveOutlet } from './keep-alive'
import { Sidebar } from './sidebar'
import { TabBar } from './tabs'
import { UserMenu } from './user-menu'
import { GlobalTools } from './global-tools'
import { LockScreen } from './lock-screen'
import { PasswordGate } from '../security/force-password-change'
import { PanelLeftClose, PanelLeft } from 'lucide-react'
import { useI18n } from '../../contexts/i18n-context'
import type { LayoutOptions } from '../../core'

export { RouteLabelsContext } from './route-labels'

const contentMinWidth = (value: LayoutOptions['contentMinWidth']): string | undefined => {
  if (typeof value === 'number' && Number.isFinite(value) && value > 0) return `${value}px`
  if (typeof value === 'string' && /^\d+(?:\.\d+)?(?:px|rem|em|ch|vw|vh|%)$/.test(value.trim())) return value.trim()
  return undefined
}

export const Layout: React.FC<{ options?: LayoutOptions }> = ({ options }) => {
  const { t } = useI18n()
  const [sidebarCollapsed, setSidebarCollapsed] = useState(() => window.matchMedia('(max-width: 767px)').matches)
  useEffect(() => {
    const viewport = window.matchMedia('(max-width: 767px)')
    const update = () => setSidebarCollapsed(viewport.matches)
    viewport.addEventListener('change', update)
    return () => viewport.removeEventListener('change', update)
  }, [])

  return (
    <PasswordGate>
    <div className="flex h-screen bg-gray-100">
      <Sidebar collapsed={sidebarCollapsed} />
      <div className="flex-1 flex flex-col min-w-0">
        <header className="h-14 bg-white border-b border-gray-200 flex items-center gap-2 px-3 shrink-0">
          <button
            aria-label={t(sidebarCollapsed ? 'navigation.expand' : 'navigation.collapse')}
            aria-expanded={!sidebarCollapsed}
            onClick={() => setSidebarCollapsed(!sidebarCollapsed)}
            className="shrink-0 p-1.5 rounded hover:bg-gray-100 text-gray-400 hover:text-gray-600 transition-colors"
          >
            {sidebarCollapsed ? <PanelLeft className="w-5 h-5" /> : <PanelLeftClose className="w-5 h-5" />}
          </button>

          <TabBar />

          <div className="ml-auto shrink-0 flex items-center gap-2">
            <GlobalTools />
            <UserMenu />
          </div>
        </header>
        <main className="flex-1 min-h-0 min-w-0 overflow-hidden">
          <div className="h-full min-w-0 overflow-x-auto overflow-y-hidden">
            <div className="h-full" style={{ minWidth: contentMinWidth(options?.contentMinWidth) }}>
              <KeepAliveOutlet />
            </div>
          </div>
        </main>
      </div>
      <LockScreen />
    </div>
    </PasswordGate>
  )
}
