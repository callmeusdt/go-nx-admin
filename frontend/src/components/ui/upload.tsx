import React from 'react'
import { api } from '../../lib/api'
import { sessionFetch } from '../../lib/session'
import { Media } from '../../types'
import { useI18n } from '../../contexts/i18n-context'

interface UploadProps {
  accept?: string
  maxSize?: number
  onSuccess?: (media: Media) => void
  onError?: (err: string) => void
  children?: React.ReactNode
  className?: string
}

export const Upload: React.FC<UploadProps> = ({ accept, maxSize, onSuccess, onError, children, className }) => {
  const { t } = useI18n()
  const [uploading, setUploading] = React.useState(false)

  const handleFile = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    if (maxSize && file.size > maxSize) {
      onError?.(t('upload.too_large'))
      return
    }
    setUploading(true)
    try {
      const form = new FormData()
      form.append('file', file)
      const res = await sessionFetch('/api/v1/media/upload', {
        method: 'POST',
        body: form,
      })
      if (!res.ok) {
        await res.json().catch(() => null)
        onError?.(t('upload.failed'))
        return
      }
      const data = await res.json()
      onSuccess?.(data as Media)
    } catch {
      onError?.(t('upload.network_error'))
    } finally {
      setUploading(false)
    }
    e.target.value = ''
  }

  return (
    <label className={className || ''}>
      {children || (uploading ? t('upload.uploading') : t('common.upload'))}
      <input type="file" aria-label={t('common.upload')} accept={accept} className="hidden" onChange={handleFile} disabled={uploading} />
    </label>
  )
}
