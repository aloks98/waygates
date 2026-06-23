# Rebrand + Re-theme — Design

**Date:** 2026-06-23
**Status:** Approved — ready for implementation plan
**Context:** Replace the warm "copper" theme + monochrome "W" logo with the new **synthwave pixel-portal** identity (assets in `logo-design/`). Frontend-only. Branch off master. Explored live in the brainstorming visual companion; the decisions below are locked.

## Goal

1. Ship the new **pixel-portal logo** everywhere it appears (favicon, sidebar, login, signup) and a VT323 "WAYGATES" wordmark lockup.
2. Re-theme the app to a **black-canvas, periwinkle-accented, square-cornered** identity that *complements* the bright logo rather than copying its neon — keeping **both dark and light** themes.

## Locked decisions

- **Canvas:** dark theme is a **true-black** (`#000000`) canvas with near-black surfaces; **light theme retained** (white/neutral canvas). Both first-class via the existing next-themes toggle.
- **Primary:** periwinkle `#6E72F0` (dark) / `#5862E0` (light, deepened for AA). **Secondary/focus accent:** cyan `#34E1E0` (dark) / `#18A6AE` (light). Bright magenta/cyan are *not* used as flat UI fills — they live in the **logo** and in **data-viz / focus** only.
- **Shape:** square — `--radius: 0` globally (cards, buttons, inputs, badges, menus, avatars). Brutalist/crisp, matching the pixel art.
- **Type system:**
  - Wordmark only → **VT323** (terminal/CRT pixel face)
  - UI / headings / body → **Space Grotesk**
  - Data / hosts / config / logs / badges / labels → **Space Mono**
- **Charts:** `--chart-1..5` switch to the logo palette (periwinkle, cyan, violet, magenta, blue-violet).
- **Dark-mode flourish:** subtle glow (low-alpha primary `box-shadow`) on primary buttons, the focus ring, and the active nav item. Light mode: no glow.

## Palette → token mapping

Overrides live in `ui/src/app.css` (`:root` = light, `.dark` = dark), replacing the copper values. Values given as hex (authoritative); keep the existing token names. (rnui-themes tokens accept hex in CSS custom properties; convert to oklch only if a token requires it.)

### Dark (`.dark`) — black canvas
| Token | Value |
|---|---|
| `--background` | `#000000` |
| `--foreground` | `#EDEDF4` |
| `--card` / `--popover` | `#0B0B12` |
| `--card-foreground` / `--popover-foreground` | `#EDEDF4` |
| `--primary` | `#6E72F0` |
| `--primary-foreground` | `#FFFFFF` |
| `--secondary` / `--muted` / `--accent` | `#15151F` |
| `--secondary-foreground` / `--accent-foreground` | `#EDEDF4` |
| `--muted-foreground` | `#8A8A99` |
| `--border` / `--input` | `#22222F` |
| `--ring` | `#34E1E0` (cyan focus) |
| `--destructive` | `#E5675F` |
| `--destructive-foreground` | `#FFFFFF` |
| `--chart-1..5` | `#6E72F0`, `#34E1E0`, `#A246E8`, `#FF3D9A`, `#6E8AF0` |

### Light (`:root`) — neutral canvas
| Token | Value |
|---|---|
| `--background` | `#F7F7FA` |
| `--foreground` | `#16171D` |
| `--card` / `--popover` | `#FFFFFF` |
| `--card-foreground` / `--popover-foreground` | `#16171D` |
| `--primary` | `#5862E0` |
| `--primary-foreground` | `#FFFFFF` |
| `--secondary` / `--muted` / `--accent` | `#F1F2F6` |
| `--secondary-foreground` / `--accent-foreground` | `#16171D` |
| `--muted-foreground` | `#5C6072` |
| `--border` / `--input` | `#E3E4EC` |
| `--ring` | `#18A6AE` (deeper cyan for visibility on white) |
| `--destructive` | `#D14A42` |
| `--destructive-foreground` | `#FFFFFF` |
| `--chart-1..5` | `#5862E0`, `#18A6AE`, `#8B5CF6`, `#E0408F`, `#6E72F0` |

