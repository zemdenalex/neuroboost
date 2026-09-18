import { useState } from 'react'
import { Check, Loader2, MessageCircle, Mail } from 'lucide-react'
import { setCredentials, submitLinkCode } from '../../api/auth'
import { useAuthContext } from '../../contexts/AuthContext'
import { errorMessage } from '../../lib/errorMessage'
import { linkingState, normalizeLinkCode } from '../../lib/auth/linking'

/**
 * Account linking on the profile page, v0.4.11.5.
 *
 * Two halves, and which one a person sees depends on what their account is
 * missing — never both questions at once. An account is one of:
 *
 *   - email only  → offer «Привязать Telegram» (the code from the bot)
 *   - Telegram only → offer «Задать email и пароль»
 *   - both        → say so and offer nothing
 *
 * 🔴 Submitting the code does NOT link anything. It opens a request that the
 * bot must confirm — Denis, 17.09: «bot asking if you're the one who's
 * connecting the profile». So the success message says «подтверди в боте», not
 * «привязано»: announcing a merge that has not happened is worse than saying
 * nothing, because the person stops watching for the question.
 */
export function AccountLinking() {
  const { user, refreshUser } = useAuthContext()

  const state = linkingState(user)

  if (state === 'linked') {
    return (
      <section className="rounded-lg border border-zinc-800 p-4">
        <h2 className="mb-2 flex items-center gap-2 text-sm font-semibold text-zinc-200">
          <Check className="h-4 w-4 text-green-400" />
          Аккаунт связан
        </h2>
        <p className="text-sm text-zinc-400">
          Вход работает и по email, и через Telegram.
        </p>
      </section>
    )
  }

  return state === 'offer-telegram'
    ? <LinkTelegram onLinked={refreshUser} />
    : <SetCredentials onSaved={refreshUser} />
}

function LinkTelegram({ onLinked }: { onLinked: () => Promise<void> }) {
  const [code, setCode] = useState('')
  const [state, setState] = useState<'idle' | 'sending' | 'pending'>('idle')
  const [error, setError] = useState('')

  // Six digits, and the leading zero matters: «048215» pasted as a number would
  // arrive as 48215 and never match.
  const digits = normalizeLinkCode(code)

  async function submit() {
    setError('')
    setState('sending')
    try {
      await submitLinkCode(digits)
      setState('pending')
      await onLinked()
    } catch (err) {
      setError(errorMessage(err, 'Код не подошёл'))
      setState('idle')
    }
  }

  if (state === 'pending') {
    return (
      <section className="rounded-lg border border-zinc-800 p-4">
        <h2 className="mb-2 flex items-center gap-2 text-sm font-semibold text-zinc-200">
          <MessageCircle className="h-4 w-4" />
          Подтверди в боте
        </h2>
        <p className="text-sm text-zinc-400">
          Бот спросил, ты ли это, и какой аккаунт оставить. Пока ты не ответишь там,
          ничего не изменится.
        </p>
      </section>
    )
  }

  return (
    <section className="rounded-lg border border-zinc-800 p-4">
      <h2 className="mb-2 flex items-center gap-2 text-sm font-semibold text-zinc-200">
        <MessageCircle className="h-4 w-4" />
        Привязать Telegram
      </h2>
      <p className="mb-3 text-sm text-zinc-400">
        В боте: ⚙️ Настройки → 🔗 Аккаунт на сайте → 🔢 Привязать сайт. Введи код сюда.
      </p>
      <div className="flex flex-wrap gap-2">
        <input
          value={digits}
          onChange={(e) => setCode(e.target.value)}
          inputMode="numeric"
          autoComplete="one-time-code"
          placeholder="000000"
          className="w-32 rounded-md border border-zinc-700 bg-zinc-900 px-3 py-2 font-mono tracking-widest text-zinc-100"
        />
        <button
          type="button"
          disabled={digits.length !== 6 || state === 'sending'}
          onClick={submit}
          className="rounded-md bg-blue-600 px-4 py-2 text-sm text-white disabled:opacity-40"
        >
          {state === 'sending' ? <Loader2 className="h-4 w-4 animate-spin" /> : 'Привязать'}
        </button>
      </div>
      {error && <p className="mt-2 text-sm text-red-400">{error}</p>}
    </section>
  )
}

function SetCredentials({ onSaved }: { onSaved: () => Promise<void> }) {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [state, setState] = useState<'idle' | 'saving'>('idle')
  const [error, setError] = useState('')

  // The same two rules the API applies. Checked here as well so the answer is
  // immediate — but the API is what decides, because only it knows whether the
  // address is taken.
  const ready = email.includes('@') && password.length >= 8

  async function submit() {
    setError('')
    setState('saving')
    try {
      await setCredentials(email.trim().toLowerCase(), password)
      await onSaved()
    } catch (err) {
      setError(errorMessage(err, 'Не получилось сохранить'))
    } finally {
      setState('idle')
    }
  }

  return (
    <section className="rounded-lg border border-zinc-800 p-4">
      <h2 className="mb-2 flex items-center gap-2 text-sm font-semibold text-zinc-200">
        <Mail className="h-4 w-4" />
        Вход без Telegram
      </h2>
      <p className="mb-3 text-sm text-zinc-400">
        Сейчас в этот аккаунт можно попасть только через бота. Задай email и пароль,
        чтобы входить обычным способом.
      </p>
      <div className="flex flex-col gap-2 sm:flex-row">
        <input
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          type="email"
          autoComplete="email"
          placeholder="you@example.com"
          className="min-w-0 flex-1 rounded-md border border-zinc-700 bg-zinc-900 px-3 py-2 text-zinc-100"
        />
        <input
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          type="password"
          autoComplete="new-password"
          placeholder="Пароль, 8+ символов"
          className="min-w-0 flex-1 rounded-md border border-zinc-700 bg-zinc-900 px-3 py-2 text-zinc-100"
        />
        <button
          type="button"
          disabled={!ready || state === 'saving'}
          onClick={submit}
          className="rounded-md bg-blue-600 px-4 py-2 text-sm text-white disabled:opacity-40"
        >
          {state === 'saving' ? <Loader2 className="h-4 w-4 animate-spin" /> : 'Сохранить'}
        </button>
      </div>
      {error && <p className="mt-2 text-sm text-red-400">{error}</p>}
    </section>
  )
}

export default AccountLinking
