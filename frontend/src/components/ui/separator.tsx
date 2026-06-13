import React from 'react'
import { cn } from '../../lib/utils'

export const Separator: React.FC<{ className?: string }> = ({ className }) => (
  <div className={cn('h-px bg-gray-200', className)} />
)
