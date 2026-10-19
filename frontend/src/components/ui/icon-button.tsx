import React from 'react'
import { cva, type VariantProps } from 'class-variance-authority'
import { cn } from '../../lib/utils'
import { Tooltip } from './tooltip'

const iconButtonVariants = cva(
  'inline-flex items-center justify-center rounded p-1.5 text-gray-500 transition-colors hover:bg-gray-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 disabled:cursor-not-allowed disabled:opacity-40 disabled:hover:bg-transparent disabled:hover:text-gray-500',
  {
    variants: {
      tone: {
        default: 'hover:text-gray-800',
        primary: 'hover:text-blue-600',
        success: 'hover:text-green-600',
        danger: 'hover:text-red-600',
      },
    },
    defaultVariants: { tone: 'default' },
  },
)

export interface IconButtonProps
  extends Omit<React.ButtonHTMLAttributes<HTMLButtonElement>, 'title'>,
    VariantProps<typeof iconButtonVariants> {
  /** Tooltip text and accessible name. */
  label: string
}

/** Icon-only action button for table rows and toolbars. Pass the icon as children. */
export const IconButton = React.forwardRef<HTMLButtonElement, IconButtonProps>(
  ({ label, tone, className, children, type = 'button', disabled, ...props }, ref) => (
    <Tooltip content={label}>
      {/* A disabled button swallows pointer events, so the tooltip hangs off a wrapper. */}
      <span className="inline-flex">
        <button
          ref={ref}
          type={type}
          aria-label={label}
          disabled={disabled}
          className={cn(iconButtonVariants({ tone }), className)}
          {...props}
        >
          {children}
        </button>
      </span>
    </Tooltip>
  ),
)
IconButton.displayName = 'IconButton'
