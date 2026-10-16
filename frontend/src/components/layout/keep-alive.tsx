import React, { createContext, useContext, useEffect, useRef, useState } from 'react'
import { useLocation, useOutlet } from 'react-router-dom'
import { useTabs } from '../../contexts/tabs-context'

const ActiveContext = createContext(true)

/** Paths (exact) that must unmount when their tab is left, e.g. realtime pages. */
export const KeepAliveExcludeContext = createContext<string[]>([])

export const TAB_REFRESH_EVENT = 'tab-refresh'

/** True while the page's tab is the visible one. Pages stay mounted when hidden. */
export const useTabActive = () => useContext(ActiveContext)

/** Runs `fn` each time the tab becomes visible again (not on first mount). */
export function useOnTabActivate(fn: () => void) {
  const active = useTabActive()
  const first = useRef(true)
  const saved = useRef(fn)
  saved.current = fn
  useEffect(() => {
    if (first.current) { first.current = false; return }
    if (active) saved.current()
  }, [active])
}

interface Entry { element: React.ReactElement; version: number }

/**
 * Renders one pane per open tab and hides the inactive ones, so form state,
 * filters, pagination and scroll position survive tab switches. Closing a tab
 * drops its pane; the tab "refresh" action remounts only that pane.
 */
export const KeepAliveOutlet: React.FC = () => {
  const outlet = useOutlet()
  const { pathname } = useLocation()
  const { tabs } = useTabs()
  const exclude = useContext(KeepAliveExcludeContext)
  const panes = useRef(new Map<string, Entry>())
  const [, bump] = useState(0)

  useEffect(() => {
    const onRefresh = (e: Event) => {
      const key = (e as CustomEvent<string | undefined>).detail ?? pathname
      const entry = panes.current.get(key)
      if (entry) { entry.version += 1; bump(n => n + 1) }
    }
    window.addEventListener(TAB_REFRESH_EVENT, onRefresh)
    return () => window.removeEventListener(TAB_REFRESH_EVENT, onRefresh)
  }, [pathname])

  if (outlet) {
    const entry = panes.current.get(pathname)
    if (entry) entry.element = outlet
    else panes.current.set(pathname, { element: outlet, version: 0 })
  }
  const open = new Set(tabs.map(t => t.key))
  for (const key of [...panes.current.keys()]) {
    if (key !== pathname && (!open.has(key) || exclude.includes(key))) panes.current.delete(key)
  }

  return (
    <>
      {[...panes.current.entries()].map(([key, { element, version }]) => {
        const active = key === pathname
        return (
          <div key={`${key}#${version}`} data-tab-pane={key} data-active={active} hidden={!active} className="h-full overflow-auto p-3 sm:p-6">
            <ActiveContext.Provider value={active}>{element}</ActiveContext.Provider>
          </div>
        )
      })}
    </>
  )
}
