import { useWallet } from "@solana/wallet-adapter-react";
import { useCallback, useMemo, useState } from "react";

import { Button } from "@/components/ui/button";
import { PageHeader, PageHeaderHeading, PageHeaderDescription } from '@/components/ui/page-header'
import { Skeleton } from "@/components/ui/skeleton";
import { OpenWagerCard } from "@/components/wager/open-wager-card";
import {
  ConfirmTxDialog,
  type ConfirmTxDetails,
} from "@/components/wager/confirm-tx-dialog";
import { useMatchesQuery } from "@/hooks/queries/use-matches";
import { useWagersQuery } from "@/hooks/queries/use-wagers";
import { useWagerMutations } from "@/hooks/mutations/use-wager-mutations";
import { useTxFeeEstimate } from "@/hooks/use-tx-fee-estimate";
import { useWagerTxBuilders } from "@/hooks/use-wager-tx-builders";
import type { Match, Side } from "@/lib/api";
import { baseUnitsToUsdt } from "@/lib/format";
import { classifyMatch, matchLabels } from "@/lib/match-display";
import { sideLabel } from "@/lib/wager-sides";
import { cn } from "@/lib/utils";

type FilterKey = "all" | "live" | "open";

const FILTERS: { key: FilterKey; label: string }[] = [
  { key: "all", label: "All" },
  { key: "live", label: "Live" },
  { key: "open", label: "Open" },
];

