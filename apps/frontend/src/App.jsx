import { useEffect, useState } from 'react'

const API = import.meta.env.VITE_API_BASE || 'http://localhost:8080'

const defaultFilters = {
  q: '',
  onlyUS: true,
  onlyMobile: false,
  onlyUnseen: true,
  verified: '',
  country: '',
  tag: '',
  hours: 24
}

export default function App() {
  const [token, setToken] = useState(localStorage.getItem('token') || '')
  const [user, setUser] = useState(null)
  const [jobs, setJobs] = useState([])
  const [selectedJob, setSelectedJob] = useState(null)
  const [filters, setFilters] = useState(defaultFilters)
  const [loadingJobs, setLoadingJobs] = useState(false)
  const [authLoading, setAuthLoading] = useState(false)
  const [error, setError] = useState('')
  const [screen, setScreen] = useState('jobs')
  const [loginForm, setLoginForm] = useState({ email: 'demo@zenfl.local', password: 'demo1234' })
  const [users, setUsers] = useState([])
  const [userForm, setUserForm] = useState({
    name: '',
    email: '',
    password: '',
    role: 'normal',
    preferences: { onlyUnseen: true, onlyUS: false, onlyMobile: false, country: '', hours: 24 }
  })

  useEffect(() => {
    if (!token) {
      localStorage.removeItem('token')
      setUser(null)
      setJobs([])
      setSelectedJob(null)
      return
    }
    localStorage.setItem('token', token)
    void bootstrap(token)
  }, [token])

  useEffect(() => {
    if (!token || !user) return
    void loadJobs()
  }, [filters, token, user])

  async function bootstrap(authToken) {
    try {
      setError('')
      const meRes = await fetch(`${API}/api/auth/me`, { headers: authHeader(authToken) })
      if (!meRes.ok) throw new Error('Failed to load account')
      const meData = await meRes.json()
      setUser(meData.user)
      setFilters((curr) => ({
        ...curr,
        onlyUS: meData.user.preferences?.only_us ?? curr.onlyUS,
        onlyMobile: meData.user.preferences?.only_mobile ?? curr.onlyMobile,
        onlyUnseen: meData.user.preferences?.only_unseen ?? curr.onlyUnseen,
        country: meData.user.preferences?.country ?? curr.country,
        hours: meData.user.preferences?.hours ?? curr.hours
      }))
      if ((meData.user.role || '') === 'admin') {
        void loadUsers(authToken)
      }
    } catch (err) {
      setError(err.message || 'Authentication failed')
      setToken('')
    }
  }

  async function login(event) {
    event.preventDefault()
    setAuthLoading(true)
    setError('')
    try {
      const res = await fetch(`${API}/api/auth/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(loginForm)
      })
      if (!res.ok) throw new Error('Invalid credentials')
      const data = await res.json()
      setToken(data.token)
    } catch (err) {
      setError(err.message || 'Login failed')
    } finally {
      setAuthLoading(false)
    }
  }

  async function loadJobs() {
    setLoadingJobs(true)
    setError('')
    try {
      const params = new URLSearchParams()
      if (filters.q) params.set('q', filters.q)
      if (filters.onlyUS) params.set('onlyUS', 'true')
      if (filters.onlyMobile) params.set('onlyMobile', 'true')
      if (filters.onlyUnseen) params.set('unseen', 'true')
      if (filters.verified) params.set('verified', filters.verified)
      if (filters.country) params.set('country', filters.country)
      if (filters.tag) params.set('tag', filters.tag)
      if (filters.hours) params.set('hours', String(filters.hours))
      params.set('limit', '100')

      const res = await fetch(`${API}/api/jobs?${params.toString()}`, { headers: authHeader(token) })
      if (!res.ok) throw new Error('Failed to load jobs')
      const data = await res.json()
      setJobs(data.items || [])
    } catch (err) {
      setError(err.message || 'Failed to load jobs')
    } finally {
      setLoadingJobs(false)
    }
  }

  async function openJob(jobID) {
    try {
      const res = await fetch(`${API}/api/jobs/${jobID}`, { headers: authHeader(token) })
      if (!res.ok) throw new Error('Failed to load job')
      const data = await res.json()
      setSelectedJob(data.item)
      setJobs((curr) => curr.filter((job) => !(filters.onlyUnseen && job.id === jobID)))
    } catch (err) {
      setError(err.message || 'Failed to open job')
    }
  }

  async function loadUsers(authToken = token) {
    const res = await fetch(`${API}/api/admin/users`, { headers: authHeader(authToken) })
    if (!res.ok) return
    const data = await res.json()
    setUsers(data.items || [])
  }

  async function createUser(event) {
    event.preventDefault()
    setError('')
    try {
      const res = await fetch(`${API}/api/admin/users`, {
        method: 'POST',
        headers: { ...authHeader(token), 'Content-Type': 'application/json' },
        body: JSON.stringify(userForm)
      })
      if (!res.ok) throw new Error('Failed to create user')
      await loadUsers()
      setUserForm({
        name: '',
        email: '',
        password: '',
        role: 'normal',
        preferences: { onlyUnseen: true, onlyUS: false, onlyMobile: false, country: '', hours: 24 }
      })
    } catch (err) {
      setError(err.message || 'Failed to create user')
    }
  }

  async function updateRole(userID, role) {
    const res = await fetch(`${API}/api/admin/users/${userID}`, {
      method: 'PATCH',
      headers: { ...authHeader(token), 'Content-Type': 'application/json' },
      body: JSON.stringify({ role })
    })
    if (res.ok) {
      await loadUsers()
    }
  }

  function logout() {
    localStorage.removeItem('token')
    setToken('')
  }

  if (!token || !user) {
    return (
      <div className="loginShell">
        <section className="loginCard">
          <div className="brandBlock">
            <span className="eyebrow">Zenfl Platform</span>
            <h1>Find fresh jobs before the queue gets crowded.</h1>
            <p>Structured Telegram intake, unseen-first review flow, and admin controls.</p>
          </div>
          <form className="formGrid" onSubmit={login}>
            <label>
              Email
              <input value={loginForm.email} onChange={(e) => setLoginForm((curr) => ({ ...curr, email: e.target.value }))} />
            </label>
            <label>
              Password
              <input type="password" value={loginForm.password} onChange={(e) => setLoginForm((curr) => ({ ...curr, password: e.target.value }))} />
            </label>
            <button className="primaryButton" disabled={authLoading}>{authLoading ? 'Signing in...' : 'Sign in'}</button>
            <p className="hint">Demo accounts: `admin@zenfl.local` / `admin1234`, `demo@zenfl.local` / `demo1234`</p>
            {error && <p className="errorText">{error}</p>}
          </form>
        </section>
      </div>
    )
  }

  const isAdmin = user.role === 'admin'

  return (
    <div className="appShell">
      <aside className="sidebar">
        <div className="sidebarTop">
          <div>
            <p className="eyebrow">Workspace</p>
            <h1>Job Feed</h1>
          </div>
          <button className="ghostButton" onClick={logout}>Log out</button>
        </div>

        <div className="userCard">
          <strong>{user.name || user.email}</strong>
          <span>{user.role}</span>
        </div>

        <div className="filterBlock">
          <h2>Display filters</h2>
          <label>
            Search
            <input value={filters.q} onChange={(e) => setFilters((curr) => ({ ...curr, q: e.target.value }))} placeholder="Title, skill, text" />
          </label>
          <label>
            Country
            <input value={filters.country} onChange={(e) => setFilters((curr) => ({ ...curr, country: e.target.value }))} placeholder="United States" />
          </label>
          <label>
            Tag
            <input value={filters.tag} onChange={(e) => setFilters((curr) => ({ ...curr, tag: e.target.value }))} placeholder="onlyus" />
          </label>
          <label>
            Last hours
            <input type="number" min="1" max="168" value={filters.hours} onChange={(e) => setFilters((curr) => ({ ...curr, hours: Number(e.target.value || 24) }))} />
          </label>
          <label className="toggle">
            <input type="checkbox" checked={filters.onlyUnseen} onChange={(e) => setFilters((curr) => ({ ...curr, onlyUnseen: e.target.checked }))} />
            <span>Only unseen</span>
          </label>
          <label className="toggle">
            <input type="checkbox" checked={filters.onlyUS} onChange={(e) => setFilters((curr) => ({ ...curr, onlyUS: e.target.checked }))} />
            <span>Only US</span>
          </label>
          <label className="toggle">
            <input type="checkbox" checked={filters.onlyMobile} onChange={(e) => setFilters((curr) => ({ ...curr, onlyMobile: e.target.checked }))} />
            <span>Only mobile</span>
          </label>
          <label>
            Payment verified
            <select value={filters.verified} onChange={(e) => setFilters((curr) => ({ ...curr, verified: e.target.value }))}>
              <option value="">Any</option>
              <option value="true">Verified only</option>
              <option value="false">Unverified only</option>
            </select>
          </label>
        </div>

        {isAdmin && (
          <div className="adminNav">
            <button className={screen === 'jobs' ? 'tab active' : 'tab'} onClick={() => setScreen('jobs')}>Jobs</button>
            <button className={screen === 'users' ? 'tab active' : 'tab'} onClick={() => { setScreen('users'); void loadUsers() }}>Users</button>
          </div>
        )}
      </aside>

      <main className="content">
        {screen === 'users' && isAdmin ? (
          <section className="adminPanel">
            <div className="sectionHeader">
              <div>
                <p className="eyebrow">Admin</p>
                <h2>User management</h2>
              </div>
            </div>
            <div className="adminGrid">
              <form className="panel" onSubmit={createUser}>
                <h3>Create user</h3>
                <label>Name<input value={userForm.name} onChange={(e) => setUserForm((curr) => ({ ...curr, name: e.target.value }))} /></label>
                <label>Email<input value={userForm.email} onChange={(e) => setUserForm((curr) => ({ ...curr, email: e.target.value }))} /></label>
                <label>Password<input type="password" value={userForm.password} onChange={(e) => setUserForm((curr) => ({ ...curr, password: e.target.value }))} /></label>
                <label>
                  Role
                  <select value={userForm.role} onChange={(e) => setUserForm((curr) => ({ ...curr, role: e.target.value }))}>
                    <option value="normal">normal</option>
                    <option value="admin">admin</option>
                  </select>
                </label>
                <button className="primaryButton">Create user</button>
              </form>

              <section className="panel">
                <h3>Existing users</h3>
                <div className="userList">
                  {users.map((item) => (
                    <article key={item.id} className="userRow">
                      <div>
                        <strong>{item.name || item.email}</strong>
                        <p>{item.email}</p>
                      </div>
                      <select value={item.role} onChange={(e) => updateRole(item.id, e.target.value)}>
                        <option value="normal">normal</option>
                        <option value="admin">admin</option>
                      </select>
                    </article>
                  ))}
                </div>
              </section>
            </div>
          </section>
        ) : (
          <div className="board">
            <section className="jobListPanel">
              <div className="sectionHeader">
                <div>
                  <p className="eyebrow">Recent feed</p>
                  <h2>{filters.onlyUnseen ? 'Unseen jobs' : 'All jobs'}</h2>
                </div>
                <button className="ghostButton" onClick={() => loadJobs()} disabled={loadingJobs}>{loadingJobs ? 'Refreshing...' : 'Refresh'}</button>
              </div>
              {error && <p className="errorText">{error}</p>}
              <div className="jobList">
                {jobs.map((job) => (
                  <button key={job.id} className={selectedJob?.id === job.id ? 'jobCard active' : 'jobCard'} onClick={() => openJob(job.id)}>
                    <div className="jobCardTop">
                      <h3>{job.title || 'Untitled job'}</h3>
                      {job.only_us && <span className="chip accent">Only US</span>}
                    </div>
                    <p className="metaLine">{job.category} {job.subcategory ? `· ${job.subcategory}` : ''} {job.experience_level ? `· ${job.experience_level}` : ''}</p>
                    <p className="metaLine">{job.client_location || 'Location unknown'} {job.payment_verified ? '· Payment verified' : ''}</p>
                    <div className="chipRow">
                      {job.skills?.slice(0, 4).map((skill) => <span className="chip" key={skill}>{skill}</span>)}
                    </div>
                  </button>
                ))}
              </div>
            </section>

            <section className="jobDetailPanel">
              {selectedJob ? (
                <>
                  <div className="detailHeader">
                    <div>
                      <p className="eyebrow">Job detail</p>
                      <h2>{selectedJob.title}</h2>
                    </div>
                    {selectedJob.job_link && <a className="primaryButton linkButton" href={selectedJob.job_link} target="_blank" rel="noreferrer">Open Upwork job</a>}
                  </div>
                  <div className="detailMeta">
                    <span>{selectedJob.client_location || 'Unknown location'}</span>
                    <span>{selectedJob.budget || 'Budget not parsed'}</span>
                    <span>{selectedJob.project_type || 'Project type unknown'}</span>
                    <span>{selectedJob.experience_level || 'Experience unknown'}</span>
                  </div>
                  <div className="detailSection">
                    <h3>Skills</h3>
                    <div className="chipRow">
                      {selectedJob.skills?.map((skill) => <span className="chip" key={skill}>{skill}</span>)}
                    </div>
                  </div>
                  <div className="detailSection">
                    <h3>Description</h3>
                    <pre>{selectedJob.description || selectedJob.raw_text}</pre>
                  </div>
                  {!!selectedJob.questions?.length && (
                    <div className="detailSection">
                      <h3>Questions</h3>
                      <ul className="questionList">
                        {selectedJob.questions.map((question) => <li key={question}>{question}</li>)}
                      </ul>
                    </div>
                  )}
                  {!!selectedJob.special_tags?.length && (
                    <div className="detailSection">
                      <h3>Tags</h3>
                      <div className="chipRow">
                        {selectedJob.special_tags.map((tag) => <span className="chip accent" key={tag}>#{tag}</span>)}
                      </div>
                    </div>
                  )}
                </>
              ) : (
                <div className="emptyState">
                  <p className="eyebrow">Ready to review</p>
                  <h2>Select a job from the left.</h2>
                  <p>Opening a job marks it seen, so it drops out of the default unseen feed.</p>
                </div>
              )}
            </section>
          </div>
        )}
      </main>
    </div>
  )
}

function authHeader(token) {
  return { Authorization: `Bearer ${token}` }
}

