# Matchlock Integration Journal

Living log of experience, hurdles, and sweetspots while building the dApp and integrating with TxLINE.

**Maintained by:** developers + Grok (via `/txline-dev-journal` skill)  
**Last updated:** 2026-07-08

---

## How to read this

| Tag              | Meaning                               |
| ---------------- | ------------------------------------- |
| 🚧 **Hurdle**    | Problem that cost time — include fix  |
| ✨ **Sweetspot** | What worked well — reuse this pattern |
| 💡 **Tip**       | Small note worth remembering          |

Areas: `blockchain` · `backend-go` · `frontend-react` · `txline-api` · `cross-cutting`

---

## Quick reference (top lessons)

1. **Network consistency** — devnet RPC + devnet program ID + `txline-dev.txodds.com` must all match. Mixing mainnet/devnet breaks auth and activation. (`txline-api`)
2. **Skill layering** — implement in production skills; test in test skills; log learnings here. Don't mix concerns in one session. (`cross-cutting`)
3. **Simulate before sign** — every keeper and frontend tx should simulate first; saves hours debugging failed sends. (`cross-cutting`)
4. **Backend owns SSE** — browser should not connect to TxLINE SSE directly; Go backend ingests and exposes HTTP API. (`backend-go` + `frontend-react`)
5. **World Cup free tier** — skip TxL purchase for hackathon; use World Cup docs for instant API access. (`txline-api`)
6. **Winner-claim settlement** — `KEEPER_AUTO_SETTLE=false` by default; backend brokers TxLINE proofs, winner wallet signs `settle_wager` as `settler` and pays SOL. (`blockchain` + `backend-go` + `frontend-react`)
7. **Outcome stat probe** — at settlement seq, probe TxLINE stats `1002`/`1003` for `value > 0`; do not trust snapshot `participant1IsHome` alone. (`backend-go`)
8. **Agenda 4 auth** — magic link via Brevo transactional email; access JWT + refresh token in httpOnly cookies (`matchlock_access` / `matchlock_refresh`); Postgres/GORM for users, sessions, wallet links, wager invites. (`backend-go` + `frontend-react`)
9. **Vite 8 + node polyfills** — `vite-plugin-node-polyfills@0.28` breaks Rolldown pre-bundling (`init_dist is not a function`); use manual `src/polyfills.ts` + `resolve.alias.buffer` instead. (`frontend-react`)

---

## Entries

### 2026-07-05 — SSE streaming with polling fallback for live matches

**Area:** frontend-react · backend-go  
**Type:** ✨ Sweetspot

#### Context

Real-time match updates (scores, game states, kickoff times) needed lower latency for live betting, but we also needed reliability if sockets drop.

#### What worked

We implemented an SSE (Server-Sent Events) endpoint (`/matches/stream`) in the Go backend that fan-outs cache updates. The React frontend uses a single global `EventSource` connection in a `useMatchStream` hook to surgically splice incoming `Match` payloads into the `react-query` cache (`queryClient.setQueryData`).

#### Takeaway

SSE is perfect for one-way score feeds. By keeping `react-query`'s standard HTTP fetching in place alongside SSE, we get the best of both worlds: instant push updates via the stream, with robust HTTP polling as a fallback for initial hydration and seamless recovery if the connection drops.

---

### 2026-07-03 — Vite 8 Rolldown breaks vite-plugin-node-polyfills

**Area:** frontend-react  
**Type:** 🚧 Hurdle

#### Context

Login page stuck on static HTML fallback “Loading Matchlock…” — React never mounted.

#### What happened

Console: `Uncaught TypeError: vite-plugin-node-polyfills: init_dist is not a function`. Vite 8.1 Rolldown pre-bundled `buffer` shim into a circular `.vite/deps/vite-plugin-node-polyfills_shims_buffer.js` that called `init_dist()` before the ESM wrapper was ready.

#### Resolution / takeaway

Removed `vite-plugin-node-polyfills` from `vite.config.ts`; kept `src/polyfills.ts` (`import { Buffer } from 'buffer'`) + `resolve.alias.buffer` + `optimizeDeps.include: ['buffer']`. Cleared `node_modules/.vite` and restarted dev server. Unrelated `ws://127.0.0.1:5500` error is a browser Live Server extension, not Matchlock.

### 2026-06-30 — TxLINE network mismatch is the #1 integration footgun

**Area:** txline-api  
**Type:** 🚧 Hurdle

#### Context

Activating API tokens or calling data endpoints after on-chain subscribe.

#### What happened

Using `https://txline.txodds.com` (mainnet origin) with a devnet subscription tx (or vice versa) causes silent auth/activation failures.

#### Resolution

Pick one network at project start and enforce in config:

| Network | API origin                      | Solana RPC  |
| ------- | ------------------------------- | ----------- |
| devnet  | `https://txline-dev.txodds.com` | devnet RPC  |
| mainnet | `https://txline.txodds.com`     | mainnet RPC |

Validate at startup in Go and document in frontend `.env.example`.

---

### 2026-06-30 — Dual-header auth is easy to miss

**Area:** txline-api  
**Type:** 🚧 Hurdle

#### Context

First successful call to TxLINE data API or SSE stream.

#### What happened

Only sending `Authorization: Bearer {jwt}` returns 401/403. TxLINE requires **both** headers after activation.

#### Resolution

```
Authorization: Bearer {guest_jwt}
X-Api-Token: {activated_api_token}
```

Flow: `POST /auth/guest/start` → (optional subscribe + activate) → use both headers on SSE and REST.

---

### 2026-06-30 — World Cup free tier is the hackathon sweetspot

**Area:** txline-api  
**Type:** ✨ Sweetspot

#### Context

Hackathon timeline; avoid TxL purchase + KYC friction.

#### What worked

World Cup free tier (service levels 1 or 12) gives instant access to World Cup + International Friendlies without on-chain TxL purchase. Faster path to SSE data for Phase 1.

#### Reuse

Start here for demo; upgrade to paid tier only if judges need broader leagues.

Docs: https://txline.txodds.com/documentation/worldcup

---

### 2026-06-30 — LiteSVM tests need anchor build first

**Area:** blockchain  
**Type:** 🚧 Hurdle

