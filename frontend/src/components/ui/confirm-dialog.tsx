import React from 'react'
import { Button } from './button'
import { Alert } from './alert'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from './dialog'
import { useI18n } from '../../contexts/i18n-context'

export interface ConfirmDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onConfirm: () => void | Promise<void>
  title?: React.ReactNode
  description?: React.ReactNode
  confirmLabel?: React.ReactNode
  /** Red confirm button for deletes and other irreversible actions. */
  destructive?: boolean
  busy?: boolean
  error?: string
}

/** Yes/no prompt. Use this instead of window.confirm and hand-built delete dialogs. */
export const ConfirmDialog: React.FC<ConfirmDialogProps> = ({
  open, onOpenChange, onConfirm, title, description, confirmLabel, destructive, busy, error,
}) => {
  const { t } = useI18n()
  return (
    <Dialog open={open} onOpenChange={v => !busy && onOpenChange(v)}>
      <DialogContent className="max-h-[90vh] overflow-y-auto break-words">
        <DialogHeader><DialogTitle>{title ?? t('common.confirm_delete')}</DialogTitle></DialogHeader>
        {description && <p className="py-2 text-sm text-gray-500">{description}</p>}
        {error && <Alert>{error}</Alert>}
        <div className="mt-2 flex flex-wrap justify-end gap-2">
          <Button variant="outline" onClick={() => onOpenChange(false)} disabled={busy}>{t('common.cancel')}</Button>
          <Button variant={destructive ? 'destructive' : 'default'} onClick={() => void onConfirm()} disabled={busy}>
            {confirmLabel ?? t('common.confirm_delete')}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}
