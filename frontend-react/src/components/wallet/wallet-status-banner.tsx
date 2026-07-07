import { Link } from 'react-router-dom'
import { AlertTriangle, CheckCircle2, Link2, Wallet } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { useAuthMutations } from '@/hooks/mutations/use-auth-mutations'
import { useWalletLinkStatus } from '@/hooks/use-wallet-link-status'
import { truncatePubkey } from '@/lib/display-name'
import { cn } from '@/lib/utils'

type WalletStatusBannerProps = {
  className?: string
  compact?: boolean
}

function WalletStatusBanner({
  className,
  compact = false,
}: WalletStatusBannerProps) {
  const status = useWalletLinkStatus()
  const { linkWallet } = useAuthMutations()

  if (status.conflict && status.connectedPubkey) {
    return (
      <div
        className={cn(
          'flex flex-wrap items-center justify-between gap-3 rounded-lg border border-destructive/40 bg-destructive/10 px-4 py-3',
          className,
        )}
        role="alert"
      >
        <div className="flex items-start gap-2 text-sm">
          <AlertTriangle className="mt-0.5 size-4 shrink-0 text-destructive" />
          <div>
            <p className="font-medium text-destructive">
              Wallet is not linked to this account
            </p>
            <p className="text-muted-foreground">
              <span className="font-mono">
                {truncatePubkey(status.connectedPubkey)}
              </span>{' '}
              is already linked to a different Matchlock account. Switch
              wallets, or sign in with the account it is linked to.
            </p>
          </div>
        </div>
        <Link
          to="/profile"
          className="inline-flex min-h-8 items-center rounded-md border border-border bg-card px-3 text-sm font-medium hover:bg-muted"
        >
          Profile
        </Link>
      </div>
    )
  }

  if (!status.connected && !status.hasLinkedWallet) {
    return (
      <div
        className={cn(
          'flex flex-wrap items-center justify-between gap-3 rounded-lg border border-dashed bg-muted/30 px-4 py-3',
          className,
        )}
        role="status"
      >
        <div className="flex items-start gap-2 text-sm">
          <Wallet className="mt-0.5 size-4 shrink-0 text-muted-foreground" />
          <div>
            <p className="font-medium">No wallet connected</p>
            <p className="text-muted-foreground">
              Connect a Solana wallet on your{' '}
              <Link to="/profile" className="text-primary hover:underline">
                Profile
              </Link>{' '}
              to create and accept wagers.
            </p>
          </div>
        </div>
        {!compact ? (
          <Link
            to="/profile"
            className="inline-flex min-h-8 items-center rounded-md bg-primary px-3 text-sm font-medium text-primary-foreground hover:bg-primary/90"
          >
            Go to Profile
          </Link>
        ) : null}
      </div>
    )
  }

  if (status.needsLink && status.connectedPubkey) {
    return (
      <div
        className={cn(
          'flex flex-wrap items-center justify-between gap-3 rounded-lg border border-amber-500/30 bg-amber-500/10 px-4 py-3',
          className,
        )}
        role="status"
      >
        <div className="flex items-start gap-2 text-sm">
          <Link2 className="mt-0.5 size-4 shrink-0 text-amber-700 dark:text-amber-300" />
          <div>
            <p className="font-medium text-amber-950 dark:text-amber-50">
              Wallet connected — link required
            </p>
            <p className="text-amber-900/80 dark:text-amber-100/80">
              <span className="font-mono">
                {truncatePubkey(status.connectedPubkey)}
              </span>{' '}
              is connected in your browser but not linked to your Matchlock
              account yet.
            </p>
          </div>
        </div>
        <Button
          size="sm"
          disabled={linkWallet.isPending || !status.canLink}
          onClick={() => linkWallet.mutate(undefined)}
        >
          {linkWallet.isPending ? 'Linking…' : 'Link wallet'}
        </Button>
      </div>
    )
  }

  if (status.mismatch && status.connectedPubkey && status.primaryWallet) {
    return (
      <div
        className={cn(
          'flex flex-wrap items-center justify-between gap-3 rounded-lg border border-amber-500/30 bg-amber-500/10 px-4 py-3',
          className,
        )}
        role="status"
      >
        <div className="flex items-start gap-2 text-sm">
          <AlertTriangle className="mt-0.5 size-4 shrink-0 text-amber-700" />
          <div>
            <p className="font-medium">Different wallet connected</p>
            <p className="text-muted-foreground">
              Connected{' '}
              <span className="font-mono">
                {truncatePubkey(status.connectedPubkey)}
              </span>{' '}
              but your primary linked wallet is{' '}
              <span className="font-mono">
                {truncatePubkey(status.primaryWallet.pubkey)}
              </span>
              .
            </p>
          </div>
        </div>
        <Link
          to="/profile"
          className="inline-flex min-h-8 items-center rounded-md border border-border bg-card px-3 text-sm font-medium hover:bg-muted"
        >
          Switch on Profile
        </Link>
      </div>
    )
  }

  if (status.isLinkedToAccount && status.connectedPubkey) {
    return (
      <div
        className={cn(
          'flex items-center gap-2 rounded-lg border border-primary/20 bg-primary/5 px-4 py-3 text-sm',
          className,
        )}
        role="status"
      >
        <CheckCircle2 className="size-4 shrink-0 text-primary" />
        <p>
          <span className="font-medium">Wallet linked to this account</span>
          {' · '}
          <span className="font-mono text-muted-foreground">
            {truncatePubkey(status.connectedPubkey)}
          </span>
        </p>
      </div>
    )
  }

  return null
}

export { WalletStatusBanner }
export default WalletStatusBanner
