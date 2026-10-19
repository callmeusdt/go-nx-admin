import React, { createContext, useCallback, useContext, useMemo, useState } from 'react'
import * as ToastPrimitive from '@radix-ui/react-toast'
import { CheckCircle2, AlertCircle, Info, X } from 'lucide-react'
import { cn } from '../../lib/utils'
import { useI18n } from '../../contexts/i18n-context'

type ToastKind = 'success' | 'error' | 'info'
interface ToastItem { id: number; kind: ToastKind; message: React.ReactNode }

interface ToastApi {
  success: (message: React.ReactNode) => void
  error: (message: React.ReactNode) => void
  info: (message: React.ReactNode) => void
}

const ToastContext = createContext<ToastApi | null>(null)

/** Light feedback after an action ("saved", "copied"). Field errors still belong in an Alert. */
export const useToast = (): ToastApi => {
  const ctx = useContext(ToastContext)
  if (!ctx) throw new Error('useToast must be used within ToastProvider')
  return ctx
}

const icons = { success: CheckCircle2, error: AlertCircle, info: Info }
const tones: Record<ToastKind, string> = { success: 'text-green-600', error: 'text-red-600', info: 'text-blue-600' }

export const ToastProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { t } = useI18n()
  const [items, setItems] = useState<ToastItem[]>([])
  const push = useCallback((kind: ToastKind, message: React.ReactNode) => {
    setItems(list => [...list, { id: Date.now() + Math.random(), kind, message }])
  }, [])
  const api = useMemo<ToastApi>(() => ({
    success: m => push('success', m),
    error: m => push('error', m),
    info: m => push('info', m),
  }), [push])

  return (
    <ToastContext.Provider value={api}>
      <ToastPrimitive.Provider swipeDirection="right" duration={3500}>
        {children}
        {items.map(item => {
          const Icon = icons[item.kind]
          return (
            <ToastPrimitive.Root
              key={item.id}
              onOpenChange={open => { if (!open) setItems(list => list.filter(i => i.id !== item.id)) }}
              className="flex items-start gap-2 rounded-lg border border-gray-200 bg-white p-3 text-sm shadow-lg"
            >
              <Icon aria-hidden className={cn('mt-0.5 h-4 w-4 shrink-0', tones[item.kind])} />
              <ToastPrimitive.Description className="min-w-0 flex-1 break-words text-gray-700">{item.message}</ToastPrimitive.Description>
              <ToastPrimitive.Close aria-label={t('tabs.close')} className="rounded p-0.5 text-gray-400 hover:bg-gray-100 hover:text-gray-600">
                <X className="h-3.5 w-3.5" />
              </ToastPrimitive.Close>
            </ToastPrimitive.Root>
          )
        })}
        <ToastPrimitive.Viewport className="fixed bottom-4 right-4 z-[80] flex w-80 max-w-[calc(100vw-2rem)] flex-col gap-2 outline-none" />
      </ToastPrimitive.Provider>
    </ToastContext.Provider>
  )
}