`--radius: 0` (both). The shadow scale stays minimal; dark mode adds a primary-glow utility (see below).

## Typography

Replace the current font imports (Clash Grotesk / Bricolage Grotesque / Geist Mono) in `app.css` with Google Fonts:
```css
@import url('https://fonts.googleapis.com/css2?family=Space+Grotesk:wght@300;400;500;600;700&family=Space+Mono:wght@400;700&family=VT323&display=swap');
```
Token assignments (`:root` and `@theme inline`):
- `--font-sans` / `--text-sans` / `--font-heading` → `'Space Grotesk', system-ui, sans-serif`
- `--font-mono` → `'Space Mono', ui-monospace, monospace`
- New `--font-display` → `'VT323', monospace` — used **only** in the wordmark lockup, not anywhere in body/UI.
- Drop the `--font-serif` (Bricolage) usage; if a token must remain, point it at Space Grotesk.

## Logo wiring

- **Favicon:** copy `logo-design/waygates-favicon.svg` → `ui/public/favicon.svg` (overwrite the old copper W). `index.html` already links `/favicon.svg`.
- **`index.html`:** keep `<title>Waygates</title>`; add `<meta name="theme-color" content="#000000">`.
- **`ui/src/components/layout/waygate-logo.tsx`:** replace the monochrome `currentColor` "W" with the **pixel portal mark** (multi-color). Render `logo-design/waygates-mark.svg`'s contents (transparent, no tile, so it sits on the black surface) as an inline SVG; `className` controls size only (the `currentColor` contract is dropped — document this). The animated variant (`waygates-icon-animated.svg`) is optional and out of scope for v1.
- **Wordmark lockup:** sidebar / login / signup render `<WaygateLogo>` + the text "WAYGATES" in `--font-display` (VT323). Update `sidebar.tsx`, `login.tsx`, `signup.tsx` to use the wordmark font for that text.
- **ACL login default branding** (`acl-login.tsx` / `DEFAULT_BRANDING`): the public ACL login is tenant-configurable, but its built-in defaults should match the new identity (dark canvas, periwinkle). Update the default `primary_color` / `background_color` and the fallback mark. (Low-risk, in scope; admins can still override per-tenant.)

## Components & scope

**In scope (frontend only):**
- `ui/src/app.css` — palette tokens (light+dark), font imports + tokens, `--radius: 0`, a dark-mode primary-glow utility (e.g. `.glow-primary` / applied to primary button + focus), keep the existing scrollbar/animation blocks.
- `waygate-logo.tsx`, `sidebar.tsx`, `login.tsx`, `signup.tsx` — new mark + VT323 wordmark.
- `ui/public/favicon.svg`, `ui/index.html`.
- `acl-login.tsx` default branding.
- A **sweep** for hardcoded `rounded-*` utilities that should square (badges, avatars, anything not honoring `--radius`) and for stale font references.
- Charts (traffic, fleet donut) pick up the new `--chart-*` automatically — verify visually.

**Out of scope:** backend; pixel-art-ifying the whole UI (pixel font is wordmark-only); per-component redesign; self-hosting fonts; the animated logo variant.

## Testing

- Gate: `pnpm --dir ui build` + `pnpm --dir ui check` + `pnpm --dir ui test` (existing tests stay green).
- Visual smoke (both themes): dashboard, a data table, a chart, buttons (primary/outline/ghost/danger), inputs + focus ring, badges, the sidebar logo + wordmark, login + signup, the theme toggle, the logo/favicon at small sizes.

## Risks & notes

- **Light theme loses the neon-on-black look** — accepted; light is a clean neutral periwinkle treatment.
- **Contrast:** white-on-periwinkle buttons are borderline AA — primary is deepened to `#5862E0` in light and buttons use 600-weight; verify periwinkle text/cyan ring meet WCAG AA on each canvas. Bright cyan is reserved for focus/data, not body text.
- **Square `--radius: 0`** may expose components that assume rounding (avatars, pills); the sweep handles overrides.
- **Fonts load from Google Fonts** at runtime (`display=swap`); self-hosting is a later optimization.
- **`currentColor` contract change** for the logo: any place that recolored the old W via text color must be updated to size-only.
