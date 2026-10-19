import React from 'react'
import { Inbox } from 'lucide-react'
import { cn } from '../../lib/utils'

/** "Nothing here" block for lists, grids and panels. Keep the text short and say what to do next if there is an action. */
export const EmptyState: React.FC<{ title: React.ReactNode; action?: React.ReactNode; className?: string }> = ({ title, action, className }) => (
  <div className={cn('flex flex-col items-center gap-2 py-12 text-sm text-gray-400', className)}>
    <Inbox aria-hidden className="h-8 w-8 text-gray-300" />
    <p>{title}</p>
    {action}
  </div>
)
