# Matchlock Design System

Inferred from `frontend-react/home.html` — the marketing landing page.

---

## Color Palette

All colors are verified from the compiled HTML/CSS of the landing page.

### Backgrounds & Surfaces

| Token | Hex | Usage |
|---|---|---|
| Page background | `#282828` | Body/section fills |
| Card / surface | `#2c2c2c` | Nav bar, workflow cards, comparison cards, match-example card, footer |
| Card border | `#373737` | Default card stroke |
| Team avatar well | `#383838` | Circular avatar placeholder |
| Team badge bg | `#3b312e` | Team abbreviation pill |
| Team badge border | `#e66140` / 0.3 opacity | Team pill outline |

### Brand — Red

| Token | Hex | Usage |
|---|---|---|
| Primary | `#e64040` | Active nav link, step numbers, highlighted headings, CTA gradient start, accent bar start |
| Primary-light | `#e64e40` | Accent bar end |
| CTA highlight | `#e66140` | Button gradient end, team badge border tint |
| Match status | `#e64d40` | "Open • 1,250.00 USDT" text |

### Semantic

| Token | Hex | Usage |
|---|---|---|
| Drawback (✕) | `#ef4444` (red-500) | Traditional sportsbook list items |
| Drawback heading | `#f87171` (red-400) | "TRADITIONAL SPORTSBOOKS" label |
| Benefit (✔) | `#34d399` (emerald-400) | Matchlock protocol list items |
| Verified badge bg | `#059669` (emerald-600) | "Verified" pill |

### Text

| Token | Hex | Usage |
|---|---|---|
| Primary text | `#ffffff` | Body, headings, nav |
| Muted | `white/80` | Hero description |
| Muted | `white/70` | Card body text, section subheading |
| Footer | `white/60` | Footer tagline |
| Gradient hero | `white` → `white` → `#999` | "Settle on-chain." display text (text-transparent bg-clip) |

### Gradients

| Gradient | Direction | Usage |
|---|---|---|
| `#e64040` → `#e66140` | to right | Primary CTA buttons |
| `#e64040` → `#e64e40` | to right | Section accent bars |
| `white` → `white` → `#999` | to right | Hero display text fill |

---

## Typography

### Font Stack

| Role | Family | Import |
|---|---|---|
| UI / body | `Manrope`, system-ui, sans-serif | CDN + `@fontsource-variable/manrope` |
| Display / headings | `Garamond`, serif | CDN + `@fontsource/eb-garamond` |

### CSS Class

```css
body { font-family: 'Manrope', system-ui, sans-serif; }
.garamond { font-family: 'Garamond', serif; }
```

### Font Weights

- **Manrope:** 400, 500, 600, 700
- **Garamond:** 400, 700, italic

### Type Scale (landing page)

| Element | Size | Weight | Tracking | Class/Notes |
|---|---|---|---|---|
| Hero heading line 1 | `text-7xl` (4.5rem) / `md:text-8xl` | `font-medium` | `tracking-tighter` | Manrope |
| Hero heading line 2 | `text-[110px]` / `md:text-[130px]` | bold (font-family default) | — | Garamond, gradient fill |
| Section heading | `text-5xl` (3rem) | `font-medium` | `tracking-tighter` | Manrope |
| Section subheading | `text-xl` | normal | — | `white/70` |
| Card number (01.) | `text-4xl` | `font-bold` | — | Primary red |
| Card title | `text-3xl` | `font-semibold` | — | White |
| Card body | base (1rem) | normal | — | `white/70` |
| Comparison heading | `text-xl` | `font-bold` | — | Uppercase style |
| Nav links | `text-sm` | `font-medium` | — | Manrope |
| VS badge | `text-5xl` | `font-bold` | — | Black on white/70 |
| Team abbreviation | `text-3xl` | `font-bold` | — | Badge pill |
| Footer links | `text-sm` | normal | — | |

---

## Spacing

| Token | Value | Usage |
|---|---|---|
| Section Y padding | `py-32` (8rem) | Workflow / Advantage sections |
| Card padding | `p-10` (2.5rem) | All cards |
| Grid gap | `gap-8` (2rem) | Card grids |
| Section max width | `max-w-6xl` (72rem) | Section containers |
| Grid max width | `max-w-5xl` (64rem) | Card grids |
| Hero content max | `max-w-4xl` (56rem) | Hero text block |
| Container side padding | `px-6` (1.5rem) | Section inner padding |
| Nav padding | `py-4 px-10` | Nav bar |
| Button padding | `px-8 py-3` / `px-10 py-4` | Default / hero CTA |
| Section top margin | `mt-20` (5rem) | Grid after section heading |
| Content max-width (sm) | `max-w-md` (28rem) | Section subheading |
| Content max-width (sm) | `max-w-lg` (32rem) | Hero description |

---

## Border Radius

