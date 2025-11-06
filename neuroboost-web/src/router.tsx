import { HashRouter, Routes, Route } from 'react-router-dom'
import Home from './pages/Home'
import Login from './pages/Login'
import Calendar from './pages/Calendar'
import Tasks from './pages/Tasks'
import Planning from './pages/Planning'
import Reflections from './pages/Reflections'
import Settings from './pages/Settings'
import Tools from './pages/Tools'
export default function Router() {
  return (
    <HashRouter>
      <Routes>
        <Route path="/" element={<Home />} />
        <Route path="/login" element={<Login />} />
        <Route path="/calendar" element={<Calendar />} />
        <Route path="/tasks" element={<Tasks />} />
        <Route path="/planning" element={<Planning />} />
        <Route path="/reflections" element={<Reflections />} />
        <Route path="/settings" element={<Settings />} />
        <Route path="/tools" element={<Tools />} />
      </Routes>
    </HashRouter>
  )
}
