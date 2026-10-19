import React from 'react'
import { Loader2 } from 'lucide-react'
import { cn } from '../../lib/utils'

/** Centered loading indicator for a page or panel. */
export const Spinner: React.FC<{ label: string; className?: string }> = ({ label, className }) => (
  <div role="status" aria-label={label} className={cn('flex justify-center py-20', className)}>
    <Loader2 className="h-6 w-6 animate-spin text-gray-400" />
  </div>
)
