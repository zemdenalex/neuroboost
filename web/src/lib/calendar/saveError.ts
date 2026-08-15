import { ApiError } from '../../api/client'
import { errorMessage } from '../errorMessage'

/**
 * What to tell someone whose event would not save.
 *
 * The editor used to print the raw `error.message`, which for a refused
 * calendar move reads "Request failed" — true, unhelpful, and impossible to
 * act on. The three codes below are all decisions the API made deliberately,
 * so each deserves a sentence saying what to do instead.
 *
 * Anything unrecognised keeps the old behaviour rather than being swallowed:
 * a message we cannot explain is still better shown than hidden.
 */
export function describeSaveError(err: unknown): string {
  if (err instanceof ApiError) {
    switch (err.code) {
      case 'CALENDAR_SCOPE_SERIES':
        return 'Moving an event to another calendar applies to the whole series. Save again and choose "All occurrences", or move it back before saving this one.'
      case 'CALENDAR_READ_ONLY':
        return 'You can read that calendar but not put events in it.'
      case 'CALENDAR_NOT_FOUND':
        return 'That calendar no longer exists. Reopen the event and pick another one.'
    }
  }
  return 'Failed to save event: ' + errorMessage(err, 'Unknown error')
}
