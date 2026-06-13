import React, { useEffect, useState } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import { useTabs } from '../../contexts/tabs-context'
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

export const Sidebar: React.FC<{ collapsed: boolean }> = ({ collapsed }) => {
  const { addTab } = useTabs()
  const navigate = useNavigate()
  const location = useLocation()
  const [menuItems, setMenuItems] = useState<MenuItem[]>(fallbackMenuItems)
  const [openKeys, setOpenKeys] = useState<Set<string>>(new Set())

  useEffect(() => {
    api.get<{ data: Menu[] }>('/auth/menus')
      .then(res => setMenuItems(menuToItems(res.data)))
      .catch(() => setMenuItems(fallbackMenuItems))
  }, [])

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
    addTab({ key: item.key, label: item.label, icon: item.icon, closable: item.key !== '/dashboard' })
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
          title={collapsed ? item.label : undefined}
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
          {!collapsed && <span className="truncate flex-1 text-left">{item.label}</span>}
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
          <span className="w-full text-center font-bold text-lg">N</span>
        ) : (
          <h1 className="font-bold text-lg w-full px-5 truncate">NX Admin</h1>
        )}
      </div>
      <nav className="flex-1 overflow-y-auto p-2 space-y-1">
        {renderMenu(menuItems)}
      </nav>
    </aside>
  )
}
