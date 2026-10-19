import React from 'react'

/** Page title on the left, primary actions on the right. Every page starts with this. */
export const PageHeader: React.FC<{ title: React.ReactNode; actions?: React.ReactNode }> = ({ title, actions }) => (
  <div className="flex flex-wrap items-center justify-between gap-2">
    <h2 className="text-xl font-bold text-slate-800">{title}</h2>
    {actions && <div className="flex items-center gap-2">{actions}</div>}
  </div>
)
