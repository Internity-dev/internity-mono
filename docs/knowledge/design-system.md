# Design System

Verified against the actual files on 2026-08-26. Every value below was read
from source, not inferred — file:line refs let you jump straight to it.

## Pipeline: tokens.json → theme.css → both apps

`packages/design-tokens/tokens.json` is the **only** place color/type/radius
values are authored. Nothing else should hardcode a hex or px value for
these (see Rule 10 below).

- Build: `pnpm tokens:build` at repo root (`package.json:13` →
  `pnpm --filter @internity/design-tokens build`), or
  `pnpm --filter @internity/design-tokens build` directly. Runs
  `packages/design-tokens/scripts/build.mjs`.
- `build.mjs` reads `tokens.json`, resolves `"primary.700"`-style references
  against `tokens.palette`, and writes `packages/design-tokens/dist/theme.css`
  — a generated file, **do not hand-edit** (it says so in its own header
  comment, `dist/theme.css:1-4`).
- `theme.css` emits three blocks: `:root` (raw `--color-{group}-{shade}`
  ramps + light semantic vars), `.dark` (dark semantic vars only — the ramps
  are shared), and `@theme inline` (maps everything to Tailwind v4 utility
  vars, e.g. `bg-primary`, `text-muted-foreground`).
- Both apps consume it as a package import, not a relative path:
  `@import "@internity/design-tokens/theme.css";` — dashboard:
  `apps/dashboard/src/assets/main.css:5`, landing:
  `apps/landing/app/assets/css/main.css:2`. Package export map is in
  `packages/design-tokens/package.json:10-13`.
- Dark mode is applied via a `.dark` class on `<html>` (Tailwind's
  `@custom-variant dark (&:where(.dark, .dark *));`, `main.css:8`). **Landing
  has no dark mode at all** — its `main.css` hardcodes `color-scheme: light`
  (`apps/landing/app/assets/css/main.css:14`) and there is no `.dark` rule,
  no `useDark`, anywhere under `apps/landing`. Dark mode is a
  dashboard-only feature.

After editing `tokens.json`, you must re-run `pnpm tokens:build` — nothing
watches the file automatically.

## Brand identity — `brand.md` (repo root)

Read this file in full before touching any color. Summary of what it says
and why:

- **Primary blue is sampled from the real logo**
  (`D:\Project\Internity\Internity\public\img\logo-internity.png`, outside
  this repo), not picked freehand. Logo gradient: `#2BADF7` (bright, top) →
  `#0263A9` (deep, bottom), hue ≈ 244° in OKLCH.
- It replaced an earlier `primary` scale (`#2e63f5`/`#0d2fa8`, hue ≈ 265°,
  more violet, more saturated) that had drifted from the actual logo, and
  whose light-mode default (`#0d2fa8`, L=0.38) read too dark for a filled
  button. If you ever see those old hex values reappear anywhere, that's a
  regression back to the pre-brand.md palette — flag it.
- **Typography is Sora everywhere** (body, display, headings) — Inter was
  deliberately dropped after comparing six candidate pairs. `JetBrains Mono`
  stays for `--font-mono`. Google Fonts is wired per-app, not through
  `main.css` in both cases: dashboard imports it directly in
  `main.css:1`; landing pulls it via `<head>` link tags in
  `apps/landing/nuxt.config.ts:45,49` (Nuxt idiom, not a CSS `@import`).
- **Light mode is the default product experience**, overriding OS
  preference. This is a deliberate product decision, not an oversight — see
  Dark Mode section below for the actual comment justifying it.
- Contrast claims in brand.md for the *light-mode* primary: `primary-700`
  (`#0065ab`) vs white = 6.1:1 (independently recomputed below: 6.09:1,
  matches). Passes WCAG AA for body text.

## Current token values (read directly from `tokens.json`)

`packages/design-tokens/tokens.json`:

**Palette — primary** (logo blue):
`50 #eaf7ff · 100 #d8eeff · 200 #bae0ff · 300 #91cbfb · 400 #63b3f1 · 500 #2e9ae4 · 600 #007ec8 · 700 #0065ab · 800 #004e8a · 900 #003866 · 950 #001c3c`

