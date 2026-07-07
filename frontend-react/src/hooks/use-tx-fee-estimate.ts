import { useAnchorWallet, useConnection } from '@solana/wallet-adapter-react'
import { useEffect, useState } from 'react'
import type { Transaction } from '@solana/web3.js'

import { estimateTransactionFee } from '@/lib/wager-tx'

export type TxFeeEstimate = {
  lamports: number | null
  loading: boolean
  error: string | null
}

type UseTxFeeEstimateOptions = {
  enabled: boolean
  /** Changes here trigger a fresh estimate (e.g. action + inputs). */
  estimateKey: string
  buildTx: () => Promise<Transaction | null>
}

const idle: TxFeeEstimate = { lamports: null, loading: false, error: null }

export function useTxFeeEstimate({
  enabled,
  estimateKey,
  buildTx,
}: UseTxFeeEstimateOptions): TxFeeEstimate {
  const { connection } = useConnection()
  const wallet = useAnchorWallet()
  const [state, setState] = useState<TxFeeEstimate>(idle)

  useEffect(() => {
    if (!enabled || !wallet?.publicKey) {
      setState(idle)
      return
    }

    let cancelled = false
    setState({ lamports: null, loading: true, error: null })

    void (async () => {
      try {
        const tx = await buildTx()
        if (!tx || cancelled) return
        const lamports = await estimateTransactionFee(connection, wallet, tx)
        if (!cancelled) {
          setState({ lamports, loading: false, error: null })
        }
      } catch (error) {
        if (!cancelled) {
          setState({
            lamports: null,
            loading: false,
            error:
              error instanceof Error
                ? error.message
                : 'Could not estimate network fee',
          })
        }
      }
    })()

    return () => {
      cancelled = true
    }
  }, [enabled, estimateKey, buildTx, connection, wallet])

  return state
}