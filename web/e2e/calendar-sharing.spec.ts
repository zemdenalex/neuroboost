import { test, expect } from './fixtures/auth'
import { request as playwrightRequest, type APIRequestContext } from '@playwright/test'

/**
 * Two people, one calendar — the whole point of P3.
 *
 * Denis, 17.08: "их вообще можно с кем-то вместе использовать? Вроде бы кнопки
 * поделиться нет, тогда смысл теряется". This walks the path he could not:
 * create a shared calendar, invite a second person, accept, and check that the
 * second person sees the first person's event — and that they saw nothing
 * before accepting.
 *
 * 🔴 The second person is a real second account, registered by this spec
 * against staging. A test that shares a calendar with ITSELF would pass on an
 * implementation with no access control at all, which is the one thing this
 * feature must never have.
 *
 * Everything is done through the API rather than the UI. The UI path is worth
 * its own spec, but this one is about the access rule, and driving two browser
 * sessions to assert a database fact would put a lot of clicking between the
 * question and the answer.
 */

const API_BASE = process.env.E2E_API_URL ?? process.env.E2E_BASE_URL ?? 'https://dev.neuroboost.website'

interface Ctx {
  api: APIRequestContext
  userId: string
  email: string
}

/** Register a throwaway account and return an authed context for it. */
async function registerSecondPerson(label: string): Promise<Ctx> {
  const email = `e2e-${label}-${Date.now()}@example.com`
  const anon = await playwrightRequest.newContext({ baseURL: API_BASE })
  const res = await anon.post('/api/auth/register', {
    data: { email, password: "e2e-Password-1234", name: `E2E ${label}` },
    headers: { 'Content-Type': 'application/json' },
  })
  if (!res.ok()) throw new Error(`register ${label}: ${res.status()} ${await res.text()}`)
  const body = (await res.json()).data
  await anon.dispose()

  const api = await playwrightRequest.newContext({
    baseURL: API_BASE,
    extraHTTPHeaders: {
      Authorization: `Bearer ${body.token}`,
      'Content-Type': 'application/json',
    },
  })
  return { api, userId: body.user.id, email }
}

