import { NavLink } from "react-router-dom";
import type { ReactNode } from "react";

import { PoweredByTxLine } from "@/components/brand/powered-by-txline";
import { UserAccountMenu } from "@/components/auth/user-account-menu";
import { ClusterBadge } from "@/components/wallet/ClusterBadge";
import { Badge } from "@/components/ui/badge";
import { useHealthQuery } from "@/hooks/queries/use-health";
import { cn } from "@/lib/utils";

const NAV_ITEMS = [
  { to: "/markets", label: "Markets" },

  { to: "/open", label: "Open" },
  { to: "/my-wagers", label: "My wagers" },
  { to: "/history", label: "History" },
  { to: "/invites", label: "Challenges" },
  { to: "/profile", label: "Profile" },
] as const;

function BackendStatus() {
  const { isPending, isSuccess, error } = useHealthQuery();

  if (isPending) {
    return <Badge variant="outline">Checking API…</Badge>;
  }

  if (isSuccess) {
    return <Badge variant="outline">API online</Badge>;
  }

  return (
    <Badge
      variant="destructive"
      title={error instanceof Error ? error.message : undefined}
    >
      API offline
    </Badge>
  );
}

export interface AppShellProps {
  children: ReactNode;
}

export function AppShell({ children }: AppShellProps) {
  return (
    <div className="flex min-h-svh flex-col bg-background">
      <header className="sticky top-0 z-50 border-b bg-background/95 backdrop-blur-sm">
        <div className="mx-auto flex max-w-5xl items-center justify-between gap-4 px-4 py-4">
          <div className="flex min-w-0 items-center gap-2 sm:gap-3">
            <NavLink
              to="/markets"
              className="font-heading shrink-0 text-xl tracking-tight hover:text-primary sm:text-2xl"
            >
              Matchlock
            </NavLink>
            <div className="hidden items-center gap-2 sm:flex">
              <ClusterBadge />
              <BackendStatus />
            </div>
          </div>
          <UserAccountMenu />
        </div>

        <div className="mx-auto flex max-w-5xl flex-col gap-3 px-4 pb-3 sm:hidden">
          <div className="flex gap-2">
            <ClusterBadge />
            <BackendStatus />
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
                    cn(
                      "inline-flex min-h-9 items-center rounded-full border px-4 text-sm font-medium transition-colors",
                      isActive
                        ? "border-primary bg-primary text-primary-foreground"
                        : "border-border bg-card text-foreground hover:bg-muted",
                    )
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
