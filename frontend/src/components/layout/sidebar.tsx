import React, { useEffect, useRef, useState } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import { useTabs } from '../../contexts/tabs-context'
import { useHomePath } from '../../dashboard/context'
import {
  ChevronDown,
  ChevronRight,
  Circle,
  FileText,
  LayoutDashboard,
  LayoutPanelLeft,
  LogIn,
  Menu as MenuIcon,
  Route,
  Shield,
  Users,
  Settings,
  Image,
  SlidersHorizontal,
} from 'lucide-react'
import { api } from '../../lib/api'
import { Menu } from '../../types'
import { useI18n } from '../../contexts/i18n-context'

interface MenuItem {
  key: string
  label: string
  icon: string
  children?: MenuItem[]
}

const IconComponent = ({ name, className }: { name: string; className?: string }) => {
  const iconMap: Record<string, React.FC<{ className?: string }>> = {
    Circle,
    FileText,
    LayoutDashboard,
    LayoutPanelLeft,
    LogIn,
    Menu: MenuIcon,
    Route,
    Shield,
    Users,
    Settings,
    Image,
    Sliders: SlidersHorizontal,
  }
  const Icon = iconMap[name] || Circle
  return Icon ? <Icon className={className || 'w-4 h-4'} /> : null
}

const fallbackMenuItems: MenuItem[] = [
  { key: '/dashboard', label: '仪表盘', icon: 'LayoutDashboard' },
  { key: '/users', label: '用户管理', icon: 'Users' },
  { key: '/roles', label: '角色管理', icon: 'Shield' },
  { key: '/menus', label: '菜单管理', icon: 'Menu' },
  { key: '/audit-logs', label: '操作日志', icon: 'FileText' },
]

function menuToItems(menus: Menu[]): MenuItem[] {
  return menus.map(menu => ({
    key: menu.path || `menu-${menu.id}`,
    label: menu.name,
    icon: menu.icon || 'Circle',
    children: menu.children?.length ? menuToItems(menu.children) : undefined,
  }))
}

// Returns the keys of the groups containing `path`, or null when no menu node
// has exactly that path. Prefix matches are deliberately not considered: a tab
// such as /users/5 opened from a row action has no node and must not move the tree.
export function findMenuAncestors(items: MenuItem[], path: string, trail: string[] = []): string[] | null {
  for (const item of items) {
    if (item.key === path) return trail
    if (item.children?.length) {
      const found = findMenuAncestors(item.children, path, [...trail, item.key])
      if (found) return found
    }
  }
  return null
}

const MenuSkeleton: React.FC<{ collapsed: boolean }> = ({ collapsed }) => (
  <div className="space-y-1" aria-hidden="true">
    {[0, 1, 2, 3, 4].map(i => (
      <div key={i} className={`flex items-center rounded-lg ${collapsed ? 'justify-center p-2.5' : 'gap-3 px-3 py-2'}`}>
        <div className="w-4 h-4 rounded bg-slate-800 animate-pulse shrink-0" />
        {!collapsed && <div className="h-3.5 rounded bg-slate-800 animate-pulse flex-1" />}
      </div>
    ))}
  </div>
)

