# Matchlock

*Peer-to-peer sports wagering on Solana.*

You pick a match, pick a side, stake some USDT. Someone on the other side picks the opposite. When the game ends, the winner claims the pot — settled by verifiable proofs from the TxLINE oracle. No house, no spread, no funny business.

Built for the Solana + TxLINE hackathon.

---

## The thing it does

Two people, one match, opposite outcomes. Escrow on-chain, settlement via CPI into TxLINE's oracle program. The winner pays the Solana fee to claim (or the keeper auto-settles if that's turned on). Loser's funds go to the winner. That's it.

The contract has a pause switch — but it's intentionally leaky. You can always cancel an open wager or settle a matched one, even when paused. Funds never get trapped by admin action.

---

## Stack

Three directories, three languages, one git root.

| Directory | What | The actual bits |
|-----------|------|-----------------|
| `blockchain/` | Anchor program | Rust, Anchor 1.1.2, LiteSVM for tests |
| `backend-go/` | Keeper + API | Go 1.24, Gin, GORM, Redis, `gagliardetto/solana-go` |
| `frontend-react/` | Web app | React 19, Vite 8 (Rolldown), TypeScript 6, Tailwind 4, pnpm |

Some notable choices that didn't make the bullet list:

- **`@coral-xyz/anchor`** — the community fork of Anchor (the official repo archived, this is what everyone uses now)
- **oxlint** instead of ESLint. It's written in Rust and it's fast. No ESLint config files anywhere in the repo.
- **`@base-ui/react`** — headless React primitives from the MUI team. Not Radix, not shadcn's default.
- **Zod 4** on the frontend for validation.
- **`canvas-confetti`** because why not.
- **No tests on the frontend.** TypeScript type-checking is the gate. That's a tradeoff we made, not an oversight.
- **No Docker.** `go run`, `pnpm dev`, `anchor build`. That's how it runs.

---

## How it all fits

```
TxLINE ──SSE──► keeper ──REST──► frontend
                  │                  │
                  ▼                  ▼
               Redis            Solana RPC
              (cache)           (program)
```

The browser talks to the keeper, not to TxLINE. API tokens stay server-side. The keeper ingests scores via SSE, caches everything in Redis, and exposes REST endpoints for matches, wagers, settlement proofs, and the leaderboard.

---

## What you'll find in each directory

### `blockchain/` — the program

The on-chain escrow. Instructions in `src/instructions/`:

- `make_wager` — create a wager, stake USDT into a PDA vault
- `accept_wager` — match an open wager, stake the opposite side
- `settle_wager` — winner or keeper submits a TxLINE Merkle proof, vault pays out
- `cancel_wager` — maker pulls the wager back (only while Open)
- `register_wallet` / `unregister_wallet` — bind a Solana address to an off-chain identity
- `update_config` — change authority, mints, or toggle pause
- `initialize` — one-time, creates the Config PDA

The tests use **LiteSVM**, an in-process Solana VM. No local validator needed. They compile the `.so` at build time and run 14 test functions covering the full lifecycle — invited taker, wrong side rejection, settlement proof validation, draw settlement, all of it.

### `backend-go/` — the keeper

One main daemon (`cmd/keeper`) and about 9 utility CLIs scattered across `cmd/`:

- `keeper` — SSE ingestor, API server, settlement worker, odds refresher, reconcile loop, leaderboard updater
- `initialize-matchlock` — one-time Config setup on devnet
- `smoke-wager` / `smoke-cancel` — full E2E tests against live devnet
- `keeper-settle` — manually drive settlement for a specific fixture
- `place-matched-wager`, `request-faucet`, `seed-leaderboard`, `activate-txline` — test helpers

Config comes from YAML, `.env`, or env vars — Viper handles the layering. There's a network consistency check at startup that catches the classic "devnet RPC + mainnet API origin" footgun.

### `frontend-react/` — the UI

Pages (all lazy-loaded):

