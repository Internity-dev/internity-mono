# Brand: Internity

_Status: applied_

## Palette

The primary blue is sampled directly from the real logo file
(`D:\Project\Internity\Internity\public\img\logo-internity.png`), not picked
freehand. The logo's gradient runs from `#2BADF7` (bright, top) to `#0263A9`
(deep, bottom). In OKLCH that's hue ≈ 244°, chroma ≈ 0.13–0.15.

The previous `primary` scale (`#2e63f5` / `#0d2fa8` etc.) had drifted to
hue ≈ 265° with chroma ≈ 0.20–0.23, more violet-indigo and more saturated
than the actual brand mark. The step used as the default light-mode
`--primary` (`#0d2fa8`, L=0.38) read as too dark. The scale below corrects
both: same hue family as the logo, same rough chroma, and a brighter default.

`packages/design-tokens/tokens.json` → `palette.primary`:

| Step | Hex | OKLCH |
|---|---|---|
| 50 | `#eaf7ff` | `oklch(0.97 0.02 244)` |
| 100 | `#d8eeff` | `oklch(0.94 0.035 244)` |
| 200 | `#bae0ff` | `oklch(0.89 0.06 244)` |
| 300 | `#91cbfb` | `oklch(0.82 0.09 244)` |
| 400 | `#63b3f1` | `oklch(0.74 0.12 244)`, dark-mode `--primary` |
| 500 | `#2e9ae4` | `oklch(0.66 0.145 244)`, near-identical to the logo's bright gradient stop |
| 600 | `#007ec8` | `oklch(0.57 0.15 244)` |
| 700 | `#0065ab` | `oklch(0.49 0.145 244)`, near-identical to the logo's deep gradient stop; light-mode `--primary` |
| 800 | `#004e8a` | `oklch(0.41 0.13 244)` |
| 900 | `#003866` | `oklch(0.33 0.105 244)` |
| 950 | `#001c3c` | `oklch(0.22 0.08 244)` |

Contrast: `primary-700` (light `--primary`) vs white is 6.1:1, `primary-400`
(dark `--primary`) vs the dark background is comfortably above 4.5:1. Both
pass WCAG AA for body-sized text.

`accent` (teal, `#0bb98e` family) and `neutral` (cool gray) are unchanged.
Only `primary` and the `info` semantic color (which mirrored the old
`primary-500`) were rebuilt.

## Elevation

Cards previously separated from the page with only a 1px ring, no shadow,
and in dark mode `--card` was only one neutral step above `--background`
(L 0.17 → 0.223), which read as flat. Fixed two ways:

- `Card.vue` now carries a real shadow (`shadow-sm shadow-black/5` in light
  mode, `shadow-md shadow-black/40` in dark mode; dark shadows need higher
  opacity to read at all against a dark background).
- Dark-mode `card`/`popover`/`sidebar` moved from `neutral.900` to
  `neutral.800` (L 0.223 → 0.291), widening the gap from `background`
  (`neutral.950`, L 0.17) to 0.12. `sidebarAccent` (nav-item hover) moved to
  `neutral.700` so it stays visibly distinct from the now-lighter sidebar
  background.
- The dashboard's home-page shortcut cards also got a hover
  lift (`hover:-translate-y-0.5`) alongside the shadow increase, 150ms
  `ease-out`.

## Mode default

The dashboard defaults to **light mode** regardless of OS preference
(`useDark({ initialValue: 'light' })` in `DefaultLayout.vue`). Dark mode is
still available via the header toggle and persists once chosen.

## Typography

Sora, used everywhere: body, display, headings. Inter is dropped. Six
Gen-Z-leaning pairs were previewed (Space Grotesk, Unbounded, Bricolage
Grotesque, Outfit, Lexend, and Sora-full). Sora-full won since the brand
already committed to Sora as its display face, and this just extends it to
body text instead of introducing a second family. `JetBrains Mono` stays for
`--font-mono` (code/numeric contexts), unaffected by this pick.

Google Fonts import (dashboard `main.css` and landing `main.css`):
`Sora:wght@400;500;600;700;800`.

## Usage

- `primary-700` / `--primary` is the default button/link/active-state color
  in light mode. Don't reach for `primary-600` or darker for large filled
  areas, it reads heavier than the logo.
- `primary-500` is closest to the logo's own bright blue, good for
  illustrative/decorative use (gradients, icon accents) where you want the
  mark to feel literally on-brand.
- Regenerate `theme.css` after any change here: `pnpm tokens:build` from
  `packages/design-tokens`.

_Applied: 2026-08-23_
