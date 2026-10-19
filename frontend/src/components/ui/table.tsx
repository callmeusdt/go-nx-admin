import React from 'react'
import { cn } from '../../lib/utils'

/** Light table for small lists inside dialogs or panels. Use DataTable for paged, selectable lists. */
export const SimpleTable: React.FC<React.TableHTMLAttributes<HTMLTableElement> & { wrapperClassName?: string }> = ({ className, wrapperClassName, ...props }) => (
  <div className={cn('overflow-x-auto rounded-md border border-gray-200', wrapperClassName)}>
    <table className={cn('w-full text-sm', className)} {...props} />
  </div>
)
export const SimpleTableHead: React.FC<React.HTMLAttributes<HTMLTableSectionElement>> = ({ className, ...props }) => (
  <thead className={cn('border-b border-gray-200 bg-gray-50', className)} {...props} />
)
export const SimpleTableRow: React.FC<React.HTMLAttributes<HTMLTableRowElement>> = ({ className, ...props }) => (
  <tr className={cn('border-b border-gray-100 last:border-0', className)} {...props} />
)
export const SimpleTableHeadCell: React.FC<React.ThHTMLAttributes<HTMLTableCellElement>> = ({ className, ...props }) => (
  <th className={cn('px-3 py-2 text-left font-medium text-gray-600', className)} {...props} />
)
export const SimpleTableCell: React.FC<React.TdHTMLAttributes<HTMLTableCellElement>> = ({ className, ...props }) => (
  <td className={cn('px-3 py-2', className)} {...props} />
)