#### Context

Running `cargo test` on Anchor program tests that load `.so` from target dir.

#### What happened

Tests fail with missing program binary if `anchor build` wasn't run first.

#### Resolution

```bash
cd blockchain && NO_DNA=1 anchor build && NO_DNA=1 cargo test
```

Uncomment/extend scaffold in `programs/blockchain/tests/test_initialize.rs`.

---

### 2026-06-30 — Three-layer split keeps velocity high

**Area:** cross-cutting  
**Type:** ✨ Sweetspot

#### Context

Full-stack TxLINE prediction platform (Anchor + Go + React).

#### What worked

Strict layer ownership via project skills:

| Layer    | Implement                  | Test                       |
| -------- | -------------------------- | -------------------------- |
| Programs | `solana-anchor-production` | `solana-anchor-tests`      |
| Backend  | `go-production`            | `go-test-production`       |
| Frontend | `react-dapp-production`    | (manual + component tests) |

#### Reuse

When stuck, classify which layer owns the bug before editing. Cross-layer bugs often show up at API contracts (IDL ↔ Go cache ↔ frontend types).

---

### 2026-06-30 — Keeper idempotency is non-negotiable for Phase 2

**Area:** backend-go + blockchain  
**Type:** 💡 Tip

#### Context

SSE may emit duplicate `is_final: true` events; reconnects replay state.

#### Takeaway

Track settled `match_id` in Go cache **before** submitting tx. Treat on-chain "already settled" as success. Prevents double payout attempts and RPC waste.

---

### 2026-06-30 — Frontend tx UX: confirm modal prevents user rage-quits

**Area:** frontend-react  
**Type:** ✨ Sweetspot

#### Context

Users signing wagers without understanding stake + cluster.

#### What worked

Flow: validate → simulate → **confirmation modal** (match, side, stake, cluster) → sign → explorer link toast.

Never auto-sign. Map Anchor `ErrorCode` to human-readable strings from IDL.

---

### 2026-06-30 — Receipt UI depends on log parsing contract

**Area:** frontend-react + blockchain  
**Type:** 💡 Tip

#### Context

Phase 3 verifiable receipt modal.

#### Takeaway

Agree early on what the settlement instruction **logs** (Merkle root, path bytes). Frontend parses `getTransaction` logs — if program doesn't emit structured logs, receipt UI can't be built. Add `msg!` or events in settlement instruction during Phase 2.

---

### 2026-07-01 — LiteSVM already bundles SPL Token + ATA; don't load /tmp .so files

**Area:** blockchain  
**Type:** ✨ Sweetspot

#### Context

`anchor build` failed because `tests/common/mod.rs` used `include_bytes!("/tmp/spl_token.so")` and `include_bytes!("/tmp/ata_program.so")`, which don't exist on a fresh machine.

#### Resolution

`LiteSVM::new()` already calls `with_builtins()` and `with_default_programs()`, which embed SPL Token and ATA from the crate. Remove manual `add_program` for those — only load `blockchain.so` and `txline_mock.so`. All 6 lifecycle tests pass after the fix.

---

### 2026-07-01 — Devnet E2E smoke: faucet → make → accept → settle

**Area:** cross-cutting  
**Type:** ✨ Sweetspot

#### Context

Phase 1 Step 2: full devnet wager lifecycle with real TxLINE `validate_stat` CPI.

#### What worked

- TxLINE `request_devnet_faucet` mints 100 USDT per wallet (`[faucet_tracker, user]` PDA seeds)
- `backend-go/bin/smoke-wager` runs faucet → make → accept → settle on fixture `17952170`
- Settlement needs `ComputeBudget` 1.4M CUs for TxLINE CPI (default 200k fails)
- Live `/api/scores/stat-validation` returns hashes as JSON byte arrays — `FlexHash` type handles both
- Successful run: settle tx `2WMLYMbqW3QrP7Jr1YSBTk74ZsMZMvCsRv9vEB2DyesBUrkRy7M7UsiEDWcJauPMzepsVJV68vofa82Ti9gufepQ`, maker net +0.1 USDT

#### Gotcha

`SimulateTransaction` returns `AccountNotFound` for brand-new wallets even after funding; send-without-preflight works for faucet/make/accept paths.

---

### 2026-07-01 — Keeper worker settled matched wager on devnet

**Area:** backend-go  
**Type:** ✨ Sweetspot

#### Context

Live keeper path: matched wager on fixture `17952170`, keeper fetches proof and submits `settle_wager`.

#### Flow

1. `bin/place-matched-wager` — faucet + make + accept (no direct settle)
2. `bin/keeper-settle` — builds final `ScoreUpdate` (snapshot or `SETTLE_SEQ`/`SETTLE_HOME_GOALS`), calls `keeper.Worker.HandleUpdate`
3. `bin/keeper` — SSE ingest running on `:8080`

#### Result

- Wager PDA: `8fwJw3XMgFgvjar6oMY72zmfE1K1ycJhQRhSL5LEJMjK`
- Keeper settle tx: `uEC8NPJDxLrG23SE3Uj8U148oyUAXNNaYVzgMecSLB1GghadXMHqK9dVVF4FH2F8P2j9i7syoDcDuM1AsGbUpfW`
- Proof: `GET /api/scores/stat-validation?fixtureId=17952170&seq=941&statKey=1002`
- Wager account closed after settle (expected)

#### Notes

- TxLINE snapshot rows may show `scheduled` even when scores exist; use `SETTLE_GAME_STATE=F2` + score overrides for replay.
- `MarkFinalOnce` in Redis is per `match_id` — clear `matchlock:final:{match_id}` before re-running keeper settle on same fixture.
- Allow ~5s after accept before `ListMatchedWagers` indexes the new account.

---

### 2026-07-03 — Winner-claim settlement architecture (permissionless settle)

**Area:** blockchain · backend-go · frontend-react · cross-cutting  
**Type:** ✨ Sweetspot · 💡 Tip

#### Context

Phase 1 originally used a **keeper-only** model: ops wallet signed `settle_wager`, paid all settlement SOL, and users saw keeper internals in the UI. We moved to **winner-initiated claim** with optional keeper crank behind a feature flag.

