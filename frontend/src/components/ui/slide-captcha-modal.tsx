import React, { useRef, useState, useCallback, useEffect } from 'react'
import { ChevronRight } from 'lucide-react'
import { useI18n } from '../../contexts/i18n-context'

interface SlideCaptchaModalProps {
  captchaId: string
  masterImg: string
  tileImg: string
  tileWidth: number
  tileHeight: number
  tileY: number
  error?: string
  cooldown?: number
  onSubmit: (slideX: number) => void
  onCancel: () => void
  onRefresh: () => void
}

export const SlideCaptchaModal: React.FC<SlideCaptchaModalProps> = ({
  captchaId, masterImg, tileImg, tileWidth, tileHeight, tileY,
  error, cooldown,
  onSubmit, onCancel, onRefresh,
}) => {
  const { t } = useI18n()
  const [offset, setOffset] = useState(0)
  const [locked, setLocked] = useState(false)
  const offsetRef = useRef(0)
  const sliderRef = useRef<HTMLDivElement>(null)
  const maxX = 300 - tileWidth

  const updateOffset = useCallback((clientX: number) => {
    if (!sliderRef.current) return
    const rect = sliderRef.current.getBoundingClientRect()
    const x = Math.max(0, Math.min(clientX - rect.left, maxX))
    offsetRef.current = x
    setOffset(x)
  }, [maxX])

  const handleSliderMouseDown = (e: React.MouseEvent) => {
    if (locked) return
    e.preventDefault()
    updateOffset(e.clientX)
    document.addEventListener('mousemove', onDocMouseMove)
    document.addEventListener('mouseup', onDocMouseUp)
  }

  const onDocMouseMove = (e: MouseEvent) => {
    e.preventDefault()
    updateOffset(e.clientX)
  }

  const onDocMouseUp = () => {
    document.removeEventListener('mousemove', onDocMouseMove)
    document.removeEventListener('mouseup', onDocMouseUp)
    if (offsetRef.current > 0) {
      setLocked(true)
      onSubmit(offsetRef.current)
    }
  }

  const handleRefresh = () => {
    setLocked(false)
    offsetRef.current = 0
    setOffset(0)
    onRefresh()
  }

  useEffect(() => {
    return () => {
      document.removeEventListener('mousemove', onDocMouseMove)
      document.removeEventListener('mouseup', onDocMouseUp)
    }
  }, [])

  const hasCooldown = !!cooldown && cooldown > 0

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-white rounded-xl shadow-2xl p-6" style={{ width: 348 }}>
        <h3 className="text-lg font-bold text-slate-800 mb-4">{t('captcha.title')}</h3>
        <p className="text-sm text-slate-500 mb-3">{t('captcha.hint')}</p>

        <div className="relative mx-auto rounded-lg overflow-hidden select-none mb-3"
          style={{ width: 300, height: 220 }}>
          <img src={masterImg} alt="" className="absolute inset-0 w-full h-full" />
          <div className="absolute"
            style={{
              left: hasCooldown ? '0px' : `${offset}px`,
              top: `${tileY || 0}px`,
              width: `${tileWidth}px`,
              height: `${tileHeight}px`,
            }}>
            <img src={tileImg} alt="" draggable={false} className="w-full h-full" />
          </div>
        </div>

        <div ref={sliderRef}
          className="relative mx-auto rounded-full select-none"
          style={{ width: 300, height: 40, background: '#e8e8e8' }}
          onMouseDown={hasCooldown ? undefined : handleSliderMouseDown}>
          <div className="absolute left-0 top-0 h-full rounded-full transition-all"
            style={{ width: `${offset + 24}px`, background: hasCooldown ? '#ccc' : '#3b82f6' }} />
          <div className="absolute top-1/2 -translate-y-1/2 rounded-full flex items-center justify-center transition-all"
            style={{
              left: `${offset}px`, width: 36, height: 36,
              background: '#fff',
              boxShadow: '0 2px 8px rgba(59,130,246,0.4), 0 0 0 1px rgba(59,130,246,0.2)',
            }}>
            <ChevronRight className="w-5 h-5" style={{ color: '#3b82f6' }} />
          </div>
        </div>

        <div className="flex items-center justify-between mt-2">
          <button type="button" onClick={handleRefresh} disabled={hasCooldown}
            className="text-sm text-blue-500 hover:underline disabled:text-gray-300 disabled:no-underline">{t('captcha.refresh')}</button>
          <button type="button" onClick={onCancel}
            className="px-4 py-2 rounded-lg border border-gray-300 text-slate-600 hover:bg-slate-50 text-sm">
            {t('captcha.cancel')}
          </button>
        </div>

        <div className="mt-2 flex min-h-5 flex-col items-center text-center leading-5">
          {error && <span className="text-red-500 text-xs">{error}</span>}
          {hasCooldown && <span className="text-amber-500 text-xs">{t('captcha.retry_after').replace('{seconds}', String(cooldown))}</span>}
        </div>
      </div>
    </div>
  )
}
