import { readonly, ref, type Ref } from 'vue'

export interface ConfirmDialogOptions {
  message: string
  title?: string
  confirmText?: string
  cancelText?: string
  danger?: boolean
}

export interface PromptDialogOptions {
  message: string
  title?: string
  confirmText?: string
  cancelText?: string
  defaultValue?: string
  placeholder?: string
  inputType?: 'text' | 'password'
  danger?: boolean
}

type DialogRequest =
  | {
      id: number
      type: 'confirm'
      options: ConfirmDialogOptions
      resolve: (value: boolean) => void
    }
  | {
      id: number
      type: 'prompt'
      options: PromptDialogOptions
      resolve: (value: string | null) => void
    }

const activeDialog = ref<DialogRequest | null>(null)
const pendingDialogs: Array<
  | { type: 'confirm'; options: ConfirmDialogOptions; resolve: (value: boolean) => void }
  | { type: 'prompt'; options: PromptDialogOptions; resolve: (value: string | null) => void }
> = []
let dialogId = 0

function showNextDialog(): void {
  if (activeDialog.value || pendingDialogs.length === 0) return

  const next = pendingDialogs.shift()
  if (!next) return

  activeDialog.value = {
    ...next,
    id: ++dialogId
  } as DialogRequest
}

function settleDialog(value: boolean | string | null): void {
  const current = activeDialog.value
  if (!current) return

  activeDialog.value = null
  if (current.type === 'confirm') {
    current.resolve(Boolean(value))
  } else {
    current.resolve(typeof value === 'string' ? value : null)
  }

  // Let the closing transition finish before opening a queued dialog.
  queueMicrotask(showNextDialog)
}

export function useDialog() {
  function confirm(options: ConfirmDialogOptions | string): Promise<boolean> {
    const normalizedOptions = typeof options === 'string' ? { message: options } : options
    return new Promise((resolve) => {
      pendingDialogs.push({ type: 'confirm', options: normalizedOptions, resolve })
      showNextDialog()
    })
  }

  function prompt(options: PromptDialogOptions | string, defaultValue = ''): Promise<string | null> {
    const normalizedOptions =
      typeof options === 'string' ? { message: options, defaultValue } : options
    return new Promise((resolve) => {
      pendingDialogs.push({ type: 'prompt', options: normalizedOptions, resolve })
      showNextDialog()
    })
  }

  return {
    confirm,
    prompt
  }
}

export function useDialogHost(): {
  activeDialog: Readonly<Ref<DialogRequest | null>>
  resolveConfirm: (confirmed: boolean) => void
  resolvePrompt: (value: string | null) => void
} {
  return {
    activeDialog: readonly(activeDialog),
    resolveConfirm: (confirmed) => settleDialog(confirmed),
    resolvePrompt: (value) => settleDialog(value)
  }
}
