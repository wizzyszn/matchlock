import type { Cluster } from '@/lib/config'
import { explorerClusterParam } from '@/lib/config'

const USDC_DECIMALS = 6
const LAMPORTS_PER_SOL = 1_000_000_000

export function truncateAddress(address: string, chars = 4): string {
  if (address.length <= chars * 2 + 1) return address
  return `${address.slice(0, chars)}…${address.slice(-chars)}`
}

export function baseUnitsToUsdc(amount: number | bigint): number {
  return Number(amount) / 10 ** USDC_DECIMALS
}

export function usdcToBaseUnits(amount: number): bigint {
  return BigInt(Math.round(amount * 10 ** USDC_DECIMALS))
}

export function formatUsdc(amount: number, options?: { maxDecimals?: number }): string {
  const maxDecimals = options?.maxDecimals ?? 6
  return amount.toLocaleString('en-US', {
    minimumFractionDigits: 0,
    maximumFractionDigits: maxDecimals,
  })
}

export function formatStakeBaseUnits(stake: number): string {
  return formatUsdc(baseUnitsToUsdc(stake), { maxDecimals: 6 })
}

export function formatSol(
  lamports: number,
  options?: { maxDecimals?: number },
): string {
  const maxDecimals = options?.maxDecimals ?? 6
  const sol = lamports / LAMPORTS_PER_SOL
  if (sol > 0 && sol < 10 ** -maxDecimals) {
    return `<${(10 ** -maxDecimals).toFixed(maxDecimals)}`
  }
  return sol.toLocaleString('en-US', {
    minimumFractionDigits: 0,
    maximumFractionDigits: maxDecimals,
  })
}

export function explorerTxUrl(signature: string, cluster: Cluster): string {
  return `https://explorer.solana.com/tx/${signature}${explorerClusterParam(cluster)}`
}

export function explorerAddressUrl(address: string, cluster: Cluster): string {
  return `https://explorer.solana.com/address/${address}${explorerClusterParam(cluster)}`
}