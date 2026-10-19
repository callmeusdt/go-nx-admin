import React from 'react'
import * as PopoverPrimitive from '@radix-ui/react-popover'
import { cn } from '../../lib/utils'

export const Popover = PopoverPrimitive.Root
export const PopoverTrigger = PopoverPrimitive.Trigger

export const PopoverContent: React.FC<React.ComponentPropsWithoutRef<typeof PopoverPrimitive.Content>> = ({ className, align = 'end', sideOffset = 6, ...props }) => (
  <PopoverPrimitive.Portal>
    <PopoverPrimitive.Content
      align={align}
      sideOffset={sideOffset}
      className={cn('z-[60] min-w-[160px] rounded-md border border-gray-200 bg-white p-1 text-gray-700 shadow-md outline-none', className)}
      {...props}
    />
  </PopoverPrimitive.Portal>
)
