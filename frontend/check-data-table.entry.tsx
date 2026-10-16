import React from 'react'
import ReactDOM from 'react-dom/client'
import { I18nProvider } from './src/contexts/i18n-context'
import { DataTable } from './src/components/ui/data-table'

type Row = { id: number; name: string; self: boolean }
const ROWS: Row[] = Array.from({ length: 6 }, (_, i) => ({ id: i + 1, name: 'row' + (i + 1), self: i === 0 }))

function Demo() {
  const [sel, setSel] = React.useState<Array<string | number>>([])
  const [sort, setSort] = React.useState<any>(null)
  const [page, setPage] = React.useState(1)
  const rows = page === 1 ? ROWS.slice(0, 3) : ROWS.slice(3)
  ;(window as any).__state = { sel, sort, page }
  return (
    <DataTable<Row>
      columns={[
        { key: 'id', title: 'ID', sortable: true },
        { key: 'name', title: 'Name' },
        { key: 'secret', title: 'Secret', defaultVisible: false, render: r => 'S' + r.id },
        { key: 'ops', title: 'Ops', required: true, render: () => 'op' },
      ]}
      rows={rows} rowKey={r => r.id} total={6} page={page} pageSize={3} onPageChange={setPage}
      sort={sort} onSortChange={setSort}
      selectable selectedKeys={sel} onSelectionChange={setSel}
      isRowDisabled={r => r.self}
      batchActions={({ selectedKeys, clear }) => <button id="batch" onClick={() => { (window as any).__batch = selectedKeys; clear() }}>run</button>}
      storageKey="check"
    />
  )
}

ReactDOM.createRoot(document.getElementById('root')!).render(<I18nProvider><Demo /></I18nProvider>)
