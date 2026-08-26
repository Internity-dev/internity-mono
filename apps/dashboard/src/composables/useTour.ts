import { driver, type DriveStep } from 'driver.js'
import 'driver.js/dist/driver.css'
// Themed override for driver.js's popover chrome — see src/assets/tour.css.
// Imported after driver.css so its rules win in the cascade.
import '@/assets/tour.css'

/**
 * A thin wrapper around driver.js scoped per named tour (one per role — see
 * src/tours/*.ts). "Seen" state is a per-viewer localStorage convenience
 * (never authoritative data — losing it just means the tour can replay), so
 * it's fine that it doesn't sync across devices/browsers.
 *
 * The dismissal flag is keyed by tourKey *and* userId so a shared browser
 * profile (e.g. a school lab machine) doesn't let the first account to ever
 * log in permanently dismiss the tour for every other account that follows.
 */
export function useTour(tourKey: string, userId: string, steps: DriveStep[]) {
  const storageKey = `tour-dismissed:${tourKey}:${userId}`

  function hasSeenTour(): boolean {
    try {
      return localStorage.getItem(storageKey) === '1'
    } catch {
      return false
    }
  }

  function markSeen() {
    try {
      localStorage.setItem(storageKey, '1')
    } catch {
      // private-browsing / storage blocked — the tour will just replay next time, which is fine
    }
  }

  function start() {
    const driverObj = driver({
      showProgress: true,
      allowClose: true,
      // --color-neutral-950 from @internity/design-tokens (packages/design-tokens/tokens.json)
      // — same dark tone the app itself uses, rather than an ad-hoc black.
      overlayColor: '#0d0f16',
      overlayOpacity: 0.6,
      steps,
      onDestroyStarted: () => {
        markSeen()
        driverObj.destroy()
      },
    })
    driverObj.drive()
  }

  function startIfFirstVisit() {
    if (!hasSeenTour()) start()
  }

  return { start, startIfFirstVisit, hasSeenTour }
}
