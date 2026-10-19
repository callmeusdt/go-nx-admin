import React from 'react'
import * as TooltipPrimitive from '@radix-ui/react-tooltip'

export const TooltipProvider = TooltipPrimitive.Provider

/** Hover/focus hint for icon-only buttons. Replaces the native `title` bubble. */
export const Tooltip: React.FC<{ content: React.ReactNode; children: React.ReactElement; side?: 'top' | 'bottom' | 'left' | 'right' }> = ({ content, children, side = 'top' }) => (
  <TooltipPrimitive.Root delayDuration={300}>
    <TooltipPrimitive.Trigger asChild>{children}</TooltipPrimitive.Trigger>
    <TooltipPrimitive.Portal>
      <TooltipPrimitive.Content side={side} sideOffset={6} className="z-[70] max-w-xs rounded-md bg-gray-900 px-2 py-1 text-xs text-white shadow-md">
        {content}
      </TooltipPrimitive.Content>
    </TooltipPrimitive.Portal>
  </TooltipPrimitive.Root>
)
