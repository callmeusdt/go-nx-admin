import React from 'react'

export interface PageHeaderProps {
  title: React.ReactNode
  /** Legacy action slot. It remains the primary action group. */
  actions?: React.ReactNode
  /** Main CRUD actions, rendered before the divider. */
  primaryActions?: React.ReactNode
  /** Supporting actions, rendered after the divider. */
  secondaryActions?: React.ReactNode
}

const hasChildren = (children: React.ReactNode) => React.Children.toArray(children).length > 0

/** Page title on the left, primary actions on the right. Every page starts with this. */
export const PageHeader: React.FC<PageHeaderProps> = ({ title, actions, primaryActions, secondaryActions }) => {
  const primary = primaryActions ?? actions
  const hasPrimary = hasChildren(primary)
  const hasSecondary = hasChildren(secondaryActions)

  return (
    <div className="flex flex-wrap items-center justify-between gap-2">
      <h2 className="text-xl font-bold text-slate-800">{title}</h2>
      {(hasPrimary || hasSecondary) && (
        <div className="flex flex-wrap items-center gap-2">
          {hasPrimary && <div className="flex flex-wrap items-center gap-2">{primary}</div>}
          {hasPrimary && hasSecondary && (
            <span role="separator" aria-orientation="vertical" aria-hidden="true" className="mx-1 h-5 w-px shrink-0 bg-slate-200" />
          )}
          {hasSecondary && <div className="flex flex-wrap items-center gap-2">{secondaryActions}</div>}
        </div>
      )}
    </div>
  )
}
