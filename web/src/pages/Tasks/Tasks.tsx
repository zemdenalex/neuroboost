import Layout from '../../components/Layout'
import TaskList from '../../components/TaskList'
import TaskEditor from '../../components/TaskEditor'
export default function Tasks(){
  return (
    <Layout>
      <div className="p-4">
        <TaskList/>
        <div className="mt-4"><TaskEditor/></div>
      </div>
    </Layout>
  )
}
