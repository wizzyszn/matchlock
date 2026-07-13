import { AnchorError, ProgramError } from '@coral-xyz/anchor'

import { ApiClientError } from '@/lib/api'

const AUTH_ERROR_MESSAGES: Record<string, string> = {
  INVALID_TOKEN:
    'This sign-in link has expired or was already used. Request a new one — links are single-use and expire in 15 minutes.',
  RATE_LIMITED:
    'Too many sign-in attempts. Wait a minute before requesting another link.',
  UNAUTHORIZED: 'Your session has expired. Please sign in again.',
}

export function mapAuthError(error: unknown): string {
  if (error instanceof ApiClientError) {
    if (error.code && AUTH_ERROR_MESSAGES[error.code]) {
      return AUTH_ERROR_MESSAGES[error.code]
    }
    return error.message
  }
  if (error instanceof Error) {
    return error.message
  }
  return 'Authentication failed. Please try again.'
}

const ERROR_MESSAGES: Record<string, string> = {
  Unauthorized: 'You are not authorized to perform this action.',
  InvalidStatus: 'This wager is no longer in the right state — it may have been accepted or cancelled.',
  InvalidMatchId: 'Invalid match ID.',
  InvalidStake: 'Stake must be greater than zero.',
  MatchClosed: 'This match is already closed for wagering.',
  CannotAcceptOwnWager: 'You cannot accept your own wager.',
  InvalidTakerSide: 'Pick a different outcome than the maker.',
  WagerNotOpen: 'This wager is no longer open.',
  InvalidMint: 'Token mint does not match the program configuration.',
  InvalidTxlineProgram: 'TxLINE program mismatch.',
  InvalidWinningSide: 'Invalid winning side for settlement.',
  ValidationFailed: 'TxLINE stat validation failed.',
  AlreadySettled: 'This wager has already been settled.',
}

export function mapTransactionError(error: unknown): string {
  if (error instanceof AnchorError) {
    const name = error.error.errorCode.code
    const mapped = ERROR_MESSAGES[name]
    if (mapped) return mapped
    return error.error.errorMessage ?? `Transaction failed (${name}).`
  }

  if (error instanceof ProgramError) {
    return error.message
  }

  if (error instanceof Error) {
    const anchorMatch = error.message.match(/Error Code: (\w+)/)
    if (anchorMatch) {
      const mapped = ERROR_MESSAGES[anchorMatch[1]]
      if (mapped) return mapped
    }

    if (error.message.includes('User rejected')) {
      return 'Transaction cancelled in wallet.'
    }

    if (error.message.includes('insufficient funds')) {
      return 'Insufficient SOL for transaction fees.'
    }

    return error.message
  }

  return 'Transaction failed. Please try again.'
}
