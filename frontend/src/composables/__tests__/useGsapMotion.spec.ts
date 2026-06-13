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

const makeDialog = () => {
  const overlay = document.createElement('div')
  const panel = document.createElement('div')
  panel.className = 'modal-content'
  overlay.appendChild(panel)

  return { overlay, panel }
}

const makeSelectDropdown = (optionCount = 10) => {
  const dropdown = document.createElement('div')
  const options = Array.from({ length: optionCount }, () => {
    const option = document.createElement('div')
    option.className = 'select-option'
    dropdown.appendChild(option)
    return option
  })

  return { dropdown, options }
}

const callTimelineComplete = () => {
  const timelineOptions = gsapTimeline.mock.calls[0]?.[0]
  timelineOptions?.onComplete?.()
}

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

const targetIncludes = (actual: unknown, target: Element) => {
  return Array.isArray(actual) ? actual.includes(target) : actual === target
}

const killOrderFor = (target: Element) => {
  const callIndex = gsapKillTweensOf.mock.calls.findIndex(([actual]) =>
    targetIncludes(actual, target)
  )

  expect(callIndex).toBeGreaterThanOrEqual(0)
  return gsapKillTweensOf.mock.invocationCallOrder[callIndex]
}

const expectKilledBefore = (target: Element, beforeOrder: number) => {
  expect(killOrderFor(target)).toBeLessThan(beforeOrder)
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
    const { overlay, panel } = makeDialog()
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

  it('kills dialog overlay and content before starting enter animation', async () => {
    const timeline = makeTimeline()
    gsapTimeline.mockReturnValue(timeline)
    const { createDialogTransitionHooks } = await import('../useGsapMotion')
    const { overlay, panel } = makeDialog()
    const done = vi.fn()

    createDialogTransitionHooks().onEnter(overlay, done)

    const timelineOrder = gsapTimeline.mock.invocationCallOrder[0]
    expectKilledBefore(overlay, timelineOrder)
    expectKilledBefore(panel, timelineOrder)
    expect(done).not.toHaveBeenCalled()
  })

  it('runs dialog leave through a GSAP timeline and completes once', async () => {
    const timeline = makeTimeline()
    gsapTimeline.mockReturnValue(timeline)
    const { createDialogTransitionHooks } = await import('../useGsapMotion')
    const { overlay, panel } = makeDialog()
    const done = vi.fn()

    createDialogTransitionHooks().onLeave(overlay, done)
    callTimelineComplete()

    expect(gsapTimeline).toHaveBeenCalledWith({ onComplete: done })
    expect(timeline.to).toHaveBeenCalledWith(panel, expect.objectContaining({ autoAlpha: 0 }))
    expect(timeline.to).toHaveBeenCalledWith(
      overlay,
      expect.objectContaining({ autoAlpha: 0 }),
      '<'
    )
    expect(done).toHaveBeenCalledTimes(1)
  })

  it('kills dialog overlay and content before starting leave animation', async () => {
    const timeline = makeTimeline()
    gsapTimeline.mockReturnValue(timeline)
    const { createDialogTransitionHooks } = await import('../useGsapMotion')
    const { overlay, panel } = makeDialog()
    const done = vi.fn()

    createDialogTransitionHooks().onLeave(overlay, done)

    const timelineOrder = gsapTimeline.mock.invocationCallOrder[0]
    expectKilledBefore(overlay, timelineOrder)
    expectKilledBefore(panel, timelineOrder)
    expect(done).not.toHaveBeenCalled()
  })

  it('sets dialog leave final state and completes once when reduced motion is enabled', async () => {
    mockMatchMedia(true)
    const { createDialogTransitionHooks } = await import('../useGsapMotion')
    const { overlay, panel } = makeDialog()
    const done = vi.fn()

    createDialogTransitionHooks().onLeave(overlay, done)

    const firstSetOrder = gsapSet.mock.invocationCallOrder[0]
    expectKilledBefore(overlay, firstSetOrder)
    expectKilledBefore(panel, firstSetOrder)
    expect(gsapSet).toHaveBeenCalledWith(overlay, expect.objectContaining({ autoAlpha: 0 }))
    expect(gsapSet).toHaveBeenCalledWith(
      panel,
      expect.objectContaining({ autoAlpha: 0, y: 18, scale: 0.96 })
    )
    expect(done).toHaveBeenCalledTimes(1)
  })

  it('runs toast enter and leave through GSAP and completes each once', async () => {
    const { createToastTransitionHooks } = await import('../useGsapMotion')
    const toast = document.createElement('div')
    const enterDone = vi.fn()
    const leaveDone = vi.fn()

    const hooks = createToastTransitionHooks()
    hooks.onEnter(toast, enterDone)
    hooks.onLeave(toast, leaveDone)

    expect(gsapFromTo).toHaveBeenCalledWith(
      toast,
      expect.objectContaining({ autoAlpha: 0 }),
      expect.objectContaining({ autoAlpha: 1, onComplete: enterDone })
    )
    expect(gsapTo).toHaveBeenCalledWith(
      toast,
      expect.objectContaining({ autoAlpha: 0, onComplete: leaveDone })
    )
    expect(enterDone).toHaveBeenCalledTimes(1)
    expect(leaveDone).toHaveBeenCalledTimes(1)
  })

  it('kills toast before starting interrupted enter and leave tweens', async () => {
    gsapFromTo.mockImplementation(() => ({ kill: vi.fn() }))
    gsapTo.mockImplementation(() => ({ kill: vi.fn() }))
    const { createToastTransitionHooks } = await import('../useGsapMotion')
    const toast = document.createElement('div')
    const enterDone = vi.fn()
    const leaveDone = vi.fn()

    const hooks = createToastTransitionHooks()
    hooks.onEnter(toast, enterDone)
    hooks.onLeave(toast, leaveDone)

    expect(enterDone).not.toHaveBeenCalled()
    expect(gsapKillTweensOf.mock.calls[0]?.[0]).toBe(toast)
    expect(gsapKillTweensOf.mock.invocationCallOrder[0]).toBeLessThan(
      gsapFromTo.mock.invocationCallOrder[0]
    )
    expect(gsapKillTweensOf.mock.calls[1]?.[0]).toBe(toast)
    expect(gsapKillTweensOf.mock.invocationCallOrder[1]).toBeLessThan(
      gsapTo.mock.invocationCallOrder[0]
    )
  })

  it('sets toast enter and leave final states when reduced motion is enabled', async () => {
    mockMatchMedia(true)
    const { createToastTransitionHooks } = await import('../useGsapMotion')
    const toast = document.createElement('div')
    const enterDone = vi.fn()
    const leaveDone = vi.fn()

    const hooks = createToastTransitionHooks()
    hooks.onEnter(toast, enterDone)
    hooks.onLeave(toast, leaveDone)

    expect(gsapKillTweensOf.mock.calls[0]?.[0]).toBe(toast)
    expect(gsapKillTweensOf.mock.invocationCallOrder[0]).toBeLessThan(
      gsapSet.mock.invocationCallOrder[0]
    )
    expect(gsapKillTweensOf.mock.calls[1]?.[0]).toBe(toast)
    expect(gsapKillTweensOf.mock.invocationCallOrder[1]).toBeLessThan(
      gsapSet.mock.invocationCallOrder[1]
    )
    expect(gsapSet).toHaveBeenCalledWith(
      toast,
      expect.objectContaining({ autoAlpha: 1, x: 0, y: 0, scale: 1 })
    )
    expect(gsapSet).toHaveBeenCalledWith(
      toast,
      expect.objectContaining({ autoAlpha: 0, x: 18, y: -6, scale: 0.98 })
    )
    expect(enterDone).toHaveBeenCalledTimes(1)
    expect(leaveDone).toHaveBeenCalledTimes(1)
  })

  it('sets final state immediately when reduced motion is enabled', async () => {
    mockMatchMedia(true)
    const { createSelectTransitionHooks } = await import('../useGsapMotion')
    const { dropdown, options } = makeSelectDropdown(3)
    const done = vi.fn()

    createSelectTransitionHooks().onEnter(dropdown, done)

    const firstSetOrder = gsapSet.mock.invocationCallOrder[0]
    expectKilledBefore(dropdown, firstSetOrder)
    for (const option of options) {
      expectKilledBefore(option, firstSetOrder)
    }
    expect(gsapSet).toHaveBeenCalledWith(
      dropdown,
      expect.objectContaining({ autoAlpha: 1, y: 0, scaleY: 1 })
    )
    expect(done).toHaveBeenCalled()
  })

  it('animates only the first eight select options on enter', async () => {
    const timeline = makeTimeline()
    gsapTimeline.mockReturnValue(timeline)
    const { createSelectTransitionHooks } = await import('../useGsapMotion')
    const { dropdown, options } = makeSelectDropdown()
    const done = vi.fn()

    createSelectTransitionHooks().onEnter(dropdown, done)
    callTimelineComplete()

    const timelineOrder = gsapTimeline.mock.invocationCallOrder[0]
    expectKilledBefore(dropdown, timelineOrder)
    for (const option of options.slice(0, 8)) {
      expectKilledBefore(option, timelineOrder)
    }
    expect(gsapTimeline).toHaveBeenCalledWith({ onComplete: done })
    expect(timeline.fromTo).toHaveBeenCalledTimes(2)
    expect(timeline.fromTo.mock.calls[1][0]).toEqual(options.slice(0, 8))
    expect(timeline.fromTo.mock.calls[1][0]).toHaveLength(8)
    expect(done).toHaveBeenCalledTimes(1)
  })

  it('runs select leave through GSAP and completes once', async () => {
    const { createSelectTransitionHooks } = await import('../useGsapMotion')
    const { dropdown, options } = makeSelectDropdown(4)
    const done = vi.fn()

    createSelectTransitionHooks().onLeave(dropdown, done)

    const toOrder = gsapTo.mock.invocationCallOrder[0]
    expectKilledBefore(dropdown, toOrder)
    for (const option of options) {
      expectKilledBefore(option, toOrder)
    }
    expect(gsapTo).toHaveBeenCalledWith(
      dropdown,
      expect.objectContaining({ autoAlpha: 0, y: -6, scaleY: 0.96, onComplete: done })
    )
    expect(done).toHaveBeenCalledTimes(1)
  })

  it('sets select leave final state and completes once when reduced motion is enabled', async () => {
    mockMatchMedia(true)
    const { createSelectTransitionHooks } = await import('../useGsapMotion')
    const { dropdown, options } = makeSelectDropdown(4)
    const done = vi.fn()

    createSelectTransitionHooks().onLeave(dropdown, done)

    const firstSetOrder = gsapSet.mock.invocationCallOrder[0]
    expectKilledBefore(dropdown, firstSetOrder)
    for (const option of options) {
      expectKilledBefore(option, firstSetOrder)
    }
    expect(gsapSet).toHaveBeenCalledWith(
      dropdown,
      expect.objectContaining({ autoAlpha: 0, y: -6, scaleY: 0.96 })
    )
    expect(done).toHaveBeenCalledTimes(1)
  })

  it('sets mounted surface final state when reduced motion is enabled', async () => {
    mockMatchMedia(true)
    const { animateMountedSurface } = await import('../useGsapMotion')
    const element = document.createElement('div')

    animateMountedSurface(element)

    expectKilledBefore(element, gsapSet.mock.invocationCallOrder[0])
    expect(gsapSet).toHaveBeenCalledWith(
      element,
      expect.objectContaining({ autoAlpha: 1, y: 0, scale: 1 })
    )
  })

  it('kills mounted surface root before animating the root target', async () => {
    const { animateMountedSurface } = await import('../useGsapMotion')
    const element = document.createElement('div')

    animateMountedSurface(element)

    expectKilledBefore(element, gsapFromTo.mock.invocationCallOrder[0])
    expect(gsapFromTo).toHaveBeenCalledWith(
      element,
      expect.objectContaining({ autoAlpha: 0, y: 10, scale: 0.99 }),
      expect.objectContaining({ autoAlpha: 1, y: 0, scale: 1 })
    )
  })

  it('kills mounted surface child targets before animating a child selector array', async () => {
    const { animateMountedSurface } = await import('../useGsapMotion')
    const element = document.createElement('div')
    const children = Array.from({ length: 3 }, () => {
      const child = document.createElement('div')
      child.className = 'surface-item'
      element.appendChild(child)
      return child
    })

    animateMountedSurface(element, '.surface-item')

    expect(gsapFromTo.mock.calls[0]?.[0]).toEqual(children)
    const fromToOrder = gsapFromTo.mock.invocationCallOrder[0]
    for (const child of children) {
      expectKilledBefore(child, fromToOrder)
    }
  })

  it('kills mounted surface child targets before reduced-motion set', async () => {
    mockMatchMedia(true)
    const { animateMountedSurface } = await import('../useGsapMotion')
    const element = document.createElement('div')
    const children = Array.from({ length: 2 }, () => {
      const child = document.createElement('div')
      child.className = 'surface-item'
      element.appendChild(child)
      return child
    })

    animateMountedSurface(element, '.surface-item')

    expect(gsapSet.mock.calls[0]?.[0]).toEqual(children)
    const setOrder = gsapSet.mock.invocationCallOrder[0]
    for (const child of children) {
      expectKilledBefore(child, setOrder)
    }
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

  it('sets hover feedback immediately without tweening when reduced motion is enabled', async () => {
    mockMatchMedia(true)
    const { animateHoverLift } = await import('../useGsapMotion')
    const element = document.createElement('div')

    animateHoverLift(element, true)

    expect(gsapKillTweensOf).toHaveBeenCalledWith(element)
    expect(gsapSet).toHaveBeenCalledWith(
      element,
      expect.objectContaining({ y: 0, scale: 1 })
    )
    expect(gsapTo).not.toHaveBeenCalled()
  })

  it('clears active motion tweens', async () => {
    const { clearMotion } = await import('../useGsapMotion')
    const element = document.createElement('div')

    clearMotion(element)

    expect(gsapKillTweensOf).toHaveBeenCalledWith(element)
  })
})