#### New architecture: Core Business Logic & Data Flow

The platform relies on a clear separation of data tracking and on-chain execution. Instead of central authorities settling wagers, the protocol behaves as a trustless escrow matching TxLINE's sports data on-chain.

**1. Match Synchronization Flow (Real-Time State)**
*How live match scores go from the field to the user interface.*
```text
┌─────────────┐   SSE    ┌──────────────┐  Write  ┌──────────────┐
│   TxLINE    │ ───────► │  backend-go  │ ──────► │    Redis     │
│             │          │  (Keeper)    │         │ (Match Cache)│
└─────────────┘          └──────┬───────┘         └──────────────┘
                                │ SSE fan-out
                                ▼
                         ┌──────────────┐
                         │   Frontend   │ (Live scores update ui)
                         └──────────────┘
```
- The Keeper worker consumes TxLINE scores directly via SSE.
- Validated state updates are pushed immediately to Redis allowing instant O(1) reads.
- Connected clients (Frontend) receive live match ticks via `/matches/stream` (SSE), which surgically updates React Query caches.

**2. Wagering Flow (P2P Play)**
*How users enter into predictions programmatically.*
```text
┌──────────────┐ 1. make_wager   ┌──────────────┐
│ Maker Wallet │ ──────────────► │  Matchlock   │ (Creates match state, escrows maker stake)
└──────────────┘                 │   Program    │
┌──────────────┐ 2. accept_wager │ (Solana)     │
│ Taker Wallet │ ──────────────► │              │ (Escrows taker stake, status = Matched)
└──────────────┘                 └──────────────┘
```
- **Make Wager**: Maker signs the tx paying the stake in USDT. Escrow vaults are initialized per wager, not pooled.
- **Accept Wager**: Taker spots an Open wager, signs it, and escrows an identical USDT amount. If the Maker configured an `invited_taker`, only that specific wallet can accept.

**3. Settlement & Resolution Flow (Permissionless Claim)**
*How winnings are safely resolved once a match ends, without keeper gas fees.*
```text
┌──────────────┐ 3. final state ┌──────────────┐                   ┌──────────────┐
│ TxLINE / SSE │ ─────────────► │  Keeper      │ 4. close match  ► │  Matchlock   │
└──────────────┘                └──────┬───────┘                   │   Program    │
                                       │                           └──────▲───────┘
                              5. GET /settlement-proof                    │ 8. CPI (Verify Merkle proof)
                                       │                                  │    + Payout 2x Stake
                                       ▼                           ┌──────┴───────┐
                                ┌──────────────┐ 6. Prompt Sign  ► │ Winner Wallet│
                                │   Frontend   │                   │ (Claim)      │
                                └──────────────┘ 7. settle_wager ► └──────────────┘
```
- **Finalization**: When TxLINE emits `is_final=true`, the Go Keeper immediately calls `close_match` on-chain to freeze betting.
- **Fetching Proofs**: The Winner visits their dashboard; frontend calls `GET /settlement-proof`. Keeper fetches the `StatValidation` (Merkle proof details) confirming the winning condition (e.g., `statKey 1002`).
- **Claim Sign**: Winner's wallet executes `settle_wager` with this proof payload as instruction args.
- **On-Chain CPI**: Matchlock executes a secure CPI to the TxLINE `validate_stat` program verifying the Oracle. 2x USDT is transferred to the Winner and wager accounts are closed.
- **Reporting**: The Go `ReconcileWorker` observes the successful `settle_wager` on-chain and updates Postgres with `leaderboard_entries` (increasing logic like net PnL and total volume).

**3b. Alternative Settlement Flow (Keeper Auto-Settle - `KEEPER_AUTO_SETTLE=true`)**
*How winnings are automatically paid out when the protocol pays for Solana network fees.*
```text
┌──────────────┐ 1. final state ┌──────────────┐ 2. close match  ┌──────────────┐
│ TxLINE / SSE │ ─────────────► │  Keeper      │ ──────────────► │  Matchlock   │
└──────────────┘                └──────┬───────┘                 │   Program    │
                                       │ 3. fetch proofs         └──────▲───────┘
                                       │                                │ 4. settle_wager
                                       ▼                                │    + Payout 2x Stake
                                ┌──────────────┐                 ┌──────┴───────┐
                                │ TxLINE API   │                 │ Winner Wallet│
                                └──────────────┘                 └──────────────┘
```
- **Finalization**: Upstream TxLINE `is_final=true` arrives over SSE, and Go Keeper calls `close_match` on-chain.
- **Background Settlement**: The Keeper worker loops over all matched wager accounts containing the `match_id` via `w.Solana.ListMatchedWagers`.
- **Proof Fetching & Settle**: For each wager, the Keeper fetches the `StatValidation` from TxLINE REST API directly, builds the Solana instruction, and executes `settle_wager` on-chain.
- **Fee Paying**: Because the user is offline, the Keeper's funded keypair must sign the transaction and pay the base network and compute fees.
- **Reporting**: The Go Keeper observes its own successful `settleOne` call and updates Postgres with `leaderboard_entries`.


| Before                                        | After                                               |
| --------------------------------------------- | --------------------------------------------------- |
| `keeper` signer must equal `config.authority` | `settler` signer: **winner** or `config.authority`  |
| Keeper always fee payer                       | **Settler** is fee payer (winner on claim path)     |
| Ops-funded settlement only                    | Permissionless winner claim + optional keeper crank |

Program upgraded on devnet (`HzpESYneBKXMp4qDAqMzM6GKq2yDciY5TaogT5hM1PeQ`). Upgrade tx: `5Svi9jauwtU1gqv6QyMgMYEkujGZi2i2ddVc2xcZ4zGtD9jqz3LFS4P7E5wpQLu2wf7h1zMdQzQEXAxUjrKGpSbB`.

**Verify deploy:** on-chain program data is padded (331,464 B) vs local `.so` (324,288 B). Compare executable bytes: `head -c 324288 onchain.so` must match `target/deploy/blockchain.so`.

#### Backend (`KEEPER_AUTO_SETTLE=false` default)

