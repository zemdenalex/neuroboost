import { Link } from 'react-router-dom'
export default function VerticalSidebar() {
  return (
    <aside className="fixed left-0 top-0 w-56 h-screen bg-zinc-900 border-r border-zinc-800 p-4">
      <nav className="flex flex-col gap-2 text-sm">
        <Link to="/">Home</Link>
        <Link to="/calendar">Calendar</Link>
        <Link to="/tasks">Tasks</Link>
        <Link to="/planning">Planning</Link>
        <Link to="/reflections">Reflections</Link>
        <Link to="/settings">Settings</Link>
        <Link to="/tools">Tools</Link>
      </nav>
    </aside>
  )
}
