import { useEffect, useRef, useState } from 'react'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from './ui/dialog'
import { Input } from './ui/input'
import { Button } from './ui/button'
import { useI18n } from '../contexts/i18n-context'
import { sessionFetch } from '../lib/session'

export function Reauthenticate() {
  const { t } = useI18n()
  const copy = { title: t('reauth.title'), password: t('reauth.password'), code: t('reauth.code'), submit: t('reauth.submit'), cancel: t('reauth.cancel'), error: t('reauth.error') }
  const [open,setOpen]=useState(false),[password,setPassword]=useState(''),[code,setCode]=useState(''),[busy,setBusy]=useState(false),[error,setError]=useState('')
  const waiting=useRef<((result:boolean)=>void)[]>([])
  const finish=(value:boolean)=>{for(const resolve of waiting.current)resolve(value);waiting.current=[];setOpen(false);setPassword('');setCode('');setError('')}
  useEffect(()=>{
    const start=(event:Event)=>{waiting.current.push((event as CustomEvent<(result:boolean)=>void>).detail);setOpen(true)}
    window.addEventListener('nx-reauthenticate',start)
    return()=>{window.removeEventListener('nx-reauthenticate',start);for(const resolve of waiting.current)resolve(false);waiting.current=[]}
  },[])
  return <Dialog open={open} onOpenChange={value=>{if(!value&&!busy)finish(false)}}><DialogContent><DialogHeader><DialogTitle>{copy.title}</DialogTitle></DialogHeader>
    <form className="space-y-4" onSubmit={async event=>{event.preventDefault();if(busy)return;setBusy(true);setError('');try{const response=await sessionFetch('/api/v1/auth/verify-password',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({password,code})});if(response.ok)finish(true);else setError(copy.error)}catch{setError(copy.error)}finally{setBusy(false)}}}>
      <label className="block">{copy.password}<Input type="password" autoComplete="current-password" value={password} onChange={e=>setPassword(e.target.value)} required/></label>
      <label className="block">{copy.code}<Input value={code} onChange={e=>setCode(e.target.value)} autoComplete="one-time-code" inputMode="numeric" maxLength={6} required/></label>
      {error?<p role="alert" className="text-red-600">{error}</p>:null}
      <div className="flex gap-2"><Button disabled={busy}>{copy.submit}</Button><Button type="button" variant="outline" disabled={busy} onClick={()=>finish(false)}>{copy.cancel}</Button></div>
    </form>
  </DialogContent></Dialog>
}
