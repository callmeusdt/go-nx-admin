import React from 'react'
import { createRoot } from 'react-dom/client'
import { DataTable, type DataTableColumn } from '../src/components/ui/data-table'
import { I18nProvider } from '../src/contexts/i18n-context'
import { TooltipProvider } from '../src/components/ui/tooltip'
import '../src/index.css'

interface Row {
  id: number
  uid: string
  account: string
  date: string
  depositCount: string
  depositAmount: string
  betAmount: string
  betFee: string
  profit: string
  note: string
}

const rows: Row[] = [
  { id: 1, uid: '1001', account: 'alice', date: '2026-11-01', depositCount: '3', depositAmount: '100.00000000', betAmount: '50.00000000', betFee: '1.00000000', profit: '-10.00000000', note: 'first' },
]

const columns: DataTableColumn<Row>[] = [
  {
    key: 'member', title: '会员', children: [
      { key: 'uid', title: 'UID', required: true, sortable: true, render: row => row.uid },
      { key: 'account', title: '账号', render: row => row.account },
    ],
  },
  { key: 'date', title: '日期', render: row => row.date },
  {
    key: 'deposit', title: '充值', children: [
      { key: 'depositCount', title: '人数', align: 'right', render: row => row.depositCount },
      { key: 'depositAmount', title: '金额', align: 'right', render: row => row.depositAmount },
    ],
  },
  {
    key: 'games', title: '游戏汇总', children: [
      {
        key: 'bet', title: '投注', children: [
          { key: 'betAmount', title: '金额', align: 'right', render: row => row.betAmount },
          { key: 'betFee', title: '手续费', align: 'right', render: row => row.betFee },
        ],
      },
      { key: 'profit', title: '盈亏', align: 'right', render: row => row.profit },
    ],
  },
  { key: 'note', title: '备注', render: row => row.note },
  { key: 'actions', title: '操作', required: true, align: 'center', render: () => <button type="button">View</button> },
]

const Fixture: React.FC = () => {
  const rtl = new URLSearchParams(window.location.search).has('rtl')
  return (
    <TooltipProvider>
    <I18nProvider options={{ defaultLocale: 'en-US' }}>
      <main id="grouped-fixture" dir={rtl ? 'rtl' : undefined}>
        <DataTable
          columns={columns}
          rows={rows}
          rowKey={row => row.id}
          minWidth={1600}
          selectable
          selectedKeys={[]}
          onSelectionChange={() => undefined}
          stickyStart={2}
          stickyEnd={1}
          storageKey="fixture.grouped-header"
          defaultSort={{ field: 'uid', order: 'asc' }}
          onSortChange={() => undefined}
          toolbar={<span>grouped header</span>}
        />
      </main>
    </I18nProvider>
    </TooltipProvider>
  )
}

createRoot(document.getElementById('root')!).render(<Fixture />)
