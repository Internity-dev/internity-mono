import { driver } from 'driver.js'
import { menuInfo } from '@/tours/menuInfo'

// Progressive disclosure for the sidebar: instead of front-loading every
// menu into one long tour (the onboard-skill's tours are capped at 3-7
// steps), only the critical-path items are in the role's core tour — every
// other menu gets a single one-time spotlight the first time a user
// actually lands on that page. Same storage pattern as useTour.ts (a
// per-key localStorage flag), just keyed per route instead of per role.
const HINT_PREFIX = 'menu-hint-seen:'

function hasSeenHint(path: string): boolean {
  try {
    return localStorage.getItem(HINT_PREFIX + path) === '1'
  } catch {
    return false
  }
}

function markHintSeen(path: string) {
  try {
    localStorage.setItem(HINT_PREFIX + path, '1')
  } catch {
    // private-browsing / storage blocked — the hint will just show again next time, which is fine
  }
}

// Called once, right before a role's core tour is offered, so the tour's
// own steps aren't immediately re-explained by a hint the moment the user
// navigates to one of them.
export function markHintsSeenFor(paths: string[]) {
  paths.forEach(markHintSeen)
}

export function showMenuHintIfFirstVisit(path: string) {
  const info = menuInfo[path]
  if (!info || hasSeenHint(path)) return
  if (document.body.classList.contains('driver-active')) return
  const el = document.querySelector(info.selector)
  if (!el) return

  markHintSeen(path)
  driver({
    showProgress: false,
    allowClose: true,
    overlayColor: '#0d0f16',
    overlayOpacity: 0.6,
    steps: [{ element: info.selector, popover: { title: info.title, description: info.description } }],
  }).drive()
}
