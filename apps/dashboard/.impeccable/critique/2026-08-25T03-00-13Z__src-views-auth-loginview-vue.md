---
target: Login/auth flow (LoginView + Register/Forgot/Reset)
total_score: 19
max_score: 40
na_heuristics: 
p0_count: 1
p1_count: 2
timestamp: 2026-08-25T03-00-13Z
slug: src-views-auth-loginview-vue
---
Method: dual-agent (A: general-purpose design-review · B: general-purpose detector+evidence)

## Design Health Score

| # | Heuristic | Score | Key Issue |
|---|-----------|-------|-----------|
| 1 | Visibility of System Status | 2 | Login has a spinner; Register/Forgot/Reset only swap button text, no spinner |
| 2 | Match System / Real World | 1 | 100% English copy on an Indonesian SMK product with no i18n layer |
| 3 | User Control and Freedom | 2 | No "resend link" path when a reset token is invalid |
| 4 | Consistency and Standards | 1 | LoginView skips the shared Card wrapper the other 3 auth views use |
| 5 | Error Prevention | 3 | Solid zod schemas; invite_code only checked non-empty |
| 6 | Recognition Rather Than Recall | 2 | Password rules only surface after a failed attempt, no upfront hint |
| 7 | Flexibility and Efficiency | 2 | Post-login redirect preserved (good); no autofocus or remember-me |
| 8 | Aesthetic and Minimalist Design | 3 | Clean on-brand panel, undercut by the Card inconsistency |
| 9 | Error Recovery | 2 | Zod errors are clear; server 401/429 is toast-only, no retry-time guidance |
| 10 | Help and Documentation | 1 | No help/contact affordance anywhere in the auth flow |
| **Total** | | **19/40** | **Poor** |

## Design Specificity Verdict

**LLM assessment**: This reads as generic shadcn-vue + vee-validate auth boilerplate wearing Internity's logo. The only SMK-specific signal is the coordinator-issued invite-code copy on Register — and even that is in English for an Indonesian vocational-school audience, with no i18n layer anywhere in `src`.

**Deterministic scan**: `detect.mjs --json apps/dashboard/src/views/auth` → exit 0, zero findings. Confirmed genuine (not a silent skip) by re-running against a nonexistent path for contrast and against a single file directly. The antipattern detector doesn't catch monolingual copy or missing ARIA wiring — those came from source-level reading, not the deterministic pass.

**Visual overlays**: Not available. No browser automation tool was exposed this session, so no live-render `[Human]` overlay could be injected. This report substitutes direct source reading (SFC templates, script, Tailwind/shadcn classes, design tokens) for what a browser pass would show. Treat layout/visual claims here as inferred from markup, not confirmed on a rendered page.

## Overall Impression

Functionally solid (zod validation, disabled-during-submit buttons, non-enumerating forgot-password copy) but visibly unfinished as a *system*: one of four auth screens breaks the shared Card layout, only one of four shows a loading spinner, and the accessible `FormControl`/`aria-invalid` wiring the design system already ships is never turned on in this flow. The single biggest opportunity is wiring the four auth views to the same shared primitives they already have access to, and deciding whether this product ships in Indonesian.

## What's Working

- `ForgotPasswordView.vue` — enumeration-safe confirmation copy is genuine security/UX craft, not a default.
- `http.ts` — login/register 401s are deliberately excluded from the forced-logout redirect, avoiding a confusing loop after a bad password.
- All 4 submit buttons correctly disable and swap label text during submission, guarding against double-submit.
- Zero hardcoded colors; every color in the auth flow is a theme token.

## Priority Issues

**[P0] English-only copy on an Indonesian SMK product**
- Why it matters: The stated audience is Indonesian vocational-school students and staff, many with low digital literacy per the product's own domain framing. A monolingual English front door is a comprehension barrier before the product is even used.
- Fix: Localize all UI strings and toast copy in the auth flow (and ideally the app shell) to Indonesian, or add an i18n layer if bilingual support is the goal.
- Suggested command: `$impeccable clarify`

