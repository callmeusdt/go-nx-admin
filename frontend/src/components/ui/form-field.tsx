import React, { useId } from 'react'
import { cn } from '../../lib/utils'

export interface FormFieldProps {
  label: React.ReactNode
  /** Secondary line under the control. */
  hint?: React.ReactNode
  required?: boolean
  className?: string
  /** Receives the id to put on the control so the label is clickable. */
  children: (id: string) => React.ReactNode
}

/** Label above the control, optional hint below. One layout for every form row. */
export const FormField: React.FC<FormFieldProps> = ({ label, hint, required, className, children }) => {
  const id = useId()
  return (
    <div className={cn('min-w-0', className)}>
      <label htmlFor={id} className="mb-1 block text-sm font-medium break-words">
        {label}
        {required && <span className="ml-0.5 text-red-500" aria-hidden>*</span>}
      </label>
      {children(id)}
      {hint && <p className="mt-1 text-xs text-gray-400 break-words">{hint}</p>}
    </div>
  )
}
