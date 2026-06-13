import { gsap } from 'gsap'

type DoneCallback = () => void
type MotionTarget = Element | Element[]

export const motionDurations = {
  fast: 0.16,
  normal: 0.26,
  expressive: 0.34,
} as const

export const motionEases = {
  enter: 'power3.out',
  expressiveEnter: 'back.out(1.35)',
  exit: 'power2.in',
  hover: 'power2.out',
} as const

export function prefersReducedMotion(): boolean {
  if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') {
    return false
  }

  return window.matchMedia('(prefers-reduced-motion: reduce)').matches
}

function getDialogContent(el: Element): Element {
  return el.querySelector('.modal-content') ?? el
}

function getSelectOptions(el: Element): Element[] {
  return Array.from(el.querySelectorAll('.select-option')).slice(0, 8)
}

export function createDialogTransitionHooks() {
  return {
    onEnter(el: Element, done: DoneCallback) {
      const content = getDialogContent(el)

      if (prefersReducedMotion()) {
        gsap.set(el, { autoAlpha: 1 })
        gsap.set(content, { autoAlpha: 1, y: 0, scale: 1 })
        done()
        return
      }

      gsap.timeline({ onComplete: done })
        .fromTo(
          el,
          { autoAlpha: 0 },
          { autoAlpha: 1, duration: motionDurations.fast, ease: motionEases.enter }
        )
        .fromTo(
          content,
          { autoAlpha: 0, y: 18, scale: 0.96 },
          {
            autoAlpha: 1,
            y: 0,
            scale: 1,
            duration: motionDurations.expressive,
            ease: motionEases.expressiveEnter,
          },
          '<'
        )
    },
    onLeave(el: Element, done: DoneCallback) {
      const content = getDialogContent(el)

      if (prefersReducedMotion()) {
        gsap.set(el, { autoAlpha: 0 })
        gsap.set(content, { autoAlpha: 0, y: 18, scale: 0.96 })
        done()
        return
      }

      gsap.timeline({ onComplete: done })
        .to(content, {
          autoAlpha: 0,
          y: 10,
          scale: 0.98,
          duration: motionDurations.fast,
          ease: motionEases.exit,
        })
        .to(
          el,
          { autoAlpha: 0, duration: motionDurations.fast, ease: motionEases.exit },
          '<'
        )
    },
  }
}

export function createToastTransitionHooks() {
  return {
    onEnter(el: Element, done: DoneCallback) {
      if (prefersReducedMotion()) {
        gsap.set(el, { autoAlpha: 1, x: 0, y: 0, scale: 1 })
        done()
        return
      }

      gsap.fromTo(
        el,
        { autoAlpha: 0, x: 18, y: -6, scale: 0.98 },
        {
          autoAlpha: 1,
          x: 0,
          y: 0,
          scale: 1,
          duration: motionDurations.normal,
          ease: motionEases.enter,
          onComplete: done,
        }
      )
    },
    onLeave(el: Element, done: DoneCallback) {
      if (prefersReducedMotion()) {
        gsap.set(el, { autoAlpha: 0, x: 18, y: -6, scale: 0.98 })
        done()
        return
      }

      gsap.to(el, {
        autoAlpha: 0,
        x: 18,
        y: -6,
        scale: 0.98,
        duration: motionDurations.fast,
        ease: motionEases.exit,
        onComplete: done,
      })
    },
    onMove(el: Element, done: DoneCallback) {
      if (prefersReducedMotion()) {
        gsap.set(el, { x: 0, y: 0 })
        done()
        return
      }

      gsap.to(el, {
        x: 0,
        y: 0,
        duration: motionDurations.fast,
        ease: motionEases.enter,
        onComplete: done,
      })
    },
  }
}

export function createSelectTransitionHooks() {
  return {
    onEnter(el: Element, done: DoneCallback) {
      const options = getSelectOptions(el)

      if (prefersReducedMotion()) {
        gsap.set(el, { autoAlpha: 1, y: 0, scaleY: 1 })
        if (options.length > 0) {
          gsap.set(options, { autoAlpha: 1, y: 0 })
        }
        done()
        return
      }

      const timeline = gsap.timeline({ onComplete: done })
      timeline.fromTo(
        el,
        { autoAlpha: 0, y: -6, scaleY: 0.96 },
        {
          autoAlpha: 1,
          y: 0,
          scaleY: 1,
          duration: motionDurations.normal,
          ease: motionEases.enter,
        }
      )

      if (options.length > 0) {
        timeline.fromTo(
          options,
          { autoAlpha: 0, y: -4 },
          {
            autoAlpha: 1,
            y: 0,
            duration: motionDurations.fast,
            ease: motionEases.enter,
            stagger: 0.018,
          },
          '-=0.12'
        )
      }
    },
    onLeave(el: Element, done: DoneCallback) {
      if (prefersReducedMotion()) {
        gsap.set(el, { autoAlpha: 0, y: -6, scaleY: 0.96 })
        done()
        return
      }

      gsap.to(el, {
        autoAlpha: 0,
        y: -6,
        scaleY: 0.96,
        duration: motionDurations.fast,
        ease: motionEases.exit,
        onComplete: done,
      })
    },
  }
}

export function animateMountedSurface(el: Element, childSelector?: string) {
  const target = childSelector ? Array.from(el.querySelectorAll(childSelector)) : el

  if (prefersReducedMotion()) {
    gsap.set(target, { autoAlpha: 1, y: 0, scale: 1 })
    return
  }

  gsap.fromTo(
    target,
    { autoAlpha: 0, y: 10, scale: 0.99 },
    {
      autoAlpha: 1,
      y: 0,
      scale: 1,
      duration: motionDurations.normal,
      ease: motionEases.enter,
      stagger: Array.isArray(target) ? 0.035 : 0,
    }
  )
}

export function animateHoverLift(el: Element, lifted: boolean) {
  gsap.killTweensOf(el)

  if (prefersReducedMotion()) {
    gsap.set(el, { y: lifted ? -3 : 0, scale: lifted ? 1.01 : 1 })
    return
  }

  gsap.to(el, {
    y: lifted ? -3 : 0,
    scale: lifted ? 1.01 : 1,
    duration: motionDurations.fast,
    ease: motionEases.hover,
    overwrite: true,
  })
}

export function clearMotion(el: MotionTarget) {
  gsap.killTweensOf(el)
}
