import { shallowRef } from 'vue'

export interface ConfirmOptions {
  title: string
  message?: string
  /** Label of the confirming button, e.g. "Delete". */
  confirmLabel?: string
  cancelLabel?: string
  /** Destructive: the confirming button is red. */
  danger?: boolean
  /** A checkbox under the message, unchecked at first, e.g. "Delete its trainings too". */
  option?: string
}

// The answer to a question with an option: whether it was confirmed, and the checkbox.
export interface ConfirmAnswer {
  ok: boolean
  option: boolean
}

interface Pending extends ConfirmOptions {
  resolve: (answer: ConfirmAnswer) => void
}

// The question shown by ConfirmDialog (mounted once in App.vue), null while none is open.
export const pendingConfirm = shallowRef<Pending | null>(null)

// Asks the user to confirm in a modal, in place of the browser's confirm(); resolves true
// when confirmed, false when cancelled or dismissed. Opening another one cancels the first.
export async function confirmDialog(options: ConfirmOptions): Promise<boolean> {
  return (await confirmWithOption(options)).ok
}

// confirmDialog for a question with a checkbox (options.option); also resolves its state.
export function confirmWithOption(options: ConfirmOptions): Promise<ConfirmAnswer> {
  pendingConfirm.value?.resolve({ ok: false, option: false })
  return new Promise((resolve) => {
    pendingConfirm.value = { ...options, resolve }
  })
}

// Called by ConfirmDialog when the user answers.
export function answerConfirm(ok: boolean, option = false) {
  pendingConfirm.value?.resolve({ ok, option: ok && option })
  pendingConfirm.value = null
}
