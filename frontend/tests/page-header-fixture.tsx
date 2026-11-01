import React, { useState } from 'react'
import { createRoot } from 'react-dom/client'
import { PageHeader } from '../src/components/ui/page-header'
import '../src/index.css'

const Fixture: React.FC = () => {
  const [showSecondary, setShowSecondary] = useState(true)
  const [rtl, setRtl] = useState(false)

  return (
    <main dir={rtl ? 'rtl' : 'ltr'}>
      <div className="mb-4 flex gap-2">
        <button type="button" onClick={() => setShowSecondary(value => !value)}>Toggle secondary</button>
        <button type="button" onClick={() => setRtl(value => !value)}>Toggle RTL</button>
      </div>
      <div data-testid="header">
        <PageHeader
          title="Members"
          primaryActions={
            <>
              <button type="button">Create</button>
              <button type="button" disabled>Batch</button>
            </>
          }
          secondaryActions={showSecondary ? <button type="button">Refresh</button> : null}
        />
      </div>
    </main>
  )
}

createRoot(document.getElementById('root')!).render(<Fixture />)
