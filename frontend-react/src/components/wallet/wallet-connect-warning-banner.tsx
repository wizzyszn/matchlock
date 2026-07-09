import { WalletReadyState, type WalletName } from '@solana/wallet-adapter-base'
import { useWallet } from '@solana/wallet-adapter-react'
import {
  AlertTriangle,
  CheckCircle2,
  ChevronDown,
  Loader2,
  Wallet,
} from 'lucide-react'
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'

import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'

export function WalletConnectWarningBanner() {
  const { connected, connect, connecting, select, wallets } = useWallet()
  const [pendingWallet, setPendingWallet] = useState<string | null>(null)
  const [connectError, setConnectError] = useState<string | null>(null)
  const [showSuccess, setShowSuccess] = useState(false)
  const previousConnectedRef = useRef(connected)
  const successTimeoutRef = useRef<number | null>(null)

  const installedWallets = useMemo(
    () =>
      wallets.filter(
        (wallet) => wallet.readyState === WalletReadyState.Installed,
      ),
    [wallets],
  )

  const handleConnect = useCallback(
    async (walletName: string) => {
      setConnectError(null)
      setPendingWallet(walletName)

      try {
        select(walletName as WalletName)
        await connect()
      } catch {
        setConnectError('Wallet connection was cancelled or could not be completed.')
      } finally {
        setPendingWallet(null)
      }
    },
    [connect, select],
  )

  useEffect(() => {
    const wasConnected = previousConnectedRef.current

    if (connected && !wasConnected) {
      setShowSuccess(true)
      if (successTimeoutRef.current) {
        window.clearTimeout(successTimeoutRef.current)
      }
      successTimeoutRef.current = window.setTimeout(() => {
        setShowSuccess(false)
        successTimeoutRef.current = null
      }, 1800)
    }

    if (!connected) {
      setShowSuccess(false)
      if (successTimeoutRef.current) {
        window.clearTimeout(successTimeoutRef.current)
        successTimeoutRef.current = null
      }
    }

    previousConnectedRef.current = connected

    return () => {
      if (successTimeoutRef.current) {
        window.clearTimeout(successTimeoutRef.current)
        successTimeoutRef.current = null
      }
    }
  }, [connected])

  if (connected && !showSuccess) {
    return null
  }

  const isConnecting = connecting || Boolean(pendingWallet)
  const hasInstalledWallets = installedWallets.length > 0

  const buttonContent = isConnecting ? (
    <>
      <Loader2 className="size-3.5 animate-spin" aria-hidden />
      Connecting...
    </>
  ) : (
    <>
      <Wallet className="size-3.5" aria-hidden />
      Connect wallet
    </>
  )

  if (showSuccess) {
    return (
      <div className="mx-auto max-w-5xl px-4 pb-3">
        <section
          className="rounded-lg border border-emerald-400/40 bg-linear-to-r from-emerald-500/18 via-emerald-400/12 to-teal-400/18"
          role="status"
          aria-live="polite"
        >
          <div className="flex flex-col gap-3 px-4 py-3 sm:flex-row sm:items-center">
            <span className="mt-0.5 flex size-8 shrink-0 items-center justify-center rounded-full bg-emerald-500/20 text-emerald-700 dark:text-emerald-200">
              <CheckCircle2 className="size-4" aria-hidden />
            </span>
            <div className="min-w-0">
              <p className="text-sm font-semibold text-emerald-950 dark:text-emerald-50">
                Wallet connected
              </p>
              <p className="text-sm text-emerald-950/75 dark:text-emerald-50/75">
                Your wallet is ready for wagers and settlement.
              </p>
            </div>
          </div>
        </section>
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-5xl px-4 pb-3">
      <section
        className="rounded-lg border border-amber-400/35 bg-amber-400/10"
        role="status"
        aria-live="polite"
      >
        <div className="flex flex-col gap-3 px-4 py-3 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex min-w-0 gap-3">
            <span className="mt-0.5 flex size-8 shrink-0 items-center justify-center rounded-full bg-amber-400/20 text-amber-700 dark:text-amber-200">
              <AlertTriangle className="size-4" aria-hidden />
            </span>
            <div className="min-w-0">
              <p className="text-sm font-semibold text-amber-950 dark:text-amber-50">
                Wallet not connected
              </p>
              <p className="text-sm text-amber-950/75 dark:text-amber-50/75">
                Connect a Solana wallet to place, accept, or settle wagers.
              </p>
              {connectError ? (
                <p className="mt-1 text-xs font-medium text-amber-950 dark:text-amber-100">
                  {connectError}
                </p>
              ) : null}
              {!hasInstalledWallets ? (
                <p className="mt-1 text-xs text-amber-950/70 dark:text-amber-50/70">
                  Install Phantom or Solflare, then refresh this page.
                </p>
              ) : null}
            </div>
          </div>

          {installedWallets.length > 1 ? (
            <DropdownMenu>
              <DropdownMenuTrigger className="inline-flex shrink-0 self-start sm:self-auto">
                <Button
                  size="sm"
                  className="h-8 gap-1.5 px-2.5 text-xs"
                  disabled={isConnecting}
                  type="button"
                >
                  {buttonContent}
                  {!isConnecting ? (
                    <ChevronDown className="size-3" aria-hidden />
                  ) : null}
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-52">
                <DropdownMenuGroup>
                  {installedWallets.map((wallet) => (
                    <DropdownMenuItem
                      key={wallet.adapter.name}
                      onClick={() => void handleConnect(wallet.adapter.name)}
                    >
                      {wallet.adapter.name}
                    </DropdownMenuItem>
                  ))}
                </DropdownMenuGroup>
              </DropdownMenuContent>
            </DropdownMenu>
          ) : (
            <Button
              size="sm"
              className="h-8 shrink-0 self-start gap-1.5 px-2.5 text-xs sm:self-auto"
              disabled={isConnecting || !hasInstalledWallets}
              onClick={() => {
                const wallet = installedWallets[0]
                if (wallet) {
                  void handleConnect(wallet.adapter.name)
                }
              }}
            >
              {buttonContent}
            </Button>
          )}
        </div>
      </section>
    </div>
  )
}
