import React from 'react'
import { cva, type VariantProps } from 'class-variance-authority'
import { cn } from '../../lib/utils'

const alertVariants = cva('text-sm', {
  variants: {
    variant: {
      error: 'text-red-600',
      warning: 'text-amber-700',
      success: 'text-green-600',
      info: 'text-gray-600',
    },
  },
  defaultVariants: { variant: 'error' },
})

export interface AlertProps extends React.HTMLAttributes<HTMLParagraphElement>, VariantProps<typeof alertVariants> {}

/** Inline message under a form or above a table. Errors announce as alerts, the rest as status. */
export const Alert: React.FC<AlertProps> = ({ className, variant, ...props }) => (
  <p role={variant === 'error' || !variant ? 'alert' : 'status'} className={cn(alertVariants({ variant }), className)} {...props} />
)
