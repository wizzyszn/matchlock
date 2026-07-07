const fs = require("fs");
const {
  Document,
  Packer,
  Paragraph,
  TextRun,
  Table,
  TableRow,
  TableCell,
  ExternalHyperlink,
  AlignmentType,
  HeadingLevel,
  LevelFormat,
  BorderStyle,
  WidthType,
  ShadingType,
  PageBreak,
} = require("docx");

const border = { style: BorderStyle.SINGLE, size: 1, color: "CCCCCC" };
const borders = { top: border, bottom: border, left: border, right: border };
const CONTENT_WIDTH = 9360;

function heading1(text) {
  return new Paragraph({
    heading: HeadingLevel.HEADING_1,
    children: [new TextRun(text)],
  });
}

function heading2(text) {
  return new Paragraph({
    heading: HeadingLevel.HEADING_2,
    children: [new TextRun(text)],
  });
}

function heading3(text) {
  return new Paragraph({
    heading: HeadingLevel.HEADING_3,
    children: [new TextRun(text)],
  });
}

function body(text, opts = {}) {
  return new Paragraph({
    spacing: { after: 120 },
    children: [new TextRun({ text, ...opts })],
  });
}

function bullet(ref, text, boldPrefix) {
  const children = [];
  if (boldPrefix) {
    children.push(new TextRun({ text: boldPrefix, bold: true }));
    children.push(new TextRun(text));
  } else {
    children.push(new TextRun(text));
  }
  return new Paragraph({
    numbering: { reference: ref, level: 0 },
    spacing: { after: 80 },
    children,
  });
}

function linkParagraph(label, url, description) {
  return new Paragraph({
    spacing: { after: 100 },
    children: [
      new ExternalHyperlink({
        children: [new TextRun({ text: label, style: "Hyperlink", bold: true })],
        link: url,
      }),
      new TextRun(` — ${description}`),
    ],
  });
}

function tableRow(cells, widths, header = false) {
  return new TableRow({
    children: cells.map((text, i) =>
      new TableCell({
        borders,
        width: { size: widths[i], type: WidthType.DXA },
        shading: {
          fill: header ? "F0E8DC" : "FFFFFF",
          type: ShadingType.CLEAR,
        },
        margins: { top: 80, bottom: 80, left: 120, right: 120 },
        children: [
          new Paragraph({
            children: [
              new TextRun({ text, bold: header, size: header ? 22 : 24 }),
            ],
          }),
        ],
      })
    ),
  });
}

function makeTable(headers, rows, colWidths) {
  return new Table({
    width: { size: CONTENT_WIDTH, type: WidthType.DXA },
    columnWidths: colWidths,
    rows: [
      tableRow(headers, colWidths, true),
      ...rows.map((row) => tableRow(row, colWidths)),
    ],
  });
}

