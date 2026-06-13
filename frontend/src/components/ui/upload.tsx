import React from 'react'
import { api } from '../../lib/api'
import { Media } from '../../types'

interface UploadProps {
  accept?: string
  maxSize?: number
  onSuccess?: (media: Media) => void
  onError?: (err: string) => void
  children?: React.ReactNode
  className?: string
}

export const Upload: React.FC<UploadProps> = ({ accept, maxSize, onSuccess, onError, children, className }) => {
  const [uploading, setUploading] = React.useState(false)

  const handleFile = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    if (maxSize && file.size > maxSize) {
      onError?.('文件大小超出限制')
      return
    }
    setUploading(true)
    try {
      const form = new FormData()
      form.append('file', file)
      const token = localStorage.getItem('auth_token')
      const res = await fetch('/api/v1/media/upload', {
        method: 'POST',
        headers: token ? { Authorization: 'Bearer ' + token } : {},
        body: form,
      })
      if (!res.ok) {
        const errData = await res.json().catch(() => ({ message: '上传失败' }))
        onError?.(errData.message)
        return
      }
      const data = await res.json()
      onSuccess?.(data as Media)
    } catch {
      onError?.('网络错误')
    } finally {
      setUploading(false)
    }
    e.target.value = ''
  }

  return (
    <label className={className || ''}>
      {children || (uploading ? '上传中...' : '上传文件')}
      <input type="file" accept={accept} className="hidden" onChange={handleFile} disabled={uploading} />
    </label>
  )
}
