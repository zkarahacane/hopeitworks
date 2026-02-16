import { onMounted, onUnmounted } from 'vue'

export function useKeyboard(bindings: Record<string, () => void>): void {
  function handler(event: KeyboardEvent) {
    const target = event.target as HTMLElement | null
    if (
      target &&
      (target.tagName === 'INPUT' ||
        target.tagName === 'TEXTAREA' ||
        target.isContentEditable)
    ) {
      return
    }

    const fn = bindings[event.key]
    if (fn) {
      fn()
    }
  }

  onMounted(() => {
    window.addEventListener('keydown', handler)
  })

  onUnmounted(() => {
    window.removeEventListener('keydown', handler)
  })
}