| Component                               | Role when auto-settle **off**                                              |
| --------------------------------------- | -------------------------------------------------------------------------- |
| SSE ingestor                            | Cache scores; mark `is_final`                                              |
| Schedule / odds workers                 | Hydrate fixtures, teams, odds                                              |
| `ReconcileWorker`                       | **No-op** (no SOL spend)                                                   |
| `HandleUpdate` on final                 | Log “winner claim required”; **no** `settle_wager`                         |
| `GET /wagers/{pubkey}/settlement`       | User-facing status (`message` only; no `last_error`)                       |
| `GET /wagers/{pubkey}/settlement-proof` | TxLINE validation payload + winning side + PDAs for claim tx               |
| `keeper-settle` CLI                     | Ops-only manual crank (still works if `KEEPER_AUTO_SETTLE=true` on worker) |

Env: `KEEPER_AUTO_SETTLE=false` in `backend-go/.env`. Set `true` only if ops should fund fallback settlement.

#### TxLINE proof resolution

1. Match final in cache → build `ScoreUpdate` (snapshot hydration preferred).
2. `winningSideFromScore` → home / away / draw.
3. **Probe** `statKey` 1002 and 1003; use whichever has `value > 0` at settlement `seq` (draw → 1001).
4. `GET /api/scores/stat-validation?fixtureId=&seq=&statKey=` → Merkle proof for on-chain CPI.

#### Frontend flows

| Action             | Fee payer  | Confirm dialog                                                   |
| ------------------ | ---------- | ---------------------------------------------------------------- |
| Create wager       | Maker      | Stake + ~network fee (`getFeeForMessage` + simulate cross-check) |
| Accept wager       | Taker      | Same                                                             |
| Cancel wager       | Maker      | Same                                                             |
| **Claim winnings** | **Winner** | Payout + ~network fee (~1.4M CU compute budget on claim)         |

- **Open wagers:** API omits unset `taker` (`Pubkey::default()` / system program). UI: “Waiting for an opponent to accept your challenge” — not `1111…111`.
- **Matched + final + user won:** “Claim winnings” → fetch proof → simulate → sign.
- Settlement banner: friendly copy only (“Settlement in progress” / “Paid out”); no keeper retry jargon.

#### Fee model summary

- **Users pay SOL** for every wallet-signed action (make, accept, cancel, claim).
- **Keeper pays SOL** only when `KEEPER_AUTO_SETTLE=true` (SSE final + reconcile loop + `keeper-settle`).

#### Rollout phases (incremental)

| Phase | Scope                                                                                            | Status  |
| ----- | ------------------------------------------------------------------------------------------------ | ------- |
| **A** | Fee estimates in confirm dialogs (`useTxFeeEstimate`, `getFeeForMessage` + simulate cross-check) | Shipped |
| **B** | On-chain `settler` constraint + `GET /settlement-proof` + frontend **Claim winnings**            | Shipped |
| **C** | Ops: `KEEPER_AUTO_SETTLE=false` default; reconcile worker no-op; keeper crank opt-in             | Shipped |

#### 💡 Tips

- Winners need a small SOL balance for claim (USDT stake alone is not enough).
- Browser never calls TxLINE directly; proof API keeps `X-Api-Token` server-side.
- Priority fees are not included in UI estimates unless the wallet adds them.

---

### 2026-07-03 — TxLINE outcome stat keys vs snapshot orientation

**Area:** backend-go / keeper  
**Type:** 🚧 Hurdle → ✨ Sweetspot

#### Context

Fixture `18179763` (Portugal 2–1 home win) failed on-chain with `ValidationFailed` (6010) while `17952170` settled fine.

#### Root cause

Soccer full-time outcome stats are **participant-based**, not home/away:

- `1001` = draw
- `1002` = participant 1 wins
- `1003` = participant 2 wins

Predicate for settlement: `threshold: 0`, `greaterThan` (stat value must be `1`).

For Portugal, `statKey=1003` was `1` at final seq even though snapshot rows often showed `participant1IsHome: true`. Hardcoding `1002` or trusting snapshot orientation alone picked a stat with `value=0` → predicate false.

#### Fix

Keeper now **probes outcome stats** (`1002`/`1003`) and uses whichever has `value > 0` at the settlement seq, with `StatKeyForWinningSide` as fallback. UI settlement API returns user-facing `message` only (no `last_error` / retry internals).

#### Result

- Wager `97Y4ptocSAQTLaPHLYcRBu44Xx5nVa27aZgCSFRRw1cM` settled tx: `56EYP6bFEAAM6E1M7nXq5W8QGChZLF6NQhxMPMmHuGK5bud9NBkNSmkBvxVLRxNxsx2JescLcU1YQRNL6rkZyjju`
- Proof path: `GET /api/scores/stat-validation?fixtureId=18179763&seq=941&statKey=1003`

---

### 2026-07-01 — Devnet deploy + Config initialize

**Area:** blockchain  
**Type:** ✨ Sweetspot

#### Context

Phase 1 Step 1: green test gate, deploy program, initialize Config on devnet.

#### What worked

- Deploy: `solana program deploy target/deploy/blockchain.so` with keeper keypair as upgrade authority
- Program ID: `HzpESYneBKXMp4qDAqMzM6GKq2yDciY5TaogT5hM1PeQ` (deploy tx `4Xocij4Hqoy67JbHU7Qzswc7FkiPiSiduTCjPz6thBPpj6fNZUPLTp5rqNpr41y9CSgBuQGhN9qGKquYjopBmFxz`)
- Initialize via `backend-go/cmd/initialize-matchlock` (tx `5zqPpV81n9uxquL2E9FkN2rC8qeuvugq3PwAEa9xd3CcD8eTFKLRVAmL74FUz2dTP5WNJ2y2HtG11edbStySV3MG`)
- Config PDA: `GkaDoDBdEv9sHMWosvvg1XRknJqrWvkzxTS4WXAzwzYD`
- Authority: `22WJxwL9WeGatNiYG2ejAsTtLs1E94isuNd8r8N14KEZ`

---

### 2026-07-02 — smoke-cancel: synthetic then live fixture

**Area:** backend-go + blockchain + txline-api  
**Type:** ✨ Sweetspot

#### Context

