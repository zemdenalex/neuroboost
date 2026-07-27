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
 * Global capture shortcut (Ctrl+K by default, configurable in Settings).
 *
 * Skips the shortcut while focus is in a text field so it can never eat what
 * the user is typing — including the quick-add row itself.
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
      if (target && (TEXT_ENTRY.has(target.tagName) || target.isContentEditable)) return
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