test.describe('sharing a calendar with another person', () => {
  // One viewport: this creates real rows on the shared staging database, and
  // two copies racing would invite each other's throwaway accounts.
  test.beforeEach(({}, testInfo) => {
    test.skip(testInfo.project.name !== 'desktop', 'writes to staging; one viewport only')
  })

  test('invite, accept, and the event becomes visible to both', async ({ session }) => {
    const owner = await playwrightRequest.newContext({
      baseURL: API_BASE,
      extraHTTPHeaders: {
        Authorization: `Bearer ${session.token}`,
        'Content-Type': 'application/json',
      },
    })
    const guest = await registerSecondPerson('guest')

    let calendarId: string | undefined
    let eventId: string | undefined

    try {
      const created = await owner.post('/api/calendars', {
        data: { name: `E2E Shared ${Date.now()}`, color: '#7c3aed' },
      })
      expect(created.status(), await created.text()).toBe(201)
      calendarId = (await created.json()).data.id

      // An event only the owner should see until the guest accepts.
      const start = new Date(Date.now() + 3600_000).toISOString()
      const end = new Date(Date.now() + 7200_000).toISOString()
      const ev = await owner.post('/api/events', {
        data: { title: `E2E shared event ${Date.now()}`, starts_at: start, ends_at: end, calendar_id: calendarId },
      })
      expect(ev.status(), await ev.text()).toBe(201)
      const evBody = (await ev.json()).data
      eventId = evBody.id

      const invited = await owner.post(`/api/calendars/${calendarId}/invites`, {
        data: { email: guest.email, role: 'editor' },
      })
      expect(invited.status(), await invited.text()).toBe(201)
      expect((await invited.json()).data.status).toBe('invited')

      // 🔴 Before accepting: the calendar is LISTED (that is how the invitation
      // is shown) but its events are not readable. Those are two different
      // questions and this asserts both, because an implementation that
      // conflated them would pass either one alone.
      const guestListBefore = (await (await guest.api.get('/api/calendars')).json()).data
      const pending = guestListBefore.find((c: { id: string }) => c.id === calendarId)
      expect(pending, 'the invitation is not visible to the invitee').toBeTruthy()
      expect(pending.status).toBe('invited')

      const window = `start=${new Date(Date.now() - 86400_000).toISOString()}&end=${new Date(Date.now() + 86400_000).toISOString()}`
      const before = (await (await guest.api.get(`/api/events?${window}`)).json()).data ?? []
      expect(
        before.some((e: { id: string }) => e.id === eventId),
        'an invited-but-not-accepted user can already read the calendar',
      ).toBe(false)

      // Accept.
      const accepted = await guest.api.post(`/api/calendars/${calendarId}/invitation`, {
        data: { accept: true },
      })
      expect(accepted.status(), await accepted.text()).toBe(204)

      // After accepting: the event is there. This is the positive control —
      // without it, an API that returned nothing to anyone would pass above.
      const after = (await (await guest.api.get(`/api/events?${window}`)).json()).data ?? []
      expect(
        after.some((e: { id: string }) => e.id === eventId),
        'accepting the invitation did not make the shared event visible',
      ).toBe(true)

      // An editor can write, which is what "shared" has to mean to be worth
      // anything: Denis's original pain was his girlfriend re-typing everything.
      const moved = await guest.api.patch(`/api/events/${eventId}`, {
        data: { title: 'E2E moved by the guest' },
      })
      expect(moved.status(), await moved.text()).toBe(200)
    } finally {
      // Order matters: the event references the calendar.
      if (eventId) await owner.delete(`/api/events/${eventId}`)
      if (calendarId) await owner.delete(`/api/calendars/${calendarId}`)
      await owner.dispose()
      await guest.api.dispose()
    }
  })

  test('is_shared and author_name are answered per viewer', async ({ session }) => {
    // Slice 4. The badge in the grid is only as good as the two fields behind
    // it, and both are decided per VIEWER — the same row is "mine" to one
    // member and "hers" to the other. A single-account test cannot see that
    // distinction at all: every event would be its own author's.
    const owner = await playwrightRequest.newContext({
      baseURL: API_BASE,
      extraHTTPHeaders: {
        Authorization: `Bearer ${session.token}`,
        'Content-Type': 'application/json',
      },
    })
    const guest = await registerSecondPerson('badge')

    let calendarId: string | undefined
    let byOwner: string | undefined
    let byGuest: string | undefined
    let personal: string | undefined

    const window = `start=${new Date(Date.now() - 86400_000).toISOString()}&end=${new Date(Date.now() + 86400_000).toISOString()}`
    const start = new Date(Date.now() + 3600_000).toISOString()
    const end = new Date(Date.now() + 7200_000).toISOString()

    try {
      const created = await owner.post('/api/calendars', {
        data: { name: `E2E Badge ${Date.now()}` },
      })
      expect(created.status(), await created.text()).toBe(201)
      calendarId = (await created.json()).data.id

      const invited = await owner.post(`/api/calendars/${calendarId}/invites`, {
        data: { email: guest.email, role: 'editor' },
      })
      expect(invited.status(), await invited.text()).toBe(201)
      const accepted = await guest.api.post(`/api/calendars/${calendarId}/invitation`, {
        data: { accept: true },
      })
      expect(accepted.status(), await accepted.text()).toBe(204)

      const mk = async (who: APIRequestContext, title: string, cal?: string) => {
        const res = await who.post('/api/events', {
          data: { title, starts_at: start, ends_at: end, ...(cal ? { calendar_id: cal } : {}) },
        })
        expect(res.status(), await res.text()).toBe(201)
        return (await res.json()).data.id as string
      }

      byOwner = await mk(owner, `E2E owner ${Date.now()}`, calendarId)
      byGuest = await mk(guest.api, `E2E guest ${Date.now()}`, calendarId)
      // No calendar_id: the personal one, which must never be marked shared.
      personal = await mk(owner, `E2E private ${Date.now()}`)

      const load = async (who: APIRequestContext) => {
        const rows = (await (await who.get(`/api/events?${window}`)).json()).data ?? []
        const byId: Record<string, { is_shared?: boolean; author_name?: string }> = {}
        for (const e of rows) byId[e.id] = e
        return byId
      }

      const asOwner = await load(owner)
      const asGuest = await load(guest.api)

      // The fixture has to have produced the rows, or everything below passes
      // by finding nothing.
      expect(asOwner[byOwner], 'the owner cannot see their own event').toBeTruthy()
      expect(asOwner[byGuest], 'the owner cannot see the guest event').toBeTruthy()
      expect(asGuest[byOwner], 'the guest cannot see the owner event').toBeTruthy()

      // The badge is about audience, not authorship: it is on your own events too.
      expect(asOwner[byOwner].is_shared, 'own event in a shared calendar is not badged').toBe(true)
      expect(asOwner[byGuest].is_shared).toBe(true)
      expect(asOwner[personal].is_shared, 'a personal event is badged as shared').toBeFalsy()

      // The half a stored column could never do.
      expect(asOwner[byOwner].author_name, 'the owner was named as the author to themselves').toBeFalsy()
      expect(asOwner[byGuest].author_name, 'the owner was not told who wrote the guest event').toBeTruthy()
      expect(asGuest[byGuest].author_name, 'the guest was named as the author to themselves').toBeFalsy()
      expect(asGuest[byOwner].author_name, 'the guest was not told who wrote the owner event').toBeTruthy()

      // And the two viewers were told DIFFERENT names, not the same one twice.
      expect(asOwner[byGuest].author_name).not.toBe(asGuest[byOwner].author_name)

      // A task lands in the shared calendar too — the other half of slice 4,
      // which used to be pinned to the personal calendar no matter what.
      const task = await guest.api.post('/api/tasks', {
        data: { title: `E2E shared task ${Date.now()}`, calendar_id: calendarId },
      })
      expect(task.status(), await task.text()).toBe(201)
      const taskBody = (await task.json()).data
      expect(taskBody.calendar_id, 'the task ignored the calendar it was given').toBe(calendarId)
      await guest.api.delete(`/api/tasks/${taskBody.id}`)
    } finally {
      for (const id of [byOwner, personal]) if (id) await owner.delete(`/api/events/${id}`)
      if (byGuest) await guest.api.delete(`/api/events/${byGuest}`)
      if (calendarId) await owner.delete(`/api/calendars/${calendarId}`)
      await owner.dispose()
      await guest.api.dispose()
    }
  })

  test('a personal calendar cannot be shared', async ({ session }) => {
    const owner = await playwrightRequest.newContext({
      baseURL: API_BASE,
      extraHTTPHeaders: {
        Authorization: `Bearer ${session.token}`,
        'Content-Type': 'application/json',
      },
    })
    const guest = await registerSecondPerson('nosy')

    try {
      const list = (await (await owner.get('/api/calendars')).json()).data
      const personal = list.find((c: { kind: string }) => c.kind === 'personal')
      expect(personal, 'no personal calendar to test with').toBeTruthy()

      const res = await owner.post(`/api/calendars/${personal.id}/invites`, {
        data: { email: guest.email, role: 'viewer' },
      })
      // 409, not a silent success: the personal calendar holds everything a
      // person never deliberately filed anywhere else.
      expect(res.status(), await res.text()).toBe(409)

      const guestList = (await (await guest.api.get('/api/calendars')).json()).data
      expect(
        guestList.some((c: { id: string }) => c.id === personal.id),
        'the personal calendar leaked to another account',
      ).toBe(false)
    } finally {
      await owner.dispose()
      await guest.api.dispose()
    }
  })

  test('an invite link works once and then does not', async ({ session }) => {
    const owner = await playwrightRequest.newContext({
      baseURL: API_BASE,
      extraHTTPHeaders: {
        Authorization: `Bearer ${session.token}`,
        'Content-Type': 'application/json',
      },
    })
    const first = await registerSecondPerson('link-first')
    const second = await registerSecondPerson('link-second')

    let calendarId: string | undefined
    try {
      const created = await owner.post('/api/calendars', {
        data: { name: `E2E Link ${Date.now()}`, color: '#2563eb' },
      })
      expect(created.status(), await created.text()).toBe(201)
      calendarId = (await created.json()).data.id

      const link = await owner.post(`/api/calendars/${calendarId}/invite-links`, {
        data: { role: 'editor' },
      })
      expect(link.status(), await link.text()).toBe(201)
      const token = (await link.json()).data.token
      expect(token, 'no token in the response').toBeTruthy()

      const used = await first.api.post('/api/calendars/invite-links/accept', { data: { token } })
      expect(used.status(), await used.text()).toBe(200)
      // Accepting a link grants ACTIVE membership straight away — opening the
      // link IS the acceptance, unlike an email invitation.
      expect((await used.json()).data.status).toBe('active')

      const reused = await second.api.post('/api/calendars/invite-links/accept', { data: { token } })
      expect(reused.status(), 'a spent link still worked').toBe(410)

      const secondList = (await (await second.api.get('/api/calendars')).json()).data
      expect(
        secondList.some((c: { id: string }) => c.id === calendarId),
        'the second person got in with a spent link',
      ).toBe(false)
    } finally {
      if (calendarId) await owner.delete(`/api/calendars/${calendarId}`)
      await owner.dispose()
      await first.api.dispose()
      await second.api.dispose()
    }
  })
})
