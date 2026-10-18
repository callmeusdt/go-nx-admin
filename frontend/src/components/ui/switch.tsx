import React from 'react'
import { cn } from '../../lib/utils'

export interface SwitchProps {
  checked: boolean
  onCheckedChange: (checked: boolean) => void
  disabled?: boolean
  /** Text shown beside the control; clicking it toggles too. */
  label?: React.ReactNode
  /** Secondary line under the label. */
  description?: React.ReactNode
  /** Required when there is no visible `label`. */
  'aria-label'?: string
  className?: string
}

/**
 * Two-state toggle for boolean options. Use it instead of a dropdown or a
 * checkbox whenever a field is only on/off.
 */
export const Switch: React.FC<SwitchProps> = ({ checked, onCheckedChange, disabled, label, description, className, ...rest }) => {
  const control = (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      aria-label={rest['aria-label']}
      disabled={disabled}
      onClick={() => onCheckedChange(!checked)}
      className={cn(
        'relative inline-flex h-5 w-9 shrink-0 items-center rounded-full transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-1 disabled:cursor-not-allowed disabled:opacity-50',
        checked ? 'bg-blue-600' : 'bg-gray-300',
      )}
    >
      <span
        aria-hidden="true"
        className={cn('inline-block h-4 w-4 rounded-full bg-white shadow transition-transform', checked ? 'translate-x-[18px]' : 'translate-x-0.5')}
      />
    </button>
  )

  if (label === undefined && description === undefined) return <span className={cn('inline-flex', className)}>{control}</span>

  return (
    <div className={cn('flex items-start gap-3', className)}>
      <span className="mt-0.5">{control}</span>
      <span
        className={cn('min-w-0 text-sm', disabled ? 'cursor-not-allowed opacity-50' : 'cursor-pointer')}
        onClick={() => !disabled && onCheckedChange(!checked)}
      >
        {label !== undefined && <span className="font-medium break-words">{label}</span>}
        {description !== undefined && <span className="block text-xs text-gray-400 break-words">{description}</span>}
      </span>
    </div>
  )
}
