export default function Tools(){
  return (
    <div className="p-8">
        <h1 className="text-2xl font-bold">Productivity Tools & Methodologies</h1>
        <p className="text-zinc-400">Frameworks and tools to help you prioritize and organize</p>
        <div className="grid grid-cols-2 gap-6 mt-8">
          <div className="bg-zinc-800 p-6 rounded border border-zinc-700"><h3>Eisenhower Matrix</h3><p>Prioritize by urgency and importance</p><button>Open Matrix</button></div>
          <div className="bg-zinc-800 p-6 rounded border border-zinc-700"><h3>Kanban Board</h3><p>Visualize workflow stages</p><button>Open Kanban</button></div>
          <div className="bg-zinc-800 p-6 rounded border border-zinc-700"><h3>Time Blocking Calculator</h3><p>Calculate realistic time budgets</p><button>Open Calculator</button></div>
          <div className="bg-zinc-800 p-6 rounded border border-zinc-700"><h3>More Coming Soon</h3><p>GTD, Pomodoro, etc.</p></div>
        </div>
    </div>
  )
}
