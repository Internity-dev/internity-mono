import { describe, it, expect, beforeEach, vi } from 'vitest'
import type { DriveStep } from 'driver.js'

interface DriverConfigArg {
  steps: DriveStep[]
  showProgress: boolean
  allowClose: boolean
  onDestroyStarted: () => void
}

const driveMock = vi.fn()
const destroyMock = vi.fn()
const driverFactory = vi.fn((_config: DriverConfigArg) => ({ drive: driveMock, destroy: destroyMock }))

vi.mock('driver.js', () => ({
  driver: (config: DriverConfigArg) => driverFactory(config),
}))
vi.mock('driver.js/dist/driver.css', () => ({}))

// Imported after the mocks so useTour picks up the mocked driver.js.
const { useTour } = await import('./useTour')

const steps: DriveStep[] = [{ element: '#a', popover: { title: 'Step 1' } }]

describe('useTour', () => {
  beforeEach(() => {
    localStorage.clear()
    driveMock.mockClear()
    destroyMock.mockClear()
    driverFactory.mockClear()
  })

  it('hasSeenTour is false before the tour has ever been dismissed', () => {
    const { hasSeenTour } = useTour('onboarding', steps)
    expect(hasSeenTour()).toBe(false)
  })

  it('markSeen persists a per-tour dismissal flag that hasSeenTour then reports', () => {
    const tour = useTour('onboarding', steps)
    expect(tour.hasSeenTour()).toBe(false)

    // start() marks the tour seen only when driver.js reports it was destroyed
    // (see below); markSeen itself is private, so we exercise it through start().
    tour.start()
    const config = driverFactory.mock.calls[0]![0]
    config.onDestroyStarted()

    expect(tour.hasSeenTour()).toBe(true)
    expect(localStorage.getItem('tour-dismissed:onboarding')).toBe('1')
  })

  it('scopes the dismissal flag per tourKey', () => {
    const studentTour = useTour('student', steps)
    const adminTour = useTour('admin', steps)

    studentTour.start()
    const studentConfig = driverFactory.mock.calls[0]![0]
    studentConfig.onDestroyStarted()

    expect(studentTour.hasSeenTour()).toBe(true)
    expect(adminTour.hasSeenTour()).toBe(false)
  })

  it('start() configures driver.js with the given steps and immediately drives it', () => {
    const tour = useTour('onboarding', steps)
    tour.start()

    expect(driverFactory).toHaveBeenCalledTimes(1)
    const config = driverFactory.mock.calls[0]![0]
    expect(config.steps).toBe(steps)
    expect(config.showProgress).toBe(true)
    expect(config.allowClose).toBe(true)
    expect(driveMock).toHaveBeenCalledTimes(1)
  })

  it('start() destroys the driver instance once onDestroyStarted fires', () => {
    const tour = useTour('onboarding', steps)
    tour.start()

    const config = driverFactory.mock.calls[0]![0]
    config.onDestroyStarted()

    expect(destroyMock).toHaveBeenCalledTimes(1)
  })

  it('startIfFirstVisit() starts the tour when it has never been seen', () => {
    const tour = useTour('onboarding', steps)
    tour.startIfFirstVisit()

    expect(driverFactory).toHaveBeenCalledTimes(1)
    expect(driveMock).toHaveBeenCalledTimes(1)
  })

  it('startIfFirstVisit() does nothing once the tour has already been dismissed', () => {
    const tour = useTour('onboarding', steps)
    tour.start()
    const config = driverFactory.mock.calls[0]![0]
    config.onDestroyStarted()
    driverFactory.mockClear()
    driveMock.mockClear()

    tour.startIfFirstVisit()

    expect(driverFactory).not.toHaveBeenCalled()
    expect(driveMock).not.toHaveBeenCalled()
  })

  it('hasSeenTour degrades to false when localStorage throws (private browsing)', () => {
    const getItemSpy = vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
      throw new Error('storage blocked')
    })

    const tour = useTour('onboarding', steps)
    expect(tour.hasSeenTour()).toBe(false)

    getItemSpy.mockRestore()
  })

  it('markSeen swallows storage errors instead of throwing (via start -> onDestroyStarted)', () => {
    const setItemSpy = vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
      throw new Error('storage blocked')
    })

    const tour = useTour('onboarding', steps)
    tour.start()
    const config = driverFactory.mock.calls[0]![0]

    expect(() => config.onDestroyStarted()).not.toThrow()
    expect(destroyMock).toHaveBeenCalledTimes(1)

    setItemSpy.mockRestore()
  })
})
