import React, { useEffect, useState } from 'react'
import { api } from '../../lib/api'
import { SystemConfig } from '../../types'
import { useI18n } from '../../contexts/i18n-context'

const builtinConfigKeys = new Set(['site_name', 'site_logo', 'upload_max_size', 'upload_allow_types'])

export const SystemConfigPage: React.FC = () => {
  const { t } = useI18n()
  const [groups, setGroups] = useState<Record<string, SystemConfig[]>>({})
  const [values, setValues] = useState<Record<string, string>>({})
  const [saving, setSaving] = useState(false)
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')
  const [loaded, setLoaded] = useState(false)

  useEffect(() => {
    api.get<{ data: Record<string, SystemConfig[]> }>('/system-configs').then(res => {
      setGroups(res.data)
      const v: Record<string, string> = {}
      Object.values(res.data).flat().forEach(item => { v[item.key] = item.value })
      setValues(v)
      setLoaded(true)
    }).catch(() => setError(t('ui.request_failed')))
  }, [])

  const handleSave = async () => {
    if (!loaded || saving) return
    setSaving(true)
    setError('')
    setMessage('')
    try {
      await api.put('/system-configs', values)
      setMessage(t('system_config.saved'))
      setTimeout(() => setMessage(''), 2000)
    } catch {
      setError(t('ui.request_failed'))
    } finally {
      setSaving(false)
    }
  }

  const groupLabel = (group: string) => {
    if (group === '站点信息') return t('system_config.groups.site')
    if (group === '上传设置') return t('system_config.groups.upload')
    return group
  }

  const fieldLabel = (item: SystemConfig) => {
    return builtinConfigKeys.has(item.key) ? t(`system_config.fields.${item.key}`) : item.label
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h2 className="text-2xl font-bold text-slate-800">{t('system_config.title')}</h2>
        <button onClick={handleSave} disabled={saving || !loaded}
          className="shrink-0 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 text-sm">
          {saving ? t('system_config.saving') : t('system_config.save_all')}
        </button>
      </div>

      {message && <div className="bg-green-50 text-green-700 px-4 py-2 rounded-lg text-sm">{message}</div>}
      {error && <p role="alert" className="text-red-600">{error}</p>}
      {!loaded && !error && <p role="status">{t('common.loading')}</p>}

      {Object.entries(groups).map(([group, items]) => (
        <div key={group} className="bg-white rounded-xl border border-gray-100 shadow-sm p-4 sm:p-6 min-w-0">
          <h3 className="text-lg font-semibold text-slate-800 mb-4 break-words">{groupLabel(group)}</h3>
          <div className="space-y-4">
            {items.map(item => (
              <div key={item.key} className="min-w-0">
                <label className="block text-sm font-medium text-slate-700 mb-1 break-words">{fieldLabel(item)}</label>
                {item.description && <p className="text-xs text-slate-400 mb-1 break-words">{item.description}</p>}
                {item.type === 'bool' ? (
                  <label className="inline-flex items-center gap-2 cursor-pointer">
                    <input type="checkbox" checked={values[item.key] === 'true'}
                      onChange={e => setValues(v => ({ ...v, [item.key]: e.target.checked ? 'true' : 'false' }))} />
                    <span className="text-sm">{values[item.key] === 'true' ? t('system_config.yes') : t('system_config.no')}</span>
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
