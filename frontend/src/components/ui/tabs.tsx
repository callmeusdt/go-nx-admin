import React from 'react'
import * as TabsPrimitive from '@radix-ui/react-tabs'
import { cn } from '../../lib/utils'

export const Tabs = TabsPrimitive.Root
export const TabsContent = TabsPrimitive.Content

export const TabsList: React.FC<React.ComponentPropsWithoutRef<typeof TabsPrimitive.List>> = ({ className, ...props }) => (
  <TabsPrimitive.List className={cn('flex gap-1 border-b border-gray-200', className)} {...props} />
)

export const TabsTrigger: React.FC<React.ComponentPropsWithoutRef<typeof TabsPrimitive.Trigger>> = ({ className, ...props }) => (
  <TabsPrimitive.Trigger
    className={cn(
      '-mb-px border-b-2 border-transparent px-3 py-2 text-sm text-gray-500 transition-colors hover:text-gray-800 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500',
      'data-[state=active]:border-blue-500 data-[state=active]:text-blue-600',
      className,
    )}
    {...props}
  />
)
