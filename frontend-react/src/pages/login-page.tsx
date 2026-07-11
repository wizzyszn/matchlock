import { Loader2, Mail } from 'lucide-react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useState } from 'react'
import { Link, useLocation } from 'react-router-dom'

import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { useAuthMutations } from '@/hooks/mutations/use-auth-mutations'
import { ApiClientError } from '@/lib/api'
import MatchLockLogo from '@/components/brand/matchlock-logo'
import { PoweredByTxLine } from '@/components/brand/powered-by-txline'

const REDIRECT_KEY = 'post_auth_redirect'

const loginSchema = z.object({
  email: z
    .string()
    .min(1, 'Email is required')
    .email('Enter a valid email address'),
})

type LoginFormValues = z.infer<typeof loginSchema>

export function LoginPage() {
  const { requestMagicLink } = useAuthMutations()
  const [sentTo, setSentTo] = useState<string | null>(null)
  const location = useLocation()
  
  const onSubmit = async ({ email }: LoginFormValues) => {
    await requestMagicLink.mutateAsync(email.trim())
    const from = (location.state as { from?: string } | null)?.from
    if (from) {
      sessionStorage.setItem(REDIRECT_KEY, from)
    }
    setSentTo(email.trim())
  }

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<LoginFormValues>({
    resolver: zodResolver(loginSchema),
  })

  return (
    <div className="w-full space-y-8">
      {/* Brand + headline */}
      <div className="space-y-4 text-center">
        <MatchLockLogo />
        <div className="space-y-2">
          <h1 className="font-heading text-3xl tracking-tight sm:text-4xl">
            Welcome back
          </h1>
          <p className="mx-auto max-w-xs text-sm text-muted-foreground">
            Peer-to-peer wagers on Solana-
          </p>
        </div>
      </div>

      <Card className="border-border/80 shadow-sahara">
        {!sentTo && (
          <CardHeader className="pb-2 text-center">
            <CardTitle className="text-lg">Sign in with a magic link</CardTitle>
            <CardDescription className="mt-1">
              Enter your email and we&apos;ll send a one-time link to your inbox.
            </CardDescription>
          </CardHeader>
        )}

        <CardContent className="pt-6">
          {sentTo ? (
            /* ── Success state ── */
            <div className="flex flex-col items-center gap-5 py-2 text-center">
              <div className="flex size-14 items-center justify-center rounded-full bg-primary/10">
                <Mail className="size-7 text-primary" aria-hidden />
              </div>
              <div className="space-y-1.5">
                <p className="font-semibold">Check your inbox</p>
                <p className="text-sm text-muted-foreground">
                  A magic link was sent to{' '}
                  <span className="font-medium text-foreground">{sentTo}</span>.
                  It expires in 15 minutes.
                </p>
              </div>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setSentTo(null)}
              >
                Use a different email
</Button>
            </div>
          ) : (
            /* ── Form state ── */
            <form onSubmit={handleSubmit(onSubmit)} className="space-y-5" noValidate>
              <div className="space-y-1.5">
                <label htmlFor="email" className="text-sm font-medium">
                  Email address
                </label>
                <Input
                  id="email"
                  type="email"
                  autoComplete="email"
                  placeholder="you@example.com"
                  className="min-h-11"
                  aria-invalid={!!errors.email}
                  {...register('email')}
                />
                {errors.email && (
                  <p className="text-xs text-destructive" role="alert">
                    {errors.email.message}
                  </p>
                )}
              </div>

              {requestMagicLink.isError && (
                <p className="text-sm text-destructive" role="alert">
                  {requestMagicLink.error instanceof ApiClientError &&
                  requestMagicLink.error.code === 'RATE_LIMITED'
                    ? 'Too many requests — wait a minute and try again.'
                    : requestMagicLink.error instanceof Error
                      ? requestMagicLink.error.message
                      : 'Could not send magic link. Please try again.'}
                </p>
              )}

              <p className="text-center text-xs text-muted-foreground">
                By signing in, you agree to our{' '}
                <Link to="/terms" className="underline underline-offset-2 hover:text-foreground">
                  Terms & Conditions
                </Link>
                .
              </p>

              <Button
                type="submit"
                className="w-full min-h-11"
                disabled={isSubmitting || requestMagicLink.isPending}
              >
                {requestMagicLink.isPending || isSubmitting ? (
                  <>
                    <Loader2 className="mr-2 size-4 animate-spin" aria-hidden />
                    Sending link…
                  </>
                ) : (
                  'Send magic link'
                )}
              </Button>
            </form>
          )}
        </CardContent>
      </Card>

      <div className="space-y-1 text-center">
        <p className="text-xs font-medium text-amber-500/80">
          Gamble responsibly. Must be 18+.
        </p>
        <p className="pt-2 text-xs text-muted-foreground">
          <PoweredByTxLine />
        </p>
      </div>
    </div>
  )
}