Deferred Phase 1 cancel path. First run used synthetic `cancel-{unix}` match_id (on-chain only). Re-ran during live World Cup fixture with `FIXTURE_ID` + TxLINE snapshot + keeper SSE cache.

#### Synthetic run (on-chain instruction check only)

| Field     | Value                                                                                      |
| --------- | ------------------------------------------------------------------------------------------ |
| match_id  | `cancel-1782950507` (not a TxLINE fixture)                                                 |
| cancel tx | `2BPJE1dbzm9G36NmuJHnU2iv5Ag6Aun7zUSG1JVg72qpKyqdHdimYFQVzyKAoJbx6si2nBS8GkgrjSExTNyknmpx` |

#### Live fixture run (World Cup kickoff window)

Fixture `18172379` — kickoff `2026-07-02T00:00:00Z`, TxLINE snapshot `gameState=scheduled` with clock running (~19′), keeper `GET /matches/18172379` `seq=253` `is_final=false`.

```bash
FIXTURE_ID=18172379 ./bin/smoke-cancel
```

| Field     | Value                                                                                      |
| --------- | ------------------------------------------------------------------------------------------ |
| match_id  | `18172379`                                                                                 |
| maker     | `3VfWWSDb19r7RuiZ9jY6AU6M612E2pFFup4LXScekk7s`                                             |
| wager PDA | `43fx3rCcE7pqkwXHBSGuZJWoS68VVzUqFyJvgLypj8C4`                                             |
| make tx   | `xAU7jUiS8DicPv3x27mAhYSHLCP8iogxPGfHJEyUH8haLYZyMRi16i4B9QvxonH2aQDRuJKzgortscau7PWj9nG`  |
| cancel tx | `31uCa5e4iTc1ooMqVqXe8PS642xBw5o1PZvnJ1DV3dTsdX67SV4UwFQoEHUkBPZP7ViJESuUTiSRdjRnnr6c2wKa` |
| USDT      | before=100000000 → after_make=99900000 → after_cancel=100000000                            |

On-chain `status=open` confirmed before cancel. Full maker refund (`100_000` stake units). `smoke-cancel` now accepts `FIXTURE_ID` / `MATCH_ID` and logs TxLINE snapshot before wagering.

---

### 2026-07-03 — Keeper offline left matched wagers unsettled

**Area:** backend-go · frontend-react · cross-cutting  
**Type:** 🚧 Hurdle · ✨ Sweetspot

#### Context

Agenda 1 PvP flow worked, but Portugal (match `18179763`) finished 2–1 while keeper was down — wagers stayed `Matched` with UI showing optimistic "Awaiting settlement".

#### What happened

`SettleMatch` used Redis `MarkFinalOnce` as a hard gate before per-wager settlement. Partial failures or keeper downtime stranded escrow with no retry path. `InferFinalState` could set `is_final` in cache without triggering settlement.

#### Resolution / takeaway

- Removed match-level settlement gate; idempotency is per-wager (`matchlock:settled:*` + on-chain status).
- Added Redis pending queue with exponential backoff, startup + periodic reconciliation (`ReconcileWorker`), and `GET /wagers/{pubkey}/settlement` for honest UI states (`queued` / `retrying` / `failed` / `match_ended_unverified`).
- Ops: restart keeper (systemd unit in `backend-go/deploy/`) or `keeper-settle` with `FIXTURE_ID`/`MATCH_ID` — no manual Redis key deletion needed anymore.
- **Superseded for production UX:** default path is now winner **Claim winnings** (`KEEPER_AUTO_SETTLE=false`). Keeper crank is opt-in fallback only.

---

### 2026-07-03 — Agenda 4: magic-link auth + Postgres + direct challenges

**Area:** backend-go · frontend-react · blockchain · cross-cutting  
**Type:** ✨ Sweetspot

#### Context

After Agenda 1 (challenge slip + draw), ship Agenda 4 before Agenda 3 (faucet UX): identity layer for friend challenges and wallet linking.

#### What shipped

**Auth (refresh cookies + Brevo)**

- `POST /auth/magic-link` → Brevo email with link to `{FRONTEND_URL}/auth/verify?token=…`
- `GET /auth/verify` → single-use token, issues **access JWT** (`matchlock_access`, 15m) + **refresh token** (`matchlock_refresh`, 7d, rotated on `POST /auth/refresh`)
- `POST /auth/logout` → revokes refresh session, clears cookies
- `GET /auth/me`, wallet link (`POST /auth/wallets/challenge` + `POST /auth/wallets/link` with ed25519 signature)

**Postgres (GORM)**

- Tables: `users`, `magic_link_tokens`, `sessions`, `wallet_links`, `wager_invites`
- `DATABASE_URL` required at keeper startup; `/readyz` includes `postgres` check

**Direct challenges**

- `GET /users/lookup?email=` — resolve friend primary wallet
- `POST /challenges/invites` + inbox `GET /challenges/invites` — email notification via Brevo
- Challenge slip: **Anyone** vs **A friend**; on-chain `invited_taker` when friend has linked wallet

**On-chain (breaking — redeploy required)**

- `Wager.invited_taker: Pubkey` (default = open); `make_wager(..., invited_taker)`; `accept_wager` enforces invitee
- Account size **150** bytes (was 118)

#### Env (`backend-go/.env.example`)

- `DATABASE_URL`, `JWT_ACCESS_SECRET`, `BREVO_*`, `FRONTEND_URL`, cookie flags

#### 💡 Tips

- Magic link = identity only; txs still wallet-signed (non-custodial).
- Frontend must use `credentials: 'include'` on API calls; CORS allows credentials on allowlisted origins.
- Friend without linked wallet: open wager + email invite (no `invited_taker` lock until they link).

---

## API documentation

- **Matchlock backend OpenAPI:** `backend-go/api/openapi.yaml` — REST contract for frontend Phase 1.16–1.22 (`GET /matches`, `GET /wagers`, health probes)
- **TxLINE upstream OpenAPI:** `https://txline-dev.txodds.com/docs/docs.yaml` (devnet)

### TxLINE Endpoints Consumed by Go Backend

