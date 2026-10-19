import React from 'react'
import { cn } from '../../lib/utils'

/** Grey placeholder block shown while data loads. Size it with className. */
export const Skeleton: React.FC<React.HTMLAttributes<HTMLDivElement>> = ({ className, ...props }) => (
  <div aria-hidden className={cn('animate-pulse rounded bg-gray-100', className)} {...props} />
)
