import { useState } from 'react'
import { Loader2, Mail,  Sparkles, } from 'lucide-react'

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

export function LoginPage() {
  const { requestMagicLink } = useAuthMutations()
  const [email, setEmail] = useState('')
  const [sent, setSent] = useState(false)

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault()
    await requestMagicLink.mutateAsync(email.trim())
    setSent(true)
  }

  return (
    <div className="w-full space-y-6">
      <div className="space-y-3 text-center">
        <div className="mx-auto flex size-14 items-center justify-center rounded-2xl border bg-card shadow-sahara">
          <Sparkles className="size-7 text-primary" aria-hidden />
        </div>
        <h1 className="font-heading text-3xl tracking-tight sm:text-4xl">
          Enter Matchlock
        </h1>
        <p className="mx-auto max-w-sm text-sm text-muted-foreground">
          Passwordless sign-in for peer-to-peer wagers on Solana devnet.
        </p>
      </div>

      <Card className="border-border/80 shadow-sahara">
        <CardHeader className="pb-4 text-center">
          <CardTitle className="text-xl">Magic link sign-in</CardTitle>
          <CardDescription>
            We&apos;ll email a one-time link — no seed phrase required to sign
            in.
          </CardDescription>
        </CardHeader>

        <CardContent>
          {sent ? (
            <div className="space-y-4 text-center">
              <Mail className="mx-auto size-10 text-primary" aria-hidden />
              <div className="space-y-1">
                <p className="font-medium">Check your inbox</p>
                <p className="text-sm text-muted-foreground">
                  If{' '}
                  <span className="font-medium text-foreground">{email}</span>{' '}
                  is registered, your secure sign-in link is on its way.
                </p>
              </div>
              <Button variant="outline" onClick={() => setSent(false)}>
                Use a different email
              </Button>
            </div>
          ) : (
            <form onSubmit={handleSubmit} className="space-y-4">
              <div className="space-y-2">
                <label htmlFor="email" className="text-sm font-medium">
                  Email address
                </label>
                <Input
                  id="email"
                  type="email"
                  autoComplete="email"
                  placeholder="you@example.com"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  required
                  className="min-h-11"
                />
              </div>

              {requestMagicLink.error ? (
                <p className="text-sm text-destructive" role="alert">
                  {requestMagicLink.error instanceof ApiClientError &&
                  requestMagicLink.error.code === 'RATE_LIMITED'
                    ? 'You’ve requested several links recently. Wait about a minute, then try again (max 8 per hour).'
                    : requestMagicLink.error instanceof Error
                      ? requestMagicLink.error.message
                      : 'Could not send magic link'}
                </p>
              ) : null}

              <Button
                type="submit"
                className="w-full min-h-11"
                disabled={requestMagicLink.isPending || !email.trim()}
              >
                {requestMagicLink.isPending ? (
                  <>
                    <Loader2 className="mr-2 size-4 animate-spin" aria-hidden />
                    Sending secure link…
                  </>
                ) : (
                  'Email me a sign-in link'
                )}
              </Button>
            </form>
          )}
        </CardContent>
      </Card>
    </div>
  )
}