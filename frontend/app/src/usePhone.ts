import { onBeforeUnmount, ref, type Ref } from 'vue'

// Phone widths, below Tailwind's md: the tracker's screens; the dashboard shows less there.
export const PHONE_QUERY = '(max-width: 767px)'

export const isPhone = () => window.matchMedia(PHONE_QUERY).matches

// Whether the screen has phone width, kept up to date while the component lives.
export function usePhone(): Ref<boolean> {
  const query = window.matchMedia(PHONE_QUERY)
  const phone = ref(query.matches)
  const update = (e: MediaQueryListEvent) => (phone.value = e.matches)
  query.addEventListener('change', update)
  onBeforeUnmount(() => query.removeEventListener('change', update))
  return phone
}
