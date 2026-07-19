# Matchlock Demo Transcript — TxLINE Hackathon

**Total target:** 5 minutes
**Presenter:** [Your name]
**Date:** 2026-07-15

---

## Part 1: The Problem & The Pitch (0:00–1:00)

_"Hey everyone — I'm [name], and this is Matchlock: a peer-to-peer sports wagering protocol built on Solana, settled trustlessly using TxLINE's cryptographic oracle data."_

**Key talking points:**
- Traditional sportsbooks take 5–10% house edge, hold your funds in custodial wallets, and settle behind closed doors
- Matchlock flips that: zero house edge (true PvP), non-custodial (USDT in program-controlled vaults), and every settlement is backed by a TxLINE Merkle proof you can verify on-chain
- Built on Solana for fast, cheap settlement; uses TxLINE as the verification layer for trustless resolution

**Show the home page (`/`):**
- "Challenge another player. Settle on-chain."
- Walk through protocol workflow: Pick a match → Challenge or accept → Stakes lock on-chain → Settle automatically
- Call out the advantage vs. traditional sportsbooks

---

## Part 2: Auth & Wallet Setup (1:00–1:30)

_"First, let me sign in and link my wallet so I can start wagering."_

**Show the login/magic-link flow (`/login`):**
- Enter email → magic link sent via Brevo transactional email
- Click verify → JWT access + refresh tokens in httpOnly cookies

**Show profile page (`/profile`):**
- Username set (e.g., "demo_player")
- Connect browser wallet (Phantom/Backpack)
- Sign the link challenge (ed25519) to associate wallet with account
- Show connected wallet appears with "Primary" badge

**Narrate:**
- "Matchlock is non-custodial — your identity is just for friend challenges and leaderboard tracking. All transactions are signed by your wallet, not your email."

---

## Part 3: Markets & Live Data (1:30–2:00)

_"Let's see what's on — World Cup fixtures with real-time scores streaming in from TxLINE."_

**Show markets page (`/markets`):**
- Table of upcoming + live World Cup fixtures
- Live matches show scores updating in real-time via the Go backend's SSE fan-out
- TxLINE odds displayed per 1-X-2 outcome
- Call out the TxLINE data pipeline: SSE stream → Go backend ingest → Redis cache → HTTP API → React Query → UI

**Narrate:**
- "On the backend, the Go keeper subscribes to TxLINE's SSE stream. Match data lands in Redis in milliseconds, fans out over our own SSE endpoint to every connected browser. No polling, no browser-to-TxLINE connection — it's all proxied."

---

## Part 4: Creating a Challenge (2:00–2:30)

_"I'll back Argentina to win. Let me create a wager."_

**Click a match → Open challenge slip:**
- Pick side (e.g., Home / Argentina)
- Select "Anyone" (open challenge) or "A friend" (invite by email)
- Enter stake amount — show balance displayed, "Max" button, insufficient balance check
- Click "Create challenge"

**Confirmation modal appears:**
- Match label, your side, stake, potential payout (2x), network fee estimate, cluster badge
- "Your balance" row shows current USDT balance
- "Confirm & sign" → wallet pops up → sign

**Narrate:**
- "Every transaction simulates before you sign — no surprises. The confirmation modal shows your stake, the 2x potential payout, the network fee in SOL, and your current balance so you know exactly where you stand. Once signed, the wager PDA is created on devnet and your USDT is locked in a program-controlled vault."

---

## Part 5: Browsing & Accepting Wagers (2:30–3:15)

_"Now let me switch to another wallet acting as the taker."_

**Show open wagers page (`/open`):**
- Grid of open challenges with match details, maker's side, stake, payout
- Outcome picker lets the taker choose the opposite side
- Click "Accept challenge"

**Confirmation modal shows:**
- Match, your side (opposite of maker), stake, potential payout
- "Your balance" row — taker can see they have enough USDT
- Fee estimate, cluster
- "Confirm & sign"

**Show wager moved to matched state (`/my-wagers`):**
- Both wallets see the wager under "My wagers" with status "Matched"
- Settlement indicator: "Match in progress" / "Awaiting settlement"

**Narrate:**
- "On-chain, the accept instruction validates: taker isn't the maker, the side is different, the match isn't finished. Both stakes are now escrowed in the vault. Nobody can touch these funds — not even us — until the match settles."

---

## Part 6: Settlement & Trustless Resolution (3:15–4:00)

_"The match ends. TxLINE emits is_final: true over the SSE stream. Here's what happens."_

**Show the keeper flow (architecture slide or CLI):**
- Go keeper detects `is_final` → fetches Merkle proof from TxLINE's `/api/scores/stat-validation`
- Keeper probes stat keys 1002/1003 to find the winning side
- Builds `settle_wager` transaction with proof accounts
- In winner-claim mode: user clicks "Claim winnings" → frontend fetches `/settlement-proof` → wallet signs settlement

