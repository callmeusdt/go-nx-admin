import React from 'react'
import { cva, type VariantProps } from 'class-variance-authority'
import { cn } from '../../lib/utils'

const badgeVariants = cva('inline-flex items-center gap-1 whitespace-nowrap rounded px-2 py-0.5 text-xs', {
  variants: {
    variant: {
      neutral: 'bg-gray-100 text-gray-600',
      info: 'bg-blue-50 text-blue-700',
      success: 'bg-green-50 text-green-700',
      warning: 'bg-amber-50 text-amber-700',
      danger: 'bg-red-50 text-red-700',
    },
    shape: { rounded: 'rounded', pill: 'rounded-full' },
  },
  defaultVariants: { variant: 'neutral', shape: 'rounded' },
})

export interface BadgeProps extends React.HTMLAttributes<HTMLSpanElement>, VariantProps<typeof badgeVariants> {}

/** Status / category tag. Pick a `variant` by meaning, not by color. */
export const Badge: React.FC<BadgeProps> = ({ className, variant, shape, ...props }) => (
  <span className={cn(badgeVariants({ variant, shape }), className)} {...props} />
)
