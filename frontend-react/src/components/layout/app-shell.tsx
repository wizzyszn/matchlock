import { NavLink } from "react-router-dom";
import { useState, type ReactNode } from "react";
import { Info, X } from "lucide-react";

import { PoweredByTxLine } from "@/components/brand/powered-by-txline";
import logoUrl from "@/assets/g17.svg";
import { UserAccountMenu } from "@/components/auth/user-account-menu";
import { ClusterBadge } from "@/components/wallet/ClusterBadge";
import { useSessionQuery } from "@/hooks/queries/use-session";
import { cn } from "@/lib/utils";
import { buttonVariants } from "@/components/ui/button";
import { UserBalance } from "@/components/wallet/user-balance";

const NAV_ITEMS = [
  { to: "/markets", label: "Markets" },

  { to: "/open", label: "Open" },
  { to: "/my-wagers", label: "Wagers" },
  { to: "/history", label: "History" },
  { to: "/invites", label: "Challenges" },
  { to: "/leaderboard", label: "Leaderboard" },
  { to: "/profile", label: "Profile" },
] as const;


export interface AppShellProps {
  children: ReactNode;
}

export function AppShell({ children }: AppShellProps) {
  const [dismissed, setDismissed] = useState(false);
  const { data: session } = useSessionQuery();

  const hasLinkedWallet = Boolean(session?.wallets?.length);
  const showBanner = session != null && !hasLinkedWallet && !dismissed;

  return (
    <div className="flex min-h-svh flex-col bg-background">
      {showBanner && (
        <div className="bg-status-open px-4 py-2.5 text-primary-foreground relative text-center text-sm font-medium z-50">
          <div className="mx-auto flex max-w-5xl items-center justify-center gap-2 pr-8">
            <Info className="size-4 shrink-0" aria-hidden />
            <span>
              Please head to your <NavLink to="/profile" className="underline underline-offset-2 font-semibold hover:text-primary-foreground/80">profile settings</NavLink> to connect and link a wallet so you can place wagers.
            </span>
          </div>
          <button 
            type="button" 
            onClick={() => setDismissed(true)} 
            className="absolute right-4 top-1/2 -translate-y-1/2 rounded-full p-1 opacity-80 hover:bg-primary-foreground/20 hover:opacity-100 transition-colors"
            aria-label="Dismiss banner"
          >
            <X className="size-4" />
          </button>
        </div>
      )}
      <header className="sticky top-0 z-50 border-b bg-background/95 backdrop-blur-sm">
        <div className="mx-auto flex max-w-5xl items-center justify-between gap-4 px-4 py-4">
          <div className="flex min-w-0 items-center gap-2 sm:gap-3">
            <NavLink
              to="/markets"
              className="font-heading shrink-0 flex items-center gap-2 text-xl tracking-tight hover:text-primary sm:text-2xl"
            >
              <img src={logoUrl} alt="" className="h-6 w-auto object-contain" />
              <span>Matchlock</span>
            </NavLink>
            <div className="hidden items-center gap-2 sm:flex">
              <ClusterBadge />
              
            </div>
          </div>
          <div className="flex items-center gap-2">
            <UserBalance />
            <UserAccountMenu />
          </div>
        </div>

        <div className="mx-auto flex max-w-5xl flex-col gap-3 px-4 pb-3 sm:hidden">
          <div className="flex gap-2">
            <ClusterBadge />
            <UserBalance />
          </div>
        </div>

        <nav
          aria-label="Main"
          className="mx-auto max-w-5xl overflow-x-auto px-4 pb-3 [-ms-overflow-style:none] [scrollbar-none] [&::-webkit-scrollbar]:hidden"
        >
          <ul className="flex list-none gap-2">
            {NAV_ITEMS.map(({ to, label }) => (
              <li key={to}>
                <NavLink
                  to={to}
                  className={({ isActive }) =>
                    isActive
                      ? cn(
                          buttonVariants({ variant: "default" }),
                          "rounded-full px-4",
                        )
                      : "inline-flex min-h-9 items-center rounded-full border border-border bg-card px-4 text-sm font-medium text-foreground transition-colors hover:bg-muted"
                  }
                >
                  {label}
                </NavLink>
              </li>
            ))}
          </ul>
        </nav>
      </header>

      <main className="flex-1 mx-auto max-w-5xl px-4 py-8 sm:py-12 w-full">{children}</main>

      <footer className="border-t border-border/80 bg-muted/20">
        <div className="mx-auto flex max-w-5xl flex-col items-center gap-3 px-4 py-8">
          <PoweredByTxLine />
          <p className="max-w-md text-center text-xs text-muted-foreground">
            Match data and settlement proofs powered by TxLINE on-chain oracles.
          </p>
        </div>
      </footer>
    </div>
  );
}
