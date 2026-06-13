import React, { createContext, useContext, useState, useCallback, useEffect } from 'react'
import { useLocation } from 'react-router-dom'

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

export const TabsProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [tabs, setTabs] = useState<TabItem[]>(() => {
    const raw = localStorage.getItem('tabs')
    return raw ? JSON.parse(raw).filter((tab: TabItem) => tab.key !== '/online-users') : []
  })
  const [activeKey, setActiveKeyState] = useState<string>('')
  const location = useLocation()

  const setActiveKey = useCallback((key: string) => {
    setActiveKeyState(key)
  }, [])

  const addTab = useCallback((tab: TabItem) => {
    setTabs(prev => {
      if (prev.some(t => t.key === tab.key)) return prev
      return [...prev, tab]
    })
    setActiveKey(tab.key)
  }, [setActiveKey])

  const removeTab = useCallback((key: string) => {
    setTabs(prev => {
      const idx = prev.findIndex(t => t.key === key)
      if (idx === -1) return prev
      const newTabs = prev.filter(t => t.key !== key)
      return newTabs
    })
  }, [])

  const closeOthers = useCallback((key: string) => {
    setTabs(prev => prev.filter(t => t.key === key))
    setActiveKey(key)
  }, [setActiveKey])

  const closeRight = useCallback((key: string) => {
    setTabs(prev => {
      const idx = prev.findIndex(t => t.key === key)
      return prev.filter((tab, i) => i <= idx || tab.pinned)
    })
  }, [])

  const closeLeft = useCallback((key: string) => {
    setTabs(prev => {
      const idx = prev.findIndex(t => t.key === key)
      return prev.filter((tab, i) => i >= idx || tab.pinned)
    })
  }, [])

  const togglePin = useCallback((key: string) => {
    setTabs(prev => prev.map(tab => tab.key === key ? { ...tab, pinned: !tab.pinned, closable: !!tab.pinned } : tab))
  }, [])

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
