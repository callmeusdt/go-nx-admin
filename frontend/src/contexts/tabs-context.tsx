import React, { createContext, useContext, useState, useCallback, useEffect } from 'react'
import { useLocation } from 'react-router-dom'
import { useHomePath } from '../dashboard/context'

export interface TabItem {
  key: string
  label: string
  icon?: string
  closable: boolean
  pinned?: boolean
}

interface TabsContextType {
  tabs: TabItem[]
  activeKey: string
  addTab: (tab: TabItem) => void
  removeTab: (key: string) => void
  setActiveKey: (key: string) => void
  closeOthers: (key: string) => void
  closeRight: (key: string) => void
  closeLeft: (key: string) => void
  togglePin: (key: string) => void
}

const TabsContext = createContext<TabsContextType | null>(null)

export const useTabs = () => {
  const ctx = useContext(TabsContext)
  if (!ctx) throw new Error('useTabs must be used within TabsProvider')
  return ctx
}

/**
 * The home tab is always first, can not be closed, and survives every bulk
 * close. Other tabs are closable unless pinned, which also repairs a stale
 * stored home tab after `homePath` changes.
 */
function withHome(tabs: TabItem[], home: string): TabItem[] {
  const existing = tabs.find(tab => tab.key === home)
  const homeTab: TabItem = {
    key: home,
    label: existing?.label ?? (home === '/dashboard' ? '仪表盘' : home),
    icon: existing?.icon ?? (home === '/dashboard' ? 'LayoutDashboard' : undefined),
    closable: false,
  }
  const rest = tabs
    .filter(tab => tab.key !== home)
    .map(tab => ({ ...tab, closable: !tab.pinned }))
  return [homeTab, ...rest]
}

export const TabsProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const home = useHomePath()
  const [tabs, setTabs] = useState<TabItem[]>(() => {
    const raw = localStorage.getItem('tabs')
    const stored: TabItem[] = raw ? JSON.parse(raw).filter((tab: TabItem) => tab.key !== '/online-users') : []
    return withHome(stored, home)
  })
  const [activeKey, setActiveKeyState] = useState<string>('')
  const location = useLocation()

  const setActiveKey = useCallback((key: string) => {
    setActiveKeyState(key)
  }, [])

  const addTab = useCallback((tab: TabItem) => {
    setTabs(prev => {
      const found = prev.find(t => t.key === tab.key)
      if (!found) return [...prev, tab]
      if (tab.key === home && (found.label !== tab.label || found.icon !== tab.icon)) {
        return prev.map(t => t.key === home ? { ...t, label: tab.label, icon: tab.icon } : t)
      }
      return prev
    })
    setActiveKey(tab.key)
  }, [setActiveKey, home])

  const removeTab = useCallback((key: string) => {
    if (key === home) return
    setTabs(prev => {
      const idx = prev.findIndex(t => t.key === key)
      if (idx === -1) return prev
      const newTabs = prev.filter(t => t.key !== key)
      return newTabs
    })
  }, [home])

  const closeOthers = useCallback((key: string) => {
    setTabs(prev => prev.filter(t => t.key === key || t.key === home))
    setActiveKey(key)
  }, [setActiveKey, home])

  const closeRight = useCallback((key: string) => {
    setTabs(prev => {
      const idx = prev.findIndex(t => t.key === key)
      return prev.filter((tab, i) => i <= idx || tab.pinned || tab.key === home)
    })
  }, [home])

  const closeLeft = useCallback((key: string) => {
    setTabs(prev => {
      const idx = prev.findIndex(t => t.key === key)
      return prev.filter((tab, i) => i >= idx || tab.pinned || tab.key === home)
    })
  }, [home])

  const togglePin = useCallback((key: string) => {
    if (key === home) return
    setTabs(prev => prev.map(tab => tab.key === key ? { ...tab, pinned: !tab.pinned, closable: !!tab.pinned } : tab))
  }, [home])

  useEffect(() => {
    setTabs(prev => withHome(prev, home))
  }, [home])

  useEffect(() => {
    localStorage.setItem('tabs', JSON.stringify(tabs))
  }, [tabs])

  useEffect(() => {
    if (location.pathname === '/login') return
    setActiveKey(location.pathname)
  }, [location.pathname, setActiveKey])

  return (
    <TabsContext.Provider value={{ tabs, activeKey, addTab, removeTab, setActiveKey, closeOthers, closeRight, closeLeft, togglePin }}>
      {children}
    </TabsContext.Provider>
  )
}
