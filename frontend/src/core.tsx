import React, { useEffect, useState } from 'react'
import type { ResourceProps } from '@refinedev/core'
import { Refine } from '@refinedev/core'
import routerBindings from '@refinedev/react-router-v6'
import { BrowserRouter, Routes, Route, Navigate, useLocation } from 'react-router-dom'
import { configureSession, usesCookieSession, sessionFetch } from './lib/session'
import { EnrollmentPage } from './pages/enrollment'
import { Reauthenticate } from './components/reauthenticate'
import { authProvider } from './providers/auth-provider'
import { dataProvider } from './providers/data-provider'
import { TabsProvider } from './contexts/tabs-context'
import { LockScreenProvider } from './contexts/lock-screen-context'
import { I18nProvider, useI18n } from './contexts/i18n-context'
import type { I18nOptions } from './contexts/i18n-context'
import { Layout, RouteLabelsContext } from './components/layout'
import { LoginPage } from './pages/login'
import { MFAVerifyPage } from './pages/mfa-verify'
import { DashboardPage } from './pages/dashboard'
import { UsersPage } from './pages/users'
import { RolesPage } from './pages/roles'
import { MenusPage } from './pages/menus'
import { AuditLogsPage } from './pages/audit-logs'
import { ApisPage } from './pages/apis'
import { LoginLogsPage } from './pages/login-logs'
import { MediaPage } from './pages/media'
import { SystemConfigPage } from './pages/system-config'

const BUILTIN_RESOURCES: ResourceProps[] = [
  { name: 'dashboard', list: '/dashboard' },
  { name: 'users', list: '/users' },
  { name: 'roles', list: '/roles' },
  { name: 'menus', list: '/menus' },
  { name: 'audit-logs', list: '/audit-logs' },
  { name: 'apis', list: '/apis' },
  { name: 'login-logs', list: '/login-logs' },
  { name: 'media', list: '/media' },
  { name: 'system-config', list: '/system-config' },
]

const ProtectedRoute: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { t } = useI18n()
  const location = useLocation()
  const [state, setState] = useState<'loading' | 'ready' | 'login' | 'enroll'>('loading')
  useEffect(() => {
    if (!usesCookieSession()) return
    let active = true
    setState('loading')
    void sessionFetch('/api/v1/auth/me').then(async response => {
      if (!response.ok) { if(active)setState('login'); return }
      const user = await response.json()
      if(active)setState(user.mfa_enabled ? 'ready' : 'enroll')
    }).catch(() => { if(active)setState('login') })
    return () => { active = false }
  }, [location.pathname])
  if (usesCookieSession()) {
    if(state==='loading')return <div role="status" aria-label={t('common.loading')} />
    if(state==='login')return <Navigate to="/login" replace />
    if(state==='enroll')return <EnrollmentPage />
    return <>{children}</>
  }
  const token = localStorage.getItem('auth_token')
  if (!token) return <Navigate to="/login" replace />
  return <>{children}</>
}

const buildRoutes = (extraRoutes?: React.ReactElement[], extraLabels?: Record<string, string>, disabled: string[] = []): React.ReactElement[] => {
  const labels = extraLabels ?? {}
  const builtin = [
    <Route key="/login" path="/login" element={<LoginPage />} />,
    <Route key="/mfa-verify" path="/mfa-verify" element={<MFAVerifyPage />} />,
    <Route
      key="protected"
      element={
        <ProtectedRoute>
          <RouteLabelsContext.Provider value={labels}>
            <Layout />
          </RouteLabelsContext.Provider>
        </ProtectedRoute>
      }
    >
      <Route path="/dashboard" element={<DashboardPage />} />
      <Route path="/users" element={<UsersPage />} />
      <Route path="/roles" element={<RolesPage />} />
      <Route path="/menus" element={<MenusPage />} />
      <Route path="/audit-logs" element={<AuditLogsPage />} />
      <Route path="/apis" element={<ApisPage />} />
      <Route path="/login-logs" element={<LoginLogsPage />} />
      {!disabled.includes('media') && <Route path="/media" element={<MediaPage />} />}
      <Route path="/system-config" element={<SystemConfigPage />} />
      {extraRoutes}
    </Route>,
    <Route key="/" path="/" element={<Navigate to="/dashboard" replace />} />,
    <Route key="*" path="*" element={<Navigate to="/dashboard" replace />} />,
  ]
  return builtin
}

export interface CreateAppOptions {
  extraResources?: ResourceProps[]
  extraRoutes?: React.ReactElement[]
  extraRouteLabels?: Record<string, string>
  sessionMode?: 'bearer' | 'cookie'
  disabledResources?: string[]
  i18n?: I18nOptions
}

export function createApp(opts?: CreateAppOptions): React.FC {
  configureSession(opts?.sessionMode)
  return function App() {
    const resources = [...BUILTIN_RESOURCES.filter(item => !opts?.disabledResources?.includes(item.name)), ...(opts?.extraResources ?? [])]
    return (
      <BrowserRouter>
        <I18nProvider options={opts?.i18n}>
          <Reauthenticate />
          <LockScreenProvider>
            <TabsProvider>
              <Refine
                authProvider={authProvider}
                dataProvider={dataProvider}
                routerProvider={routerBindings}
                resources={resources}
                options={{
                  syncWithLocation: false,
                  warnWhenUnsavedChanges: true,
                  disableTelemetry: true,
                }}
              >
                <Routes>
                  {buildRoutes(opts?.extraRoutes, opts?.extraRouteLabels, opts?.disabledResources)}
                </Routes>
              </Refine>
            </TabsProvider>
          </LockScreenProvider>
        </I18nProvider>
      </BrowserRouter>
    )
  }
}
