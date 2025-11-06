import EnergySlider from './EnergySlider'
import MoodSlider from './MoodSlider'
import FocusSlider from './FocusSlider'
export default function ReflectionForm(){
  return (
    <div className="p-2 border border-zinc-700 rounded">
      <EnergySlider/><MoodSlider/><FocusSlider/>
    </div>
  )
}