**Show My Wagers with settled state:**
- Status badge: "Paid out" with green settlement banner
- View transaction link to Solana Explorer
- Winner's USDT balance increased by 2x stake

**Show the Verifiable Resolution / Merkle receipt panel:**
- The settlement page displays a Merkle proof map
- Interactive tree showing leaf → root path for the verified stat
- "You can verify this data directly against TxLINE's on-chain program — no trust required."

**Narrate:**
- "The settlement instruction CPIs into TxLINE's `validate_stat` program. The program checks the stat proof on-chain — if it's valid, funds are released to the winner. The entire proof path is logged in the transaction and displayed in our Merkle receipt panel. You can audit every settlement."

---

## Part 7: Leaderboard & Close (4:00–4:45)

_"After settlement, the leaderboard updates automatically."_

**Show leaderboard page (`/leaderboard`):**
- Stats cards: Total volume, total wagers, active users, avg win rate
- Your rank card (when logged in)
- Player list ranked by net PnL descending, with win rate and volume
- Gold/silver/bronze icons for top 3

**Narrate:**
- "The keeper records settlement results to Postgres, computing net PnL, win rate, and volume. The leaderboard polls every 60 seconds. It's gamified — competitive players can track their standing against the field."

---

## Part 8: TxLINE Integration Summary & Closing (4:45–5:00)

_"Here's every TxLINE endpoint we consumed and what we learned."_

**TxLINE endpoints used:**

| Endpoint | Matchlock usage |
|---|---|
| `POST /auth/guest/start` | Guest JWT for dual-header auth |
| `GET /api/fixtures/snapshot` | Upcoming fixture schedule |
| `GET /api/odds/snapshot/{fixtureId}` | 1X2 odds for market display |
| `GET /api/odds/updates/{fixtureId}` | Live odds from in-memory cache |
| `GET /api/scores/snapshot/{fixtureId}` | Final score snapshots for settlement |
| `GET /api/scores/stat-validation` | Merkle proofs for on-chain CPI |
| `{stream_url}` / `api/scores/stream` | SSE stream for real-time match data |

**Feedback / friction points (from `docs/integration-journal.md`):**
- Sweetspot: Dual-header auth is clear once you know; World Cup free tier is perfect for hackathons (no TxL purchase/KYC)
- Sweetspot: Stat validation CPI with Merkle proofs is elegant — enables truly trustless settlement
- Friction: Network mismatch (devnet vs mainnet API origin) is the #1 footgun — validate at startup
- Friction: SSE event schema discovery — capturing the first real payload took some trial and error
- Friction: Outcome stat keys (1002/1003) are participant-based, not home/away — had to probe both values

**Closing:**
- "Matchlock is live on Solana devnet. Full repo is public. Phase 2 (prediction pools & AMM) and Phase 3 (interactive Merkle receipt trees) are in progress. Thanks to TxLINE for the data layer and the hackathon team for the support."

---

## Appendix: Key Technical Highlights for Judges

### Architecture wins
- **Three-layer split:** Anchor program (escrow + CPI) · Go backend (SSE + keeper + API) · React frontend (wallet UX)
- **Backend-proxied SSE:** Browser never calls TxLINE directly; `X-Api-Token` stays server-side
- **Keeper idempotency:** Redis deduplication + on-chain status guards prevent double settlement
- **Winner-claim settlement:** Default path — winner pays SOL, signs `settle_wager`. Keeper auto-settle is opt-in fallback

### On-chain security
- USDT mint pinned via `Config` PDA — users cannot be tricked into staking TxLINE credit tokens
- `cancel_wager` and `settle_wager` skip `paused` check — funds are never trapped
- `Side::Unset` prevents accidental settlement matches
- `simulateTransaction` before every user-facing tx

### Key files for reference

| What | Path |
|---|---|
| Wager lifecycle (Anchor) | `blockchain/programs/blockchain/src/instructions/` |
| Keeper SSE + settlement | `backend-go/internal/keeper/worker.go` |
| Confirm tx modal | `frontend-react/src/components/wager/confirm-tx-dialog.tsx` |
| Challenge slip (create) | `frontend-react/src/components/wager/challenge-slip.tsx` |
| Open wagers (accept) | `frontend-react/src/components/wager/open-wager-list.tsx` |
| Settlement status UI | `frontend-react/src/components/wager/settlement-status.tsx` |
| Merkle proof map | `frontend-react/src/components/wager/merkle-proof-map.tsx` |
| Leaderboard | `frontend-react/src/pages/leaderboard-page.tsx` |
| Integration journal | `docs/integration-journal.md` |
