import { useState } from 'react'
import Layout from '../../components/Layout'
import DreamsView from '../../components/DreamsView'
import GoalsView from '../../components/GoalsView'
import ProjectsView from '../../components/ProjectsView'
import OpportunitiesView from '../../components/OpportunitiesView'
import NeedsView from '../../components/NeedsView'
import GraphView from '../../components/GraphView'
import TimelineView from '../../components/TimelineView'
export default function Planning(){
  const [activeView, setActiveView] = useState<'dreams'|'goals'|'projects'|'opportunities'|'needs'|'graph'|'timeline'>('dreams')
  return (
    <Layout>
      <div className="flex gap-2 p-4 border-b border-zinc-700">
        <button onClick={()=>setActiveView('dreams')}>Dreams</button>
        <button onClick={()=>setActiveView('goals')}>Goals</button>
        <button onClick={()=>setActiveView('projects')}>Projects</button>
        <button onClick={()=>setActiveView('opportunities')}>Opportunities</button>
        <button onClick={()=>setActiveView('needs')}>Needs</button>
        <button onClick={()=>setActiveView('graph')}>Graph</button>
        <button onClick={()=>setActiveView('timeline')}>Timeline</button>
      </div>
      <div className="p-4">
        {activeView==='dreams' && <DreamsView/>}
        {activeView==='goals' && <GoalsView/>}
        {activeView==='projects' && <ProjectsView/>}
        {activeView==='opportunities' && <OpportunitiesView/>}
        {activeView==='needs' && <NeedsView/>}
        {activeView==='graph' && <GraphView/>}
        {activeView==='timeline' && <TimelineView/>}
      </div>
    </Layout>
  )
}
