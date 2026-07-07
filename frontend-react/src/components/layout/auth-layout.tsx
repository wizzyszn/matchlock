import { Outlet } from 'react-router-dom'

import { ClusterBadge } from '@/components/wallet/ClusterBadge'
import { PoweredByTxLine } from '@/components/brand/powered-by-txline'

export function AuthLayout() {
  return (
    <div className="relative flex min-h-svh flex-col overflow-hidden bg-background">
      <div
        className="pointer-events-none absolute inset-0 opacity-40"
        aria-hidden
        style={{
          backgroundImage:
            'radial-gradient(circle at 20% 20%, rgba(194,101,42,0.12), transparent 45%), radial-gradient(circle at 80% 0%, rgba(140,60,60,0.08), transparent 40%)',
        }}
      />
      <div
        className="pointer-events-none absolute inset-0 opacity-[0.35]"
        aria-hidden
        style={{
          backgroundImage:
            'linear-gradient(rgba(42,36,32,0.04) 1px, transparent 1px), linear-gradient(90deg, rgba(42,36,32,0.04) 1px, transparent 1px)',
          backgroundSize: '32px 32px',
        }}
      />

      <header className="relative z-10 border-b border-border/60 bg-background/80 backdrop-blur-sm">
        <div className="mx-auto flex max-w-lg items-center justify-between px-4 py-4">
          <span className="font-heading text-2xl tracking-tight">Matchlock</span>
          <ClusterBadge />
        </div>
      </header>

      <main className="relative z-10 mx-auto flex w-full max-w-lg flex-1 px-4 py-10 sm:py-14">
        <Outlet />
      </main>

      <footer className="relative z-10 border-t border-border/60 bg-muted/20 px-4 py-8">
        <div className="mx-auto flex max-w-lg flex-col items-center gap-3">
          <PoweredByTxLine />
          <p className="text-center text-xs text-muted-foreground">
            Peer-to-peer wagers secured by Solana and TxLINE oracles.
          </p>
        </div>
      </footer>
    </div>
  )
}