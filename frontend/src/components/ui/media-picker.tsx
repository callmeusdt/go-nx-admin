import React, { useEffect, useRef, useState } from 'react'
import { File as FileIcon, Image, Upload, Video } from 'lucide-react'
import { api } from '../../lib/api'
import { sessionFetch } from '../../lib/session'
import type { Media } from '../../types'
import { useI18n } from '../../contexts/i18n-context'
import { Button } from './button'
import { Alert } from './alert'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from './dialog'
import { EmptyState } from './empty-state'
import { Spinner } from './spinner'
import { Tabs, TabsList, TabsTrigger } from './tabs'

export type MediaPickerValue = Pick<Media, 'id' | 'url' | 'original_name' | 'size' | 'type'> & { path?: string }

export interface MediaPickerProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onSelect: (media: MediaPickerValue) => void
  type?: Media['type'] | 'all'
  title?: React.ReactNode
}

const PAGE_SIZE = 12

export const MediaPicker: React.FC<MediaPickerProps> = ({ open, onOpenChange, onSelect, type: initialType = 'all', title }) => {
  const { t } = useI18n()
  const [items, setItems] = useState<Media[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [type, setType] = useState<Media['type'] | 'all'>(initialType)
  const [loading, setLoading] = useState(false)
  const [uploading, setUploading] = useState(false)
  const [error, setError] = useState('')
  const inputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    if (open) {
      setType(initialType)
      setPage(1)
    }
  }, [open, initialType])

  useEffect(() => {
    if (!open) return
    let active = true
    setLoading(true)
    setError('')
    const query = new URLSearchParams({ page: String(page), page_size: String(PAGE_SIZE) })
    if (type !== 'all') query.set('type', type)
    void api.get<{ data: Media[]; total: number }>(`/media?${query}`).then(result => {
      if (!active) return
      setItems(result.data)
      setTotal(result.total)
    }).catch(() => active && setError(t('ui.request_failed'))).finally(() => active && setLoading(false))
    return () => { active = false }
  }, [open, page, type, t])

  const upload = async (file: File | undefined) => {
    if (!file) return
    setUploading(true)
    setError('')
    try {
      const body = new FormData()
      body.append('file', file)
      const response = await sessionFetch('/api/v1/media/upload', { method: 'POST', body })
      if (!response.ok) throw new Error('upload failed')
      const media = await response.json() as MediaPickerValue
      onSelect(media)
      onOpenChange(false)
    } catch {
      setError(t('ui.request_failed'))
    } finally {
      setUploading(false)
      if (inputRef.current) inputRef.current.value = ''
    }
  }

  const icon = (media: Media) => media.type === 'image' ? <Image aria-hidden className="h-8 w-8" /> : media.type === 'video' ? <Video aria-hidden className="h-8 w-8" /> : <FileIcon aria-hidden className="h-8 w-8" />
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="z-[70] max-h-[90vh] max-w-3xl overflow-y-auto">
        <DialogHeader><DialogTitle>{title ?? t('media.picker_title')}</DialogTitle></DialogHeader>
        <div className="flex flex-wrap items-center justify-between gap-2">
          <Tabs value={type} onValueChange={value => { setType(value as Media['type'] | 'all'); setPage(1) }}>
            <TabsList>
              <TabsTrigger value="all">{t('media.all')}</TabsTrigger>
              <TabsTrigger value="image">{t('media.image')}</TabsTrigger>
              <TabsTrigger value="file">{t('media.file')}</TabsTrigger>
              <TabsTrigger value="video">{t('media.video')}</TabsTrigger>
            </TabsList>
          </Tabs>
          <Button type="button" size="sm" disabled={uploading} onClick={() => inputRef.current?.click()}>
            <Upload aria-hidden className="mr-1 h-4 w-4" />{uploading ? t('media.uploading') : t('media.upload')}
          </Button>
          <input ref={inputRef} type="file" className="hidden" aria-label={t('media.upload')} onChange={event => void upload(event.target.files?.[0])} />
        </div>
        {error && <Alert>{error}</Alert>}
        {loading ? <Spinner label={t('common.loading')} /> : items.length === 0 ? <EmptyState title={t('media.empty')} /> : (
          <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 md:grid-cols-4">
            {items.map(media => (
              <button key={media.id} type="button" className="group overflow-hidden rounded-lg border border-gray-200 text-left hover:border-blue-500 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500" onClick={() => { onSelect(media); onOpenChange(false) }}>
                <div className="flex h-24 items-center justify-center bg-gray-50 text-gray-400">
                  {media.type === 'image' ? <img src={media.url} alt={media.original_name} className="h-full w-full object-cover" /> : icon(media)}
                </div>
                <div className="truncate px-2 py-1.5 text-xs text-gray-600">{media.original_name}</div>
              </button>
            ))}
          </div>
        )}
        {totalPages > 1 && <div className="flex items-center justify-between text-sm text-gray-500"><span>{t('logs.total').replace('{total}', String(total))}</span><div className="flex items-center gap-2"><Button type="button" variant="outline" size="sm" disabled={page <= 1 || loading} onClick={() => setPage(page - 1)}>{t('logs.previous')}</Button><span>{t('logs.page').replace('{page}', String(page)).replace('{pages}', String(totalPages))}</span><Button type="button" variant="outline" size="sm" disabled={page >= totalPages || loading} onClick={() => setPage(page + 1)}>{t('logs.next')}</Button></div></div>}
      </DialogContent>
    </Dialog>
  )
}