const doc = new Document({
  styles: {
    default: {
      document: { run: { font: "Arial", size: 24 } },
    },
    paragraphStyles: [
      {
        id: "Heading1",
        name: "Heading 1",
        basedOn: "Normal",
        next: "Normal",
        quickFormat: true,
        run: { size: 36, bold: true, font: "Arial", color: "2A2420" },
        paragraph: { spacing: { before: 360, after: 200 }, outlineLevel: 0 },
      },
      {
        id: "Heading2",
        name: "Heading 2",
        basedOn: "Normal",
        next: "Normal",
        quickFormat: true,
        run: { size: 30, bold: true, font: "Arial", color: "2A2420" },
        paragraph: { spacing: { before: 280, after: 160 }, outlineLevel: 1 },
      },
      {
        id: "Heading3",
        name: "Heading 3",
        basedOn: "Normal",
        next: "Normal",
        quickFormat: true,
        run: { size: 26, bold: true, font: "Arial", color: "6B5F56" },
        paragraph: { spacing: { before: 200, after: 120 }, outlineLevel: 2 },
      },
    ],
  },
  numbering: {
    config: [
      {
        reference: "bullets",
        levels: [
          {
            level: 0,
            format: LevelFormat.BULLET,
            text: "•",
            alignment: AlignmentType.LEFT,
            style: {
              paragraph: { indent: { left: 720, hanging: 360 } },
            },
          },
        ],
      },
      {
        reference: "numbers",
        levels: [
          {
            level: 0,
            format: LevelFormat.DECIMAL,
            text: "%1.",
            alignment: AlignmentType.LEFT,
            style: {
              paragraph: { indent: { left: 720, hanging: 360 } },
            },
          },
        ],
      },
    ],
  },
  sections: [
    {
      properties: {
        page: {
          size: { width: 12240, height: 15840 },
          margin: { top: 1440, right: 1440, bottom: 1440, left: 1440 },
        },
      },
      children: [
        new Paragraph({
          alignment: AlignmentType.CENTER,
          spacing: { after: 80 },
          children: [
            new TextRun({
              text: "MATCHLOCK",
              bold: true,
              size: 52,
              color: "C2652A",
            }),
          ],
        }),
        new Paragraph({
          alignment: AlignmentType.CENTER,
          spacing: { after: 40 },
          children: [
            new TextRun({
              text: "Landing Page Design Brief",
              size: 32,
              italics: true,
            }),
          ],
        }),
        new Paragraph({
          alignment: AlignmentType.CENTER,
          spacing: { after: 320 },
          children: [
            new TextRun({
              text: "For UI/UX Designer  ·  July 2026  ·  v1.0",
              size: 22,
              color: "6B5F56",
            }),
          ],
        }),

        heading1("1. Purpose of this document"),
        body(
          "This brief gives a UI/UX designer everything needed to design a production-quality landing page for Matchlock — a decentralized, peer-to-peer sports wagering platform on Solana. The landing page must convert curious visitors into users who understand what makes Matchlock different, trust the settlement model, and click through to the app."
        ),
        body(
          "Scope note: Matchlock is currently shipping Phase 1 only (head-to-head PvP wagers with trustless settlement). Features planned for later phases may appear on the landing page as “Coming soon” teasers — not as live product claims."
        ),

        heading1("2. Product summary"),
        heading2("What is Matchlock?"),
        body(
          "Matchlock lets sports fans wager USDC on match outcomes directly against other players — not against a house or bookmaker. Wagers are escrowed in Solana program-controlled vaults, matched peer-to-peer, and settled automatically when TxLINE (an on-chain sports data oracle) finalizes match statistics and supplies cryptographic proofs."
        ),
        body(
          "Tagline direction: “Challenge another player. Settle on-chain.” or “Peer-to-peer sports wagers, settled by truth.”"
        ),

        heading2("Core value propositions (lead with these)"),
        bullet("bullets", " — No house edge. You wager against another player; the platform never takes the other side of your bet."),
        bullet("bullets", " — Trustless settlement. Payouts execute on-chain only after TxLINE validates final match data via a Merkle proof CPI."),
        bullet("bullets", " — Transparent custody. Funds stay in program-controlled vault PDAs until cancel or verified settlement — never in a platform hot wallet."),
        bullet("bullets", " — Verifiable outcomes. Every settled market is backed by TxLINE consensus data, not operator discretion."),
        bullet("bullets", " — Solana-native. Fast transactions, wallet-native UX, devnet today with a mainnet path."),

        heading2("What Matchlock is NOT"),
        bullet("bullets", "Not a traditional sportsbook or casino taking house edge."),
        bullet("bullets", "Not a pool-based prediction market (yet — see Coming soon)."),
        bullet("bullets", "Not custodial — users sign every stake transaction from their own wallet."),
        bullet("bullets", "Not hype-driven DeFi — avoid slot-machine aesthetics, neon gradients, or “get rich quick” framing."),

        heading1("3. Launch scope — Phase 1 (shipping now)"),
        body(
          "Design the landing page around these live capabilities. Every hero claim, feature card, and screenshot caption should map to something a user can actually do today on devnet."
        ),

        makeTable(
          ["User journey", "What happens", "Landing-page angle"],
          [
            [
              "Browse markets",
              "Live match list from TxLINE World Cup data with status badges (upcoming, live, final)",
              "Show a curated match browser — type-led cards, no stock sports photography",
            ],
            [
              "Create wager (Maker)",
              "Pick match, pick side (Home / Draw / Away), set USDC stake, sign make_wager tx",
              "“Post a challenge” — emphasize you pick the side and stake",
            ],
            [
              "Accept wager (Taker)",
              "Browse open wagers, take the opposite side at matching stake, sign accept_wager tx",
              "“Accept a challenge” — show counterparty relationship clearly",
            ],
            [
              "Cancel (Maker only)",
              "While wager is Open, maker can cancel and receive full stake refund",
              "Trust signal: “Your funds, your control — cancel anytime before a match”",
            ],
            [
              "Settlement",
              "When match goes final, keeper submits settle_wager with Merkle proof; winner receives 2× stake",
              "“Paid automatically when the match ends — verified by TxLINE”",
            ],
          ],
          [2200, 3580, 3580]
        ),

        new Paragraph({ spacing: { before: 200, after: 120 }, children: [] }),

        heading2("Payout model (critical for copy)"),
        body(
          "Matched PvP wagers pay exactly 2× stake to the winner. Both players lock equal stakes; the winner takes the combined pool. There are no odds from a bookmaker — only a clear payout multiple."
        ),
        body(
          "Example copy: “Stake 50 USDC. Beat your opponent. Win 100 USDC.”"
        ),

        heading2("Authentication & wallet"),
        bullet("bullets", "Email magic-link login establishes a Matchlock profile and devnet session."),
        bullet("bullets", "Solana wallet connection for signing stake transactions."),
        bullet("bullets", "Username onboarding for social identity (challenges, invites)."),
        bullet("bullets", "Cluster badge (Devnet) and API health indicator in app chrome — landing page can show “Built on Solana · Powered by TxLINE”."),

        heading1("4. Coming soon (tease, do not ship)"),
        body(
          "These features are on the roadmap but not live. They may appear in a subdued “On the horizon” or “Coming soon” section — never in the hero or primary CTA area."
        ),

        makeTable(
          ["Feature", "Description", "UI treatment on landing page"],
          [
            [
              "Prediction pools & AMM",
              "Shared liquidity markets alongside PvP — deposit, swap positions, settle pro-rata via same TxLINE pipeline",
              "Ghost card with lock icon, “Coming soon” badge, brief one-liner",
            ],
            [
              "Verifiable receipt UI",
              "Interactive Merkle proof tree showing settlement used authentic TxLINE data",
              "Small trust-section teaser: “Audit any settlement” with proof-tree wireframe",
            ],
            [
              "Leaderboard & gamification",
              "Rankings by volume, win rate, net PnL; streak badges",
              "Optional podium wireframe in footer section — labeled Coming soon",
            ],
          ],
          [2400, 3960, 3000]
        ),

        new Paragraph({ children: [new PageBreak()] }),

        heading1("5. Brand & design system — Sahara"),
        body(
          "Matchlock uses the Sahara design system: “Sun-Baked Simplicity” — warm minimalism that feels editorial and trustworthy, not like a dark crypto casino or generic AI-generated Web3 template."
        ),

        heading2("Color palette"),
        makeTable(
          ["Token", "Hex", "Usage"],
          [
            ["Background (linen)", "#faf5ee", "Page ground, hero background"],
            ["Foreground", "#2a2420", "Body text"],
            ["Primary (burnt sienna)", "#c2652a", "CTAs, focus rings, key accents"],
            ["Accent (dusty rose)", "#8c3c3c", "Sparse emphasis, secondary highlights"],
            ["Card surface", "#fffdf9", "Feature cards, panels"],
            ["Border", "rgba(216,208,200,0.6)", "Hairline warm borders"],
            ["Status: Open", "#8a6914 on #f5edd8", "Warm amber — wager waiting for taker"],
            ["Status: Matched", "#6b5344 on #ede4dc", "Terracotta — both stakes locked"],
            ["Status: Settled", "#4a5c3a on #e8ede0", "Olive-gold — payout complete"],
          ],
          [2800, 2200, 4360]
        ),

        new Paragraph({ spacing: { before: 160, after: 120 }, children: [] }),

        heading2("Typography"),
        bullet("bullets", " — EB Garamond. Headlines, hero matchups, editorial moments. One serif moment per screen maximum."),
        bullet("bullets", " — Manrope. UI labels, body copy, buttons, data. Tabular nums for stakes and payouts."),
        bullet("bullets", "Never use Garamond for wallet addresses, stakes, or dense tables."),

        heading2("Visual rules"),
        bullet("bullets", "No stock sports photography — type-led match cards with team names and optional flag icons."),
        bullet("bullets", "No purple gradients, glassmorphism, or untouched shadcn defaults."),
        bullet("bullets", "Whitespace is the primary design tool on editorial sections; ledger-tier sections can be denser."),
        bullet("bullets", "Warm status semantics always pair icon + label + color (WCAG 2.1 AA)."),
        bullet("bullets", "Dark mode variant “Ember” exists for in-app night sessions — landing page can default to Sahara light with optional dark hero variant."),

        heading2("Signature layout: Duel Frame"),
        body(
          "The app’s signature head-to-head composition centers every wager screen on a matchup hierarchy:"
        ),
        new Paragraph({
          spacing: { after: 120 },
          indent: { left: 720 },
          children: [
            new TextRun({ text: "Arsenal", bold: true, font: "Georgia", size: 28 }),
          ],
        }),
        new Paragraph({
          spacing: { after: 60 },
          alignment: AlignmentType.CENTER,
          indent: { left: 720 },
          children: [new TextRun({ text: "vs", italics: true, size: 24 })],
        }),
        new Paragraph({
          spacing: { after: 120 },
          indent: { left: 720 },
          children: [
            new TextRun({ text: "Chelsea", bold: true, font: "Georgia", size: 28 }),
          ],
        }),
        body(
          "Consider adapting Duel Frame for the landing hero — a live or upcoming matchup as the visual anchor instead of abstract 3D renders."
        ),

        heading1("6. Landing page structure (recommended)"),
        body(
          "Below is a suggested information architecture. Adjust section order for mobile scroll priority — hero and primary CTA must appear above the fold on 390px viewports."
        ),

        heading2("Section A — Hero"),
        bullet("numbers", "Headline: peer-to-peer challenge framing, not “bet with us”."),
        bullet("numbers", "Subhead: TxLINE-verified settlement in one sentence."),
        bullet("numbers", "Primary CTA: “Enter Matchlock” or “Start challenging” → /login."),
        bullet("numbers", "Secondary CTA: “See how it works” → scroll to explainer."),
        bullet("numbers", "Visual: Duel Frame matchup mockup or animated wager state (Open → Matched → Settled)."),
        bullet("numbers", "Trust strip: Solana logo + “Powered by TxLINE” + “Non-custodial” badges."),

        heading2("Section B — How it works (3–4 steps)"),
        makeTable(
          ["Step", "Title", "Copy direction"],
          [
            ["1", "Pick a match", "Browse live World Cup fixtures with real-time status"],
            ["2", "Challenge or accept", "Create a wager on your side, or take the opposite side of an open challenge"],
            ["3", "Stakes lock on-chain", "USDC moves to a program vault — not our wallet"],
            ["4", "Settle automatically", "When the match ends, TxLINE proves the result; winner gets 2× stake"],
          ],
          [800, 2400, 6160]
        ),

        new Paragraph({ spacing: { before: 160, after: 120 }, children: [] }),

        heading2("Section C — Why PvP / Why on-chain"),
        bullet("bullets", "Contrast card: “Sportsbook” (house edge, opaque settlement) vs “Matchlock” (peer vs peer, on-chain proof)."),
        bullet("bullets", "Highlight cancel-while-open and non-custodial vaults as trust differentiators."),
        bullet("bullets", "Avoid jargon walls — explain Merkle proof as “cryptographic receipt anyone can verify.”"),

        heading2("Section D — Product preview"),
        bullet("bullets", "Screenshot or high-fidelity mock of Markets page (match browser)."),
        bullet("bullets", "Open wagers list showing maker side, stake, and Accept CTA."),
        bullet("bullets", "Wager confirmation modal showing simulate → sign flow."),
        bullet("bullets", "Settlement progress indicator (Live → Final → Settling → Paid out)."),

        heading2("Section E — Coming soon"),
        body(
          "Subdued row of 2–3 ghost feature cards for Pools, Verifiable Receipts, and Leaderboard. Use muted borders, lock or clock icon, and explicit “Coming soon” labels. Do not place these above the primary CTA."
        ),

        heading2("Section F — Footer"),
        bullet("bullets", "Powered by TxLINE (link to txodds.com)."),
        bullet("bullets", "Links: Docs, GitHub (if public), Terms, Privacy."),
        bullet("bullets", "Devnet disclaimer: “Currently on Solana devnet — mainnet launch forthcoming.”"),
        bullet("bullets", "Social / community links if available."),

        heading1("7. Copy & tone guidelines"),
        makeTable(
          ["Avoid", "Use instead"],
          [
            ["Place a bet", "Create a wager / Accept a challenge"],
            ["Bet with us", "Challenge another player"],
            ["The house", "Your opponent / Another player"],
            ["Odds from us", "Payout if you win: 2× stake"],
            ["Bet slip", "Wager summary / Challenge slip"],
            ["Guaranteed profits", "Transparent payout rules"],
            ["Web3 gambling", "Peer-to-peer sports wagers on Solana"],
          ],
          [4680, 4680]
        ),

        new Paragraph({ spacing: { before: 160, after: 120 }, children: [] }),

        body(
          "Tone: confident, warm, precise. Matchlock is for sports fans who want fair head-to-head competition with transparent rules — not degens chasing 100× leverage."
        ),

        heading1("8. Web3 & product inspiration"),
        body(
          "Study these projects for landing-page patterns, trust signaling, and interaction design. Matchlock should borrow structural clarity and conversion patterns — not their color palettes or house-edge models."
        ),

        heading2("Prediction markets & sports"),
        linkParagraph(
          "Polymarket",
          "https://polymarket.com",
          "Clean prediction-market landing; strong “how it works” funnel; probability-forward cards. Borrow: clarity of market cards and trust section. Avoid: pool/AMM framing — Matchlock is PvP-first."
        ),
        linkParagraph(
          "Azuro",
          "https://azuro.org",
          "Decentralized sports betting protocol. Borrow: protocol credibility framing, developer/trust split. Note: Azuro is liquidity-pool based — opposite of Phase 1 PvP."
        ),
        linkParagraph(
          "Overtime Markets",
          "https://overtimemarkets.xyz",
          "On-chain sports markets on Optimism/Base. Borrow: sports-specific hero imagery treatment (type-led, not stock photos), live market energy."
        ),
        linkParagraph(
          "SX Bet",
          "https://sx.bet",
          "Peer-to-peer sports betting exchange. Borrow: order-book / counterparty mental model — closest analog to Matchlock’s maker-taker flow."
        ),
        linkParagraph(
          "Zeitgeist",
          "https://zeitgeist.pm",
          "Substrate-based prediction markets. Borrow: educational onboarding for non-crypto users, governance/trust copy."
        ),

        heading2("Solana ecosystem (landing craft)"),
        linkParagraph(
          "Jupiter",
          "https://jup.ag",
          "Best-in-class Solana product landing: instant value prop, crisp hero, social proof strip, minimal chrome. Borrow: hero brevity and CTA hierarchy."
        ),
        linkParagraph(
          "Drift",
          "https://www.drift.trade",
          "Perps on Solana. Borrow: product screenshot carousel, stats strip (volume/TVL equivalent → Matchlock: wagers settled, matches covered)."
        ),
        linkParagraph(
          "Phantom",
          "https://phantom.app",
          "Wallet landing page. Borrow: trust/security section layout, friendly illustration style (adapt to warm Sahara palette)."
        ),
        linkParagraph(
          "Marinade Finance",
          "https://marinade.finance",
          "Staking landing. Borrow: single-message hero, step-by-step explainer with icons, restrained animation."
        ),
        linkParagraph(
          "Jito",
          "https://www.jito.network",
          "Infrastructure landing. Borrow: “how the system works” diagram style for keeper + TxLINE settlement pipeline."
        ),

        heading2("Sportsbook layout patterns (palette only — not brand)"),
        linkParagraph(
          "Stake.com",
          "https://stake.com",
          "Borrow: match row density, live badges, filter chips, mobile-first bet flow layout. Do NOT borrow: dark-neon palette, casino gamification, house-edge copy."
        ),
        linkParagraph(
          "1xBet",
          "https://1xbet.com",
          "Borrow: matchup hierarchy (home vs away equal weight), compact odds grid → adapt to PvP payout display. Do NOT borrow: visual clutter, aggressive promotions."
        ),

        heading2("Trust & transparency UX"),
        linkParagraph(
          "Etherscan",
          "https://etherscan.io",
          "Borrow: “verify on explorer” link pattern for settled wagers — Matchlock should make on-chain verification feel accessible."
        ),
        linkParagraph(
          "Hyperliquid",
          "https://hyperliquid.xyz",
          "Borrow: ultra-minimal hero, data-dense secondary sections, no fluff. Good reference for “serious money” tone without casino aesthetics."
        ),

        new Paragraph({ children: [new PageBreak()] }),

        heading1("9. Technical context for accurate visuals"),
        heading2("Architecture (simplified for diagrams)"),
        body(
          "User wallets → Matchlock Solana program (escrow vaults) → TxLINE program (validate_stat CPI). Off-chain: TxLINE SSE stream → Go backend (Redis cache + keeper) → settlement transaction on is_final."
        ),
        body(
          "A landing-page diagram could show: Two players → On-chain vault → TxLINE oracle → Automatic payout. Keep it non-technical in the hero; offer an expandable “For the curious” technical accordion."
        ),

        heading2("Settlement flow (for explainer animation)"),
        bullet("numbers", "Maker creates wager → USDC locked in vault PDA."),
        bullet("numbers", "Taker accepts → matching stake locked; status Matched."),
        bullet("numbers", "Match plays → live status from TxLINE SSE."),
        bullet("numbers", "Match goes final → keeper fetches Merkle proof."),
        bullet("numbers", "settle_wager CPI validates stat → winner receives 2× stake."),
        bullet("numbers", "Optional: maker cancels while Open → full refund."),

        heading2("Collateral"),
        body(
          "Wagers use USDC (devnet test stablecoin today; mainnet USDC at launch). TxLINE credit tokens are for API authorization only — never shown to end users as wager collateral."
        ),

        heading1("10. Accessibility & responsive requirements"),
        bullet("bullets", "Design mobile-first at 320px width; enhance at 640px, 1024px."),
        bullet("bullets", "Text contrast ≥ 4.5:1 on all body copy (Sahara palette already tuned for this)."),
        bullet("bullets", "Touch targets ≥ 44×44px on all CTAs."),
        bullet("bullets", "Visible focus rings on interactive elements."),
        bullet("bullets", "Respect prefers-reduced-motion — no autoplay slot animations or flashing win banners."),
        bullet("bullets", "Status and outcomes never communicated by color alone."),

        heading1("11. Deliverables checklist"),
        bullet("bullets", "Desktop and mobile landing page mockups (1440px and 390px)."),
        bullet("bullets", "Hero variant with Duel Frame matchup composition."),
        bullet("bullets", "“How it works” step illustrations or icon set in Sahara palette."),
        bullet("bullets", "PvP vs sportsbook comparison card."),
        bullet("bullets", "Coming soon section with ghost cards (Pools, Receipts, Leaderboard)."),
        bullet("bullets", "Component specs: buttons, badges, trust strip, footer."),
        bullet("bullets", "Optional: Ember dark-mode hero variant."),
        bullet("bullets", "Copy deck with final headlines, subheads, and CTA labels."),

        heading1("12. Key links & resources"),
        linkParagraph("TxLINE Quickstart", "https://txline.txodds.com/documentation/quickstart", "Oracle and settlement documentation"),
        linkParagraph("TxLINE World Cup API", "https://txline.txodds.com/documentation/worldcup", "Match data source for markets"),
        linkParagraph("TxOdds", "https://txodds.com", "Parent company — Powered by TxLINE attribution"),
        body("Internal references: docs/plan.md (engineering roadmap), frontend-react/src/index.css (Sahara tokens), .grok/skills/blockchain-ui/ (PvP UI patterns)."),

        new Paragraph({ spacing: { before: 400 }, children: [] }),
        new Paragraph({
          alignment: AlignmentType.CENTER,
          children: [
            new TextRun({
              text: "Questions? Reach out to the engineering team with wireframes for feasibility review before final handoff.",
              italics: true,
              size: 22,
              color: "6B5F56",
            }),
          ],
        }),
      ],
    },
  ],
});

const outPath = "/home/ubuntu/Desktop/matchlock/docs/Matchlock-Landing-Page-Brief.docx";

Packer.toBuffer(doc).then((buffer) => {
  fs.writeFileSync(outPath, buffer);
  console.log("Written:", outPath);
});