**Palette — accent** (teal):
`50 #e8fdf7 · 100 #c9fbec · 200 #96f5da · 300 #5be8c4 · 400 #22d3a8 · 500 #0bb98e · 600 #059271 · 700 #08745c · 800 #0b5c4a · 900 #0c4c3e · 950 #032b23`

**Palette — neutral** (cool gray):
`50 #f7f8fa · 100 #eef0f4 · 200 #dfe3ea · 300 #c5cbd6 · 400 #98a1b3 · 500 #717a8f · 600 #545d70 · 700 #3d4457 · 800 #262b3a · 900 #171b26 · 950 #0d0f16`

**Flat semantics**: `success #0fb782` · `warning #e9b207` · `danger #f03e61`
· `info #2e9ae4` (== `primary.500`).

**Typography**: `sans`/`display` both `["Sora", "ui-sans-serif", "system-ui", "sans-serif"]`
(same array — there is no separate display face despite the two token
names existing). `mono`: `["JetBrains Mono", "ui-monospace", "monospace"]`.

**Radius**: `base: 0.875rem` → generates `--radius-sm` (base−4px),
`--radius-md` (base−2px), `--radius-lg` (=base), `--radius-xl` (base+4px)
in `build.mjs:68-71`.

## Semantic mapping — light vs dark

Both blocks live in `tokens.json:33-101`. Key pattern: **light mode uses
white `card`/`popover`/`sidebar` on a `neutral.50` page background**; **dark
mode uses `neutral.800` for those same surfaces on a `neutral.900` page
background** — dark surfaces are one step *lighter* than the page, light
surfaces are pure white above a barely-tinted page. Not symmetric, and
that's intentional (see Elevation below).

| token | light | dark |
|---|---|---|
| background | `neutral.50` `#f7f8fa` | `neutral.900` `#171b26` |
| foreground | `neutral.900` `#171b26` | `neutral.200` `#dfe3ea` |
| card / popover / sidebar | `#ffffff` | `neutral.800` `#262b3a` |
| primary | `primary.700` `#0065ab` | `primary.400` `#63b3f1` |
| primaryForeground | `#ffffff` | `neutral.950` `#0d0f16` |
| muted / secondary | `neutral.100` `#eef0f4` | `neutral.800` `#262b3a` |
| mutedForeground | `neutral.500` `#717a8f` | `neutral.400` `#98a1b3` |
| accent (surface, not brand accent) | `neutral.100` | `neutral.700` `#3d4457` |
| border / input | `neutral.200` `#dfe3ea` | `neutral.700` `#3d4457` |
| ring | `primary.500` `#2e9ae4` | `primary.400` `#63b3f1` |
| destructive | `danger` `#f03e61` (both modes) | same |

Note `accent` the *semantic* token (nav/menu hover surface, resolves to a
neutral) is a completely different thing from `accent` the *palette* group
(the teal brand color, `#0bb98e` family). Don't confuse
`--color-accent-500` (teal) with `--accent` (neutral hover surface) — they
resolve to unrelated colors. Same ambiguity exists for `primary`
(`--color-primary-500` = a specific ramp step, `--primary` = whichever step
the current mode aliases to).

## Dark mode contrast — corrected value, don't regress

Dark `background`/`foreground` is `#171b26` / `#dfe3ea` — recomputed
(WCAG relative-luminance formula) contrast ratio is **13.4:1**. That's the
*current, correct* state, already well past WCAG AA's 4.5:1 floor for body
text, and it's already been through one correction pass — brand.md's own
Elevation section (`brand.md:42-58`) describes moving dark `card`/
`popover`/`sidebar` from `neutral.900` to `neutral.800` specifically
because the previous background/card pairing read as flat/harsh. Recomputed
for reference, a `neutral.950` (`#0d0f16`) background against a
`neutral.50` (`#f7f8fa`) foreground — the kind of pairing this was pulled
back from — hits **18:1**, which is the "harsher than necessary" end of the
range being avoided.