| Token | Value | Usage |
|---|---|---|
| Nav pill | `rounded-[100px]` | Nav bar container |
| Bottom card pill | `rounded-[58px]` | Match example card |
| Buttons | `rounded-full` | All CTAs |
| Cards | `rounded-3xl` (1.5rem) | Workflow / comparison cards |
| Team badge | `rounded-xl` (0.75rem) | ARG / POR pill |
| VS badge | `rounded-2xl` (1rem) | Center VS badge |
| Avatar | `rounded-full` | Team avatar circle |
| Verified badge | `rounded-full` | "Verified" pill |
| Accent bar | `rounded-full` | Section accent lines |

---

## Effects & Shadows

| Effect | Pattern |
|---|---|
| CTA gradient | `bg-gradient-to-r from-[#e64040] to-[#e66140]` |
| Ghost button | `border border-white/30 hover:bg-white/5` |
| Text gradient | `bg-clip-text text-transparent bg-gradient-to-r from-white via-white to-[#999]` |
| Accent bar | `bg-gradient-to-r from-[#e64040] to-[#e64e40] h-1.5 w-28 rounded-full` |
| Nav link hover | `hover:text-white transition-colors` |
| Hero bg overlay | `absolute inset-0 opacity-20` (image) |
| No box shadows | All depth via borders and color contrast, not shadows |

---

## Layout Patterns

### Page Shell
- Full dark page (`#282828`)
- Fixed pill nav at top center
- Full-viewport hero sections
- Footer at bottom
- No sidebar, no secondary chrome

### Navigation
- **Position:** Fixed top, centered (`top-8 left-1/2 -translate-x-1/2`)
- **Shape:** Pill (`rounded-[100px]`)
- **Max width:** 1282px (`w-[1282px]`)
- **Pattern:** Logo left → links center → CTAs right
- Active link highlighted in `#e64040`

### Sections
1. **Accent bar** — centered gradient line above title
2. **Heading** — centered, Manrope, `text-5xl`, `tracking-tighter`
3. **Subtitle** — centered, `text-xl`, muted
4. **Content grid** — appears ~80px below (`mt-20`)

### Hero
- Full viewport height (`min-h-screen`)
- Content centered vertically & horizontally
- Decorative match-example card anchored to bottom `12`
- Background overlay image at `opacity-20`

### Cards
- Dark surface (`#2c2c2c`)
- Border stroke (`#373737`)
- `rounded-3xl`
- `p-10` padding
- Numbered steps (01., 02., etc.) in primary red

### Comparison (side-by-side)
- 2-column grid
- "Traditional" side: red X icons, red headings
- "Matchlock" side: green check icons, red heading, "Verified" badge

### Footer
- Dark background (`#282828`)
- `border-t border-white/10`
- `py-16`
- Brand left, link row right
- `flex justify-between items-center` (stacks on smaller screens)

---

## Component Inventory

| Component | Status | Notes |
|---|---|---|
| Navigation bar | `home-page.tsx` | Fixed pill, 4 links + 2 CTAs |
| Hero section | `home-page.tsx` | Full-screen, gradient text, description, 2 CTAs |
| Match example card | `home-page.tsx` | Team-avatar-team layout with VS center |
| Section accent bar | `home-page.tsx` | Gradient line |
| Workflow card | `home-page.tsx` | Numbered, title, description |
| Comparison card | `home-page.tsx` | Pro/con with icons, optional "Verified" badge |
| Footer | `home-page.tsx` | Brand, links |
| Button (primary) | — | Gradient red, rounded-full |
| Button (ghost) | — | White border, rounded-full |
| Avatar (team) | — | Circular well for flag/logo |
| Team badge | — | Rounded pill with abbreviation |
| VS badge | — | Centered large-text badge |
| Verified badge | — | Small emerald pill, absolute positioned |

---

## Dark Theme Context

The landing page uses a dark theme that is **distinct** from the app's built-in dark mode ("Ember"):

| Property | Landing Page | App Ember (`.dark`) |
|---|---|---|
| Background | `#282828` | `#1e1814` |
| Card | `#2c2c2c` | `#2a2420` |
| Primary | `#e64040` | `#d4763f` |
| Accent | `#e66140` | `#a85555` |
| Aesthetic | High-contrast, pure dark, red-forward | Warm minimalism, brown-tinged |

The landing page is a **standalone dark experience**, not using the app's theme variables. It applies colors directly via arbitrary Tailwind values (`bg-[#282828]`, etc.).

---

## Icons & Symbols

| Use | Glyph | React Icon |
|---|---|---|
| Drawback | ✕ | `lucide-react` `X` (red-500) |
| Benefit | ✔ | `lucide-react` `Check` (emerald-400) |
| CTA arrow | — | `lucide-react` `ArrowRight` |

---

## Responsive Behavior

- Hero text: stacks naturally (br tag), sizes scale down on mobile
- Hero display text: `text-[110px]` → `md:text-[130px]`
- Workflow grid: `grid-cols-2` on md+, single column below
- Comparison grid: `grid-cols-2` on md+, single column below
- Nav: hidden links on mobile (`hidden lg:flex`)
- Footer: stacks vertically `flex-col` on `md:`
- Match example card: uses `max-w-[95vw]` to stay within viewport
