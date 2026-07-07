import { useWallet } from "@solana/wallet-adapter-react";
import { Loader2 } from "lucide-react";
import { useCallback, useMemo, useState } from "react";

import { Button } from "@/components/ui/button";
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
import type { Side } from "@/lib/api";
import { baseUnitsToUsdc } from "@/lib/format";
import { matchLabels } from "@/lib/match-display";
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
  } | null>(null);
  const [txError, setTxError] = useState<string | null>(null);
  const [signature, setSignature] = useState<string | null>(null);

  const matchMap = useMemo(
    () => new Map(matches?.map((m) => [m.match_id, m]) ?? []),
    [matches],
  );

  const filteredWagers = useMemo(() => {
    return (wagers ?? []).filter((wager) => {
      if (walletAddress && wager.maker === walletAddress) return false;

      const match = matchMap.get(wager.match_id);
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

    setAcceptTarget({ wagerPubkey, maker, takerSide });
    setConfirmDetails({
      action: "accept",
      matchLabel: labels?.league ?? `Match ${wager.match_id}`,
      sideLabel: outcomeLabel,
      stakeUsdc: baseUnitsToUsdc(wager.stake),
      payoutUsdc: baseUnitsToUsdc(wager.stake) * 2,
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
      <div className="mb-8 flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h2
            id="open-wagers-heading"
            className="font-heading text-3xl leading-tight sm:text-4xl"
          >
            Open challenges
          </h2>
          <p className="mt-2 max-w-prose text-sm text-muted-foreground">
            Head-to-head wagers waiting for a taker. Pick your outcome and match
            the stake.
          </p>
        </div>

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
        <div className="flex items-center gap-2 text-sm text-muted-foreground">
          <Loader2 className="size-4 animate-spin" />
          Loading open wagers…
        </div>
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
                  disabled={!walletAddress || acceptWager.isPending}
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
