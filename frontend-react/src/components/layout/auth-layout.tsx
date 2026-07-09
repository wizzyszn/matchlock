import { Outlet } from "react-router-dom";

import { ClusterBadge } from "@/components/wallet/ClusterBadge";
import { PoweredByTxLine } from "@/components/brand/powered-by-txline";
import logoUrl from "@/assets/g17.svg";

interface AuthLayoutPropsInt {
  showHeader: boolean;
  showFooter: boolean;
}
export function AuthLayout({
  showHeader = false,
  showFooter = false,
}: AuthLayoutPropsInt) {
  return (
    <div className="relative flex min-h-svh flex-col overflow-hidden bg-background">
      <div
        className="pointer-events-none absolute inset-0 opacity-40"
        aria-hidden
        style={{
          backgroundImage:
            "radial-gradient(circle at 20% 20%, rgba(230,64,64,0.12), transparent 45%), radial-gradient(circle at 80% 0%, rgba(230,97,64,0.08), transparent 40%)",
        }}
      />
      <div
        className="pointer-events-none absolute inset-0 opacity-[0.35]"
        aria-hidden
        style={{
          backgroundImage:
            "linear-gradient(rgba(255,255,255,0.04) 1px, transparent 1px), linear-gradient(90deg, rgba(255,255,255,0.04) 1px, transparent 1px)",
          backgroundSize: "32px 32px",
        }}
      />

      {showHeader && (
        <header className="relative z-10 border-b border-border/60 bg-background/80 backdrop-blur-sm">
          <div className="mx-auto flex max-w-lg items-center justify-between px-4 py-4">
            <span className="font-heading flex items-center gap-2 text-2xl tracking-tight">
              <img src={logoUrl} alt="" className="h-6 w-auto object-contain" />
              <span>Matchlock</span>
            </span>
            <ClusterBadge />
          </div>
        </header>
      )}

      <main className="relative z-10 mx-auto flex w-full max-w-lg flex-1 px-4 py-10 sm:py-14">
        <Outlet />
      </main>

      {showFooter && (
        <footer className="relative z-10 border-t border-border/60 bg-muted/20 px-4 py-8">
          <div className="mx-auto flex max-w-lg flex-col items-center gap-3">
            <PoweredByTxLine />
            <p className="text-center text-xs text-muted-foreground">
              Peer-to-peer wagers secured by Solana and TxLINE oracles.
            </p>
          </div>
        </footer>
      )}
    </div>
  );
}
