import { Loader2 } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Dialog } from '@/components/ui/dialog'
import type { TxFeeEstimate } from '@/hooks/use-tx-fee-estimate'
import { clusterLabel } from '@/lib/config'
import { explorerTxUrl, formatSol, formatUsdc, truncateAddress } from '@/lib/format'
import { useConfig } from '@/hooks/use-api'
import { useStablecoinLabel } from '@/hooks/use-stablecoin-label'

export type ConfirmTxDetails = {
  action: 'make' | 'accept' | 'cancel' | 'claim'
  matchLabel: string
  sideLabel: string
  stakeUsdc: number
  payoutUsdc?: number
}

type ConfirmTxDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  details: ConfirmTxDetails | null
  pending: boolean
  error: string | null
  signature: string | null
  feeEstimate?: TxFeeEstimate
  feePayerAddress?: string | null
  onConfirm: () => void
}

const ACTION_LABELS: Record<ConfirmTxDetails['action'], string> = {
  make: 'Create wager',
  accept: 'Accept wager',
  cancel: 'Cancel wager',
  claim: 'Claim winnings',
}

export function ConfirmTxDialog({
  open,
  onOpenChange,
  details,
  pending,
  error,
  signature,
  feeEstimate,
  feePayerAddress,
  onConfirm,
}: ConfirmTxDialogProps) {
  const config = useConfig()
  const stablecoin = useStablecoinLabel()

  if (!details) return null

  const title = ACTION_LABELS[details.action]
  const payout =
    details.payoutUsdc ??
    (details.action === 'cancel' ? details.stakeUsdc : details.stakeUsdc * 2)

  const feeLoading = feeEstimate?.loading ?? false
  const feeLamports = feeEstimate?.lamports
  const feeError = feeEstimate?.error

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title={signature ? 'Transaction confirmed' : title}
      description={
        signature
          ? 'Your transaction was submitted successfully.'
          : 'Review the details below before signing in your wallet.'
      }
    >
      <dl className="space-y-3 text-sm">
        <div className="flex justify-between gap-4">
          <dt className="text-muted-foreground">Match</dt>
          <dd className="text-right font-medium">{details.matchLabel}</dd>
        </div>
        {details.action !== 'cancel' ? (
          <div className="flex justify-between gap-4">
            <dt className="text-muted-foreground">Your side</dt>
            <dd className="text-right font-medium">{details.sideLabel}</dd>
          </div>
        ) : null}
        <div className="flex justify-between gap-4">
          <dt className="text-muted-foreground">Stake</dt>
          <dd className="tabular-nums font-medium">
            {formatUsdc(details.stakeUsdc)} {stablecoin}
          </dd>
        </div>
        <div className="flex justify-between gap-4">
          <dt className="text-muted-foreground">
            {details.action === 'cancel'
              ? 'Refund'
              : details.action === 'claim'
                ? 'Payout'
                : 'Potential payout'}
          </dt>
          <dd className="tabular-nums font-semibold text-primary">
            {formatUsdc(payout)} {stablecoin}
          </dd>
        </div>
        <div className="flex justify-between gap-4">
          <dt className="text-muted-foreground">Network</dt>
          <dd>{clusterLabel(config.cluster)}</dd>
        </div>
        <div className="flex justify-between gap-4">
          <dt className="text-muted-foreground">Network fee</dt>
          <dd className="text-right tabular-nums">
            {feeLoading ? (
              <span className="inline-flex items-center gap-1 text-muted-foreground">
                <Loader2 className="size-3 animate-spin" aria-hidden />
                Estimating…
              </span>
            ) : feeLamports != null ? (
              <span>~{formatSol(feeLamports, { maxDecimals: 5 })} SOL</span>
            ) : feeError ? (
              <span className="text-muted-foreground">Unavailable</span>
            ) : (
              <span className="text-muted-foreground">—</span>
            )}
          </dd>
        </div>
        {feePayerAddress ? (
          <div className="flex justify-between gap-4">
            <dt className="text-muted-foreground">Fee payer</dt>
            <dd className="font-mono text-xs">
              {truncateAddress(feePayerAddress)}
            </dd>
          </div>
        ) : null}
      </dl>

      {!feeLoading ? (
        <p className="mt-3 text-xs text-muted-foreground">
          Network fee is an estimate from the exact transaction you will sign.
          Priority fees are not included unless your wallet adds them.
        </p>
      ) : null}

      {error ? (
        <p className="mt-4 rounded-md border border-destructive/30 bg-destructive/5 px-3 py-2 text-sm text-destructive">
          {error}
        </p>
      ) : null}

      {signature ? (
        <a
          href={explorerTxUrl(signature, config.cluster)}
          target="_blank"
          rel="noreferrer"
          className="mt-4 block text-sm font-medium text-primary underline-offset-4 hover:underline"
        >
          View on Solana Explorer
        </a>
      ) : null}

      <div className="mt-6 flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
        <Button
          type="button"
          variant="outline"
          onClick={() => onOpenChange(false)}
          disabled={pending}
        >
          {signature ? 'Close' : 'Back'}
        </Button>
        {!signature ? (
          <Button type="button" onClick={onConfirm} disabled={pending || feeLoading}>
            {pending ? (
              <>
                <Loader2 className="size-4 animate-spin" />
                Signing…
              </>
            ) : (
              'Confirm & sign'
            )}
          </Button>
        ) : null}
      </div>
    </Dialog>
  )
}