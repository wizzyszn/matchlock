import { useEffect, useState } from 'react'
import { useWallet } from '@solana/wallet-adapter-react'
import { WalletReadyState, type WalletName } from '@solana/wallet-adapter-base'
import {
  Check,
  Copy,
  LogOut,
  Star,
  Trash2,
  Wallet,
} from 'lucide-react'

import { WalletConnectButton } from '@/components/wallet/WalletConnectButton'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { PageHeader, PageHeaderHeading, PageHeaderDescription } from '@/components/ui/page-header'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import { useAuthMutations } from '@/hooks/mutations/use-auth-mutations'
import { useSessionQuery } from '@/hooks/queries/use-session'
import { useWalletLinkStatus } from '@/hooks/use-wallet-link-status'
import {
  displayNameHint,
  isValidDisplayName,
  truncatePubkey,
  userDisplayLabel,
} from '@/lib/display-name'

export function ProfilePage() {
  const { data: session, isLoading } = useSessionQuery()
  const walletStatus = useWalletLinkStatus()
  const {
    updateProfile,
    linkWallet,
    unlinkWallet,
    setPrimaryWallet,
    logout,
  } = useAuthMutations()
  const { connected, publicKey, wallets, select, connect } = useWallet()
  const [username, setUsername] = useState('')
  const [copied, setCopied] = useState<string | null>(null)

  useEffect(() => {
    if (session?.display_name) {
      setUsername(session.display_name)
    }
  }, [session?.display_name])

  if (isLoading || !session) {
    return (
      <div className="mx-auto max-w-2xl space-y-5">
        <div className="space-y-2 mb-6">
          <Skeleton className="h-9 w-32" />
          <Skeleton className="h-4 w-48" />
        </div>
        
        <div className="rounded-xl border bg-card text-card-foreground shadow space-y-4">
           <div className="p-6 pb-3 space-y-2">
              <Skeleton className="h-6 w-24" />
              <Skeleton className="h-4 w-64 max-w-full" />
           </div>
           <div className="p-6 pt-0 space-y-4">
              <div className="space-y-2">
                 <Skeleton className="h-4 w-12" />
                 <Skeleton className="h-4 w-40" />
              </div>
              <div className="space-y-3">
                 <Skeleton className="h-4 w-20" />
                 <Skeleton className="h-10 w-full" />
                 <Skeleton className="h-10 w-32 mt-2" />
              </div>
           </div>
        </div>
      </div>
    )
  }

  const installedWallets = wallets.filter(
    (w) => w.readyState === WalletReadyState.Installed,
  )

  const handleSaveUsername = async (event: React.FormEvent) => {
    event.preventDefault()
    const value = username.trim()
    if (!isValidDisplayName(value)) return
    await updateProfile.mutateAsync({ display_name: value })
  }

  const copyPubkey = async (pubkey: string) => {
    await navigator.clipboard.writeText(pubkey)
    setCopied(pubkey)
    window.setTimeout(() => setCopied(null), 2000)
  }

  return (
    <div className="mx-auto max-w-2xl space-y-5">
      <PageHeader>
        <PageHeaderHeading>Profile</PageHeaderHeading>
        <PageHeaderDescription>
          Identity, linked wallets, and session.
        </PageHeaderDescription>
      </PageHeader>

      <Card>
        <CardHeader className="pb-3">
          <CardTitle>Identity</CardTitle>
          <CardDescription>
            Public username for challenges and wagers.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-1">
            <p className="text-xs font-medium text-muted-foreground">Email</p>
            <p className="text-sm">{session.email}</p>
          </div>

          <form onSubmit={handleSaveUsername} className="space-y-3">
            <div className="space-y-2">
              <label htmlFor="profile-username" className="text-sm font-medium">
                Username
              </label>
              <Input
                id="profile-username"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                placeholder="matchlock_ace"
              />
              <p className="text-xs text-muted-foreground">
                {displayNameHint()}
              </p>
            </div>
            {updateProfile.error ? (
              <p className="text-sm text-destructive" role="alert">
                {updateProfile.error instanceof Error
                  ? updateProfile.error.message
                  : 'Could not save username'}
              </p>
            ) : null}
            <Button
              type="submit"
              disabled={
                !isValidDisplayName(username) ||
                updateProfile.isPending ||
                username.trim() === (session.display_name ?? '')
              }
            >
              {updateProfile.isPending ? 'Saving…' : 'Save username'}
            </Button>
          </form>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="pb-3">
          <CardTitle>Wallets</CardTitle>
          <CardDescription>
            Connect a browser wallet, then link it to this signed-in account
            for on-chain wagers.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex flex-wrap items-center gap-2">
            <WalletConnectButton />
          </div>

          {linkWallet.error ? (
            <p className="text-sm text-destructive" role="alert">
              {linkWallet.error instanceof Error
                ? linkWallet.error.message
                : 'Could not link wallet'}
            </p>
          ) : null}

          {connected && publicKey && (walletStatus.needsLink || walletStatus.conflict) ? (
            <div className="space-y-2 rounded-lg border bg-amber-500/5 px-3 py-3 shadow-sm">
              <div className="flex items-center justify-between">
                <span className="font-mono text-sm font-medium">
                  {truncatePubkey(publicKey.toBase58())}
                </span>
                <span className="rounded-full bg-amber-500/10 px-2 py-0.5 text-xs font-medium text-amber-800">
                  {walletStatus.conflict ? 'Linked to another account' : 'Not linked'}
                </span>
              </div>
              {walletStatus.conflict ? (
                 <p className="text-sm text-amber-950">
                   This browser wallet is linked to a different Matchlock account. Switch wallets, or sign in with that account.
                 </p>
              ) : (
                <div className="flex flex-wrap items-center justify-between gap-3 text-sm">
                   <p className="text-red-400">
                     Link this wallet to your account to use it for wagers.
                   </p>
                   <Button
                     size="sm"
                     disabled={linkWallet.isPending || !walletStatus.canLink}
                     onClick={() => linkWallet.mutate(undefined)}
                   >
                     {linkWallet.isPending ? 'Linking…' : 'Link account'}
                   </Button>
                </div>
              )}
            </div>
          ) : null}

          <div className="space-y-1.5">
            <p className="text-xs font-medium text-muted-foreground">
              Linked to {userDisplayLabel(session)}
            </p>
            {walletStatus.linkedWallets.length > 0 ? (
              <ul className="space-y-2">
                {walletStatus.linkedWallets.map((wallet) => {
                  const isConnected = connected && publicKey?.toBase58() === wallet.pubkey
                  return (
                    <li
                      key={wallet.pubkey}
                      className="flex flex-wrap items-center justify-between gap-2 rounded-lg border px-3 py-2.5"
                    >
                      <div className="min-w-0 flex flex-col items-start gap-1">
                        <div className="flex items-center gap-2">
                          <p className="font-mono text-sm font-medium">
                            {truncatePubkey(wallet.pubkey)}
                          </p>
                          {isConnected ? (
                            <span className="rounded-full bg-muted px-1.5 py-0.5 text-[0.65rem] font-medium text-muted-foreground">
                              Connected
                            </span>
                          ) : null}
                        </div>
                        {wallet.label ? (
                          <p className="text-xs text-muted-foreground">
                            {wallet.label}
                          </p>
                        ) : null}
                      </div>
                      <div className="flex items-center gap-1.5">
                        {wallet.is_primary ? (
                          <span className="inline-flex items-center gap-1 rounded-full bg-primary/10 px-2 py-0.5 text-xs font-medium text-primary">
                            <Star className="size-3" />
                            Primary
                          </span>
                        ) : (
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() =>
                              setPrimaryWallet.mutate(wallet.pubkey)
                            }
                          >
                            Set primary
                          </Button>
                        )}
                        <Button
                          variant="ghost"
                          size="icon-sm"
                          aria-label="Copy address"
                          onClick={() => void copyPubkey(wallet.pubkey)}
                        >
                          {copied === wallet.pubkey ? (
                            <Check className="size-4 text-primary" />
                          ) : (
                            <Copy className="size-4" />
                          )}
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon-sm"
                          aria-label="Unlink wallet"
                          disabled={unlinkWallet.isPending}
                          onClick={() => unlinkWallet.mutate(wallet.pubkey)}
                        >
                          <Trash2 className="size-4 text-destructive" />
                        </Button>
                      </div>
                    </li>
                  )
                })}
              </ul>
            ) : (
              <p className="rounded-lg border border-dashed px-3 py-4 text-center text-sm text-muted-foreground">
                No wallets linked to your account. Connect a wallet to get started.
              </p>
            )}
          </div>

          {installedWallets.length > 1 && !connected ? (
            <div className="rounded-lg border bg-muted/20 p-3">
              <p className="mb-2 text-sm font-medium">Quick connect</p>
              <div className="flex flex-wrap gap-2">
                {installedWallets.map((wallet) => (
                  <Button
                    key={wallet.adapter.name}
                    variant="outline"
                    size="sm"
                    onClick={() => {
                      select(wallet.adapter.name as WalletName)
                      void connect()
                    }}
                  >
                    <Wallet className="mr-1 size-3.5" />
                    {wallet.adapter.name}
                  </Button>
                ))}
              </div>
            </div>
          ) : null}
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="pb-3">
          <CardTitle>Session</CardTitle>
          <CardDescription>Sign out on this device.</CardDescription>
        </CardHeader>
        <CardContent>
          <Button
            variant="destructive"
            disabled={logout.isPending}
            onClick={() => logout.mutate()}
          >
            <LogOut className="mr-2 size-4" />
            Sign out
          </Button>
        </CardContent>
      </Card>
    </div>
  )
}
