import { useWallet } from '@solana/wallet-adapter-react'
import { useCallback, useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import confetti from 'canvas-confetti'
import { toast } from 'sonner'

import {
  ConfirmTxDialog,
  type ConfirmTxDetails,
} from '@/components/wager/confirm-tx-dialog'
import { Skeleton } from '@/components/ui/skeleton'
import { MyWagerCard } from '@/components/wager/my-wager-card'
import { useMatchesQuery } from '@/hooks/queries/use-matches'
import { useWagerSettlementQuery, useWagersQuery } from '@/hooks/queries/use-wagers'
import { useWagerMutations } from '@/hooks/mutations/use-wager-mutations'
import { useApi } from '@/hooks/use-api'
import type { Match, Wager } from '@/lib/api'
import { useTxFeeEstimate } from '@/hooks/use-tx-fee-estimate'
import { useWagerTxBuilders } from '@/hooks/use-wager-tx-builders'
import { baseUnitsToUsdt } from '@/lib/format'
import { matchLabels } from '@/lib/match-display'
import { canClaimWinnings, userBackedSide } from '@/lib/wager-outcome'
import { sideLabel } from '@/lib/wager-sides'
import { useOptimisticWagersStore } from '@/stores/optimistic-wagers-store'

export function MyWagersPanel() {
  const { publicKey } = useWallet()
  const walletAddress = publicKey?.toBase58()
  const navigate = useNavigate()
  const { data: wagers, isLoading } = useWagersQuery(
    walletAddress ? { wallet: walletAddress } : {},
  )
  const { data: matches } = useMatchesQuery()
  const api = useApi()
  const { cancelWager, claimWager, mapError } = useWagerMutations()
  const { buildCancel, buildClaim, wallet } = useWagerTxBuilders()
  const optimisticEntries = useOptimisticWagersStore((state) => state.wagers)
  const reconcileOptimistic = useOptimisticWagersStore((state) => state.reconcile)
  const pruneExpired = useOptimisticWagersStore((state) => state.pruneExpired)

  const [dialogOpen, setDialogOpen] = useState(false)
  const [confirmDetails, setConfirmDetails] = useState<ConfirmTxDetails | null>(
    null,
  )
  const [cancelTarget, setCancelTarget] = useState<Wager | null>(null)
  const [claimTarget, setClaimTarget] = useState<string | null>(null)
  const [txError, setTxError] = useState<string | null>(null)
  const [signature, setSignature] = useState<string | null>(null)

  const matchMap = useMemo(
    () => new Map<string, Match>(matches?.map((m) => [m.match_id, m]) ?? []),
    [matches],
  )

  const myWagers = useMemo(() => {
    const serverWagers = wagers ?? []
    const merged = new Map(serverWagers.map((wager) => [wager.pubkey, wager]))
    const optimisticOnly: typeof serverWagers = []
    const now = Date.now()

    for (const entry of Object.values(optimisticEntries)) {
      const wager = entry.wager
      const hidden =
        entry.hidden || (entry.visibleUntil !== undefined && entry.visibleUntil <= now)
      if (
        walletAddress &&
        (wager.maker === walletAddress || wager.taker === walletAddress)
      ) {
        if (hidden) {
          merged.delete(wager.pubkey)
        } else if (merged.has(wager.pubkey)) {
          merged.set(wager.pubkey, wager)
        } else {
          optimisticOnly.push(wager)
        }
      }
    }

    return [...optimisticOnly, ...Array.from(merged.values())]
  }, [optimisticEntries, wagers, walletAddress])

  useEffect(() => {
    if (wagers) {
      reconcileOptimistic(wagers)
    }
  }, [reconcileOptimistic, wagers])

  useEffect(() => {
    const timer = window.setInterval(() => {
      pruneExpired()
    }, 500)
    return () => window.clearInterval(timer)
  }, [pruneExpired])

  const openCancel = (wager: Wager, stakeUsdt: number, matchLabel: string) => {
    setClaimTarget(null)
    setCancelTarget(wager)
    setConfirmDetails({
      action: 'cancel',
      matchLabel,
      sideLabel: '—',
      stakeUsdt,
    })
    setTxError(null)
    setSignature(null)
    setDialogOpen(true)
  }

  const openClaim = (
    wagerPubkey: string,
    stakeUsdt: number,
    matchLabel: string,
    side: string,
  ) => {
    setCancelTarget(null)
    setClaimTarget(wagerPubkey)
    setConfirmDetails({
      action: 'claim',
      matchLabel,
      sideLabel: side,
      stakeUsdt,
      payoutUsdt: stakeUsdt * 2,
    })
    setTxError(null)
    setSignature(null)
    setDialogOpen(true)
  }

  const estimateFee = useCallback(async () => {
    if (confirmDetails?.action === 'cancel' && cancelTarget) {
      return buildCancel({ wagerPubkey: cancelTarget.pubkey })
    }
    if (confirmDetails?.action === 'claim' && claimTarget) {
      const proof = await api.getWagerSettlementProof(claimTarget)
      return buildClaim({ wagerPubkey: claimTarget, proof })
    }
    return null
  }, [
    api,
    buildCancel,
    buildClaim,
    cancelTarget,
    claimTarget,
    confirmDetails?.action,
  ])

  const feeEstimate = useTxFeeEstimate({
    enabled: dialogOpen && Boolean(confirmDetails),
    estimateKey: `${confirmDetails?.action ?? 'idle'}-${cancelTarget?.pubkey ?? ''}-${claimTarget ?? ''}`,
    buildTx: estimateFee,
  })

  const handleConfirm = async () => {
    setTxError(null)
    try {
      if (confirmDetails?.action === 'cancel' && cancelTarget) {
        const sig = await cancelWager.mutateAsync({
          wagerPubkey: cancelTarget.pubkey,
          wager: cancelTarget,
        })
        setSignature(sig.signature)
        toast.success('Wager cancelled.')
        return
      }
      if (confirmDetails?.action === 'claim' && claimTarget) {
        const sig = await claimWager.mutateAsync({ wagerPubkey: claimTarget })
        setSignature(sig.signature)
        confetti({
          particleCount: 150,
          spread: 80,
          origin: { y: 0.6 }
        })
      }
    } catch (error) {
      const message = mapError(error)
      setTxError(message)
      toast.error(message)
    }
  }

  const pending = cancelWager.isPending || claimWager.isPending

  if (!walletAddress) {
    return (
      <div className="rounded-lg border border-dashed bg-muted/40 px-6 py-12 text-center">
        <p className="font-heading text-2xl">Your wagers</p>
        <p className="mx-auto mt-2 max-w-sm text-sm text-muted-foreground">
          Connect your wallet to view wagers you created or accepted.
        </p>
      </div>
    )
  }

  if (isLoading && myWagers.length === 0) {
    return (
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
    )
  }

  if (myWagers.length === 0) {
    return (
      <div className="rounded-lg border border-dashed bg-muted/40 px-6 py-12 text-center">
        <p className="font-heading text-2xl">No wagers yet</p>
        <p className="mx-auto mt-2 max-w-sm text-sm text-muted-foreground">
          Create a challenge or accept an open wager to get started.
        </p>
      </div>
    )
  }

  return (
    <>
      <ul className="grid list-none gap-3 sm:grid-cols-2 lg:grid-cols-3">
        {myWagers.map((wager) => (
          <MyWagerListItem
            key={wager.pubkey}
            wager={wager}
            match={matchMap.get(wager.match_id)}
            walletAddress={walletAddress}
            claimPending={claimWager.isPending}
            onSelect={() => navigate(`/my-wagers/${wager.pubkey}`)}
            onClaim={openClaim}
            onCancel={openCancel}
          />
        ))}
      </ul>

      <ConfirmTxDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        details={confirmDetails}
        pending={pending}
        error={txError}
        signature={signature}
        feeEstimate={feeEstimate}
        feePayerAddress={wallet?.publicKey?.toBase58() ?? walletAddress}
        onConfirm={() => void handleConfirm()}
      />
    </>
  )
}

type MyWagerListItemProps = {
  wager: Wager
  match?: Match
  walletAddress: string
  claimPending: boolean
  onSelect: () => void
  onClaim: (
    wagerPubkey: string,
    stakeUsdt: number,
    matchLabel: string,
    side: string,
  ) => void
  onCancel: (wager: Wager, stakeUsdt: number, matchLabel: string) => void
}

function MyWagerListItem({
  wager,
  match,
  walletAddress,
  claimPending,
  onSelect,
  onClaim,
  onCancel,
}: MyWagerListItemProps) {
  const { data: settlement } = useWagerSettlementQuery(
    wager.status === 'matched' ? wager.pubkey : undefined,
  )
  const labels = match ? matchLabels(match) : null
  const claimable = canClaimWinnings(wager, match, walletAddress, settlement)
  const backed = match
    ? sideLabel(userBackedSide(wager, walletAddress), match)
    : userBackedSide(wager, walletAddress)

  return (
    <li>
      <MyWagerCard
        wager={wager}
        match={match}
        walletAddress={walletAddress}
        claimable={claimable}
        claimPending={claimPending}
        onSelect={onSelect}
        onClaim={() =>
          onClaim(
            wager.pubkey,
            baseUnitsToUsdt(wager.stake),
            labels?.league ?? `Match ${wager.match_id}`,
            backed,
          )
        }
        onCancel={() =>
          onCancel(
            wager,
            baseUnitsToUsdt(wager.stake),
            labels?.league ?? `Match ${wager.match_id}`,
          )
        }
      />
    </li>
  )
}