**Principle to preserve**: dark-mode text contrast should be softened
toward the AA floor, not maximized. If you're about to change dark
`background` or `foreground`, recompute the ratio (formula: WCAG relative
luminance, `L = 0.2126R + 0.7152G + 0.0722B` on linearized sRGB channels;
ratio = `(Lmax+0.05)/(Lmin+0.05)`) and keep it in roughly the 10–14:1 band —
enough headroom to be safely AA, not so much it looks like pure
white-on-black. Don't reach for `neutral.950`/`neutral.50` as a dark
foreground/background pair again; that's the regression this note exists to
prevent.

Other dark-mode pairs worth knowing (recomputed, all pass AA):
card `#262b3a` vs cardForeground `#dfe3ea` → 10.96:1. Primary `#63b3f1` vs
background `#171b26` → 7.57:1. mutedForeground `#98a1b3` vs background →
6.62:1. Card vs page background itself is deliberately *low* contrast
(1.22:1, `#262b3a` vs `#171b26`) — that's a surface separation, not a text
pair, and is why `Card.vue` compensates with a real shadow instead of
relying on background contrast (see below).

## Light-mode-by-default — the actual code and reasoning

`apps/dashboard/src/layouts/DefaultLayout.vue:56-62`:

```ts
// Defaults to light regardless of OS preference — a school admin tool reads
// friendlier light-first. Once the user toggles it manually, that choice is
// remembered (useDark persists to localStorage under the hood). Storage key
// changed from vueuse's default so anyone who toggled dark earlier in this
// project's life (before light became the default) gets a clean reset
// instead of staying stuck on a stale "dark" value forever.
const isDark = useDark({ initialValue: 'light', storageKey: 'internity-color-scheme' })
const toggleDark = useToggle(isDark)
```

Two non-obvious things here if you ever touch this: (1) `initialValue:
'light'` overrides `prefers-color-scheme` — this is deliberate product
positioning (school-admin tool = friendlier light-first), not a bug to
"fix" by respecting OS preference. (2) the `storageKey` is **not** vueuse's
default (`vueuse-color-scheme`) — it was intentionally renamed to
`internity-color-scheme` specifically so old localStorage values from
before the light-default decision don't leak through. If you ever rename it
again, you'll cause the same one-time reset for whoever has the
in-between key.

Toggle UI: header sun/moon button, `DefaultLayout.vue:184-187`. Persists
across sessions once touched — it's real `localStorage`, not per-tab state.

## Elevation (from brand.md, verified in code)

`apps/dashboard/src/components/ui/card/Card.vue:17` — the actual class
string:

```
ring-foreground/10 bg-card text-card-foreground ... ring-1
shadow-sm shadow-black/5 dark:shadow-md dark:shadow-black/40
```

Confirms brand.md: light mode gets a subtle `shadow-black/5`, dark mode
needs a visibly stronger `shadow-black/40` (dark shadows need much higher
opacity to read at all against a dark background — a `shadow-black/5` in
dark mode would be invisible). If you add a new elevated surface, follow
this same light/dark shadow-opacity split rather than one shadow value for
both modes.

## Component library — shadcn-vue

Config: `apps/dashboard/components.json`.

```json
{
  "style": "reka-nova",
  "font": "inter",
  "tailwind": { "css": "src/assets/main.css", "baseColor": "neutral", "cssVariables": true, "prefix": "" },
  "iconLibrary": "lucide",
  "aliases": { "components": "@/components", "ui": "@/components/ui", "lib": "@/lib", "composables": "@/composables" }
}
```

**Gotcha**: `"font": "inter"` in this config is stale/misleading — it's the
shadcn-vue scaffold default and was never updated when the project switched
to Sora-everywhere (see brand.md Typography section and `main.css:1`).
`components.json`'s `font` field doesn't actually drive anything at runtime
here (the CLI only reads it when scaffolding new font-related boilerplate);
don't trust it as documentation of the real typeface, and don't "fix" it by
regenerating font imports from it — `main.css` is the actual source.

