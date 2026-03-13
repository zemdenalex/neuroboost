import ReflectionsList from '../../components/ReflectionsList'
import ReflectionForm from '../../components/ReflectionForm'
export default function Reflections(){
  return (
    <div className="p-4 grid gap-4">
      <ReflectionsList/>
      <ReflectionForm/>
    </div>
  )
}
