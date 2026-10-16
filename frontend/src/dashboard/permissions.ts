import { useEffect, useState } from 'react'
import { api } from '../lib/api'

/** 'all' for a super role, otherwise the set of permission keys. Empty until loaded. */
export type Granted = 'all' | Set<string>

let cached: Promise<Granted> | null = null

const load = (): Promise<Granted> => {
  cached ??= api.get<{ all: boolean; granted: string[] }>('/auth/permissions')
    .then(res => (res.all ? 'all' as const : new Set(res.granted)))
    .catch(() => { cached = null; return new Set<string>() })
  return cached
}

/** Forget the cached answer, e.g. after signing in as someone else. */
export const resetPermissions = () => { cached = null }

export function useMyPermissions(): Granted {
  const [granted, setGranted] = useState<Granted>(() => new Set())
  useEffect(() => {
    let active = true
    load().then(value => { if (active) setGranted(value) })
    return () => { active = false }
  }, [])
  return granted
}
