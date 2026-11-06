import React from 'react'
import { createRoot } from 'react-dom/client'
import { DataTable } from '../src/components/ui/data-table'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '../src/components/ui/dialog'
import { I18nProvider } from '../src/contexts/i18n-context'
import '../src/index.css'

const rows = [{ id: 1, name: 'First row' }]

const Fixture: React.FC = () => (
  <I18nProvider>
    <main>
      <DataTable
        columns={[
          { key: 'name', title: 'A very long column heading that must stay together', required: true },
          { key: 'actions', title: 'Actions', required: true, align: 'center', render: () => <button type="button">Action</button> },
        ]}
        rows={rows}
        rowKey={row => row.id}
        page={1}
        pageSize={20}
        total={41}
        onPageChange={() => undefined}
        paginationToolbar={<button type="button">Custom page control</button>}
      />
      <Dialog open>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>A very long dialog title that must remain readable</DialogTitle>
          </DialogHeader>
          <p>Dialog content</p>
        </DialogContent>
      </Dialog>
    </main>
  </I18nProvider>
)

createRoot(document.getElementById('root')!).render(<Fixture />)
