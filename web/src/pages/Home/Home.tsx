import { useAuthContext } from '../../contexts/AuthContext'
import { Landing } from './Landing'
import { Dashboard } from './Dashboard'

export default function Home() {
  const { isAuthenticated, loading } = useAuthContext()
  if (loading) return null
  return isAuthenticated ? <Dashboard /> : <Landing />
}
