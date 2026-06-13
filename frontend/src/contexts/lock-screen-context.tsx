import React, { createContext, useContext, useState, useCallback } from 'react'

interface LockScreenContextType {
  isLocked: boolean
  lock: () => void
  unlock: (password: string) => Promise<boolean>
}

const LockScreenContext = createContext<LockScreenContextType | null>(null)

export const useLockScreen = () => {
  const ctx = useContext(LockScreenContext)
  if (!ctx) throw new Error('useLockScreen must be used within LockScreenProvider')
  return ctx
}

export const LockScreenProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [isLocked, setIsLocked] = useState(() => {
    return localStorage.getItem('lock-screen') === 'true'
  })

  const lock = useCallback(() => {
    localStorage.setItem('lock-screen', 'true')
    setIsLocked(true)
  }, [])

  const unlock = useCallback(async (password: string): Promise<boolean> => {
    try {
      const token = localStorage.getItem('auth_token')
      const res = await fetch('/api/v1/auth/verify-password', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({ password }),
      })
      if (res.ok) {
        localStorage.removeItem('lock-screen')
        setIsLocked(false)
        return true
      }
      return false
    } catch {
      return false
    }
  }, [])

  return (
    <LockScreenContext.Provider value={{ isLocked, lock, unlock }}>
      {children}
    </LockScreenContext.Provider>
  )
}
