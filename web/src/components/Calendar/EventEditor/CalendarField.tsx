import { CalendarPicker } from '../../Calendars/CalendarPicker'

interface Props {
  /** Selected calendar id, or '' for "my personal calendar". */
  value: string
  onChange: (calendarId: string) => void
  disabled?: boolean
}

/**
 * The event editor's calendar field.
 *
 * The picker itself moved to components/Calendars/CalendarPicker when tasks
 * needed the same control. Copying it would have been the third duplicated
 * stack in this frontend, after the two NbEvent declarations and the two task
 * API stacks — each of which has already cost a defect that only showed up in
 * the copy nobody opened.
 */
export function CalendarField({ value, onChange, disabled = false }: Props) {
  return (
    <CalendarPicker id="event-calendar" value={value} onChange={onChange} disabled={disabled} />
  )
}