- `POST {guest_url}`: Fetches Guest JSON Web Tokens (`token`) for dual-header API authentication (used in `EnsureGuestJWT`).
- `GET /api/fixtures/snapshot`: Fetches upcoming fixture schedules and match identifiers (optionally grouped by `?startEpochDay=X`).
- `GET /api/odds/snapshot/{fixtureId}`: Yields historical and latest Demargined odds lines (e.g., `1X2_PARTICIPANT_RESULT`) optionally taking an `?asOf={timestamp}`.
- `GET /api/odds/updates/{fixtureId}`: Retrieves live betting odds directly from the TxLINE in-memory fast cache.
- `GET /api/scores/snapshot/{fixtureId}`: Fetches terminal score snapshots to determine final values and fallback results for the Go Keeper.
- `GET /api/scores/stat-validation`: Pulls the cryptographic Merkle proofs (`StatValidation` + Events Tree Root) for verifying oracle stats on Solana (`?fixtureId=X&seq=Y&statKey=Z`).
- `GET {stream_url}`: Consumes the Server-Sent Events (`text/event-stream`) feeding live match ticks for instant latency updates (e.g., `/api/scores/stream`).

## Open questions

- [ ] Exact SSE event schema for World Cup matches — capture first real payload here
- [ ] TxLINE `validate_stat` CPI account layout for Phase 2 settlement
- [x] Devnet USDT mint address for wager tests — `ELWTKspHKCnCfCiCiqYw1EDH77k8VCP74dK9qytG2Ujh` (TxLINE devnet USDT)
- [x] Merkle proof REST endpoint path after `is_final` — `GET /wagers/{pubkey}/settlement-proof` (backend brokers `GET /api/scores/stat-validation`)

---

### 2026-07-08 — E2E Leaderboard: backend service + API + frontend + tests

**Area:** backend-go · frontend-react  
**Type:** ✨ Sweetspot

#### Context

Production-level ranking system tracking user PnL, win rate, and volume from settled wagers.

#### Architecture

```
keeper worker ──► RecordSettlement() ──► leaderboard_entries (Postgres)
                                              │
                                              ▼
                                       GET /leaderboard
                                       GET /leaderboard/me  (auth required)
                                       GET /leaderboard/stats
                                              │
                                              ▼
                                       React Query hooks
                                       (60s polling)
                                              │
                                              ▼
                                       Leaderboard page
                                       (stats cards, your rank, player list)
```

#### How it works

1. **DB model** (`internal/db/models.go:135`) — `LeaderboardEntry` keyed by `UserID`, stores `TotalWagers`, `Wins`, `Losses`, `TotalVolume`, `NetPnL`, `UpdatedAt`. GORM auto-migrated. Note: GORM snake-cased `NetPnL` to `net_pn_l` — the `gorm:"column:net_pnl"` tag fixes it.

