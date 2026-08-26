import { driver, type DriveStep } from 'driver.js'
import 'driver.js/dist/driver.css'

/**
 * A thin wrapper around driver.js scoped per named tour (one per role — see
 * src/tours/*.ts). "Seen" state is a per-viewer localStorage convenience
 * (never authoritative data — losing it just means the tour can replay), so
 * it's fine that it doesn't sync across devices/browsers.
 */
export function useTour(tourKey: string, steps: DriveStep[]) {
  const storageKey = `tour-dismissed:${tourKey}`

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
      overlayColor: 'rgba(15, 15, 20, 0.55)',
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
