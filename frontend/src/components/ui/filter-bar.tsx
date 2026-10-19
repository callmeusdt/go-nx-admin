import React from 'react'
import { Button } from './button'
import { useI18n } from '../../contexts/i18n-context'
import { cn } from '../../lib/utils'

export interface FilterBarProps {
  onQuery: () => void
  onReset: () => void
  children: React.ReactNode
  className?: string
}

/**
 * White filter card. Fields keep a fixed width and sit on the left; reset and
 * query stay on the right. Wrap each field in <FilterField> for its width.
 */
export const FilterBar: React.FC<FilterBarProps> = ({ onQuery, onReset, children, className }) => {
  const { t } = useI18n()
  return (
    <form
      className={cn('flex flex-wrap items-center gap-2 rounded-lg border border-gray-200 bg-white p-3', className)}
      onSubmit={e => { e.preventDefault(); onQuery() }}
    >
      {children}
      <div className="ml-auto flex gap-2">
        <Button type="button" variant="outline" onClick={onReset}>{t('logs.reset')}</Button>
        <Button type="submit">{t('logs.query')}</Button>
      </div>
    </form>
  )
}

export const FilterField: React.FC<{ className?: string; children: React.ReactNode }> = ({ className, children }) => (
  <div className={cn('w-48', className)}>{children}</div>
)
