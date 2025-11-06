import AllDaySection from './AllDaySection'
import HourGrid from './HourGrid'
import TimeIndicator from './TimeIndicator'
import GhostPreview from './GhostPreview'
import useWeekGridDrag from './useWeekGridDrag'
import useWeekGridResize from './useWeekGridResize'
import useWeekGridCreate from './useWeekGridCreate'
export default function WeekGrid(){ 
  const drag = useWeekGridDrag(); const resize = useWeekGridResize(); const create = useWeekGridCreate();
  return (
    <div className="border border-zinc-700 rounded p-2">
      <AllDaySection/>
      <HourGrid/>
      <TimeIndicator/>
      <GhostPreview/>
    </div>
  )
}
