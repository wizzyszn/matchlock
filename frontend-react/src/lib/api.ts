import type { AppConfig } from '@/lib/config'

export type Side = 'home' | 'draw' | 'away'

export type WagerStatus = 'open' | 'matched' | 'settled' | 'cancelled'

export type MatchOdds = {
  home?: number | null
  draw?: number | null
  away?: number | null
}

export type SettlementState =
  | 'match_live'
  | 'match_ended_unverified'
  | 'queued'
  | 'retrying'
  | 'settled'
  | 'failed'
  | 'not_applicable'

export type SettlementProof = {
  winning_side_code: number
  winning_side: string
  fixture_id: number
  seq: number
  stat_key: number
  validation: Record<string, unknown>
  merkle_root: string
  epoch_day: number
  daily_scores_pda: string
  txline_program_id: string
}

export type WagerSettlementStatus = {
  state: SettlementState
  message: string
  match_final: boolean
  settled_at?: string | null
  tx_signature?: string
  updated_at: string
}

export type Match = {
  match_id: string
  fixture_id: number
  status: string
  is_final: boolean
  final_source?: string
  home_goals?: number | null
  away_goals?: number | null
  seq: number
  updated_at: string
  finalized_at?: string | null
  start_time?: number
  competition_id?: number
  competition?: string
  fixture_group_id?: number
  participant1_id?: number
  participant2_id?: number
  home_team?: string
  away_team?: string
  sport_id?: number
  country_id?: number
  odds?: MatchOdds | null
}

export type Wager = {
  pubkey: string
  maker: string
  invited_taker?: string
  taker: string
  match_id: string
  maker_side: Side
  taker_side?: Side
  stake: number
  status: WagerStatus
}

export type WalletLink = {
  pubkey: string
  label?: string
  is_primary: boolean
  linked_at: string
}

export type WalletBindingStatus =
  | 'unlinked'
  | 'linked_to_you'
  | 'linked_to_other'

export type WalletBinding = {
  pubkey: string
  status: WalletBindingStatus
  owner_label?: string
  owner_user_id?: string
  linked_to_you: boolean
  owned_by_other: boolean
}

export type UserProfile = {
  id: string
  email: string
  display_name?: string
  wallets: WalletLink[]
}

export type UserLookup = {
  email: string
  user_id?: string
  has_account: boolean
  primary_wallet?: string
}

export type WagerInvite = {
  id: string
  maker_email: string
  recipient_email: string
  wager_pubkey?: string
  match_id: string
  maker_side: Side
  stake: number
  home_team?: string
  away_team?: string
  status: 'pending' | 'accepted' | 'declined' | 'expired'
  created_at: string
}

export type HealthStatus = { status: 'ok' }

export type ReadyStatus = {
  status: 'ready'
  checks: {
    redis: string
    rpc: string
    txline: string
  }
}

export type ApiError = {
  error: string
  code: string
}

export class ApiClientError extends Error {
  readonly status: number
  readonly code?: string

  constructor(message: string, status: number, code?: string) {
    super(message)
    this.name = 'ApiClientError'
    this.status = status
    this.code = code
  }
}

type ListWagersParams = {
  match_id?: string
  status?: WagerStatus
  wallet?: string
}

export class MatchlockApi {
  readonly baseUrl: string

  constructor(config: Pick<AppConfig, 'backendUrl'>) {
    this.baseUrl = config.backendUrl.replace(/\/$/, '')
  }

  private async request<T>(path: string, init?: RequestInit): Promise<T> {
    const response = await fetch(`${this.baseUrl}${path}`, {
      credentials: 'include',
      ...init,
      headers: {
        Accept: 'application/json',
        ...init?.headers,
      },
    })

    if (!response.ok) {
      let message = `Request failed (${response.status})`
      let code: string | undefined

      try {
        const body = (await response.json()) as ApiError
        message = body.error ?? message
        code = body.code
      } catch {
        // ignore non-JSON error bodies
      }

      throw new ApiClientError(message, response.status, code)
    }

    if (response.status === 204) {
      return undefined as T
    }
    return (await response.json()) as T
  }

