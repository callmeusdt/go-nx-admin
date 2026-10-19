import React from 'react'
import * as SelectPrimitive from '@radix-ui/react-select'
import { Check, ChevronDown } from 'lucide-react'
import { cn } from '../../lib/utils'

export interface SelectOption {
  value: string
  label: React.ReactNode
  disabled?: boolean
}

export interface SelectProps {
  value: string
  onValueChange: (value: string) => void
  options: SelectOption[]
  placeholder?: string
  disabled?: boolean
  'aria-label'?: string
  /** Applied to the trigger, e.g. a fixed width. */
  className?: string
}

// Radix reserves '' for "no selection", but our forms use '' for "all" / "none".
const EMPTY = '__empty__'
const toInner = (v: string) => (v === '' ? EMPTY : v)
const toOuter = (v: string) => (v === EMPTY ? '' : v)

/**
 * Single-choice dropdown. Both the trigger and the opened list are styled
 * here, so no native select UI shows up. Pass `''` as a value for an
 * "all / none" option.
 */
export const Select: React.FC<SelectProps> = ({ value, onValueChange, options, placeholder, disabled, className, ...rest }) => (
  <SelectPrimitive.Root value={toInner(value)} onValueChange={v => onValueChange(toOuter(v))} disabled={disabled}>
    <SelectPrimitive.Trigger
      aria-label={rest['aria-label']}
      className={cn(
        'flex h-10 w-full items-center justify-between gap-2 rounded-md border border-gray-300 bg-white pl-3 pr-3 text-left text-sm',
        'focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent',
        'data-[placeholder]:text-gray-400 disabled:cursor-not-allowed disabled:opacity-50',
        className,
      )}
    >
      <span className="min-w-0 flex-1 truncate">
        <SelectPrimitive.Value placeholder={placeholder} />
      </span>
      <SelectPrimitive.Icon asChild>
        <ChevronDown aria-hidden className="h-4 w-4 shrink-0 text-gray-400" />
      </SelectPrimitive.Icon>
    </SelectPrimitive.Trigger>
    <SelectPrimitive.Portal>
      <SelectPrimitive.Content
        position="popper"
        sideOffset={4}
        className="z-[60] max-h-72 min-w-[var(--radix-select-trigger-width)] overflow-hidden rounded-md border border-gray-200 bg-white p-1 text-gray-700 shadow-md"
      >
        <SelectPrimitive.Viewport>
          {options.map(o => (
            <SelectPrimitive.Item
              key={o.value}
              value={toInner(o.value)}
              disabled={o.disabled}
              className="relative flex cursor-pointer select-none items-center rounded-sm py-1.5 pl-2 pr-8 text-sm outline-none data-[highlighted]:bg-gray-100 data-[disabled]:pointer-events-none data-[disabled]:opacity-50 data-[state=checked]:font-medium data-[state=checked]:text-blue-600"
            >
              <SelectPrimitive.ItemText>{o.label}</SelectPrimitive.ItemText>
              <SelectPrimitive.ItemIndicator className="absolute right-2 inline-flex items-center">
                <Check className="h-4 w-4" />
              </SelectPrimitive.ItemIndicator>
            </SelectPrimitive.Item>
          ))}
        </SelectPrimitive.Viewport>
      </SelectPrimitive.Content>
    </SelectPrimitive.Portal>
  </SelectPrimitive.Root>
)
