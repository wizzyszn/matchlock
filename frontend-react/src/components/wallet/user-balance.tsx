import { ChevronUp, ChevronDown } from "lucide-react";
import usdtLogo from "@/assets/usdt-svgrepo-com.svg";

import { useTokenBalanceQuery } from "@/hooks/queries/use-token-balance";
import { useBalanceChange } from "@/hooks/use-balance-change";
import { baseUnitsToUsdt, formatUsdt } from "@/lib/format";

export function UserBalance() {
  const { data: balance = BigInt(0) } = useTokenBalanceQuery();
  const usdt = baseUnitsToUsdt(balance);
  const formatted = formatUsdt(usdt, { maxDecimals: 2 });
  const change = useBalanceChange(usdt);

  return (
    <span className="shimmer relative inline-flex items-center gap-2 overflow-hidden rounded-full border border-border/50 bg-card/60 px-3 py-1.5 text-xs font-medium text-foreground backdrop-blur-sm">
      <img src={usdtLogo} alt="USDT" className="size-4 rounded-full" />
      <span className="tabular-nums">{formatted}</span>
      <span className="text-muted-foreground">USDT</span>

      {change && (
        <span
          className={`animate-delta-pop ml-0.5 inline-flex items-center gap-0.5 rounded-full px-1.5 py-0.5 text-[10px] font-semibold tabular-nums ${
            change.direction === "up"
              ? "bg-emerald-500/15 text-emerald-400"
              : "bg-red-500/15 text-red-400"
          }`}
        >
          {change.direction === "up" ? (
            <ChevronUp className="size-3" strokeWidth={3} />
          ) : (
            <ChevronDown className="size-3" strokeWidth={3} />
          )}
          {change.delta.toFixed(2)}
        </span>
      )}
    </span>
  );
}
