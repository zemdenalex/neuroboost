import { useCallback, useEffect, useState } from 'react'
import { useAuthContext } from '../contexts/AuthContext'
import { resolveQuickTaskSettings } from '../lib/quickTask/settings'

/** Fields where a bare shortcut would steal the user's typing. */
const TEXT_ENTRY = new Set(['INPUT', 'TEXTAREA', 'SELECT'])

interface Binding {
  ctrl: boolean
  alt: boolean
  shift: boolean
  key: string
}

/**
 * Parse a stored binding like "Ctrl+K" or "Alt+Shift+N".
 * Returns null for anything unparseable, which simply disables the shortcut
 * rather than binding something surprising.
 */
const MODIFIER_NAMES = new Set(['ctrl', 'control', 'alt', 'shift', 'meta'])

export function parseBinding(binding: string): Binding | null {
  // Split without dropping empties first: "Ctrl+" must fail rather than
  // collapsing to the modifier itself and binding the bare Ctrl key.
  const parts = binding.split('+').map(part => part.trim())
  const key = parts.pop()
  if (!key || MODIFIER_NAMES.has(key.toLowerCase())) return null
  const mods = parts.filter(Boolean).map(part => part.toLowerCase())
  return {
    ctrl: mods.includes('ctrl') || mods.includes('control'),
    alt: mods.includes('alt'),
    shift: mods.includes('shift'),
    key: key.toLowerCase(),
  }
}

/**
 * Whether this binding must stand down while the user is typing.
 *
 * Only bare bindings do. A shortcut carrying Ctrl or Alt is not a keystroke
 * anyone is trying to type, so it has nothing to steal — and blanket-skipping
 * every text field made the default Ctrl+K look broken on /tasks, which
 * auto-focuses the quick-add row and therefore always has focus in an input.
 *
 * Shift alone does not count as a modifier here: Shift+N is just "N".
 */
export function shouldSkipInTextField(binding: Binding): boolean {
  return !binding.ctrl && !binding.alt
}

/**
 * Global capture shortcut (Ctrl+K by default, configurable in Settings).
 */
export function useGlobalQuickAdd() {
  const { user } = useAuthContext()
  const [open, setOpen] = useState(false)
  const binding = parseBinding(resolveQuickTaskSettings(user?.settings).keys.global_capture)

  const close = useCallback(() => setOpen(false), [])

  useEffect(() => {
    if (!binding) return
    const onKey = (e: KeyboardEvent) => {
      const target = e.target as HTMLElement | null
      const inText = !!target && (TEXT_ENTRY.has(target.tagName) || target.isContentEditable)
      if (inText && shouldSkipInTextField(binding)) return
      if (e.ctrlKey !== binding.ctrl || e.altKey !== binding.alt || e.shiftKey !== binding.shift) return
      if (e.key.toLowerCase() !== binding.key) return
      e.preventDefault()
      setOpen(true)
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [binding?.ctrl, binding?.alt, binding?.shift, binding?.key])

  return { open, close }
}
