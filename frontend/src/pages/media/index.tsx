import React, { useEffect, useState } from 'react'
import { File, Trash2 } from 'lucide-react'
import { api } from '../../lib/api'
import { sessionFetch } from '../../lib/session'
import { Media } from '../../types'
import { useI18n } from '../../contexts/i18n-context'

export const MediaPage: React.FC = () => {
  const { t } = useI18n()
  const [items, setItems] = useState<Media[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [type, setType] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
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

  const handleDelete = async (id: number) => {
    if (!confirm(t('media.delete_hint'))) return
    setError('')
    try { await api.del('/media/' + id); await fetchList() }
    catch { setError(t('ui.request_failed')) }
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
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h2 className="text-2xl font-bold text-slate-800">{t('media.title')}</h2>
        <label className="px-4 py-2 bg-blue-600 text-white rounded-lg cursor-pointer hover:bg-blue-700 text-sm text-center">
          {t('common.upload')}
          <input type="file" aria-label={t('common.upload')} className="hidden" onChange={handleUpload} />
        </label>
      </div>

      <div className="flex flex-wrap gap-2">
        {filters.map(filter => (
          <button key={filter.value} onClick={() => { setType(filter.value); setPage(1) }}
            className={`px-3 py-1.5 rounded-lg text-sm ${type === filter.value ? 'bg-blue-600 text-white' : 'bg-white border text-slate-600 hover:bg-slate-50'}`}>
            {filter.label}
          </button>
        ))}
      </div>

      {error && <p role="alert" className="text-red-600">{error}</p>}
      {loading ? (
        <div className="text-center text-slate-400 py-12">{t('common.loading')}</div>
      ) : items.length === 0 ? (
        <div className="text-center text-slate-400 py-12">{t('media.empty')}</div>
      ) : (
        <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-6 gap-4">
          {items.map(item => (
            <div key={item.id} className="bg-white rounded-lg border border-gray-100 shadow-sm overflow-hidden group">
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
                <button onClick={() => handleDelete(item.id)}
                  className="mt-1 text-xs text-red-500 hover:text-red-700 opacity-0 group-hover:opacity-100 transition-opacity">
                  <Trash2 aria-hidden="true" className="w-3.5 h-3.5 inline" /> {t('common.delete')}
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      {totalPages > 1 && (
        <div className="flex justify-center gap-2 mt-4">
          {Array.from({ length: totalPages }, (_, i) => i + 1).map(p => (
            <button key={p} onClick={() => setPage(p)}
              className={`px-3 py-1 rounded text-sm ${p === page ? 'bg-blue-600 text-white' : 'bg-white border text-slate-600'}`}>
              {p}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}
