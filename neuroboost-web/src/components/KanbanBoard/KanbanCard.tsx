export default function KanbanCard({card}:{card:{id:string;title:string}}){return <div className='p-2 border border-zinc-700 rounded text-sm'>{card.title}</div>}
