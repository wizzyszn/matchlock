import { useEffect, useRef } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { LogIn } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { useSessionQuery } from '@/hooks/queries/use-session'

export function SessionExpiredBanner() {
  const navigate = useNavigate()
  const location = useLocation()
  const { data: session, isLoading, isFetched } = useSessionQuery()
  const hadSessionRef = useRef(false)

  useEffect(() => {
    if (session) {
      hadSessionRef.current = true
    }
  }, [session])

  const isPublicRoute =
    location.pathname.startsWith('/login') ||
    location.pathname.startsWith('/auth/verify')

  if (isLoading || !isFetched || isPublicRoute) return null

  const expired = hadSessionRef.current && !session
  if (!expired) return null

  return (
    <div
      className="border-b border-destructive/20 bg-destructive/10 px-4 py-3"
      role="alert"
    >
      <div className="mx-auto flex max-w-5xl flex-wrap items-center justify-between gap-3">
        <p className="text-sm">
          Your session expired. Sign in again to continue.
        </p>
        <Button size="sm" onClick={() => navigate('/login')}>
          <LogIn className="mr-2 size-4" />
          Sign in
        </Button>
      </div>
    </div>
  )
}