import { useEffect, useMemo, useState } from 'react'

const API = import.meta.env.VITE_API_BASE || 'http://localhost:8080'

export default function App() {
  const [email, setEmail] = useState('demo@zenfl.local')
  const [password, setPassword] = useState('demo1234')
  const [token, setToken] = useState(localStorage.getItem('token') || '')
  const [items, setItems] = useState([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  const loggedIn = useMemo(() => !!token, [token])

  useEffect(() => {
    if (!token) return
    localStorage.setItem('token', token)
    void loadJobs(token)
  }, [token])

  async function login(e) {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      const res = await fetch(`${API}/api/auth/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password })
      })
      if (!res.ok) throw new Error('Invalid credentials')
      const data = await res.json()
      setToken(data.token || '')
    } catch (err) {
      setError(err.message || 'Login failed')
    } finally {
      setLoading(false)
    }
  }

  async function loadJobs(authToken = token) {
    setLoading(true)
    setError('')
    try {
      const res = await fetch(`${API}/api/jobs?limit=100`, {
        headers: { Authorization: `Bearer ${authToken}` }
      })
      if (!res.ok) throw new Error('Failed to load jobs')
      const data = await res.json()
      setItems(data.items || [])
    } catch (err) {
      setError(err.message || 'Failed to load')
    } finally {
      setLoading(false)
    }
  }

  function logout() {
    localStorage.removeItem('token')
    setToken('')
    setItems([])
  }

  return (
    <div className="page">
      <header className="topbar">
        <h1>Zenfl Job Board</h1>
        {loggedIn && <button onClick={logout}>Logout</button>}
      </header>

      {!loggedIn && (
        <form className="card" onSubmit={login}>
          <h2>Demo Login</h2>
          <label>Email<input value={email} onChange={(e) => setEmail(e.target.value)} /></label>
          <label>Password<input type="password" value={password} onChange={(e) => setPassword(e.target.value)} /></label>
          <button disabled={loading}>{loading ? 'Signing in...' : 'Sign in'}</button>
          <p className="hint">Default: demo@zenfl.local / demo1234</p>
        </form>
      )}

      {loggedIn && (
        <section className="card">
          <div className="sectionHead">
            <h2>Latest Jobs</h2>
            <button onClick={() => loadJobs()} disabled={loading}>{loading ? 'Loading...' : 'Refresh'}</button>
          </div>
          {error && <p className="error">{error}</p>}
          <div className="jobs">
            {items.map((j) => (
              <article key={j.id || `${j.telegram_msg_id}-${j.received_at}`} className="job">
                <p className="meta">#{j.telegram_msg_id} · {new Date(j.received_at).toLocaleString()}</p>
                <pre>{j.text}</pre>
                {j.job_link && <a href={j.job_link} target="_blank" rel="noreferrer">Open Upwork Job</a>}
              </article>
            ))}
          </div>
        </section>
      )}

      {error && !loggedIn && <p className="error">{error}</p>}
    </div>
  )
}