export const Sidebar: React.FC<{ collapsed: boolean }> = ({ collapsed }) => {
  const { t, routeLabel } = useI18n()
  const { addTab } = useTabs()
  const navigate = useNavigate()
  const home = useHomePath()
  const location = useLocation()
  // null until /auth/menus settles, so the first paint is the full tree (base +
  // extension menus together) instead of the base fallback replaced a moment later.
  const [menuItems, setMenuItems] = useState<MenuItem[] | null>(null)
  const [openKeys, setOpenKeys] = useState<Set<string>>(new Set())
  const navRef = useRef<HTMLElement>(null)

  useEffect(() => {
    let active = true
    api.get<{ data: Menu[] }>('/auth/menus')
      .then(res => { if (active) setMenuItems(menuToItems(res.data)) })
      .catch(() => { if (active) setMenuItems(fallbackMenuItems) })
    return () => { active = false }
  }, [])

  // Keep the tree in step with the current route (refresh, tab switch, deep link).
  useEffect(() => {
    if (!menuItems) return
    const ancestors = findMenuAncestors(menuItems, location.pathname)
    if (!ancestors) return
    // A bookmarked/refreshed route has not gone through handleMenuClick yet.
    // Register its authorised leaf so leaving it does not discard its pane.
    let siblings = menuItems
    for (const key of ancestors) siblings = siblings.find(item => item.key === key)?.children ?? []
    const current = siblings.find(item => item.key === location.pathname)
    if (current && !current.children?.length && current.key.startsWith('/')) {
      addTab({ key: current.key, label: current.label, icon: current.icon, closable: current.key !== home })
    }
    if (ancestors.length) {
      setOpenKeys(prev => ancestors.every(k => prev.has(k)) ? prev : new Set([...prev, ...ancestors]))
    }
    requestAnimationFrame(() => {
      navRef.current?.querySelector('[data-menu-active="true"]')?.scrollIntoView?.({ block: 'nearest' })
    })
  }, [location.pathname, menuItems, addTab, home])

  const handleMenuClick = (item: MenuItem) => {
    if (item.children?.length) {
      setOpenKeys(prev => {
        const next = new Set(prev)
        next.has(item.key) ? next.delete(item.key) : next.add(item.key)
        return next
      })
      return
    }
    if (!item.key.startsWith('/')) return
    addTab({ key: item.key, label: item.label, icon: item.icon, closable: item.key !== home })
    navigate(item.key)
  }

  const renderMenu = (items: MenuItem[], depth = 0) => items.map(item => {
    const isActive = location.pathname === item.key
    const hasChildren = !!item.children?.length
    const isOpen = openKeys.has(item.key)

    return (
      <div key={item.key}>
        <button
          onClick={() => handleMenuClick(item)}
          data-menu-active={isActive}
          title={collapsed ? routeLabel(item.key, item.label) : undefined}
          aria-label={routeLabel(item.key, item.label)}
          className={`w-full flex items-center rounded-lg text-sm transition-colors ${
            collapsed
              ? 'justify-center p-2.5'
              : 'gap-3 px-3 py-2'
          } ${
            isActive
              ? 'bg-blue-600 text-white'
              : 'text-slate-300 hover:bg-slate-800 hover:text-white'
          }`}
          style={!collapsed ? { paddingLeft: 12 + depth * 16 } : undefined}
        >
          <IconComponent name={item.icon} className="w-4 h-4 shrink-0" />
          {!collapsed && <span className="truncate flex-1 text-left">{routeLabel(item.key, item.label)}</span>}
          {!collapsed && hasChildren && (
            isOpen ? <ChevronDown className="w-3.5 h-3.5" /> : <ChevronRight className="w-3.5 h-3.5" />
          )}
        </button>
        {!collapsed && hasChildren && isOpen && (
          <div className="mt-1 space-y-1">
            {renderMenu(item.children!, depth + 1)}
          </div>
        )}
      </div>
    )
  })

  return (
    <aside className={`${collapsed ? 'w-16' : 'w-56'} bg-slate-900 text-white flex flex-col h-full shrink-0 transition-all duration-200`}>
      <div className="h-14 flex items-center border-b border-slate-700 overflow-hidden shrink-0">
        {collapsed ? (
          <span className="w-full text-center font-bold text-lg">{Array.from(t('login.title'))[0]}</span>
        ) : (
          <h1 className="font-bold text-lg w-full px-5 truncate">{t('login.title')}</h1>
        )}
      </div>
      <nav ref={navRef} className="flex-1 overflow-y-auto p-2 space-y-1" aria-busy={!menuItems}>
        {menuItems ? renderMenu(menuItems) : <MenuSkeleton collapsed={collapsed} />}
      </nav>
    </aside>
  )
}