- `/` — landing page
- `/markets` — match browser with live/finished/upcoming tabs, odds cells, challenge slip
- `/my-wagers/` — active wagers list and detail view
- `/open` — wagers waiting for an opponent
- `/history` — settled wagers and PnL
- `/invites` — direct email challenges
- `/leaderboard` — ranked players by PnL
- `/profile` — wallet linking, stats
- `/login` — magic link auth

The wager flow goes: **select match → pick side → enter stake → simulate → confirm dialog → sign**. The confirm dialog shows the match, side, stake, network fee estimate, and cluster badge before the wallet pops up.

There's a `MatchStreamSubscriber` component (renders `null`) that opens a single global `EventSource` to the keeper and splices live score updates directly into the React Query cache. SSE for push, HTTP polling as fallback.

An `OptimisticWagersStore` (zustand) tracks wagers that haven't confirmed yet — they show up immediately in the UI and disappear after 120s if the tx never lands.

---

## Running it

### Prerequisites you'll actually need

- Node.js 22+, pnpm 10
- Go 1.24+
- Rust 1.84+, SBF target installed
- Anchor 1.1.2 CLI
- PostgreSQL 16+
- Redis 7+
- Solana CLI configured for devnet
- A Brevo account + API key (magic link emails) — skip this if you don't need auth

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

### Tests

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

## Design philosophy (the 253-line spec)

There's a `design.md` in the root. It specifies everything — colors, typography (Manrope for UI, Garamond for display), spacing, border radii, component inventory with implementation status. The interesting bit: **zero box shadows**. All depth comes from borders and color contrast, which makes the UI feel flatter and more deliberate.

There are actually two themes: the landing page uses a dark scheme (`#282828` bg, `#e64040` red), while the app itself uses an "Ember" theme (`#1e1814` bg, `#d4763f` amber). They're intentionally different — the landing page sells the product, the app is where you work.

---

## Things that went wrong (and you should know about)

The [`docs/integration-journal.md`](docs/integration-journal.md) has the full list, but the highlights:

- **Vite 8 + `vite-plugin-node-polyfills`** — Rolldown broke the polyfill shim. The fix was to remove the plugin and manually polyfill `Buffer` via `src/polyfills.ts`. If you upgrade Vite, watch for `init_dist is not a function`.
- **TxLINE network mismatch** — using mainnet API origin with a devnet subscription (or vice versa) fails silently. The keeper config now validates this at startup.
- **GORM column naming** — `NetPnL` auto-migrates to `net_pn_l` (not `net_pnl`). Fixed with a `gorm:"column:net_pnl"` tag.
- **LiteSVM bundles SPL Token + ATA** — don't try to load them from `/tmp/`. Just use `with_default_programs()`.
- **SimulateTransaction returns AccountNotFound** — for brand-new wallets even after funding. `send-without-preflight` works for faucet/make/accept.
- **Keeper offline + matched wagers** — when the keeper goes down, matched wagers stay matched. The winner-claim settlement model (`KEEPER_AUTO_SETTLE=false`) solves this because users can claim without the keeper.

---

## Project structure

```
matchlock/
├── blockchain/               # Anchor program (Rust)
│   ├── programs/blockchain/src/
│   ├── programs/blockchain/tests/
│   └── Anchor.toml
├── backend-go/               # Go keeper service
│   ├── cmd/                  # daemon + CLIs
│   ├── internal/             # api, keeper, solana, db, leaderboard, config
│   └── api/openapi.yaml
├── frontend-react/           # React SPA
│   └── src/
│       ├── hooks/            # React Query hooks
│       ├── pages/            # route pages
│       ├── components/       # UI components
│       └── lib/              # API client, Anchor helpers, etc.
├── docs/
│   └── integration-journal.md
└── design.md                 # design system spec
```

---

The integration journal has 6000+ words of actual field notes from building this thing. If you're extending the project or debugging something weird, read it first.
