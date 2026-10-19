import React from 'react'
import * as RadioPrimitive from '@radix-ui/react-radio-group'
import { cn } from '../../lib/utils'

export interface RadioOption { value: string; label: React.ReactNode; disabled?: boolean }

/** One-of-few choice shown inline. Use Select when there are more than ~4 options. */
export const RadioGroup: React.FC<{
  value: string
  onValueChange: (value: string) => void
  options: RadioOption[]
  'aria-label'?: string
  className?: string
}> = ({ value, onValueChange, options, className, ...rest }) => (
  <RadioPrimitive.Root value={value} onValueChange={onValueChange} aria-label={rest['aria-label']} className={cn('flex flex-wrap gap-x-4 gap-y-2', className)}>
    {options.map(o => (
      <label key={o.value} className="inline-flex cursor-pointer items-center gap-2 text-sm">
        <RadioPrimitive.Item
          value={o.value}
          disabled={o.disabled}
          className="flex h-4 w-4 items-center justify-center rounded-full border border-gray-300 bg-white focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 data-[state=checked]:border-blue-600 disabled:opacity-50"
        >
          <RadioPrimitive.Indicator className="h-2 w-2 rounded-full bg-blue-600" />
        </RadioPrimitive.Item>
        {o.label}
      </label>
    ))}
  </RadioPrimitive.Root>
)
