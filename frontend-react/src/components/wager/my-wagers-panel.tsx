import { useWallet } from '@solana/wallet-adapter-react'
import { Loader2 } from 'lucide-react'
import { useCallback, useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import confetti from 'canvas-confetti'
import { toast } from 'sonner'

import {
  ConfirmTxDialog,
  type ConfirmTxDetails,
} from '@/components/wager/confirm-tx-dialog'
import { MyWagerCard } from '@/components/wager/my-wager-card'
import { useMatchesQuery } from '@/hooks/queries/use-matches'
import { useWagersQuery } from '@/hooks/queries/use-wagers'
import { useWagerMutations } from '@/hooks/mutations/use-wager-mutations'
import { useApi } from '@/hooks/use-api'
import type { Match, Wager } from '@/lib/api'
import { useTxFeeEstimate } from '@/hooks/use-tx-fee-estimate'
import { useWagerTxBuilders } from '@/hooks/use-wager-tx-builders'
import { baseUnitsToUsdc } from '@/lib/format'
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

  const openCancel = (wager: Wager, stakeUsdc: number, matchLabel: string) => {
    setClaimTarget(null)
    setCancelTarget(wager)
    setConfirmDetails({
      action: 'cancel',
      matchLabel,
      sideLabel: '—',
      stakeUsdc,
    })
    setTxError(null)
    setSignature(null)
    setDialogOpen(true)
  }

  const openClaim = (
    wagerPubkey: string,
    stakeUsdc: number,
    matchLabel: string,
    side: string,
  ) => {
    setCancelTarget(null)
    setClaimTarget(wagerPubkey)
    setConfirmDetails({
      action: 'claim',
      matchLabel,
      sideLabel: side,
      stakeUsdc,
      payoutUsdc: stakeUsdc * 2,
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
      <div className="flex items-center gap-2 text-sm text-muted-foreground">
        <Loader2 className="size-4 animate-spin" />
        Loading your wagers…
      </div>
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
        {myWagers.map((wager) => {
          const match = matchMap.get(wager.match_id)
          const labels = match ? matchLabels(match) : null
          const claimable = canClaimWinnings(wager, match, walletAddress)
          const backed = match
            ? sideLabel(userBackedSide(wager, walletAddress), match)
            : userBackedSide(wager, walletAddress)

          return (
            <li key={wager.pubkey}>
              <MyWagerCard
                wager={wager}
                match={match}
                walletAddress={walletAddress}
                claimable={claimable}
                claimPending={claimWager.isPending}
                onSelect={() => navigate(`/my-wagers/${wager.pubkey}`)}
                onClaim={() =>
                  openClaim(
                    wager.pubkey,
                    baseUnitsToUsdc(wager.stake),
                    labels?.league ?? `Match ${wager.match_id}`,
                    backed,
                  )
                }
                onCancel={() =>
                  openCancel(
                    wager,
                    baseUnitsToUsdc(wager.stake),
                    labels?.league ?? `Match ${wager.match_id}`,
                  )
                }
              />
            </li>
          )
        })}
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