export function OpenWagerList() {
  const { publicKey } = useWallet();
  const walletAddress = publicKey?.toBase58();
  const {
    data: wagers,
    isLoading,
    isError,
    error,
  } = useWagersQuery({ status: "open" });
  const { data: matches } = useMatchesQuery();
  const { acceptWager, mapError } = useWagerMutations();
  const { buildAccept, wallet } = useWagerTxBuilders();

  const [filter, setFilter] = useState<FilterKey>("all");
  const [dialogOpen, setDialogOpen] = useState(false);
  const [confirmDetails, setConfirmDetails] = useState<ConfirmTxDetails | null>(
    null,
  );
  const [acceptTarget, setAcceptTarget] = useState<{
    wagerPubkey: string;
    maker: string;
    takerSide: Side;
    matchId: string;
  } | null>(null);
  const [txError, setTxError] = useState<string | null>(null);
  const [signature, setSignature] = useState<string | null>(null);

  const matchMap = useMemo(() => {
    const map = new Map<string, Match>();
    matches?.forEach((m) => map.set(m.match_id, m));
    return map;
  }, [matches]);

  const filteredWagers = useMemo(() => {
    return (wagers ?? []).filter((wager) => {
      if (walletAddress && wager.maker === walletAddress) return false;

      const match = matchMap.get(wager.match_id);
      if (match && classifyMatch(match) === "finished") return false;
      if (filter === "open") return wager.status === "open";
      if (filter === "live") return match ? matchLabels(match).isLive : false;
      return true;
    });
  }, [wagers, walletAddress, matchMap, filter]);

  const openAccept = (wagerPubkey: string, maker: string, takerSide: Side) => {
    const wager = wagers?.find((w) => w.pubkey === wagerPubkey);
    if (!wager) return;

    const match = matchMap.get(wager.match_id);
    const labels = match ? matchLabels(match) : null;
    const outcomeLabel = match
      ? sideLabel(takerSide, match)
      : takerSide;

    setAcceptTarget({ wagerPubkey, maker, takerSide, matchId: wager.match_id });
    setConfirmDetails({
      action: "accept",
      matchLabel: labels?.league ?? `Match ${wager.match_id}`,
      sideLabel: outcomeLabel,
      stakeUsdt: baseUnitsToUsdt(wager.stake),
      payoutUsdt: baseUnitsToUsdt(wager.stake) * 2,
    });
    setTxError(null);
    setSignature(null);
    setDialogOpen(true);
  };

  const estimateFee = useCallback(async () => {
    if (!acceptTarget) return null;
    return buildAccept(acceptTarget);
  }, [acceptTarget, buildAccept]);

  const feeEstimate = useTxFeeEstimate({
    enabled: dialogOpen && confirmDetails?.action === "accept",
    estimateKey: acceptTarget
      ? `accept-${acceptTarget.wagerPubkey}-${acceptTarget.takerSide}`
      : "idle",
    buildTx: estimateFee,
  });

  const handleAccept = async () => {
    if (!acceptTarget) return;
    setTxError(null);
    try {
      const sig = await acceptWager.mutateAsync(acceptTarget);
      setSignature(sig);
    } catch (err) {
      setTxError(mapError(err));
    }
  };

  return (
    <section aria-labelledby="open-wagers-heading">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
        <PageHeader className="mb-8 sm:mb-0">
          <PageHeaderHeading id="open-wagers-heading">Open challenges</PageHeaderHeading>
          <PageHeaderDescription>
            Head-to-head wagers waiting for a taker. Pick your outcome and match
            the stake.
          </PageHeaderDescription>
        </PageHeader>

        <div
          className="flex gap-2 overflow-x-auto pb-1 [-ms-overflow-style:none] [scrollbar-none] [&::-webkit-scrollbar]:hidden"
          role="tablist"
          aria-label="Filter wagers"
        >
          {FILTERS.map(({ key, label }) => {
            const selected = filter === key;
            return (
              <Button 
                key={key}
                type="button"
                role="tab"
                aria-selected={selected}
                variant={selected ? "default" : "outline"}
                size="sm"
                className={cn("min-h-9 shrink-0 rounded-full px-4")}
                onClick={() => setFilter(key)}
              >
                {label}
              </Button>
            );
          })}
        </div>
      </div>

      {isLoading ? (
        <ul className="grid list-none gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {[1, 2, 3, 4, 5, 6].map((i) => (
            <li key={i}>
              <div className="flex flex-col gap-4 rounded-lg border border-border bg-card p-4 shadow-sahara">
                <div className="flex items-start justify-between mb-2">
                  <div className="space-y-2">
                    <Skeleton className="h-4 w-32" />
                    <Skeleton className="h-3 w-20" />
                  </div>
                  <Skeleton className="h-6 w-16 rounded-md" />
                </div>
                <div className="flex items-center justify-between py-2 border-t border-border/60 mt-2">
                   <div className="space-y-1.5">
                     <Skeleton className="h-3 w-12" />
                     <Skeleton className="h-5 w-16" />
                   </div>
                   <Skeleton className="h-9 w-24 rounded-md" />
                </div>
              </div>
            </li>
          ))}
        </ul>
      ) : isError ? (
        <p className="text-sm text-destructive">
          {error instanceof Error ? error.message : "Failed to load wagers"}
        </p>
      ) : filteredWagers.length === 0 ? (
        <div className="rounded-lg border border-dashed bg-muted/40 px-6 py-16 text-center">
          <p className="font-heading text-2xl">No open challenges</p>
          <p className="mx-auto mt-2 max-w-sm text-sm text-muted-foreground">
            Be the first to create a wager, or check back when new matchups are
            posted.
          </p>
        </div>
      ) : (
        <ul className="grid list-none gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {filteredWagers.map((wager) => {
            const match = matchMap.get(wager.match_id);
            return (
              <li key={wager.pubkey}>
                <OpenWagerCard
                  wager={wager}
                  match={match}
                  disabled={
                    !walletAddress ||
                    acceptWager.isPending ||
                    (match ? classifyMatch(match) === "finished" : false)
                  }
                  onAccept={(takerSide) =>
                    openAccept(wager.pubkey, wager.maker, takerSide)
                  }
                />
              </li>
            );
          })}
        </ul>
      )}

      <ConfirmTxDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        details={confirmDetails}
        pending={acceptWager.isPending}
        error={txError}
        signature={signature}
        feeEstimate={feeEstimate}
        feePayerAddress={wallet?.publicKey?.toBase58() ?? walletAddress}
        onConfirm={() => void handleAccept()}
      />
    </section>
  );
}
