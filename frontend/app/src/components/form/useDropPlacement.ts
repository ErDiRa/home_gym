import { ref } from 'vue'

// The option lists are at most max-h-60 (15rem) tall.
const OPTIONS_HEIGHT = 240

// Where a field's option list opens: below it, or above it (openUp) when it doesn't fit below but
// there is more room above (e.g. fields near the bottom of the page). Bind `field` to the element
// the list hangs from and call place() as the list opens.
export function useDropPlacement() {
  const field = ref<HTMLElement | null>(null)
  const openUp = ref(false)

  function place() {
    const rect = field.value?.getBoundingClientRect()
    if (!rect) return
    const below = window.innerHeight - rect.bottom
    openUp.value = below < OPTIONS_HEIGHT && rect.top > below
  }

  return { field, openUp, place }
}
