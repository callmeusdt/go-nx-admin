import React from 'react'
import * as CheckboxPrimitive from '@radix-ui/react-checkbox'
import { Check, Minus } from 'lucide-react'
import { cn } from '../../lib/utils'

export interface CheckboxProps {
  checked: boolean | 'indeterminate'
  onCheckedChange: (checked: boolean) => void
  disabled?: boolean
  'aria-label'?: string
  className?: string
}

/** Square tick box. `checked="indeterminate"` shows a dash (partial selection). */
export const Checkbox = React.forwardRef<HTMLButtonElement, CheckboxProps>(
  ({ checked, onCheckedChange, disabled, className, ...rest }, ref) => (
    <CheckboxPrimitive.Root
      ref={ref}
      checked={checked}
      onCheckedChange={v => onCheckedChange(v === true)}
      disabled={disabled}
      aria-label={rest['aria-label']}
      className={cn(
        'inline-flex h-4 w-4 shrink-0 items-center justify-center rounded border border-gray-300 bg-white transition-colors',
        'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-1',
        'data-[state=checked]:border-blue-600 data-[state=checked]:bg-blue-600 data-[state=indeterminate]:border-blue-600 data-[state=indeterminate]:bg-blue-600',
        'disabled:cursor-not-allowed disabled:opacity-50',
        className,
      )}
    >
      <CheckboxPrimitive.Indicator className="text-white">
        {checked === 'indeterminate' ? <Minus className="h-3 w-3" strokeWidth={3} /> : <Check className="h-3 w-3" strokeWidth={3} />}
      </CheckboxPrimitive.Indicator>
    </CheckboxPrimitive.Root>
  ),
)
Checkbox.displayName = 'Checkbox'
