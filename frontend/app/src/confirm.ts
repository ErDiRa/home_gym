import { shallowRef } from 'vue'

export interface ConfirmOptions {
  title: string
  message?: string
  /** Label of the confirming button, e.g. "Delete". */
  confirmLabel?: string
  cancelLabel?: string
  /** Destructive: the confirming button is red. */
  danger?: boolean
}

interface Pending extends ConfirmOptions {
  resolve: (ok: boolean) => void
}

// The question shown by ConfirmDialog (mounted once in App.vue), null while none is open.
export const pendingConfirm = shallowRef<Pending | null>(null)

// Asks the user to confirm in a modal, in place of the browser's confirm(); resolves true
// when confirmed, false when cancelled or dismissed. Opening another one cancels the first.
export function confirmDialog(options: ConfirmOptions): Promise<boolean> {
  pendingConfirm.value?.resolve(false)
  return new Promise((resolve) => {
    pendingConfirm.value = { ...options, resolve }
  })
}

// Called by ConfirmDialog when the user answers.
export function answerConfirm(ok: boolean) {
  pendingConfirm.value?.resolve(ok)
  pendingConfirm.value = null
}
