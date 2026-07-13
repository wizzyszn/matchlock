import type { ReactNode } from "react";

import { MatchlockLogoAnimated } from "@/components/brand/matchlock-logo-animated";
import { cn } from "@/lib/utils";

type AuthTransitionLoaderProps = {
  title?: string;
  subtitle?: string;
  className?: string;
  icon?: ReactNode;
};

export function AuthTransitionLoader({
  title,
  subtitle,
  className,
  icon,
}: AuthTransitionLoaderProps) {
  return (
    <div
      className={cn(
        "flex min-h-full w-full flex-col items-center justify-center gap-6 py-16 text-center",
        className,
      )}
      role="status"
      aria-live="polite"
    >
      {icon ?? <MatchlockLogoAnimated size={32} />}
      <div className="space-y-1">
        <p className="font-heading text-lg tracking-tight">{title}</p>
        {subtitle ? (
          <p className="max-w-sm text-xs text-muted-foreground">{subtitle}</p>
        ) : null}
      </div>
    </div>
  );
}
