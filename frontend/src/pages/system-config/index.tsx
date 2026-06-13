import React, { useEffect, useState } from 'react'
import { api } from '../../lib/api'
import { SystemConfig } from '../../types'

export const SystemConfigPage: React.FC = () => {
  const [groups, setGroups] = useState<Record<string, SystemConfig[]>>({})
  const [values, setValues] = useState<Record<string, string>>({})
  const [saving, setSaving] = useState(false)
  const [message, setMessage] = useState('')

  useEffect(() => {
    api.get<{ data: Record<string, SystemConfig[]> }>('/system-configs').then(res => {
      setGroups(res.data)
      const v: Record<string, string> = {}
      Object.values(res.data).flat().forEach(item => { v[item.key] = item.value })
      setValues(v)
    })
  }, [])

  const handleSave = async () => {
    setSaving(true)
    try {
      await api.put('/system-configs', values)
      setMessage('保存成功')
      setTimeout(() => setMessage(''), 2000)
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold text-slate-800">系统配置</h2>
        <button onClick={handleSave} disabled={saving}
          className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 text-sm">
          {saving ? '保存中...' : '保存全部'}
        </button>
      </div>

      {message && <div className="bg-green-50 text-green-700 px-4 py-2 rounded-lg text-sm">{message}</div>}

      {Object.entries(groups).map(([group, items]) => (
        <div key={group} className="bg-white rounded-xl border border-gray-100 shadow-sm p-6">
          <h3 className="text-lg font-semibold text-slate-800 mb-4">{group}</h3>
          <div className="space-y-4">
            {items.map(item => (
              <div key={item.key}>
                <label className="block text-sm font-medium text-slate-700 mb-1">{item.label}</label>
                {item.description && <p className="text-xs text-slate-400 mb-1">{item.description}</p>}
                {item.type === 'bool' ? (
                  <label className="inline-flex items-center gap-2 cursor-pointer">
                    <input type="checkbox" checked={values[item.key] === 'true'}
                      onChange={e => setValues(v => ({ ...v, [item.key]: e.target.checked ? 'true' : 'false' }))} />
                    <span className="text-sm">{values[item.key] === 'true' ? '是' : '否'}</span>
                  </label>
                ) : (
                  <input type={item.type === 'number' ? 'number' : 'text'} value={values[item.key] || ''}
                    onChange={e => setValues(v => ({ ...v, [item.key]: e.target.value }))}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:border-blue-500 focus:ring-1 focus:ring-blue-500 outline-none text-sm" />
                )}
              </div>
            ))}
          </div>
        </div>
      ))}
    </div>
  )
}
