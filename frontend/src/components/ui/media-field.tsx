import React, { useState } from 'react'
import { File as FileIcon, Image, Video, X } from 'lucide-react'
import { useI18n } from '../../contexts/i18n-context'
import { normalizeMediaUrl } from '../../lib/media-url'
import type { MediaPickerValue } from './media-picker'
import { MediaPicker } from './media-picker'
import { Button } from './button'

export interface MediaFieldProps {
  value?: MediaPickerValue | null
  onChange: (value: MediaPickerValue | null) => void
  type?: MediaPickerValue['type'] | 'all'
  label?: React.ReactNode
  pickerTitle?: React.ReactNode
  disabled?: boolean
}

export const MediaField: React.FC<MediaFieldProps> = ({ value, onChange, type = 'all', label, pickerTitle, disabled }) => {
  const { t } = useI18n()
  const [open, setOpen] = useState(false)
  const previewUrl = normalizeMediaUrl(value?.url ?? value?.path)
  const isImage = value?.type === 'image'

  return (
    <div className="space-y-2">
      {label && <div className="text-sm font-medium text-slate-700">{label}</div>}
      <div className="flex flex-wrap items-center gap-3">
        {value ? (
          <div className="relative h-20 w-20 overflow-hidden rounded-lg border border-gray-200 bg-gray-50">
            {isImage && previewUrl ? <img src={previewUrl} alt={value.original_name ?? t('media.preview')} className="h-full w-full object-cover" /> : (
              <div className="flex h-full w-full items-center justify-center text-gray-400">
                {value.type === 'video' ? <Video aria-hidden className="h-8 w-8" /> : <FileIcon aria-hidden className="h-8 w-8" />}
              </div>
            )}
            <button
              type="button"
              aria-label={t('media.remove')}
              title={t('media.remove')}
              disabled={disabled}
              onClick={() => onChange(null)}
              className="absolute right-1 top-1 rounded-full bg-black/60 p-1 text-white opacity-90 transition hover:bg-black/80 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 disabled:cursor-not-allowed disabled:opacity-50"
            >
              <X aria-hidden className="h-3 w-3" />
            </button>
          </div>
        ) : (
          <div className="flex h-20 w-20 items-center justify-center rounded-lg border border-dashed border-gray-300 bg-gray-50 text-gray-400">
            <Image aria-hidden className="h-8 w-8" />
          </div>
        )}
        <Button type="button" variant="outline" disabled={disabled} onClick={() => setOpen(true)}>
          {value ? t('media.change') : t('media.choose')}
        </Button>
      </div>
      <MediaPicker open={open} onOpenChange={setOpen} onSelect={onChange} type={type} title={pickerTitle} />
    </div>
  )
}
