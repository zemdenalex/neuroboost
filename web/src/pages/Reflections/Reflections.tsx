import Layout from '../../components/Layout'
import ReflectionsList from '../../components/ReflectionsList'
import ReflectionForm from '../../components/ReflectionForm'
export default function Reflections(){
  return (
    <Layout>
      <div className="p-4 grid gap-4">
        <ReflectionsList/>
        <ReflectionForm/>
      </div>
    </Layout>
  )
}