2. **Settlement recording** (`internal/leaderboard/service.go:72`) — `RecordSettlement` is called by the keeper worker after each successful `settleOne`. For each pubkey (winner + loser):
   - Resolves wallet → user via `WalletLink` table (Preloads `User` for email/display name)
   - Upserts `LeaderboardEntry`: winner gets `+stake` PnL, loser gets `-stake` PnL, both get `2x stake` volume and `+1 wager`
   - Skips silently if pubkey has no linked wallet entry (unlinked wallets don't create phantom leaderboard rows)

3. **Leaderboard query** (`internal/leaderboard/service.go:34`) — `GetLeaderboard` orders by `net_pnl DESC, wins DESC, total_volume DESC`, limits to `100` max (default `20`). Returns `Entry` structs with computed `Rank` (1-indexed) and `WinRate` (wins/total\*100).

4. **Single-user rank** (`internal/leaderboard/service.go:127`) — `GetRank` counts how many entries have strictly higher PnL (or same PnL but more wins) to compute the user's rank.

5. **API handlers** (`internal/api/leaderboard_handlers.go`) — Three endpoints:
   - `GET /leaderboard?limit=N` — public, returns ranked entries
   - `GET /leaderboard/me` — auth required, returns caller's rank + entry
   - `GET /leaderboard/stats` — returns aggregate: total players, total volume, avg win rate

6. **Keeper wiring** (`internal/keeper/worker.go`) — After `settleOne` succeeds, calls `Leaderboard.RecordSettlement` with winner/loser pubkeys and stake. Only fires when `KEEPER_AUTO_SETTLE=true`.

7. **Frontend** — React Query hooks (`src/hooks/queries/use-leaderboard.ts`) poll every 60s. The leaderboard page (`src/pages/leaderboard-page.tsx`) shows:
   - Stats cards (total players, total volume, avg win rate)
   - "Your rank" card (when logged in)
   - Player list with gold/silver/bronze rank icons

#### Files changed

| File                                                  | What                                                            |
| ----------------------------------------------------- | --------------------------------------------------------------- |
| `internal/db/models.go`                               | `LeaderboardEntry` model + `WinRate()`                          |
| `internal/db/db.go`                                   | Added `LeaderboardEntry` to AutoMigrate                         |
| `internal/leaderboard/service.go`                     | `Service`, `RecordSettlement`, `GetLeaderboard`, `GetRank`      |
| `internal/keeper/worker.go`                           | Calls `RecordSettlement` after `settleOne`                      |
| `internal/api/leaderboard_handlers.go`                | `GET /leaderboard`, `/leaderboard/me`, `/leaderboard/stats`     |
| `internal/api/server.go`                              | Dependencies + route registration                               |
| `cmd/keeper/main.go`                                  | Wired leaderboard service into Worker + API                     |
| `.env`                                                | `KEEPER_AUTO_SETTLE=true`                                       |
| `frontend-react/src/lib/api.ts`                       | `getLeaderboard`, `getMyLeaderboardRank`, `getLeaderboardStats` |
| `frontend-react/src/hooks/queries/use-leaderboard.ts` | 3 React Query hooks (60s polling)                               |
| `frontend-react/src/pages/leaderboard-page.tsx`       | Leaderboard UI                                                  |
| `frontend-react/src/app/router.tsx`                   | `/leaderboard` route                                            |
| `frontend-react/src/components/layout/app-shell.tsx`  | Nav item                                                        |

#### Tests (`internal/leaderboard/service_test.go`)

8 tests covering:

- `RecordSettlement` creates entries for both winner/loser with correct PnL (+stake / -stake), volume (2x stake), wager counts
- `RecordSettlement` updates existing entries (accumulates wins, PnL, volume)
- `RecordSettlement` silently skips unlinked pubkeys (no DB rows created)
- `GetLeaderboard` returns entries ranked by net PnL descending
- `GetLeaderboard` respects the limit parameter (and defaults to 20 when 0)
- `GetRank` returns correct rank (2nd when one user has higher PnL)
- `GetRank` returns nil for unknown user (no entry yet)
- `WinRate` is computed correctly (70% for 7/10)

Tests use real Postgres via `db.Open` with cleanup in `t.Cleanup`, same pattern as `internal/db/db_test.go`.

#### 🚧 Hurdle — GORM column naming

GORM auto-migrates `NetPnL` to `net_pn_l` instead of `net_pnl`. Fixed with `gorm:"column:net_pnl"` tag and manual `ALTER TABLE leaderboard_entries RENAME COLUMN net_pn_l TO net_pnl;` in Postgres.

---

### 2026-07-11 — Production hardening: pause/unpause, Side::Unset, update_config

**Area:** blockchain · backend-go · frontend-react · cross-cutting
**Type:** ✨ Sweetspot

#### Context

Before opening the platform to real users, we hardened the program to prevent abuse and give the admin a safety kill switch.

#### Changes

**On-chain (`blockchain/`)**

| Change                      | Details                                                                                                                                         |
| --------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------- |
| `Side::Unset` variant       | Added to `state.rs` — `taker_side` initializes as `Unset` instead of `Home`, preventing accidental settlement matches                           |
| `Config.paused` bool        | New field; defaults to `false` on initialize                                                                                                    |
| `ContractPaused` error      | Error code 6018 in `error.rs`                                                                                                                   |
| `update_config` instruction | Permissioned (config authority) — partial updates via `Option<>` fields: `new_authority`, `new_stablecoin_mint`, `new_txline_program`, `paused` |
| Paused gating               | `make_wager`, `accept_wager`, `register_wallet` check `!config.paused` and return `ContractPaused`                                              |
| Unpaused operations         | `cancel_wager`, `settle_wager`, `unregister_wallet` intentionally **skip** the paused check so funds are never trapped                          |
| `stat_key_for_winning_side` | Now returns `Result<u32>` — rejects `Side::Unset` with `InvalidWinningSide` instead of silently matching                                        |

**Backend (`backend-go/`)**

- Added `SideUnset = 3` constant; `SideName("unset")` returns `"unset"` instead of defaulting to `"home"`

**Frontend (`frontend-react/`)**

- `Side` type widened to `'home' | 'draw' | 'away' | 'unset'`
- `toAnchorSide` throws on `'unset'` (can't submit it as an instruction arg)
- `sideLabel`, `sideShortLabel`, `referenceOddsForSide` all handle `'unset'` (return `'—'` or `null`)
- IDL regenerated with new `update_config` instruction + `paused` field + `ContractPaused` error
- `useContractPaused` hook fetches `Config.paused` every 30s from the chain

#### Deployment

- New program keypair generated: `7jbdwJLrePo6dr6Jo5sSmK4RQC5tYRrGebnkMFTuPGq5`
- Deployed to devnet; Config PDA initialized at `FU8myLVAMXBZfsEE9PbpmRmeqR8R82NR2gYGphqjcqcS`
- Program ID updated across 12 files: `lib.rs`, `Anchor.toml`, frontend IDL/env/store, backend config/constants/env/OpenAPI/cli/test

#### Design rationale

Paused is **not** a full halt — users can always withdraw funds (cancel open wagers) and the keeper can still settle matched wagers. This follows the principle that contract admin actions should never trap user funds.

---

## Template (copy for new entries)

```markdown
### YYYY-MM-DD — Short title

**Area:** blockchain | backend-go | frontend-react | txline-api | cross-cutting
**Type:** 🚧 Hurdle | ✨ Sweetspot | 💡 Tip

#### Context

What you were trying to do.

#### What happened

What went wrong or surprisingly well.

#### Resolution / takeaway

Fix, pattern to reuse, or link to PR/commit.
```

---

### 2026-07-14 — Void wager instruction: draw/third-outcome refunds

**Area:** blockchain · backend-go · frontend-react · cross-cutting
**Type:** ✨ Sweetspot

#### Context

When a match ends in a draw and neither participant backed the draw, or more generally when the actual outcome is neither the maker's nor the taker's selected side, the vault holds 2× stake with no valid winner. Previously this was an unrecoverable state — the keeper had no path to release funds.

#### Solution: `void_wager` instruction (on-chain)

A new Anchor instruction in `blockchain/programs/blockchain/src/instructions/void_wager.rs`.

**Account model:**
- `settler` — pays SOL fees; must be config authority, maker, or taker
- `wager` — status must be `Matched`
- `vault` — token account seeded from wager PDA
- `maker` / `maker_stablecoin` — maker receives refund via ATA
- `taker` / `taker_stablecoin` — taker receives refund via ATA
- `daily_scores_merkle_roots` — TxLINE PDA for CPI verification
- `txline_program` — TxLINE program for CPI

**Guards:**
- `winning_side` must not equal `Unset`, `maker_side`, or `taker_side` (enforced by `InvalidVoidOutcome`)
- Calls `validate_settlement_proof` (same as settle_wager) — verifies TxLINE stat proof
- CPIs `invoke_validate_stat` against TxLINE — same on-chain verification as payout

**Payout:**
- Refunds `stake_amount` to maker's ATA
- Refunds `stake_amount` to taker's ATA
- Closes vault (rent returned to maker)
- Sets `wager.status = Cancelled`
- Emits structured `msg!` log with maker, taker, refund amount, and outcome

**New error code:** `InvalidVoidOutcome` (6019) — "Void outcome must differ from both selected wager outcomes"

#### Keeper integration (backend-go)

In `backend-go/internal/keeper/worker.go`, `settleOne` now branches on `WinnerPubkey`:

```go
if _, winnerErr := wager.WinnerPubkey(winningSide); winnerErr != nil {
    resolution = "refund"
    sig, err = w.Solana.VoidWager(ctx, params)
} else {
    sig, err = w.Solana.SettleWager(ctx, params)
}
```

If neither participant selected the winning side (i.e. a draw outcome when both backed a side), `WinnerPubkey` returns an error → keeper calls `VoidWager` instead of `SettleWager`. Leaderboard recording is skipped for refunds (no winner/loser to record).

#### Go client (`backend-go/internal/solana/`)

- New `VoidParams = SettleParams` type alias (`client.go`)
- `VoidWager` method — builds accounts for both maker + taker ATAs and vault; calls `EncodeVoidWagerData`
- `EncodeVoidWagerData` in `encode.go` — uses the `void_wager` Anchor discriminator; shares `encodeResolutionPayload` with settle
- `ListActiveWagers` — new method filtering out settled/cancelled (used by reconcile)
- `GetMatchState` + `DecodeMatchState` in new `match_state.go` — reads the on-chain wagering gate PDA
- `ErrMatchAlreadyClosed` — new sentinel error for idempotent close

#### Settlement API changes (`backend-go/internal/api/settlement.go`)

Two new settlement states added:

| State | Meaning |
|---|---|
| `claimable` | Winner determined, proof ready — winner can claim payout |
| `refundable` | Neither side won — both participants entitled to full refund |

`resolveWagerSettlement` now distinguishes between `claimable` and `refundable` using `winningSideFromMatch` + `WinnerPubkey`. Previously all verified finals went to `queued`.

`refreshVerifiedFinalForWager` now hydrates matches from TxLINE snapshot on cache miss (resilience against Redis loss).

#### Match phase tracking (`backend-go/internal/cache/match_phase.go`)

New file with time-based bounds:

- `MatchDuration = 105min` — typical match window before probing for missed final
- `MaxLiveStatusAge = 4h` — how long a non-final match can show as "live" before UI degrades to "result pending"
- `FinalVerificationEligible` — checks if non-final match is old enough for snapshot probe
- `LiveStatusExpired` — non-final fixture too old to show as live

These feed into settlement resolution: `LiveStatusExpired` causes `match_ended_unverified` instead of `match_live`, prompting keeper reconciliation.

#### Reconcile worker hardening (`backend-go/internal/keeper/reconcile.go`)

`ReconcileFinalMatches` now:
1. Scans on-chain active wagers (open + matched) via `ListActiveWagers`
2. Hydrates missing matches from TxLINE score snapshot
3. Closes overdue fixtures on-chain
4. Queues pending settlements

This ensures Redis cache loss or schedule window gaps don't strand wagers.

#### Fixture validation API endpoint

New backend endpoints:
- `GET /api/fixtures/validation?fixtureId=X&timestamp=Y` — proxies TxLINE fixture validation (Merkle proof material)
- `backend-go/internal/api/fixture_validation.go` — handler
- `backend-go/internal/txline/fixture_validation.go` — TxLINE client for `FetchFixtureValidation`

#### Frontend changes

**Settlement states** (`frontend-react/src/lib/api.ts`):
- `SettlementState` union widened with `'claimable'` and `'refundable'`
- New `FixtureValidation` + `FixtureValidationProofNode` types for the fixture validation API
- `getFixtureValidation` API method

**Wager outcome logic** (`frontend-react/src/lib/wager-outcome.ts`):
- `winningSideFromMatch` — extracts `'home' | 'away' | 'draw'` from match score, returns `null` if inconclusive
- `isSettlementClaimable` — now includes `claimable` alongside `queued` / `retrying` / `failed`
- `canClaimWinnings` — winner check + claimable state check

**SettlementStatus UI** (`frontend-react/src/components/wager/settlement-status.tsx`):
- `claimable`: "Ready to claim" / "The final result is verified. The winner can claim the payout."
- `refundable`: "Refund due" / "Neither selected outcome won. Both stakes will be returned."
- `queued`: "Settlement queued" / "The keeper has scheduled the payout..."
- `settled`: "Resolved" / "Escrow funds have been released on-chain."

**Match stale status** (`frontend-react/src/lib/match-display.ts`):
- `isStatusStale` flag surfaces "Result pending" badge on cards when a match is past `MaxLiveStatusAge`

**Confirm dialog** (`frontend-react/src/components/wager/confirm-tx-dialog.tsx`):
- Added "Your balance" row for `accept` action — shows user's current USDT balance before signing

**Wager detail page** (`frontend-react/src/pages/wager-detail-page.tsx`):
- Full-width layout when settlement resolution panel is shown (`claimable` / `refundable` / `queued` / `settled`)
- Merkle receipt panel wired in after settlement

**Wager detail page** (`frontend-react/src/pages/wager-detail-page.tsx`):
- Full-width layout when settlement resolution panel is shown (`claimable` / `refundable` / `queued` / `settled`)
- Merkle receipt panel wired in after settlement

#### Tests

**Rust (LiteSVM)** — `test_wager_lifecycle.rs`:
- `draw_wager_settles_for_draw_backer` — maker backs draw, taker backs home, draw wins → maker gets 2× payout via normal settle path
- `unselected_third_outcome_refunds_both_participants` — maker backs home, taker backs draw, away wins → void_wager refunds both
- `selected_outcome_cannot_use_void_path` — trying to void with a winning_side that matches a participant's side fails (must use settle)

**Go** — `client_test.go`:
- `TestVoidWagerSuccess` — void wager with away winning (unselected) succeeds
- `TestVoidWagerRejectsSelectedOutcome` — void wager with home winning (selected by maker) fails
- `TestListActiveWagersFiltersTerminalStatuses` — only Open + Matched returned

#### Open questions / follow-ups

- Frontend has no "Claim refund" button for `refundable` state — refunds currently require keeper auto-settle (`KEEPER_AUTO_SETTLE=true`) or manual keeper crank
- `sideLabel`, `referenceOddsForSide` need `'draw'` handling in all components (mostly done but audit needed)
- `void_wager` sets status to `Cancelled` — this conflates "cancelled by maker before match" with "refunded after draw"; consider a separate `Voided` status variant in future

---