**Adding a new shadcn-vue component**: from `apps/dashboard/`, run the
shadcn-vue CLI (devDependency `"shadcn-vue": "^2.8.2"`,
`apps/dashboard/package.json:33`) — `pnpm dlx shadcn-vue@latest add
<component>`. It reads `components.json`, drops the new component under
`src/components/ui/<name>/` following the same per-component folder +
`index.ts` barrel pattern every existing one uses (see
`src/components/ui/card/` for the shape: `Card.vue`, `CardHeader.vue`, etc.
+ `index.ts` re-exporting them). Currently present under
`apps/dashboard/src/components/ui/`: `alert, avatar, badge, button,
calendar, card, checkbox, dialog, drawer, dropdown-menu, form, input,
label, native-select, pagination, popover, select, separator, skeleton,
sonner, table, tabs, textarea`. Check this list before adding — per
`docs/RULES.md` rule 1, if shadcn already has it, don't hand-roll a custom
component instead.

**Rule 10** (`docs/RULES.md:30`): "Warna, spacing, dan font wajib dari
`packages/design-tokens`. Jangan hardcode hex/px." — color/spacing/font
must come from the design tokens; no hardcoded hex or px. In practice this
means: reach for the Tailwind utility classes the `@theme inline` block
generates (`bg-primary`, `text-muted-foreground`, `border-border`,
`rounded-lg`, etc.) or the raw `--color-*`/semantic CSS vars, never a
literal `#0065ab` or `12px` in a component. When shadcn-vue scaffolds a new
component with its own default palette assumptions, those get overridden
through the token CSS vars (which is exactly the mechanism — shadcn-vue
components already read `--primary`, `--border`, etc. by convention, so
correctly-scaffolded components pick up this project's tokens for free;
only hand-written one-off styles are at risk of hardcoding).

## Real brand/visual assets in the repo

These are the only non-screenshot visual assets that actually exist —
useful when asked to build a document/marketing surface without inventing
generic AI-style imagery:

- `apps/dashboard/src/assets/logo.png` (33.7 KB) — the Internity logo,
  used in the sidebar header (`DefaultLayout.vue:17,122`).
- `apps/dashboard/src/assets/illustrations/online-security.svg` (9.4 KB).
- `apps/landing/public/illustrations/` — five SVGs: `career-growth.svg`
  (16.6 KB), `grading-papers.svg` (6.1 KB), `monitoring-data.svg`
  (33.4 KB), `person-search.svg` (18.2 KB), `winner.svg` (5.4 KB).

That's the complete set. No other logo variants (no dark-mode logo, no
favicon source, no icon set beyond Lucide) exist in the repo as of this
writing — don't assume e.g. a white-on-transparent logo variant exists
without checking again.

## Onboarding tour theming — `tour.css` as the reskinning pattern

`apps/dashboard/src/assets/tour.css`, imported from
`apps/dashboard/src/composables/useTour.ts:5` (`import
'@/assets/tour.css'`) — loaded alongside driver.js's own popover, not
instead of it.

Pattern demonstrated: driver.js (`"driver.js": "^1.8.0"`,
`apps/dashboard/package.json:30`) ships its own plain white/black popover
chrome. Rather than vendoring/forking driver.css, this file targets
driver.js's own class names (`.driver-popover`, `.driver-popover-title`,
`.driver-popover-footer-btn`, `.driver-popover-next-btn`,
`.driver-popover-prev-btn`, etc.) and overrides colors/radius/shadow/type
using **this app's own CSS custom properties** (`var(--popover)`,
`var(--border)`, `var(--radius)`, `var(--font-sans)`, `var(--primary)` …).
Because those vars already flip under `.dark`, `tour.css` needs no separate
dark-mode branch — it inherits dark mode for free.

The file's own header comment (`tour.css:16-18`) explicitly flags the
fragility: *"Class names verified against
`node_modules/driver.js/dist/driver.css` (driver.js@1.8.0)."* If driver.js
is ever upgraded, **re-diff `node_modules/driver.js/dist/driver.css`
against this file's selectors before assuming the reskin still applies** —
a version bump could rename or restructure the classes this file targets,
and nothing will error, it'll just silently stop reskinning (you'd fall
back to driver.js's stock white/black look with no warning).

This is the reference pattern for reskinning *any* third-party widget in
this codebase: target the library's real class names with a comment
pinning the verified version, pull all values from the design tokens'
CSS vars, and keep it a from-scratch override rather than a copy of the
vendor's stylesheet.
