import React, { useState } from 'react'
import { sessionFetch, clearSession } from '../lib/session'
import { useI18n } from '../contexts/i18n-context'

// Mandatory enrollment uses the same white controls as the existing admin.
export const EnrollmentPage: React.FC = () => {
  const { locale } = useI18n()
  const copy = locale === 'zh-CN' ? {title:'设置两步验证',hint:'管理操作前需要启用认证器。',error:'操作失败，请重试',save:'请安全保存恢复码。',continue:'已保存，继续',setup:'设置认证器',code:'验证码',enable:'启用',logout:'退出'} : {title:'Set up two-step verification',hint:'An authenticator is required for admin access.',error:'Request failed. Try again.',save:'Save these recovery codes securely.',continue:'Saved, continue',setup:'Set up authenticator',code:'Verification code',enable:'Enable',logout:'Sign out'}
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
      {error?<p role="alert" className="text-red-600">{error}</p>:null}
      {codes.length?<><p>{copy.save}</p><pre className="p-4 bg-gray-50">{codes.join('\n')}</pre><button className="border rounded px-4 py-2" onClick={()=>location.reload()}>{copy.continue}</button></>:<>
        {!secret?<button disabled={busy} className="border rounded px-4 py-2" onClick={()=>void run(async()=>{const data=await call('mfa/setup');setSecret(data.secret)})}>{copy.setup}</button>:<form className="space-y-4" onSubmit={e=>{e.preventDefault();void run(async()=>{const data=await call('mfa/enable',{code});setCodes(data.recovery_codes);setSecret('');setCode('')})}}>
          <p className="break-all font-mono" aria-label="Authenticator secret">{secret}</p>
          <label className="block">{copy.code}<input className="block w-full border rounded p-2" value={code} onChange={e=>setCode(e.target.value)} autoComplete="one-time-code" inputMode="numeric" maxLength={6}/></label>
          <button className="bg-blue-600 text-white rounded px-4 py-2" disabled={busy||code.length!==6}>{copy.enable}</button>
        </form>}
      </>}
      <button disabled={busy} className="border rounded px-4 py-2" onClick={()=>void run(async()=>{await call('logout');clearSession();location.href='/login'})}>{copy.logout}</button>
    </section>
  </main>
}
