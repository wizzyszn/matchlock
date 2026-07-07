import { useNavigate } from "react-router-dom";
import { ChevronDown, Loader2, LogOut, Mail, UserRound } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useAuthMutations } from "@/hooks/mutations/use-auth-mutations";
import { useSessionQuery } from "@/hooks/queries/use-session";
import { needsUsername, userDisplayLabel } from "@/lib/display-name";
import { cn } from "@/lib/utils";

function UserAvatar({ label }: { label: string }) {
  const initial = (label[0] ?? "?").toUpperCase();
  return (
    <span
      className="flex size-7 shrink-0 items-center justify-center rounded-full bg-primary/15 font-medium text-primary"
      aria-hidden
    >
      {initial}
    </span>
  );
}

export function UserAccountMenu() {
  const navigate = useNavigate();
  const { data: session, isLoading } = useSessionQuery();
  const { logout } = useAuthMutations();

  if (isLoading) {
    return (
      <Button variant="outline" size="sm" disabled>
        <Loader2 className="size-4 animate-spin" aria-hidden />
      </Button>
    );
  }

  if (!session) {
    return (
      <Button
        variant="default"
        size="sm"
        className="min-h-9"
        onClick={() => navigate("/login")}
      >
        Sign in
      </Button>
    );
  }

  const label = userDisplayLabel(session);
  const showSetupBadge = needsUsername(session);

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        className={cn(
          "inline-flex min-h-9 max-w-[min(100%,16rem)] items-center gap-2 rounded-md border border-border bg-card px-2.5 py-1.5 text-sm font-medium shadow-sm transition-colors",
          "hover:bg-muted focus-visible:outline-none focus-visible:ring-3 focus-visible:ring-ring/40",
        )}
      >
        <UserAvatar label={label} />
        <span className="flex min-w-0 flex-col items-start leading-tight">
          <span className="truncate font-medium">
            {showSetupBadge ? "Set username" : label}
          </span>
        </span>
        <ChevronDown className="size-3.5 shrink-0 text-muted-foreground" />
      </DropdownMenuTrigger>

      <DropdownMenuContent align="end" className="w-80">
        <DropdownMenuGroup>
          <DropdownMenuLabel className="font-normal">
            <div className="flex items-center gap-2">
              <UserAvatar label={label} />
              <div className="min-w-0">
                <p className="truncate text-sm font-medium">
                  {showSetupBadge ? session.email : label}
                </p>
                <p className="truncate text-xs text-muted-foreground">
                  {session.email}
                </p>
              </div>
            </div>
          </DropdownMenuLabel>
        </DropdownMenuGroup>

        <DropdownMenuSeparator />

        <DropdownMenuItem onClick={() => navigate("/profile")}>
          <UserRound className="mr-2 size-4" />
          Settings
        </DropdownMenuItem>

        <DropdownMenuItem onClick={() => navigate("/invites")}>
          <Mail className="mr-2 size-4" />
          Challenges
        </DropdownMenuItem>

        <DropdownMenuSeparator />

        <DropdownMenuItem
          variant="destructive"
          disabled={logout.isPending}
          onClick={() => logout.mutate()}
        >
          <LogOut className="mr-2 size-4" />
          Sign out
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
