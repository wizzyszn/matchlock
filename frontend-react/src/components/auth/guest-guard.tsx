import { Navigate, Outlet } from 'react-router-dom'

import { AuthTransitionLoader } from '@/components/auth/auth-transition-loader'
import { useSessionQuery } from '@/hooks/queries/use-session'

export function GuestGuard() {
  const { data: session, isLoading } = useSessionQuery()

  if (isLoading) {
    return (
      <AuthTransitionLoader
        subtitle="Preparing sign-in…"
      />
    )
  }

  if (session) {
    return <Navigate to="/markets" replace />
  }

  return <Outlet />
}