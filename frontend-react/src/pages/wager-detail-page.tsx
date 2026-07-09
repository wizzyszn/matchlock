import { useWallet } from '@solana/wallet-adapter-react'
import { ArrowLeft, ExternalLink, Loader2, Swords, Clock } from 'lucide-react'
import { useCallback, useMemo, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { ConfirmTxDialog, type ConfirmTxDetails } from '@/components/wager/confirm-tx-dialog'
import { DuelFrame } from '@/components/wager/duel-frame'
import { OutcomePicker } from '@/components/wager/outcome-picker'
import { SettlementStatus } from '@/components/wager/settlement-status'
import { WagerStatusBadge } from '@/components/wager/wager-status-badge'
import { useMatchesQuery } from '@/hooks/queries/use-matches'
import { useWagerQuery } from '@/hooks/queries/use-wagers'
import { useWagerMutations } from '@/hooks/mutations/use-wager-mutations'
import { useApi } from '@/hooks/use-api'
import { useStablecoinLabel } from '@/hooks/use-stablecoin-label'
import { useTxFeeEstimate } from '@/hooks/use-tx-fee-estimate'
import { useWagerTxBuilders } from '@/hooks/use-wager-tx-builders'
import type { Side } from '@/lib/api'
import { baseUnitsToUsdc, formatStakeBaseUnits, truncateAddress, explorerAddressUrl } from '@/lib/format'
import { matchLabels } from '@/lib/match-display'
import { sideLabel, availableTakerSides, defaultTakerSide } from '@/lib/wager-sides'
import { canClaimWinnings } from '@/lib/wager-outcome'
import { isPlaceholderAddress } from '@/lib/accounts'
import { useConfig } from '@/hooks/use-api'

export function WagerDetailPage() {
  const { wagerPubkey } = useParams<{ wagerPubkey: string }>()
  // Removing unused navigate
  const { publicKey } = useWallet()
  const walletAddress = publicKey?.toBase58()
  const config = useConfig()
  const { data: wager, isLoading, isError, error } = useWagerQuery(wagerPubkey)
  const { data: matches } = useMatchesQuery()
  const stablecoin = useStablecoinLabel()
  const api = useApi()
  const { acceptWager, cancelWager, claimWager, mapError } = useWagerMutations()
  const { buildCancel, buildClaim, buildAccept, wallet } = useWagerTxBuilders()

  const match = useMemo(
    () => matches?.find((m) => m.match_id === wager?.match_id),
    [matches, wager?.match_id],
  )

  const labels = match ? matchLabels(match) : null

  const isMaker = walletAddress ? wager?.maker === walletAddress : false
  const canCancel = Boolean(isMaker && wager?.status === 'open')
  const claimable = wager ? canClaimWinnings(wager, match, walletAddress ?? '') : false

  const roleLabel = useMemo(() => {
    if (!wager || !walletAddress) return ''
    if (wager.status === 'open') {
      return isMaker ? 'Your open challenge' : 'Challenge you joined'
    }
    return isMaker ? 'You posted this wager' : 'You accepted this wager'
  }, [wager, walletAddress, isMaker])

  const opponent = wager ? (isMaker ? wager.taker : wager.maker) : ''
  const awaitingOpponent = !wager || wager.status === 'open' || isPlaceholderAddress(opponent)
  const showOpponent = !awaitingOpponent && opponent.length > 0

  const backedSide = wager
    ? isMaker
      ? wager.maker_side
      : wager.taker_side ?? wager.maker_side
    : null
  const backedLabel = backedSide && match
    ? sideLabel(backedSide, match)
    : backedSide ?? ''

  const takerOptions = wager ? availableTakerSides(wager.maker_side) : []
  const [takerSide, setTakerSide] = useState<Side>(wager ? defaultTakerSide(wager.maker_side) : 'away')

  const [dialogOpen, setDialogOpen] = useState(false)
  const [confirmDetails, setConfirmDetails] = useState<ConfirmTxDetails | null>(null)
  const [acceptTarget, setAcceptTarget] = useState<{ wagerPubkey: string; maker: string; takerSide: Side } | null>(null)
  const [txError, setTxError] = useState<string | null>(null)
  const [signature, setSignature] = useState<string | null>(null)

  const openCancel = () => {
    if (!wager) return
    setConfirmDetails({
      action: 'cancel',
      matchLabel: labels?.league ?? `Match ${wager.match_id}`,
      sideLabel: '—',
      stakeUsdc: baseUnitsToUsdc(wager.stake),
    })
    setTxError(null)
    setSignature(null)
    setDialogOpen(true)
  }

  const openClaim = () => {
    if (!wager) return
    setConfirmDetails({
      action: 'claim',
      matchLabel: labels?.league ?? `Match ${wager.match_id}`,
      sideLabel: backedLabel,
      stakeUsdc: baseUnitsToUsdc(wager.stake),
      payoutUsdc: baseUnitsToUsdc(wager.stake * 2),
    })
    setTxError(null)
    setSignature(null)
    setDialogOpen(true)
  }

  const openAccept = (takerSide: Side) => {
    if (!wager) return
    const outcomeLabel = match ? sideLabel(takerSide, match) : takerSide
    setAcceptTarget({ wagerPubkey: wager.pubkey, maker: wager.maker, takerSide })
    setConfirmDetails({
      action: 'accept',
      matchLabel: labels?.league ?? `Match ${wager.match_id}`,
      sideLabel: outcomeLabel,
      stakeUsdc: baseUnitsToUsdc(wager.stake),
      payoutUsdc: baseUnitsToUsdc(wager.stake * 2),
    })
    setTxError(null)
    setSignature(null)
    setDialogOpen(true)
  }

  const estimateFee = useCallback(async () => {
    if (confirmDetails?.action === 'cancel' && wager) {
      return buildCancel({ wagerPubkey: wager.pubkey })
    }
    if (confirmDetails?.action === 'claim' && wager) {
      const proof = await api.getWagerSettlementProof(wager.pubkey)
      return buildClaim({ wagerPubkey: wager.pubkey, proof })
    }
    if (confirmDetails?.action === 'accept' && acceptTarget) {
      return buildAccept(acceptTarget)
    }
    return null
  }, [api, buildCancel, buildClaim, buildAccept, acceptTarget, confirmDetails?.action, wager])

  const feeEstimate = useTxFeeEstimate({
    enabled: dialogOpen && Boolean(confirmDetails),
    estimateKey: `${confirmDetails?.action ?? 'idle'}-${wager?.pubkey ?? ''}`,
    buildTx: estimateFee,
  })

  const handleConfirm = async () => {
    setTxError(null)
    try {
      if (confirmDetails?.action === 'cancel' && wager) {
        const sig = await cancelWager.mutateAsync({
          wagerPubkey: wager.pubkey,
          wager,
        })
        setSignature(sig.signature)
        toast.success('Wager cancelled.')
        return
      }
      if (confirmDetails?.action === 'claim' && wager) {
        const sig = await claimWager.mutateAsync({ wagerPubkey: wager.pubkey })
        setSignature(sig.signature)
        return
      }
      if (confirmDetails?.action === 'accept' && acceptTarget) {
        const sig = await acceptWager.mutateAsync(acceptTarget)
        setSignature(sig)
      }
    } catch (error) {
      setTxError(mapError(error))
    }
  }

  const pending = cancelWager.isPending || claimWager.isPending || acceptWager.isPending

  const notOwnWager = walletAddress && wager && wager.maker !== walletAddress && wager.status === 'open'

  if (isLoading) {
    return (
      <div className="flex items-center gap-2 text-sm text-muted-foreground">
        <Loader2 className="size-4 animate-spin" />
        Loading wager…
      </div>
    )
  }

  if (isError) {
    return (
      <div className="space-y-4">
        <Link to="/my-wagers" className="inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground">
          <ArrowLeft className="size-4" aria-hidden />
          Back to My wagers
        </Link>
        <div className="rounded-lg border border-dashed bg-muted/40 px-6 py-12 text-center">
          <p className="font-heading text-2xl">Wager not found</p>
          <p className="mx-auto mt-2 max-w-sm text-sm text-muted-foreground">
            {error instanceof Error ? error.message : 'This wager could not be loaded.'}
          </p>
        </div>
      </div>
    )
  }

  if (!wager) {
    return (
      <div className="space-y-4">
        <Link to="/my-wagers" className="inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground">
          <ArrowLeft className="size-4" aria-hidden />
          Back to My wagers
        </Link>
        <div className="rounded-lg border border-dashed bg-muted/40 px-6 py-12 text-center">
          <p className="font-heading text-2xl">Wager not found</p>
          <p className="mx-auto mt-2 max-w-sm text-sm text-muted-foreground">
            This wager does not exist or has been removed.
          </p>
        </div>
      </div>
    )
  }

  const stake = formatStakeBaseUnits(wager.stake)
  const payout = formatStakeBaseUnits(wager.stake * 2)

  return (
    <div className="mx-auto max-w-2xl space-y-6">
      <Link
        to="/my-wagers"
        className="inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground"
      >
        <ArrowLeft className="size-4" aria-hidden />
        Back to My wagers
      </Link>

      <Card className="overflow-hidden">
        <CardContent className="space-y-5 p-6">
          <div className="flex items-start justify-between gap-3">
            <div className="min-w-0 space-y-1">
              <p className="flex items-center gap-1.5 text-sm font-medium text-muted-foreground">
                <Swords className="size-4 shrink-0" aria-hidden />
                {roleLabel}
              </p>
            </div>
            <WagerStatusBadge status={wager.status} />
          </div>

          {labels ? (
            <>
              <div className="flex min-w-0 items-center gap-2 text-sm text-muted-foreground">
                <span className="truncate">{labels.league}</span>
                {labels.isLive ? (
                  <>
                    <span aria-hidden>·</span>
                    <span className="inline-flex shrink-0 items-center gap-1 font-medium text-status-open">
                      <span
                        className="size-1.5 rounded-full bg-status-open motion-safe:animate-pulse"
                        aria-hidden
                      />
                      Live
                    </span>
                  </>
                ) : null}
              </div>

              <DuelFrame home={labels.homeTeam} away={labels.awayTeam} size="editorial" layout="stack" />
            </>
          ) : (
            <p className="font-heading text-xl">Match {wager.match_id}</p>
          )}

          <div className="grid gap-3 rounded-lg bg-muted/40 px-4 py-3.5 text-sm">
            <div className="flex flex-wrap items-center justify-between gap-x-3 gap-y-1">
              <span className="text-muted-foreground">
                {isMaker ? 'Your pick' : 'They pick'}{' '}
                <span className="font-medium text-foreground">
                  {isMaker ? backedLabel : match ? sideLabel(wager.maker_side, match) : wager.maker_side}
                </span>
              </span>
              <span className="tabular-nums text-foreground">
                <span className="font-medium">{stake}</span>{' '}
                <span className="text-muted-foreground">{stablecoin}</span>
              </span>
            </div>

            {wager.status !== 'open' ? (
              <>
                <div className="flex items-center justify-between border-t border-border/50 pt-2">
                  <span className="text-muted-foreground">If you win</span>
                  <span className="tabular-nums font-semibold text-primary">
                    {payout} {stablecoin}
                  </span>
                </div>

                <div className="flex items-center justify-between border-t border-border/50 pt-2">
                  <span className="text-muted-foreground">Opponent</span>
                  <a
                    href={explorerAddressUrl(opponent, config.cluster)}
                    target="_blank"
                    rel="noreferrer"
                    className="inline-flex items-center gap-1 font-mono text-xs underline-offset-4 hover:underline"
                  >
                    {truncateAddress(opponent)}
                    <ExternalLink className="size-3 shrink-0" aria-hidden />
                  </a>
                </div>
              </>
            ) : null}

            <div className="flex items-center justify-between border-t border-border/50 pt-2">
              <span className="text-muted-foreground">Status</span>
              <WagerStatusBadge status={wager.status} />
            </div>

            <div className="flex items-center justify-between border-t border-border/50 pt-2">
              <span className="text-muted-foreground">Wager ID</span>
              <a
                href={explorerAddressUrl(wager.pubkey, config.cluster)}
                target="_blank"
                rel="noreferrer"
                className="inline-flex items-center gap-1 font-mono text-xs underline-offset-4 hover:underline"
              >
                {truncateAddress(wager.pubkey, 6)}
                <ExternalLink className="size-3 shrink-0" aria-hidden />
              </a>
            </div>
          </div>

          {awaitingOpponent ? (
            <p className="flex items-center gap-1.5 text-sm text-muted-foreground">
              <Clock className="size-4 shrink-0" aria-hidden />
              Waiting for an opponent to accept your challenge
            </p>
          ) : showOpponent ? null : null}

          {wager.status === 'matched' ? (
            <SettlementStatus wagerPubkey={wager.pubkey} match={match} />
          ) : null}

          {notOwnWager ? (
            <div className="space-y-4 border-t pt-4">
              <OutcomePicker
                match={match}
                sides={takerOptions}
                selected={takerSide}
                onSelect={setTakerSide}
                label="Your outcome"
                showOdds={Boolean(match)}
                density="compact"
              />
              <Button
                className="min-h-12 w-full text-base"
                disabled={!walletAddress || pending}
                onClick={() => openAccept(takerSide)}
              >
                Accept challenge
              </Button>
            </div>
          ) : null}
        </CardContent>

        {canCancel || claimable ? (
          <div className="border-t px-6 py-4">
            {claimable ? (
              <Button
                className="min-h-12 w-full text-base"
                disabled={claimWager.isPending}
                onClick={openClaim}
              >
                Claim winnings
              </Button>
            ) : (
              <Button
                variant="outline"
                className="min-h-12 w-full text-base"
                onClick={openCancel}
              >
                Cancel wager
              </Button>
            )}
          </div>
        ) : null}
      </Card>

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
    </div>
  )
}
