import { Link } from 'react-router-dom'
export default function HorizontalHeader() {
  return (
    <header className="fixed top-0 left-0 right-0 bg-zinc-900 border-b border-zinc-800 z-50">
      <nav className="px-4 py-3 flex gap-4 text-sm">
        <Link to="/">Home</Link>
        <Link to="/calendar">Calendar</Link>
        <Link to="/tasks">Tasks</Link>
        <Link to="/planning">Planning</Link>
        <Link to="/reflections">Reflections</Link>
        <Link to="/settings">Settings</Link>
        <Link to="/tools">Tools</Link>
      </nav>
    </header>
  )
}
