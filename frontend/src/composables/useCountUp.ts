import { ref, watch, onBeforeUnmount, unref, type Ref, type MaybeRefOrGetter, toValue } from 'vue'

/** Options for the count-up animation. */
export interface UseCountUpOptions {
  /** Animation duration in ms toward a new target. Default 600. */
  durationMs?: number
  /** Initial displayed value. Default 0 (or the initial target if provided). */
  initial?: number
  /** Easing applied to progress t∈[0,1]. Default easeOutCubic. */
  easing?: (t: number) => number
  /**
   * If true, the value may animate downward when the target drops. Default
   * false — costs/durations only ever climb, so a lower target snaps instantly
   * (avoids a misleading backwards count for live tickers).
   */
  allowDecrease?: boolean
}

const easeOutCubic = (t: number) => 1 - Math.pow(1 - t, 3)

/**
 * Animates a displayed number upward toward a reactive `target`.
 *
 * Pure: drives the tween with requestAnimationFrame (no external lib). Designed
 * for SSE-driven live tickers — cost ($) and elapsed (s) — where the target is
 * updated by a store and the UI should smoothly count up to it.
 *
 * Returns a readonly-ish `current` ref. Cleans up its frame on unmount.
 *
 * Testable without rAF: pass a target and call no timers — the first watch tick
 * schedules a frame; tests can stub `requestAnimationFrame`/`performance.now`
 * to step the tween deterministically.
 */
export function useCountUp(
  target: MaybeRefOrGetter<number>,
  options: UseCountUpOptions = {},
): { current: Ref<number> } {
  const {
    durationMs = 600,
    easing = easeOutCubic,
    allowDecrease = false,
  } = options

  const initial = options.initial ?? toValue(target) ?? 0
  const current = ref<number>(initial)

  let rafId: number | null = null
  let from = current.value
  let to = current.value
  let startTime = 0

  const now = () =>
    typeof performance !== 'undefined' && performance.now
      ? performance.now()
      : Date.now()

  function cancel() {
    if (rafId !== null && typeof cancelAnimationFrame !== 'undefined') {
      cancelAnimationFrame(rafId)
    }
    rafId = null
  }

  function step() {
    const elapsed = now() - startTime
    const t = durationMs <= 0 ? 1 : Math.min(elapsed / durationMs, 1)
    const eased = easing(t)
    current.value = from + (to - from) * eased
    if (t < 1) {
      rafId = requestAnimationFrame(step)
    } else {
      current.value = to
      rafId = null
    }
  }

  function animateTo(next: number) {
    cancel()
    // Guard against NaN / non-finite targets.
    if (!Number.isFinite(next)) return

    if (!allowDecrease && next < current.value) {
      // Target dropped (e.g. ticker reset for a new run) — snap, don't rewind.
      current.value = next
      return
    }

    if (next === current.value || durationMs <= 0) {
      current.value = next
      return
    }

    if (typeof requestAnimationFrame === 'undefined') {
      // Non-browser env without rAF — snap to target.
      current.value = next
      return
    }

    from = current.value
    to = next
    startTime = now()
    rafId = requestAnimationFrame(step)
  }

  watch(
    () => toValue(target),
    (next) => animateTo(unref(next)),
    { immediate: true },
  )

  onBeforeUnmount(cancel)

  return { current }
}
