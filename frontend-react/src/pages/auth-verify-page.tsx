import { useEffect, useRef, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { AlertCircle, CheckCircle2 } from 'lucide-react'
import { useQueryClient } from '@tanstack/react-query'

import { AuthTransitionLoader } from '@/components/auth/auth-transition-loader'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { useAuthMutations } from '@/hooks/mutations/use-auth-mutations'
import { useApi } from '@/hooks/use-api'
import { delay } from '@/hooks/use-auth-transition'
import {
  ApiClientError,
  MatchlockApi,
  type UserProfile,
} from '@/lib/api'
import { mapAuthError } from '@/lib/errors'
import { queryKeys } from '@/lib/query-keys'

type VerifyStatus =
  | 'verifying'
  | 'securing'
  | 'expired'
  | 'already_signed_in'

async function resolveSession(api: MatchlockApi): Promise<UserProfile | null> {
  try {
    return await api.getMe()
  } catch (error) {
    if (error instanceof ApiClientError && error.status === 401) {
      try {
        return await api.refreshSession()
      } catch {
        return null
      }
    }
    return null
  }
}

export function AuthVerifyPage() {
  const [params] = useSearchParams()
  const navigate = useNavigate()
  const api = useApi()
  const queryClient = useQueryClient()
  const token = params.get('token') ?? ''
  const { verifyMagicLink } = useAuthMutations()
  const verifyAttemptedRef = useRef(false)
  const [status, setStatus] = useState<VerifyStatus>('verifying')
  const [errorMessage, setErrorMessage] = useState<string | null>(null)

  useEffect(() => {
    if (!token || verifyAttemptedRef.current) return
    verifyAttemptedRef.current = true

    const run = async () => {
      try {
        await verifyMagicLink.mutateAsync(token)
        setStatus('securing')
        await delay(2200)
        navigate('/markets', { replace: true })
      } catch (error) {
        const existing = await resolveSession(api)
        if (existing) {
          queryClient.setQueryData(queryKeys.auth.session, existing)
          setStatus('already_signed_in')
          await delay(1600)
          navigate('/markets', { replace: true })
          return
        }

        setErrorMessage(mapAuthError(error))
        setStatus('expired')
      }
    }

    void run()
  }, [token, verifyMagicLink, navigate, api, queryClient])

  if (!token) {
    return (
      <div className="w-full space-y-4 text-center">
        <h1 className="font-heading text-2xl">Invalid link</h1>
        <p className="text-sm text-muted-foreground">
          This sign-in link is missing or malformed.
        </p>
        <Button onClick={() => navigate('/login')}>Request a new link</Button>
      </div>
    )
  }

  if (status === 'expired') {
    return (
      <Card className="w-full border-border/80 shadow-sahara">
        <CardHeader className="text-center">
          <AlertCircle
            className="mx-auto size-10 text-destructive"
            aria-hidden
          />
          <CardTitle className="text-xl">Link expired</CardTitle>
          <CardDescription>
            {errorMessage ??
              'This sign-in link is no longer valid. Request a fresh one below.'}
          </CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col gap-3">
          <Button className="min-h-11" onClick={() => navigate('/login')}>
            Request a new sign-in link
          </Button>
          <Button variant="outline" onClick={() => navigate('/login')}>
            Back to sign in
          </Button>
        </CardContent>
      </Card>
    )
  }

  if (status === 'already_signed_in') {
    return (
      <AuthTransitionLoader
        title="Already signed in"
        subtitle="This link was already used. Taking you to Matchlock…"
        icon={
          <CheckCircle2 className="size-10 text-primary" aria-hidden />
        }
      />
    )
  }

  const loaderCopy =
    status === 'securing'
      ? {
          title: 'Securing your session',
          subtitle:
            'Establishing your Matchlock profile and devnet session…',
        }
      : {
          title: 'Verifying magic link',
          subtitle:
            'Confirming your sign-in token on-chain-ready infrastructure…',
        }

  return <AuthTransitionLoader {...loaderCopy} />
}
