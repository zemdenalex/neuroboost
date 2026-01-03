import MatrixQuadrant from './MatrixQuadrant'
export default function EisenhowerMatrix(){
  return (
    <div className="grid grid-cols-2 gap-2">
      <MatrixQuadrant title="Do" description="Urgent & Important"/>
      <MatrixQuadrant title="Plan" description="Not Urgent & Important"/>
      <MatrixQuadrant title="Delegate" description="Urgent & Not Important"/>
      <MatrixQuadrant title="Eliminate" description="Not Urgent & Not Important"/>
    </div>
  )
}
