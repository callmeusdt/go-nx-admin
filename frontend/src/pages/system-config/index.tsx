import React, { useEffect, useState } from 'react'
import { api } from '../../lib/api'
import { SystemConfig } from '../../types'
import { useI18n } from '../../contexts/i18n-context'
import { Switch } from '../../components/ui/switch'
import { Input } from '../../components/ui/input'
import { Button } from '../../components/ui/button'
import { Alert } from '../../components/ui/alert'
import { FormField } from '../../components/ui/form-field'
import { PageHeader } from '../../components/ui/page-header'

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
      <PageHeader
        title={t('system_config.title')}
        actions={<Button size="sm" onClick={handleSave} disabled={saving || !loaded}>{saving ? t('system_config.saving') : t('system_config.save_all')}</Button>}
      />

      {message && <Alert variant="success">{message}</Alert>}
      {error && <Alert>{error}</Alert>}
      {!loaded && !error && <Alert variant="info">{t('common.loading')}</Alert>}

      {Object.entries(groups).map(([group, items]) => (
        <div key={group} className="bg-white rounded-xl border border-gray-100 shadow-sm p-4 sm:p-6 min-w-0">
          <h3 className="text-lg font-semibold text-slate-800 mb-4 break-words">{groupLabel(group)}</h3>
          <div className="space-y-4">
            {items.map(item => (
              <FormField key={item.key} label={fieldLabel(item)} hint={item.description}>{id => (
                item.type === 'bool' ? (
                  <Switch
                    checked={values[item.key] === 'true'}
                    onCheckedChange={on => setValues(v => ({ ...v, [item.key]: on ? 'true' : 'false' }))}
                    aria-label={fieldLabel(item)}
                    label={values[item.key] === 'true' ? t('system_config.yes') : t('system_config.no')}
                  />
                ) : (
                  <Input id={id} type={item.type === 'number' ? 'number' : 'text'} value={values[item.key] || ''}
                    onChange={e => setValues(v => ({ ...v, [item.key]: e.target.value }))} />
                )
              )}</FormField>
            ))}
          </div>
        </div>
      ))}
    </div>
  )
}
