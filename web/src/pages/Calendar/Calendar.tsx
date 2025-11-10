import { useState } from 'react'
import Layout from '../../components/Layout'
import WeekGrid from '../../components/WeekGrid'
import MonthView from '../../components/MonthView'
import TaskSidebar from '../../components/TaskSidebar'
export default function Calendar(){
  const [view,setView]=useState<'week'|'month'>('week')
  return (
    <Layout>
      <div className="p-4 flex gap-2 border-b border-zinc-800">
        <button onClick={()=>setView('week')}>Week</button>
        <button onClick={()=>setView('month')}>Month</button>
      </div>
      <div className="p-4">{view==='week'? <WeekGrid/> : <MonthView/>}</div>
      <TaskSidebar/>
    </Layout>
  )
}
