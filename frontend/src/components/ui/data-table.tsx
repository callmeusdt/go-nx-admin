import React, { useEffect, useMemo, useRef, useState } from 'react'
import { ArrowDown, ArrowUp, ChevronsUpDown, Columns3, Loader2 } from 'lucide-react'
import { useI18n } from '../../contexts/i18n-context'
import { Button } from './button'

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

  /** Persists the column picker choice; pass a stable id per table. */
  storageKey?: string
  toolbar?: React.ReactNode
}

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
    storageKey, toolbar,
  } = props
  const { t } = useI18n()

  const [hidden, setHidden] = useState<string[]>(() =>
    readHidden(storageKey) ?? columns.filter(c => c.defaultVisible === false).map(c => c.key))
  const [pickerOpen, setPickerOpen] = useState(false)
  const pickerRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (storageKey) localStorage.setItem(`nx-table-cols:${storageKey}`, JSON.stringify(hidden))
  }, [hidden, storageKey])

  useEffect(() => {
    if (!pickerOpen) return
    const onDown = (e: MouseEvent) => {
      if (!pickerRef.current?.contains(e.target as Node)) setPickerOpen(false)
    }
    document.addEventListener('mousedown', onDown)
    return () => document.removeEventListener('mousedown', onDown)
  }, [pickerOpen])

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

  const headerCheckbox = useRef<HTMLInputElement>(null)
  useEffect(() => {
    if (headerCheckbox.current) headerCheckbox.current.indeterminate = someOnPage && !allOnPage
  }, [someOnPage, allOnPage])

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

  const totalPages = total === undefined ? 1 : Math.max(1, Math.ceil(total / pageSize))
  const colSpan = visible.length + (selectable ? 1 : 0)
  const hasSelection = selectable && selectedKeys.length > 0
  const showToolbar = hasSelection || toolbar || pickable.length > 0

  return (
    <div className="space-y-3">
      {showToolbar && (
        <div className="flex flex-wrap items-center justify-between gap-2 min-h-8">
          <div className="flex flex-wrap items-center gap-2 text-sm">
            {hasSelection && (
              <>
                <span className="text-gray-600">{t('table.selected').replace('{count}', String(selectedKeys.length))}</span>
                {batchActions?.({ selectedKeys, selectedRows, clear })}
                <Button variant="ghost" size="sm" onClick={clear}>{t('table.clear_selection')}</Button>
              </>
            )}
          </div>
          <div className="flex items-center gap-2">
            {toolbar}
            {pickable.length > 0 && (
              <div className="relative" ref={pickerRef}>
                <Button variant="outline" size="sm" aria-expanded={pickerOpen} onClick={() => setPickerOpen(o => !o)}>
                  <Columns3 className="w-4 h-4 mr-1" />{t('table.columns')}
                </Button>
                {pickerOpen && (
                  <div className="absolute right-0 z-20 mt-1 min-w-[160px] rounded-md border border-gray-200 bg-white py-1 shadow-lg">
                    {pickable.map(col => (
                      <label key={col.key} className="flex cursor-pointer items-center gap-2 px-3 py-1.5 text-sm text-gray-700 hover:bg-gray-50">
                        <input
                          type="checkbox"
                          checked={!hidden.includes(col.key)}
                          onChange={() => setHidden(h => h.includes(col.key) ? h.filter(k => k !== col.key) : [...h, col.key])}
                        />
                        {col.label ?? (typeof col.title === 'string' ? col.title : col.key)}
                      </label>
                    ))}
                  </div>
                )}
              </div>
            )}
          </div>
        </div>
      )}

      <div className="relative bg-white rounded-lg border border-gray-200 overflow-x-auto">
        <table className="w-full text-sm" style={{ minWidth }}>
          <thead className="bg-gray-50 border-b border-gray-200">
            <tr>
              {selectable && (
                <th className="w-10 px-4 py-3">
                  <input
                    ref={headerCheckbox}
                    type="checkbox"
                    aria-label={t('table.select_all')}
                    checked={allOnPage}
                    disabled={selectableRows.length === 0}
                    onChange={toggleAll}
                  />
                </th>
              )}
              {visible.map(col => {
                const active = sort?.field === col.key
                const sortable = col.sortable && !!onSortChange
                return (
                  <th
                    key={col.key}
                    aria-sort={active ? (sort!.order === 'asc' ? 'ascending' : 'descending') : undefined}
                    className={`text-left px-4 py-3 font-medium text-gray-600 ${col.headerClassName ?? ''}`}
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
                <tr key={key} className={`border-b border-gray-100 hover:bg-gray-50 ${checked ? 'bg-blue-50/40' : ''}`}>
                  {selectable && (
                    <td className="w-10 px-4 py-2.5">
                      <input
                        type="checkbox"
                        aria-label={t('table.select_row')}
                        checked={checked}
                        disabled={disabled}
                        onChange={() => toggleRow(key)}
                      />
                    </td>
                  )}
                  {visible.map(col => (
                    <td key={col.key} className={`px-4 py-2.5 ${col.className ?? ''}`}>
                      {col.render ? col.render(row, i) : String((row as Record<string, unknown>)[col.key] ?? '')}
                    </td>
                  ))}
                </tr>
              )
            })}
            {rows.length === 0 && !loading && (
              <tr><td colSpan={colSpan} className="px-4 py-8 text-left text-gray-400">{emptyText ?? t('table.empty')}</td></tr>
            )}
          </tbody>
        </table>
        {loading && (
          <div role="status" aria-label={t('table.loading')} className="absolute inset-0 flex items-center justify-center bg-white/60">
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
