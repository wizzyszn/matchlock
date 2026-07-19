# Matchlock

*Peer-to-peer sports wagering on Solana.*

You pick a match, pick a side, stake some USDT. Someone on the other side picks the opposite. When the game ends, the winner claims the pot — settled by verifiable proofs from the TxLINE oracle. No house, no spread, no funny business.

Built for the Solana + TxLINE hackathon.

**Deployed app:** [https://matchlock-u1ww.onrender.com/](https://matchlock-u1ww.onrender.com/)  
**Demo video:** [https://www.youtube.com/watch?v=I43HTTdQVLw](https://www.youtube.com/watch?v=I43HTTdQVLw)

---

## Table of contents

- [Architecture](#architecture)
  - [Match synchronization](#1-match-synchronization-real-time-state)
  - [Wagering](#2-wagering-flow-p2p-play)
  - [Settlement (winner-claim)](#3-settlement--resolution-flow-permissionless-claim)
  - [Settlement (keeper auto-settle)](#3b-alternative-settlement-flow-keeper-auto-settle)
  - [Leaderboard](#4-leaderboard)
- [TxLINE endpoints](#txline-endpoints)
- [Stack](#stack)
- [What's in each directory](#whats-in-each-directory)
  - [blockchain/](#blockchain--the-program)
  - [backend-go/](#backend-go--the-keeper)
  - [frontend-react/](#frontend-react--the-ui)
- [Running it](#running-it)
- [Tests](#tests)
- [Current devnet addresses](#current-devnet-addresses)
- [Design philosophy](#design-philosophy)
- [Known rough edges](#known-rough-edges)

---

## Architecture

The platform relies on a clear separation of data tracking and on-chain execution. Instead of a central authority settling wagers, the protocol behaves as a trustless escrow matching TxLINE's sports data on-chain.

### 1. Match synchronization (real-time state)

*How live match scores go from the field to the user interface.*

```text
┌─────────────┐   SSE    ┌──────────────┐  Write  ┌──────────────┐
│   TxLINE    │ ───────► │  backend-go  │ ──────► │    Redis     │
│   Oracle    │          │  (Keeper)    │         │ (Match Cache)│
└─────────────┘          └──────┬───────┘         └──────────────┘
                                │ SSE fan-out
                                ▼
                         ┌──────────────┐
                         │   Frontend   │ (Live scores update UI)
                         └──────────────┘
```

The Keeper worker consumes TxLINE scores directly via SSE. Validated state updates are pushed immediately to Redis for O(1) reads. Connected clients receive live match ticks via `/matches/stream` (SSE), which surgically updates React Query caches.

### 2. Wagering flow (P2P play)

*How users enter into predictions programmatically.*

```text
┌──────────────┐ 1. make_wager   ┌──────────────┐
│ Maker Wallet │ ──────────────► │  Matchlock   │ (Creates wager, escrows maker stake)
└──────────────┘                 │   Program    │
┌──────────────┐ 2. accept_wager │ (Solana)     │
│ Taker Wallet │ ──────────────► │              │ (Escrows taker stake, status = Matched)
└──────────────┘                 └──────────────┘
```

Maker signs the tx paying the stake in USDT. Escrow vaults are initialized per wager (not pooled). Taker spots an open wager, signs it, and escrows an identical USDT amount. If the Maker set an `invited_taker`, only that specific wallet can accept.

### 3. Settlement & resolution flow (permissionless claim)

*How winnings are resolved once a match ends, without keeper gas fees. (Default mode — `KEEPER_AUTO_SETTLE=false`)*

```text
┌──────────────┐ 3. final state ┌──────────────┐ 4. close match ┌──────────────┐
│ TxLINE / SSE │ ─────────────► │   Keeper     │ ─────────────► │  Matchlock   │
└──────────────┘                └──────┬───────┘                │   Program    │
                                       │                        └──────▲───────┘
                              5. GET /settlement-proof                │ 8. CPI (Verify proof)
                                       │                               │    + Payout 2x Stake
                                       ▼                        ┌──────┴───────┐
                                ┌──────────────┐ 6. Sign       │ Winner Wallet│
                                │   Frontend   │ ────────────► │ (Claimant)   │
                                └──────────────┘ 7. settle_wager              │
                                                                └──────────────┘
```

When TxLINE emits `is_final=true`, the Keeper calls `close_match` on-chain to freeze betting. The winner visits their dashboard; the frontend calls `GET /settlement-proof`. The Keeper fetches the `StatValidation` (Merkle proof) from TxLINE confirming the winning condition. The winner's wallet signs `settle_wager` with the proof as instruction args. Matchlock CPIs into TxLINE's `validate_stat` to verify the oracle, then transfers 2× USDT to the winner.

### 3b. Alternative settlement flow (keeper auto-settle)

*How winnings are automatically paid out when the protocol pays Solana fees. (Opt-in — `KEEPER_AUTO_SETTLE=true`)*

```text
┌──────────────┐ 1. final state ┌──────────────┐ 2. close match ┌──────────────┐
│ TxLINE / SSE │ ─────────────► │   Keeper     │ ─────────────► │  Matchlock   │
└──────────────┘                └──────┬───────┘                │   Program    │
                                       │ 3. fetch proofs        └──────▲───────┘
                                       │                                │ 4. settle_wager
                                       ▼                                │    + Payout 2x Stake
                                ┌──────────────┐                ┌──────┴───────┐
                                │ TxLINE API   │                │ Winner Wallet│
                                └──────────────┘                │ (Receives)   │
                                                                 └──────────────┘
```

Same finalization trigger, but the Keeper worker loops over all matched wager accounts, fetches proofs from TxLINE, and submits `settle_wager` on-chain using its own keypair as fee payer.

### 4. Leaderboard

*Post-settlement ranking tracked in Postgres.*

```text
Keeper Worker ──► RecordSettlement() ──► leaderboard_entries (Postgres)
                                               │
                                               ▼
                                        GET /leaderboard
                                        GET /leaderboard/me  (auth)
                                        GET /leaderboard/stats
                                               │
                                               ▼
                                        React Query hooks (60s polling)
                                               │
                                               ▼
                                        Leaderboard page
                                               (stats cards, rank, player list)
```

After each successful `settleOne`, the keeper upserts `LeaderboardEntry` rows keyed by user ID: winner gets `+stake` PnL, loser gets `-stake` PnL, both get `2× stake` volume. Unlinked wallets are skipped silently.

---

## TxLINE endpoints

The following TxLINE API endpoints power the application, covering auth, live scores, odds, fixture data, and on-chain settlement proofs.

| Endpoint | Purpose | Used by |
|---|---|---|
| `POST {origin}/auth/guest/start` | Fetch guest JWT for dual-header auth (`Authorization` + `X-Api-Token`) | Keeper startup, all downstream requests |
| `GET /api/scores/stream` | SSE feed of live score updates (goals, game state, final signal) | Keeper `StreamScores` — real-time match sync |
| `GET /api/scores/snapshot/{fixtureId}` | Terminal score rows with game state, clock, home/away goals | Keeper settlement (build `ScoreUpdate`, verify `is_final`) |
| `GET /api/scores/stat-validation` | Merkle proof for on-chain CPI (`?fixtureId=X&seq=Y&statKey=Z`) | Settlement — keeper fetches proof; winner or keeper submits `settle_wager` |
| `GET /api/fixtures/snapshot` | Upcoming fixture schedule (teams, competition, start time) | Keeper schedule hydration, frontend market browser |
| `GET /api/fixtures/validation` | Merkle proof for fixture integrity (`?fixtureId=X&timestamp=Y`) | Settlement proof enrichment |
| `GET /api/odds/snapshot/{fixtureId}` | Latest demargined 1X2 odds lines | Keeper odds refresher, frontend odds display |
| `GET /api/odds/snapshot/{fixtureId}?asOf={timestamp}` | Historical odds at a specific time | Keeper odds hydration (pre-kickoff baseline) |
| `GET /api/odds/updates/{fixtureId}` | Live odds from the 5-minute in-memory cache | Keeper odds refresher during live matches |

All data requests use dual-header auth: `Authorization: Bearer {guest_jwt}` + `X-Api-Token: {activated_api_token}`. The guest JWT is fetched lazily at startup via `/auth/guest/start` with automatic retry and 401-triggered refresh.

---

## Stack

Three directories, three languages, one git root.

| Directory | What | The actual bits |
|-----------|------|-----------------|
| `blockchain/` | Anchor program | Rust, Anchor 1.1.2, LiteSVM tests |
| `backend-go/` | Keeper + API | Go 1.24, Gin, GORM, Redis, `gagliardetto/solana-go` |
| `frontend-react/` | Web app | React 19, Vite 8 (Rolldown), TypeScript 6, Tailwind 4, pnpm |

Notable choices:

- **`@coral-xyz/anchor`** — community fork of Anchor (official repo archived, this is what everyone uses now)
- **oxlint** instead of ESLint. Written in Rust, fast. No ESLint config files anywhere.
- **`@base-ui/react`** — headless React primitives from the MUI team. Not Radix, not shadcn.
- **Zod 4** on the frontend for validation.
- **`canvas-confetti`** because why not.
- **No frontend tests.** TypeScript type-checking is the gate. Tradeoff, not oversight.
- **No Docker.** `go run`, `pnpm dev`, `anchor build`. That's how it runs.

---

## What's in each directory

### `blockchain/` — the program

The on-chain escrow. Instructions in `src/instructions/`:

| Instruction | What it does |
|---|---|
| `make_wager` | Create a wager, stake USDT into a PDA vault |
| `accept_wager` | Match an open wager, stake the opposite side |
| `settle_wager` | Winner or keeper submits a TxLINE Merkle proof, vault pays out |
| `void_wager` | Refund both participants when neither side won (draw outcome) |
| `cancel_wager` | Maker pulls the wager back (only while Open) |
| `register_wallet` / `unregister_wallet` | Bind a Solana address to an off-chain identity |
| `update_config` | Change authority, mints, or toggle pause |
| `close_match` | Prevent new wagers on a finished match |
| `initialize` | One-time, creates the Config PDA |

Tests use **LiteSVM** — an in-process Solana VM. No local validator needed.

### `backend-go/` — the keeper

One main daemon and several utility CLIs across `cmd/`:

| CLI | Purpose |
|---|---|
| `keeper` | Main daemon — SSE ingestor, API server, settlement worker, reconcile loop, leaderboard updater |
| `initialize-matchlock` | One-time Config setup on devnet |
| `smoke-wager` / `smoke-cancel` | Full E2E tests against live devnet |
| `keeper-settle` | Manual ops crank — settle a specific fixture by ID |
| `place-matched-wager`, `request-faucet`, `seed-leaderboard`, `activate-txline` | Test helpers |

Config comes from YAML, `.env`, or env vars — Viper handles the layering. Network consistency check at startup catches the classic "devnet RPC + mainnet API origin" footgun.

### `frontend-react/` — the UI

Pages (all lazy-loaded):

| Route | Page |
|---|---|
| `/` | Landing page |
| `/markets` | Match browser (live/finished/upcoming tabs, odds cells, challenge slip) |
| `/my-wagers/` | Active wagers list + detail view |
| `/open` | Wagers waiting for an opponent |
| `/history` | Settled wagers and PnL |
| `/invites` | Direct email challenges |
| `/leaderboard` | Ranked players by PnL |
| `/profile` | Wallet linking, stats |
| `/login` | Magic link auth |

Wager flow: **select match → pick side → enter stake → simulate → confirm dialog → sign**. The confirm dialog shows match, side, stake, network fee, and cluster badge before the wallet pops up.

`MatchStreamSubscriber` (renders `null`) opens a single global `EventSource` to the keeper and splices live score updates directly into React Query cache. SSE for push, HTTP polling as fallback.

`OptimisticWagersStore` (zustand) tracks pre-confirmation wagers — they show immediately and disappear after 120s if the tx never lands.

---

## Running it

### Prerequisites

- Node.js 22+, pnpm 10
- Go 1.24+
- Rust 1.84+, SBF target installed
- Anchor 1.1.2 CLI
- PostgreSQL 16+
- Redis 7+
- Solana CLI configured for devnet
- Brevo account + API key (skip if you don't need auth)

### Setup

```bash
# Frontend
cd frontend-react
pnpm install
cp .env.example .env
# edit VITE_PROGRAM_ID, VITE_SOLANA_RPC_URL, VITE_BACKEND_URL

# Backend
cd ../backend-go
go mod download
cp .env.example .env
# edit DATABASE_URL, JWT_ACCESS_SECRET, BREVO_*, TXLINE_API_TOKEN

# Database
createdb matchlock  # tables auto-migrate on startup

# Redis
redis-server
```

### Run locally

```bash
# Terminal 1 — keeper (API + SSE ingestor + workers)
cd backend-go && go run ./cmd/keeper

# Terminal 2 — frontend
cd frontend-react && pnpm dev
```

Open `http://localhost:5173`.

### Deploy the program

```bash
cd blockchain
anchor build
anchor program deploy --provider.cluster devnet

# Initialize the Config PDA
cd ../backend-go && go run ./cmd/initialize-matchlock
```

---

## Tests

```bash
# On-chain tests (LiteSVM — fast, no validator needed)
cd blockchain && NO_DNA=1 anchor build && NO_DNA=1 cargo test

# Backend tests (uses miniredis, no real Redis needed)
cd backend-go && go test ./...

# Frontend type-check (the only test gate we have)
cd frontend-react && pnpm run build
```

---

## Current devnet addresses

| Thing | Address |
|-------|---------|
| Matchlock program | `B39Vk22T2VPpqBEbGkW51BzFC6sNeeiQQ1mqwdvCJ2H4` |
| Config PDA | `GJLWM4UNP445RJvDHMr38N8XcFMwn18cLTDLyAEx7psN` |
| USDT (TxLINE devnet) | `ELWTKspHKCnCfCiCiqYw1EDH77k8VCP74dK9qytG2Ujh` |
| TxLINE oracle program | `6pW64gN1s2uqjHkn1unFeEjAwJkPGHoppGvS715wyP2J` |

---

## Design philosophy

There's a `design.md` in the root specifying colors, typography (Manrope for UI, Garamond for display), spacing, border radii, and component inventory with implementation status. The notable choice: **zero box shadows**. All depth comes from borders and color contrast.

Two themes: the landing page uses a dark scheme (`#282828` bg, `#e64040` red), while the app uses an "Ember" theme (`#1e1814` bg, `#d4763f` amber). Intentionally different — the landing page sells the product, the app is where you work.

---

## Known rough edges

The full list lives in [`docs/integration-journal.md`](docs/integration-journal.md), but the highlights:

- **Vite 8 + `vite-plugin-node-polyfills`** — Rolldown broke the polyfill shim. Fix: remove the plugin and manually polyfill `Buffer` via `src/polyfills.ts`.
- **TxLINE network mismatch** — mainnet API origin + devnet subscription fails silently. Config validates this at startup.
- **GORM column naming** — `NetPnL` auto-migrates to `net_pn_l`. Fixed with `gorm:"column:net_pnl"` tag.
- **LiteSVM bundles SPL Token + ATA** — don't load them from `/tmp/`. Use `with_default_programs()`.
- **SimulateTransaction returns AccountNotFound** — for brand-new wallets even after funding. `send-without-preflight` works for faucet/make/accept.
- **Keeper offline + matched wagers** — winner-claim settlement model (`KEEPER_AUTO_SETTLE=false`) solves this: users claim without the keeper.
- **Refundable state is keeper-only** — `refundable` (draw outcome) currently requires keeper auto-settle or manual crank; no frontend "Claim refund" button yet.
