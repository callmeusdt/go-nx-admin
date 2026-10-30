import React, { useState } from 'react'
import { sessionFetch, clearSession } from '../lib/session'
import { useI18n } from '../contexts/i18n-context'
import { Button } from '../components/ui/button'
import { Input } from '../components/ui/input'
import { Alert } from '../components/ui/alert'
import { FormField } from '../components/ui/form-field'

// Optional enrollment uses the same white controls as the existing admin.
export const EnrollmentPage: React.FC = () => {
  const { t } = useI18n()
  const copy = { title: t('enrollment.title'), hint: t('enrollment.hint'), error: t('enrollment.error'), save: t('enrollment.save'), continue: t('enrollment.continue'), setup: t('enrollment.setup'), code: t('enrollment.code'), enable: t('enrollment.enable'), logout: t('enrollment.logout'), secret: t('enrollment.secret') }
  const [secret,setSecret]=useState(''),[code,setCode]=useState(''),[codes,setCodes]=useState<string[]>([])
  const [busy,setBusy]=useState(false),[error,setError]=useState('')
  const call=async(path:string,body:object={})=>{
    const response=await sessionFetch('/api/v1/auth/'+path,{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(body)})
    const data=await response.json();if(!response.ok)throw new Error(data.code||data.message);return data
  }
  const run=async(action:()=>Promise<void>)=>{if(busy)return;setBusy(true);setError('');try{await action()}catch{setError(copy.error)}finally{setBusy(false)}}
  return <main className="min-h-screen bg-gray-50 flex items-center justify-center p-6">
    <section className="bg-white border rounded-xl p-8 w-full max-w-lg space-y-4">
      <h1 className="text-xl font-semibold">{copy.title}</h1>
      <p>{copy.hint}</p>
      {error?<Alert>{error}</Alert>:null}
      {codes.length?<><p>{copy.save}</p><pre className="rounded-md bg-gray-50 p-4 font-mono text-sm">{codes.join('\n')}</pre><div className="flex flex-wrap gap-3"><Button onClick={()=>location.reload()}>{copy.continue}</Button></div></>:<>
        {!secret?<Button disabled={busy} onClick={()=>void run(async()=>{const data=await call('mfa/setup');setSecret(data.secret)})}>{copy.setup}</Button>:<form className="space-y-4" onSubmit={e=>{e.preventDefault();void run(async()=>{const data=await call('mfa/enable',{code});setCodes(data.recovery_codes);setSecret('');setCode('')})}}>
          <p className="break-all font-mono" aria-label={copy.secret}>{secret}</p>
          <FormField label={copy.code}>{id=><Input id={id} value={code} onChange={e=>setCode(e.target.value)} autoComplete="one-time-code" inputMode="numeric" maxLength={6}/>}</FormField>
          <Button disabled={busy||code.length!==6}>{copy.enable}</Button>
        </form>}
      </>}
       <div className="flex flex-wrap gap-3"><Button variant="outline" disabled={busy} onClick={()=>void run(async()=>{await call('logout');clearSession();location.href='/login'})}>{copy.logout}</Button></div>
    </section>
  </main>
}
