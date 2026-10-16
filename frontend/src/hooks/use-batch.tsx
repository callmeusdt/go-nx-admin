import React, { useState } from 'react'
import { api } from '../lib/api'
import { useI18n } from '../contexts/i18n-context'
import { Button } from '../components/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '../components/ui/dialog'

export interface BatchFailure { id: number; reason: string }
interface BatchResponse { deleted?: number[]; updated?: number[]; failed?: BatchFailure[] }

export interface BatchAction {
  /** Message key for the confirm button and, unless `confirmKey` is given, the dialog title. */
  labelKey: string
  /** Message key for the confirm text; `{count}` is replaced. */
  confirmKey: string
  destructive?: boolean
  run: (ids: number[]) => Promise<BatchResponse>
}

export const batchDelete = (path: string): Pick<BatchAction, 'run'> => ({
  run: ids => api.del<BatchResponse>(path, { ids }),
})

export const batchSetStatus = (path: string, status: 0 | 1): Pick<BatchAction, 'run'> => ({
  run: ids => api.put<BatchResponse>(path, { ids, status }),
})

/**
 * Confirm -> run -> summarise. Bulk endpoints report per-id outcomes, so a
 * partial failure is shown (with the server's reason) rather than swallowed.
 * `onDone` receives the ids that actually succeeded so the caller can reload
 * and drop them from the selection while keeping the ones that failed.
 */
export function useBatch(onDone: (succeeded: number[]) => void) {
  const { t, locale } = useI18n()
  const [pending, setPending] = useState<{ action: BatchAction; ids: number[] } | null>(null)
  const [running, setRunning] = useState(false)
  const [failures, setFailures] = useState<BatchFailure[]>([])
  const [error, setError] = useState('')
  const [summary, setSummary] = useState('')

  const request = (action: BatchAction, ids: number[]) => {
    setFailures([]); setError(''); setSummary('')
    setPending({ action, ids })
  }

  const confirm = async () => {
    if (!pending) return
    setRunning(true)
    try {
      const res = await pending.action.run(pending.ids)
      const ok = res.deleted ?? res.updated ?? []
      const failed = res.failed ?? []
      setFailures(failed)
      setSummary(t(failed.length ? 'batch.partial' : 'batch.result').replace('{ok}', String(ok.length)).replace('{fail}', String(failed.length)))
      onDone(ok)
      if (!failed.length) setPending(null)
      else setPending({ ...pending, ids: failed.map(f => f.id) })
    } catch {
      setError(t('ui.request_failed'))
    } finally {
      setRunning(false)
    }
  }

  const close = () => { setPending(null); setFailures([]); setError('') }

  const reasonText = (reason: string) => {
    const key = `batch.reason.${reason}`
    const text = t(key)
    return text === key ? reason : text
  }

  const dialog = (
    <Dialog open={!!pending} onOpenChange={v => !v && !running && close()}>
      <DialogContent className="max-h-[90vh] overflow-y-auto break-words">
        <DialogHeader><DialogTitle>{t('batch.confirm_title')}</DialogTitle></DialogHeader>
        {pending && (
          <p className="text-gray-500 text-sm py-2">{t(pending.action.confirmKey).replace('{count}', String(pending.ids.length))}</p>
        )}
        {summary && failures.length > 0 && <p role="status" className="text-sm text-amber-700">{summary}</p>}
        {failures.length > 0 && (
          <ul className="max-h-40 overflow-y-auto rounded border border-gray-200 text-xs divide-y" data-locale={locale}>
            {failures.map(f => (
              <li key={f.id} className="flex justify-between gap-3 px-3 py-1.5"><span>#{f.id}</span><span className="text-red-600 text-right">{reasonText(f.reason)}</span></li>
            ))}
          </ul>
        )}
        {error && <p role="alert" className="text-red-500 text-sm">{error}</p>}
        <div className="flex flex-wrap justify-end gap-2">
          <Button variant="outline" onClick={close} disabled={running}>{t('common.cancel')}</Button>
          {pending && (
            <Button variant={pending.action.destructive ? 'destructive' : 'default'} onClick={confirm} disabled={running}>
              {running ? t('batch.running') : t(pending.action.labelKey)}
            </Button>
          )}
        </div>
      </DialogContent>
    </Dialog>
  )

  return { request, dialog, summary: pending ? '' : summary }
}

export const BatchSummary: React.FC<{ text: string }> = ({ text }) =>
  text ? <p role="status" className="text-sm text-gray-600">{text}</p> : null
