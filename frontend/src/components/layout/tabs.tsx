import React, { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTabs } from '../../contexts/tabs-context'
import { Pin, RotateCw, X } from 'lucide-react'
import { useI18n } from '../../contexts/i18n-context'

export const TabBar: React.FC = () => {
  const { routeLabel } = useI18n()
  const { tabs, activeKey, removeTab, setActiveKey, closeOthers, closeRight, closeLeft, togglePin } = useTabs()
  const navigate = useNavigate()
  const [contextMenu, setContextMenu] = useState<{ x: number; y: number; key: string } | null>(null)

  const handleTabClick = (key: string) => {
    setActiveKey(key)
    navigate(key)
  }

  const handleTabClose = (e: React.MouseEvent, key: string) => {
    e.stopPropagation()
    const idx = tabs.findIndex(t => t.key === key)
    removeTab(key)
    if (activeKey === key) {
      const remaining = tabs.filter(t => t.key !== key)
      if (remaining.length > 0) {
        const newIdx = Math.min(idx, remaining.length - 1)
        navigate(remaining[newIdx].key)
      } else {
        navigate('/dashboard')
      }
    }
  }

  const handleContextMenu = (e: React.MouseEvent, key: string) => {
    e.preventDefault()
    setContextMenu({ x: e.clientX, y: e.clientY, key })
  }

  const execCloseOthers = () => {
    if (!contextMenu) return
    closeOthers(contextMenu.key)
    navigate(contextMenu.key)
    setContextMenu(null)
  }

  const execCloseRight = () => {
    if (!contextMenu) return
    closeRight(contextMenu.key)
    setContextMenu(null)
  }

  const execCloseLeft = () => {
    if (!contextMenu) return
    closeLeft(contextMenu.key)
    setContextMenu(null)
  }

  const execRefresh = () => {
    if (!contextMenu) return
    navigate(contextMenu.key)
    setTimeout(() => window.dispatchEvent(new Event('tab-refresh')), 0)
    setContextMenu(null)
  }

  const execTogglePin = () => {
    if (!contextMenu) return
    togglePin(contextMenu.key)
    setContextMenu(null)
  }

  const execCloseAll = () => {
    closeOthers('/dashboard')
    navigate('/dashboard')
    setContextMenu(null)
  }

  React.useEffect(() => {
    const handler = () => setContextMenu(null)
    document.addEventListener('click', handler)
    return () => document.removeEventListener('click', handler)
  }, [])

  if (tabs.length === 0) return null

  return (
    <>
      <div className="h-8 bg-white border-b border-gray-200 flex items-stretch overflow-hidden shrink-0">
        <div className="flex-1 flex items-stretch overflow-x-auto scrollbar-none">
          {tabs.map(tab => (
            <div
              key={tab.key}
              onClick={() => handleTabClick(tab.key)}
              onContextMenu={e => handleContextMenu(e, tab.key)}
              className={`flex items-center gap-1 px-2.5 text-xs cursor-pointer border-r border-gray-200 whitespace-nowrap select-none transition-colors shrink-0 ${
                activeKey === tab.key
                  ? 'bg-blue-50 text-blue-600 border-t-2 border-t-blue-500'
                  : 'text-gray-600 hover:bg-gray-50'
              }`}
            >
              <span>{routeLabel(tab.key, tab.label)}</span>
              {tab.pinned && <Pin className="w-3 h-3" />}
              {tab.closable && !tab.pinned && (
                <span
                  onClick={e => handleTabClose(e, tab.key)}
                  className="inline-flex items-center justify-center w-4 h-4 rounded hover:bg-gray-200 transition-colors"
                >
                  <X className="w-3 h-3" />
                </span>
              )}
            </div>
          ))}
        </div>
      </div>

      {contextMenu && (
        <div
          className="fixed bg-white rounded-md shadow-lg border border-gray-200 py-1 z-50 min-w-[130px]"
          style={{ left: contextMenu.x, top: contextMenu.y }}
          onClick={e => e.stopPropagation()}
        >
          <button onClick={execCloseOthers} className="w-full text-left px-3 py-1.5 text-sm text-gray-700 hover:bg-gray-100">关闭其他</button>
          <button onClick={execCloseLeft} className="w-full text-left px-3 py-1.5 text-sm text-gray-700 hover:bg-gray-100">关闭左侧</button>
          <button onClick={execCloseRight} className="w-full text-left px-3 py-1.5 text-sm text-gray-700 hover:bg-gray-100">关闭右侧</button>
          <button onClick={execRefresh} className="w-full text-left px-3 py-1.5 text-sm text-gray-700 hover:bg-gray-100"><RotateCw className="inline w-3 h-3 mr-1" />刷新当前</button>
          <button onClick={execTogglePin} className="w-full text-left px-3 py-1.5 text-sm text-gray-700 hover:bg-gray-100">固定/取消固定</button>
          <button onClick={execCloseAll} className="w-full text-left px-3 py-1.5 text-sm text-gray-700 hover:bg-gray-100">关闭全部</button>
        </div>
      )}
    </>
  )
}
