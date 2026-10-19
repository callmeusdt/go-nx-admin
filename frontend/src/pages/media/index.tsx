import React, { useEffect, useState } from 'react'
import { File, Trash2, Upload } from 'lucide-react'
import { api } from '../../lib/api'
import { sessionFetch } from '../../lib/session'
import { Media } from '../../types'
import { useI18n } from '../../contexts/i18n-context'
import { Button } from '../../components/ui/button'
import { Alert } from '../../components/ui/alert'
import { Checkbox } from '../../components/ui/checkbox'
import { ConfirmDialog } from '../../components/ui/confirm-dialog'
import { EmptyState } from '../../components/ui/empty-state'
import { PageHeader } from '../../components/ui/page-header'
import { Spinner } from '../../components/ui/spinner'
import { Tabs, TabsList, TabsTrigger } from '../../components/ui/tabs'
import { useBatch, BatchSummary, batchDelete } from '../../hooks/use-batch'

export const MediaPage: React.FC = () => {
  const { t } = useI18n()
  const [items, setItems] = useState<Media[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [type, setType] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [selected, setSelected] = useState<number[]>([])
  const [delId, setDelId] = useState<number | null>(null)
  const pageSize = 20

  const fetchList = async () => {
    setLoading(true)
    setError('')
    try {
      const params = `page=${page}&page_size=${pageSize}${type ? '&type=' + type : ''}`
      const res = await api.get<{ data: Media[]; total: number }>('/media?' + params)
      setItems(res.data)
      setTotal(res.total)
    } catch {
      setError(t('ui.request_failed'))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { fetchList() }, [page, type])

  const batch = useBatch(succeeded => {
    setSelected(prev => prev.filter(id => !succeeded.includes(id)))
    fetchList()
  })
  const toggle = (id: number) => setSelected(prev => prev.includes(id) ? prev.filter(v => v !== id) : [...prev, id])
  const pageIds = items.map(i => i.id)
  const allOnPage = pageIds.length > 0 && pageIds.every(id => selected.includes(id))
  const togglePage = () => setSelected(prev => allOnPage
    ? prev.filter(id => !pageIds.includes(id))
    : [...prev, ...pageIds.filter(id => !prev.includes(id))])

  const handleDelete = async () => {
    if (delId === null) return
    setError('')
    try { await api.del('/media/' + delId); setDelId(null); await fetchList() }
    catch { setDelId(null); setError(t('ui.request_failed')) }
  }

  const handleUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    const form = new FormData()
    form.append('file', file)
    setError('')
    try {
      const response = await sessionFetch('/api/v1/media/upload', { method: 'POST', body: form })
      if (!response.ok) throw new Error('upload failed')
      await fetchList()
    } catch { setError(t('ui.request_failed')) }
    finally { e.target.value = '' }
  }

  const totalPages = Math.ceil(total / pageSize)
  const filters = [
    { value: '', label: t('media.all') },
    { value: 'image', label: t('media.image') },
    { value: 'file', label: t('media.file') },
  ]

  return (
    <div className="space-y-4">
      <PageHeader
        title={t('media.title')}
        actions={
          <Button size="sm" asChild>
            <label className="cursor-pointer">
              <Upload aria-hidden className="mr-1 h-4 w-4" />{t('common.upload')}
              <input type="file" aria-label={t('common.upload')} className="hidden" onChange={handleUpload} />
            </label>
          </Button>
        }
      />

      <Tabs value={type} onValueChange={v => { setType(v); setPage(1) }}>
        <TabsList>
          {filters.map(filter => <TabsTrigger key={filter.value || 'all'} value={filter.value}>{filter.label}</TabsTrigger>)}
        </TabsList>
      </Tabs>

      {error && <Alert>{error}</Alert>}
      <BatchSummary text={batch.summary} />
      {items.length > 0 && (
        <div className="flex flex-wrap items-center gap-3 text-sm text-slate-600">
          <label className="inline-flex cursor-pointer items-center gap-2">
            <Checkbox aria-label={t('table.select_all')} checked={selected.length > 0 && !allOnPage ? 'indeterminate' : allOnPage} onCheckedChange={togglePage} />
            {t('table.select_all')}
          </label>
          {selected.length > 0 && (
            <>
              <span>{t('table.selected').replace('{count}', String(selected.length))}</span>
              <Button size="sm" variant="destructive" onClick={() => batch.request({ labelKey: 'batch.delete', confirmKey: 'batch.confirm_delete', destructive: true, ...batchDelete('/media') }, selected)}>{t('batch.delete')}</Button>
              <Button size="sm" variant="outline" onClick={() => setSelected([])}>{t('table.clear_selection')}</Button>
            </>
          )}
        </div>
      )}
      {loading ? (
        <Spinner label={t('common.loading')} />
      ) : items.length === 0 ? (
        <EmptyState title={t('media.empty')} />
      ) : (
        <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-6 gap-4">
          {items.map(item => (
            <div key={item.id} className={`bg-white rounded-lg border shadow-sm overflow-hidden group relative ${selected.includes(item.id) ? 'border-blue-500 ring-1 ring-blue-500' : 'border-gray-100'}`}>
              <Checkbox aria-label={`${t('table.select_row')} ${item.original_name}`} checked={selected.includes(item.id)} onCheckedChange={() => toggle(item.id)} className="absolute left-2 top-2 z-10" />
              <div className="h-32 bg-slate-100 flex items-center justify-center">
                {item.type === 'image' ? (
                  <img src={item.url} alt={item.original_name} className="w-full h-full object-cover" />
                ) : (
                  <File aria-hidden="true" className="w-10 h-10 text-slate-400" />
                )}
              </div>
              <div className="p-2">
                <p className="text-xs text-slate-600 truncate">{item.original_name}</p>
                <p className="text-xs text-slate-400 mt-1">{(item.size / 1024).toFixed(1)} KB</p>
                <button onClick={() => setDelId(item.id)}
                  className="mt-1 text-xs text-red-500 hover:text-red-700 opacity-0 group-hover:opacity-100 transition-opacity">
                  <Trash2 aria-hidden="true" className="w-3.5 h-3.5 inline" /> {t('common.delete')}
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
      {batch.dialog}
      <ConfirmDialog
        open={delId !== null}
        onOpenChange={v => !v && setDelId(null)}
        onConfirm={handleDelete}
        description={t('media.delete_hint')}
        destructive
      />

      {totalPages > 1 && (
        <div className="flex flex-wrap items-center justify-between gap-2 text-sm text-gray-500">
          <span>{t('logs.total').replace('{total}', String(total))}</span>
          <div className="flex items-center gap-2">
            <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => setPage(page - 1)}>{t('logs.previous')}</Button>
            <span>{t('logs.page').replace('{page}', String(page)).replace('{pages}', String(totalPages))}</span>
            <Button variant="outline" size="sm" disabled={page >= totalPages} onClick={() => setPage(page + 1)}>{t('logs.next')}</Button>
          </div>
        </div>
      )}
    </div>
  )
}
