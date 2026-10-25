import React, { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react'
import { ArrowDown, ArrowUp, ChevronsUpDown, Columns3, Loader2 } from 'lucide-react'
import { useI18n } from '../../contexts/i18n-context'
import { Button } from './button'
import { Checkbox } from './checkbox'
import { Popover, PopoverContent, PopoverTrigger } from './popover'
import { Tooltip } from './tooltip'

export type SortOrder = 'asc' | 'desc'
export interface SortState { field: string; order: SortOrder }

export interface DataTableColumn<T> {
  key: string
  title: React.ReactNode
  /** Plain-text title used in the column picker when `title` is not a string. */
  label?: string
  render?: (row: T, index: number) => React.ReactNode
  sortable?: boolean
  /** Column is shown by default. Defaults to true. */
  defaultVisible?: boolean
  /** Column can not be hidden from the picker (e.g. the actions column). */
  required?: boolean
  className?: string
  headerClassName?: string
}

export interface DataTableProps<T> {
  columns: DataTableColumn<T>[]
  rows: T[]
  rowKey: (row: T) => string | number
  loading?: boolean
  emptyText?: React.ReactNode
  minWidth?: number

  /** Server-side pagination. Omit `total` to hide the footer. */
  page?: number
  pageSize?: number
  total?: number
  onPageChange?: (page: number) => void

  sort?: SortState | null
  onSortChange?: (sort: SortState | null) => void

  /** Enables the checkbox column. */
  selectable?: boolean
  selectedKeys?: Array<string | number>
  onSelectionChange?: (keys: Array<string | number>) => void
  /** Rows that must not be selected (e.g. the current user). */
  isRowDisabled?: (row: T) => boolean
  /** Rendered in the toolbar while at least one row is selected. */
  batchActions?: (ctx: { selectedKeys: Array<string | number>; selectedRows: T[]; clear: () => void }) => React.ReactNode

  /**
   * Makes the first N visible data columns stick to the start (left) edge while the table
   * scrolls horizontally. The selection checkbox column, when present, sticks
   * together with them. Counts follow the column picker: a hidden
   * column no longer takes a sticky slot.
   */
  stickyStart?: number
  /** Makes the last N visible data columns stick to the end (right) edge (e.g. 1 for the actions column). */
  stickyEnd?: number

  /** Persists the column picker choice; pass a stable id per table. */
  storageKey?: string
  toolbar?: React.ReactNode
}

interface PinInfo { side: 'left' | 'right'; offset: number; edge: boolean }

const clampCount = (n: number | undefined, max: number) => Math.max(0, Math.min(Math.floor(n ?? 0) || 0, max))

// Sticky cells need an opaque background or the cells scrolling underneath show
// through. These match the row states: default, hover, and selected.
const PIN_BG = 'bg-white group-hover:bg-gray-50'
const PIN_BG_SELECTED = 'bg-[#f9fbff] group-hover:bg-gray-50'

function readHidden(storageKey?: string): string[] | null {
  if (!storageKey) return null
  try {
    const raw = localStorage.getItem(`nx-table-cols:${storageKey}`)
    const parsed = raw ? JSON.parse(raw) : null
    return Array.isArray(parsed) ? parsed.filter((v): v is string => typeof v === 'string') : null
  } catch {
    return null
  }
}

export function DataTable<T>(props: DataTableProps<T>) {
  const {
    columns, rows, rowKey, loading, emptyText, minWidth = 720,
    page = 1, pageSize = 20, total, onPageChange,
    sort, onSortChange,
    selectable, selectedKeys = [], onSelectionChange, isRowDisabled, batchActions,
    stickyStart, stickyEnd,
    storageKey, toolbar,
  } = props
  const { t } = useI18n()

  const [hidden, setHidden] = useState<string[]>(() =>
    readHidden(storageKey) ?? columns.filter(c => c.defaultVisible === false).map(c => c.key))

  useEffect(() => {
    if (storageKey) localStorage.setItem(`nx-table-cols:${storageKey}`, JSON.stringify(hidden))
  }, [hidden, storageKey])

  const visible = useMemo(
    () => columns.filter(c => c.required || !hidden.includes(c.key)),
    [columns, hidden],
  )
  const pickable = columns.filter(c => !c.required)

  const selectableRows = useMemo(() => rows.filter(r => !isRowDisabled?.(r)), [rows, isRowDisabled])
  const selected = useMemo(() => new Set(selectedKeys), [selectedKeys])
  const selectedRows = useMemo(() => rows.filter(r => selected.has(rowKey(r))), [rows, selected, rowKey])
  const allOnPage = selectableRows.length > 0 && selectableRows.every(r => selected.has(rowKey(r)))
  const someOnPage = selectableRows.some(r => selected.has(rowKey(r)))

  const toggleAll = () => {
    if (!onSelectionChange) return
    const pageKeys = new Set(selectableRows.map(rowKey))
    onSelectionChange(allOnPage
      ? selectedKeys.filter(k => !pageKeys.has(k))
      : [...selectedKeys, ...selectableRows.map(rowKey).filter(k => !selected.has(k))])
  }
  const toggleRow = (key: string | number) => {
    if (!onSelectionChange) return
    onSelectionChange(selected.has(key) ? selectedKeys.filter(k => k !== key) : [...selectedKeys, key])
  }
  const clear = () => onSelectionChange?.([])

  const nextSort = (field: string) => {
    if (!onSortChange) return
    if (sort?.field !== field) onSortChange({ field, order: 'asc' })
    else if (sort.order === 'asc') onSortChange({ field, order: 'desc' })
    else onSortChange(null)
  }

  const nStart = clampCount(stickyStart, visible.length)
  const nEnd = clampCount(stickyEnd, visible.length - nStart)
  const pinned = nStart + nEnd > 0

  // Sticky offsets depend on the rendered width of every column before (left)
  // or after (right) the cell, so the header cells are measured.
  const scrollRef = useRef<HTMLDivElement>(null)
  const tableRef = useRef<HTMLTableElement>(null)
  const [widths, setWidths] = useState<number[]>([])
  const [scrolled, setScrolled] = useState({ left: false, right: false })

  const measure = useCallback(() => {
    const table = tableRef.current
    const box = scrollRef.current
    if (!table || !box) return
    const cells = Array.from(table.tHead?.rows[0]?.cells ?? [])
    const next = cells.map(c => c.getBoundingClientRect().width)
    setWidths(prev => prev.length === next.length && prev.every((w, i) => Math.abs(w - next[i]) < 0.5) ? prev : next)
    const left = box.scrollLeft > 0
    const right = box.scrollLeft + box.clientWidth < box.scrollWidth - 1
    setScrolled(prev => prev.left === left && prev.right === right ? prev : { left, right })
  }, [])

  useLayoutEffect(() => {
    if (!pinned) return
    measure()
    const table = tableRef.current
    if (!table || typeof ResizeObserver === 'undefined') return
    const observer = new ResizeObserver(measure)
    observer.observe(table)
    if (scrollRef.current) observer.observe(scrollRef.current)
    return () => observer.disconnect()
  }, [pinned, measure, visible, selectable, rows])

  // ci is the cell's position in the row, counting the selection column.
  const cellCount = visible.length + (selectable ? 1 : 0)
  const pinInfo = (ci: number): PinInfo | null => {
    if (!pinned || widths.length !== cellCount) return null
    const leftCells = nStart > 0 ? nStart + (selectable ? 1 : 0) : 0
    if (ci < leftCells) {
      return { side: 'left', offset: widths.slice(0, ci).reduce((a, b) => a + b, 0), edge: ci === leftCells - 1 }
    }
    if (nEnd > 0 && ci >= cellCount - nEnd) {
      return { side: 'right', offset: widths.slice(ci + 1).reduce((a, b) => a + b, 0), edge: ci === cellCount - nEnd }
    }
    return null
  }
  const pinStyle = (pin: PinInfo | null): React.CSSProperties | undefined => {
    if (!pin) return undefined
    const style: React.CSSProperties = { position: 'sticky', [pin.side]: pin.offset }
    if (pin.edge && pin.side === 'left' && scrolled.left) style.boxShadow = 'inset -8px 0 6px -6px rgba(0,0,0,0.15)'
    if (pin.edge && pin.side === 'right' && scrolled.right) style.boxShadow = 'inset 8px 0 6px -6px rgba(0,0,0,0.15)'
    return style
  }
  const selOffset = selectable ? 1 : 0

  const totalPages = total === undefined ? 1 : Math.max(1, Math.ceil(total / pageSize))
  const colSpan = visible.length + (selectable ? 1 : 0)
  const hasSelection = selectable && selectedKeys.length > 0

  const columnPicker = pickable.length > 0 && (
    <Popover>
      <Tooltip content={t('table.columns')} side="top">
        <PopoverTrigger asChild>
          <button
            type="button"
            aria-label={t('table.columns')}
            className="-my-1 rounded p-1 text-gray-400 hover:bg-gray-200 hover:text-gray-600 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500"
          >
            <Columns3 className="h-4 w-4" />
          </button>
        </PopoverTrigger>
      </Tooltip>
      <PopoverContent>
        {pickable.map(col => (
          <label key={col.key} className="flex cursor-pointer items-center gap-2 rounded-sm px-2 py-1.5 text-sm hover:bg-gray-100">
            <Checkbox
              checked={!hidden.includes(col.key)}
              onCheckedChange={() => setHidden(h => h.includes(col.key) ? h.filter(k => k !== col.key) : [...h, col.key])}
              aria-label={col.label ?? (typeof col.title === 'string' ? col.title : col.key)}
            />
            {col.label ?? (typeof col.title === 'string' ? col.title : col.key)}
          </label>
        ))}
      </PopoverContent>
    </Popover>
  )

  return (
    <div className="space-y-3">
      {toolbar && <div className="flex flex-wrap items-center justify-end gap-2">{toolbar}</div>}

      <div className="relative bg-white rounded-lg border border-gray-200 overflow-hidden">
        {hasSelection && (
          <div className="flex flex-wrap items-center gap-2 border-b border-blue-100 bg-blue-50/60 px-4 py-2 text-sm">
            <span className="text-gray-700">{t('table.selected').replace('{count}', String(selectedKeys.length))}</span>
            {batchActions?.({ selectedKeys, selectedRows, clear })}
            <Button variant="ghost" size="sm" onClick={clear}>{t('table.clear_selection')}</Button>
          </div>
        )}
        <div ref={scrollRef} className="overflow-x-auto" onScroll={pinned ? measure : undefined}>
        <table ref={tableRef} className="w-full text-sm" style={{ minWidth }}>
          <thead className="bg-gray-50 border-b border-gray-200">
            <tr>
              {selectable && (
                <th
                  className={`w-10 px-4 py-3 ${pinInfo(0) ? 'z-20 bg-gray-50' : ''}`}
                  style={pinStyle(pinInfo(0))}
                >
                  <Checkbox
                    aria-label={t('table.select_all')}
                    checked={someOnPage && !allOnPage ? 'indeterminate' : allOnPage}
                    disabled={selectableRows.length === 0}
                    onCheckedChange={toggleAll}
                  />
                </th>
              )}
              {visible.map((col, colIndex) => {
                const active = sort?.field === col.key
                const sortable = col.sortable && !!onSortChange
                const pin = pinInfo(colIndex + selOffset)
                return (
                  <th
                    key={col.key}
                    aria-sort={active ? (sort!.order === 'asc' ? 'ascending' : 'descending') : undefined}
                    className={`text-left px-4 py-3 font-medium text-gray-600 ${pin ? 'z-20 bg-gray-50' : ''} ${col.headerClassName ?? ''}`}
                    style={pinStyle(pin)}
                  >
                    {sortable ? (
                      <button
                        type="button"
                        onClick={() => nextSort(col.key)}
                        className="inline-flex items-center gap-1 hover:text-gray-900"
                        title={active ? t(sort!.order === 'asc' ? 'table.sort_asc' : 'table.sort_desc') : undefined}
                      >
                        {col.title}
                        {active
                          ? (sort!.order === 'asc' ? <ArrowUp className="w-3 h-3" /> : <ArrowDown className="w-3 h-3" />)
                          : <ChevronsUpDown className="w-3 h-3 text-gray-300" />}
                      </button>
                    ) : col.title}
                    {colIndex === visible.length - 1 && columnPicker && <span className="ml-2 inline-flex align-middle">{columnPicker}</span>}
                  </th>
                )
              })}
            </tr>
          </thead>
          <tbody>
            {rows.map((row, i) => {
              const key = rowKey(row)
              const disabled = isRowDisabled?.(row) ?? false
              const checked = selected.has(key)
              return (
                <tr key={key} className={`group border-b border-gray-100 hover:bg-gray-50 ${checked ? 'bg-blue-50/40' : ''}`}>
                  {selectable && (
                    <td
                      className={`w-10 px-4 py-2.5 ${pinInfo(0) ? `z-10 ${checked ? PIN_BG_SELECTED : PIN_BG}` : ''}`}
                      style={pinStyle(pinInfo(0))}
                    >
                      <Checkbox
                        aria-label={t('table.select_row')}
                        checked={checked}
                        disabled={disabled}
                        onCheckedChange={() => toggleRow(key)}
                      />
                    </td>
                  )}
                  {visible.map((col, colIndex) => {
                    const pin = pinInfo(colIndex + selOffset)
                    return (
                      <td
                        key={col.key}
                        className={`px-4 py-2.5 ${pin ? `z-10 ${checked ? PIN_BG_SELECTED : PIN_BG}` : ''} ${col.className ?? ''}`}
                        style={pinStyle(pin)}
                      >
                        {col.render ? col.render(row, i) : String((row as Record<string, unknown>)[col.key] ?? '')}
                      </td>
                    )
                  })}
                </tr>
              )
            })}
            {rows.length === 0 && !loading && (
              <tr><td colSpan={colSpan} className="px-4 py-8 text-left text-gray-400">{emptyText ?? t('table.empty')}</td></tr>
            )}
          </tbody>
        </table>
        </div>
        {loading && (
          <div role="status" aria-label={t('table.loading')} className="absolute inset-0 z-30 flex items-center justify-center bg-white/60">
            <Loader2 className="w-5 h-5 animate-spin text-gray-400" />
          </div>
        )}
      </div>

      {total !== undefined && onPageChange && (
        <div className="flex flex-wrap gap-2 items-center justify-between text-sm text-gray-500">
          <span>{t('logs.total').replace('{total}', String(total))}</span>
          <div className="flex items-center gap-2">
            <Button variant="outline" size="sm" disabled={page <= 1 || loading} onClick={() => onPageChange(page - 1)}>{t('logs.previous')}</Button>
            <span>{t('logs.page').replace('{page}', String(page)).replace('{pages}', String(totalPages))}</span>
            <Button variant="outline" size="sm" disabled={page >= totalPages || loading} onClick={() => onPageChange(page + 1)}>{t('logs.next')}</Button>
          </div>
        </div>
      )}
    </div>
  )
}
