import { ref } from 'vue'

/** Default key used when a guard tracks a single control (no per-item id). */
const DEFAULT_KEY = '__default__'

/**
 * Re-entrancy guard for async actions, keyed by an optional item id.
 *
 * Prevents a control from firing the same async action twice while a previous
 * invocation is still in flight (anti double-click, bug #295). The first call
 * for a given key runs `fn`; any call for the same key issued before it settles
 * is ignored (returns `undefined`) without invoking `fn`. The key is released
 * in a `finally`, so the action is re-triggerable after both success AND error
 * (RG4).
 *
 * For a single control (e.g. one Delete button), call `run(fn)` with no key.
 * For a list where each row deletes its own item, pass the item id so a second
 * click on the SAME row is ignored while other rows stay clickable.
 */
export function useInFlightGuard() {
  const inFlight = ref<Set<string>>(new Set())

  /** Whether an action for `key` (or the default control) is currently running. */
  function isBusy(key: string = DEFAULT_KEY): boolean {
    return inFlight.value.has(key)
  }

  /**
   * Run `fn` guarded by `key`. Returns the resolved value of `fn`, or
   * `undefined` if the call was ignored because `key` is already in flight.
   * Rejections from `fn` propagate to the caller after the key is released.
   */
  async function run<T>(fn: () => Promise<T>, key: string = DEFAULT_KEY): Promise<T | undefined> {
    if (inFlight.value.has(key)) {
      return undefined
    }
    const next = new Set(inFlight.value)
    next.add(key)
    inFlight.value = next
    try {
      return await fn()
    } finally {
      const after = new Set(inFlight.value)
      after.delete(key)
      inFlight.value = after
    }
  }

  return { isBusy, run }
}
