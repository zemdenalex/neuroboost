import { describe, it, expect } from 'vitest'
import { safeNextPath } from './nextPath'

/**
 * The redirect target after login, taken from the URL.
 *
 * 🔴 This is a security control, so the tests that matter are the refusals.
 * Used as given, `?next=` is an open redirect: a link that looks like ours
 * bounces the visitor to somebody else's login page, carrying our domain in the
 * referrer. It exists for calendar invite links, where losing the destination
 * costs a single-use token that lives two hours.
 */
describe('safeNextPath', () => {
  it('keeps a same-site path', () => {
    expect(safeNextPath('/i/abc123')).toBe('/i/abc123')
    expect(safeNextPath('/calendar')).toBe('/calendar')
    expect(safeNextPath('/settings?tab=calendars')).toBe('/settings?tab=calendars')
  })

  it('decodes the form our own code writes', () => {
    // AcceptInvite builds this with encodeURIComponent.
    expect(safeNextPath(encodeURIComponent('/i/abc123'))).toBe('/i/abc123')
  })

  it('refuses another origin', () => {
    expect(safeNextPath('https://evil.example/login')).toBe('/home')
    expect(safeNextPath('http://evil.example')).toBe('/home')
  })

  it('refuses a protocol-relative URL', () => {
    // Browsers read //host as another origin, and it starts with a slash —
    // which is exactly why "starts with /" alone is not the rule.
    expect(safeNextPath('//evil.example/login')).toBe('/home')
    expect(safeNextPath(encodeURIComponent('//evil.example'))).toBe('/home')
  })

  it('refuses a backslash escape', () => {
    // Some browsers normalise /\host to //host.
    expect(safeNextPath('/\\evil.example')).toBe('/home')
  })

  it('refuses a scheme that is not http', () => {
    expect(safeNextPath('javascript:alert(1)')).toBe('/home')
    expect(safeNextPath('data:text/html,<script>')).toBe('/home')
  })

  it('falls back on absent, empty and undecodable input', () => {
    expect(safeNextPath(null)).toBe('/home')
    expect(safeNextPath(undefined)).toBe('/home')
    expect(safeNextPath('')).toBe('/home')
    // A lone % is not valid percent-encoding; decodeURIComponent throws.
    expect(safeNextPath('/broken%')).toBe('/home')
  })

  it('honours a caller-supplied fallback', () => {
    expect(safeNextPath('https://evil.example', '/calendar')).toBe('/calendar')
  })
})
