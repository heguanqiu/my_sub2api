import { beforeEach, describe, expect, it, vi } from 'vitest'

const gsapTimeline = vi.fn()
const gsapTo = vi.fn()
const gsapFromTo = vi.fn()
const gsapSet = vi.fn()
const gsapKillTweensOf = vi.fn()

vi.mock('gsap', () => ({
  gsap: {
    timeline: gsapTimeline,
    to: gsapTo,
    fromTo: gsapFromTo,
    set: gsapSet,
    killTweensOf: gsapKillTweensOf,
  },
}))

const makeTimeline = () => ({
  fromTo: vi.fn().mockReturnThis(),
  to: vi.fn().mockReturnThis(),
})

const mockMatchMedia = (matches: boolean) => {
  Object.defineProperty(window, 'matchMedia', {
    configurable: true,
    writable: true,
    value: vi.fn().mockImplementation((query: string) => ({
      matches: query.includes('prefers-reduced-motion') ? matches : false,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })),
  })
}

describe('useGsapMotion', () => {
  beforeEach(() => {
    vi.resetModules()
    vi.clearAllMocks()
    mockMatchMedia(false)
    gsapTimeline.mockImplementation(makeTimeline)
    gsapTo.mockImplementation((_target, vars) => {
      vars?.onComplete?.()
      return { kill: vi.fn() }
    })
    gsapFromTo.mockImplementation((_target, _fromVars, toVars) => {
      toVars?.onComplete?.()
      return { kill: vi.fn() }
    })
  })

  it('detects reduced motion from matchMedia', async () => {
    mockMatchMedia(true)
    const { prefersReducedMotion } = await import('../useGsapMotion')

    expect(prefersReducedMotion()).toBe(true)
  })

  it('runs dialog enter through a GSAP timeline', async () => {
    const timeline = makeTimeline()
    gsapTimeline.mockReturnValue(timeline)
    const { createDialogTransitionHooks } = await import('../useGsapMotion')
    const overlay = document.createElement('div')
    const panel = document.createElement('div')
    panel.className = 'modal-content'
    overlay.appendChild(panel)
    const done = vi.fn()

    createDialogTransitionHooks().onEnter(overlay, done)

    expect(gsapTimeline).toHaveBeenCalledWith({ onComplete: done })
    expect(timeline.fromTo).toHaveBeenCalledWith(
      overlay,
      { autoAlpha: 0 },
      expect.objectContaining({ autoAlpha: 1 })
    )
    expect(timeline.fromTo).toHaveBeenCalledWith(
      panel,
      expect.objectContaining({ y: 18, scale: 0.96 }),
      expect.objectContaining({ y: 0, scale: 1 }),
      '<'
    )
  })

  it('sets final state immediately when reduced motion is enabled', async () => {
    mockMatchMedia(true)
    const { createSelectTransitionHooks } = await import('../useGsapMotion')
    const dropdown = document.createElement('div')
    const done = vi.fn()

    createSelectTransitionHooks().onEnter(dropdown, done)

    expect(gsapSet).toHaveBeenCalledWith(
      dropdown,
      expect.objectContaining({ autoAlpha: 1, y: 0, scaleY: 1 })
    )
    expect(done).toHaveBeenCalled()
  })

  it('kills active tweens for hover feedback before starting a new hover tween', async () => {
    const { animateHoverLift } = await import('../useGsapMotion')
    const element = document.createElement('div')

    animateHoverLift(element, true)

    expect(gsapKillTweensOf).toHaveBeenCalledWith(element)
    expect(gsapTo).toHaveBeenCalledWith(
      element,
      expect.objectContaining({ y: -3, scale: 1.01, overwrite: true })
    )
  })
})
