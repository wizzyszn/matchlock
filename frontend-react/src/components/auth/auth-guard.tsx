import { Navigate, Outlet, useLocation } from 'react-router-dom'

import { AuthTransitionLoader } from '@/components/auth/auth-transition-loader'
import { useSessionQuery } from '@/hooks/queries/use-session'

export function AuthGuard() {
  const location = useLocation()
  const { data: session, isLoading, isFetching } = useSessionQuery()

  if (isLoading || (isFetching && !session)) {
    return (
      <AuthTransitionLoader
        title="Checking session"
  
      />
    )
  }

  if (!session) {
    return <Navigate to="/login" replace state={{ from: location.pathname }} />
  }

  return <Outlet />
}