  getHealthz() {
    return this.request<HealthStatus>('/healthz')
  }

  getReadyz() {
    return this.request<ReadyStatus>('/readyz')
  }

  listMatches() {
    return this.request<Match[]>('/matches')
  }

  getMatch(id: string) {
    return this.request<Match>(`/matches/${encodeURIComponent(id)}`)
  }

  listWagers(params: ListWagersParams = {}) {
    const search = new URLSearchParams()
    if (params.match_id) search.set('match_id', params.match_id)
    if (params.status) search.set('status', params.status)
    if (params.wallet) search.set('wallet', params.wallet)
    const query = search.toString()
    return this.request<Wager[]>(`/wagers${query ? `?${query}` : ''}`)
  }

  getWager(pubkey: string) {
    return this.request<Wager>(`/wagers/${encodeURIComponent(pubkey)}`)
  }

  getWagerSettlement(pubkey: string) {
    return this.request<WagerSettlementStatus>(
      `/wagers/${encodeURIComponent(pubkey)}/settlement`,
    )
  }

  getWagerSettlementProof(pubkey: string) {
    return this.request<SettlementProof>(
      `/wagers/${encodeURIComponent(pubkey)}/settlement-proof`,
    )
  }

  requestMagicLink(email: string) {
    return this.request<void>('/auth/magic-link', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email }),
    })
  }

  verifyMagicLink(token: string) {
    return this.request<UserProfile>(
      `/auth/verify?token=${encodeURIComponent(token)}`,
    )
  }

  refreshSession() {
    return this.request<UserProfile>('/auth/refresh', { method: 'POST' })
  }

  logout() {
    return this.request<void>('/auth/logout', { method: 'POST' })
  }

  getMe() {
    return this.request<UserProfile>('/auth/me')
  }

  updateProfile(input: { display_name: string }) {
    return this.request<UserProfile>('/auth/me', {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(input),
    })
  }

  checkWalletBinding(pubkey: string) {
    return this.request<WalletBinding>(
      `/auth/wallets/check?pubkey=${encodeURIComponent(pubkey)}`,
    )
  }

  getWalletLinkChallenge(pubkey: string) {
    return this.request<{ message: string; pubkey: string }>(
      '/auth/wallets/challenge',
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ pubkey }),
      },
    )
  }

  linkWallet(input: {
    pubkey: string
    message: string
    signature: string
    label?: string
  }) {
    return this.request<WalletLink>('/auth/wallets/link', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(input),
    })
  }

  setPrimaryWallet(pubkey: string) {
    return this.request<void>(
      `/auth/wallets/${encodeURIComponent(pubkey)}/primary`,
      { method: 'POST' },
    )
  }

  unlinkWallet(pubkey: string) {
    return this.request<void>(`/auth/wallets/${encodeURIComponent(pubkey)}`, {
      method: 'DELETE',
    })
  }

  lookupUser(email: string) {
    return this.request<UserLookup>(
      `/users/lookup?email=${encodeURIComponent(email)}`,
    )
  }

  createInvite(input: {
    recipient_email: string
    wager_pubkey?: string
    match_id: string
    maker_side: Side
    stake: number
    home_team?: string
    away_team?: string
  }) {
    return this.request<WagerInvite>('/challenges/invites', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(input),
    })
  }

  listInvites() {
    return this.request<WagerInvite[]>('/challenges/invites')
  }

  getInvite(id: string) {
    return this.request<WagerInvite>(
      `/challenges/invites/${encodeURIComponent(id)}`,
    )
  }

  updateInvite(id: string, status: 'accepted' | 'declined') {
    return this.request<WagerInvite>(
      `/challenges/invites/${encodeURIComponent(id)}`,
      {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ status }),
      },
    )
  }

  attachInviteWager(id: string, wagerPubkey: string) {
    return this.request<WagerInvite>(
      `/challenges/invites/${encodeURIComponent(id)}/wager`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ wager_pubkey: wagerPubkey }),
      },
    )
  }
}