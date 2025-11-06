import GraphControls from './GraphControls'
import GraphNode from './GraphNode'
import GraphEdge from './GraphEdge'
export default function GraphView(){
  return (
    <div className="border border-zinc-700 rounded p-2">
      <GraphControls/>
      <svg className="w-full h-[480px] bg-zinc-900 rounded">
        {/* TODO: render nodes & edges */}
        <GraphEdge edge={{}}/>
        <GraphNode node={{}}/>
      </svg>
    </div>
  )
}
