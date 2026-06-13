import React, { useState } from 'react'
import { Download } from 'lucide-react'

interface ColumnDef {
  key: string
  title: string
  render?: (value: any, row: any) => string
}

interface ExportButtonProps {
  columns: ColumnDef[]
  fetcher: () => Promise<any[]>
  filename?: string
}

export const ExportButton: React.FC<ExportButtonProps> = ({ columns, fetcher, filename }) => {
  const [loading, setLoading] = useState(false)

  const handleExport = async () => {
    setLoading(true)
    try {
      const data = await fetcher()
      const bom = '\uFEFF'
      const header = columns.map(c => c.title).join(',')
      const rows = data.map(row =>
        columns.map(col => {
          let val = col.render ? col.render(row[col.key], row) : row[col.key]
          const s = String(val ?? '')
          return s.includes(',') || s.includes('"') || s.includes('\n')
            ? '"' + s.replace(/"/g, '""') + '"'
            : s
        }).join(',')
      )
      const csv = bom + header + '\n' + rows.join('\n')
      const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = filename || `export-${Date.now()}.csv`
      a.click()
      URL.revokeObjectURL(url)
    } catch (err) {
      console.error('export error', err)
    } finally {
      setLoading(false)
    }
  }

  return (
    <button onClick={handleExport} disabled={loading}
      className="flex items-center gap-1.5 px-3 py-1.5 bg-white border border-gray-300 text-slate-600 rounded-lg hover:bg-slate-50 disabled:opacity-50 text-sm">
      <Download className="w-4 h-4" />
      {loading ? '导出中...' : '导出'}
    </button>
  )
}