**[P1] No persistent error state for server-side failures**
- Why it matters: 401/403/429 responses are toast-only and ephemeral; a stressed user mid-login can miss the message entirely and be left staring at an unchanged form with no explanation.
- Fix: Surface server errors into the existing (currently unused) `Alert` component near the form, in addition to or instead of the toast. Add attempts-remaining/retry-after messaging for the Redis rate limiter specifically.
- Suggested command: `$impeccable harden`

**[P1] Accessible form-control wiring exists but is never turned on**
- Why it matters: The shared `FormControl` component already wires `aria-invalid`/`aria-describedby`, and `Input.vue` already ships `aria-invalid:*` Tailwind styling, but all 4 auth views hand-roll `label + Input + <p>` without binding `aria-invalid` or `aria-describedby`. Screen-reader users get no signal a field failed.
- Fix: Bind `:aria-invalid="!!errors.field"` and `aria-describedby` on every auth input, or switch these 4 views to the shared `FormControl` component they're already paying the maintenance cost of keeping.
- Suggested command: `$impeccable audit` (verify fix) then `$impeccable polish`

**[P2] Drifted layout and loading-state treatment across the 4 auth screens**
- Why it matters: Register/Forgot/Reset use the shared `Card`/`CardHeader`/`CardTitle` shell; Login uses a bare hand-rolled div instead. Only Login shows a spinner icon during submit; the other three do text-only swaps. Small, but it reads as unfinished rather than designed.
- Fix: Move Login onto the shared Card shell; add the spinner icon to Register/Forgot/Reset's loading state.
- Suggested command: `$impeccable polish`

**[P2] Reset-password token isn't validated until submit**
- Why it matters: A user who lands on `/reset-password` with a missing/expired token only learns the link is broken after filling out the entire form.
- Fix: Check `route.query.token` on mount and show a clear "this link is invalid or expired" state immediately, with a path back to requesting a new one.
- Suggested command: `$impeccable harden`

## Persona Red Flags

**Jordan (First-Timer, SMK student, low digital literacy)**: Hits `/register` with no invite code and finds no "don't have one?" escape hatch — dead end. A wrong password produces a toast he may not read in time; retrying into the 5-attempt rate limit shows "Too many requests. Slow down" with no ETA, reading as a scold rather than help. On his phone (the primary device for this persona), the marketing/brand panel is hidden below `lg:`, leaving a bare, context-free form.

**Sam (Accessibility-Dependent, screen reader/keyboard)**: Labels are correctly paired via `for`/`id` throughout, which is real credit — but every inline error `<p>` is unlinked (`aria-invalid`/`aria-describedby` never bound), so a screen reader gives no indication a field failed validation. Submit failure produces no additional live-region announcement beyond a toast that may not be reliably announced either.

## Minor Observations

- "Sign in" (LoginView copy) vs. "Log in" (buttons and cross-links elsewhere) used interchangeably — pick one term.
- `errorMessage()` is duplicated verbatim between `RegisterView.vue` and `ResetPasswordView.vue` instead of shared.
- `RegisterView.vue`'s invite-code input has a cosmetic `uppercase` class that doesn't affect the submitted value.
- No password-visibility toggle on any of the 5 password inputs across the flow.
- No autofocus on any form's first field.
- Numbered token-scale colors (`--color-primary-800`, `--color-accent-400`) used directly in `GuestLayout`'s hero panel don't flip in dark mode the way the semantic tokens do, since they're only defined in `:root`, not re-defined under `.dark`.

## Questions to Consider

- Every string in this flow is English — was that a deliberate scope decision for this take-home, or should Indonesian localization be treated as in-scope?
- The design system already ships an accessible `FormControl` primitive — why do all four auth screens quietly opt out of it, and is that worth fixing now or tracking separately?
- The backend enforces a real 5-attempt Redis lockout — should the UI turn that into a visible, reassuring countdown instead of an unexplained wall?
