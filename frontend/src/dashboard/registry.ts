import type React from 'react'

export interface DashboardWidget {
  /** Unique id. Reusing a built-in key replaces that built-in widget. */
  key: string
  component: React.ComponentType
  /** Grid columns on large screens, 1-4. Defaults to 1. */
  span?: 1 | 2 | 3 | 4
  /** Ascending. Built-ins use 10, 20, 30, 40. Defaults to 100. */
  order?: number
  /**
   * Menu-style permission key. The widget is shown only when the signed-in
   * role holds it. This hides the widget; the API behind it must still
   * authorise the request itself.
   */
  permission?: string
}

export interface DashboardOptions {
  /** Added after (or over) the built-in widgets. */
  widgets?: DashboardWidget[]
  /** Keys of built-in widgets to remove. */
  hide?: string[]
  /** Replaces the whole dashboard page; widgets and hide are then ignored. */
  page?: React.ComponentType
}

const DEFAULT_ORDER = 100

/** Built-ins first, extras override by key, hidden keys dropped, then ordered. */
export function resolveWidgets(builtin: DashboardWidget[], options?: DashboardOptions): DashboardWidget[] {
  const byKey = new Map<string, DashboardWidget>()
  for (const widget of builtin) byKey.set(widget.key, widget)
  for (const widget of options?.widgets ?? []) byKey.set(widget.key, widget)
  for (const key of options?.hide ?? []) byKey.delete(key)
  return [...byKey.values()]
    .map((widget, index) => ({ widget, index }))
    .sort((a, b) => (a.widget.order ?? DEFAULT_ORDER) - (b.widget.order ?? DEFAULT_ORDER) || a.index - b.index)
    .map(({ widget }) => widget)
}

export const DEFAULT_HOME_PATH = '/dashboard'
