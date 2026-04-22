import { useState } from 'react'

const initialResponse = {
  title: 'Последний ответ',
  status: 'Пока пусто',
  payload: null,
}

function App() {
  const [apiBaseUrl, setApiBaseUrl] = useState(
    import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080'
  )
  const [registerForm, setRegisterForm] = useState({ email: '', password: '' })
  const [loginForm, setLoginForm] = useState({ email: '', password: '' })
  const [refreshToken, setRefreshToken] = useState('')
  const [responseCard, setResponseCard] = useState(initialResponse)
  const [isLoading, setIsLoading] = useState(false)

  const updateCard = (title, status, payload) => {
    setResponseCard({ title, status, payload })
  }

  const request = async (title, path, options = {}) => {
    setIsLoading(true)

    try {
      const response = await fetch(`${apiBaseUrl}${path}`, {
        credentials: 'include',
        headers: {
          'Content-Type': 'application/json',
          ...(options.headers || {}),
        },
        ...options,
      })

      const rawText = await response.text()
      let data

      try {
        data = rawText ? JSON.parse(rawText) : null
      } catch {
        data = rawText
      }

      updateCard(
        title,
        `${response.status} ${response.statusText}`,
        data
      )

      if (data?.refresh_token) {
        setRefreshToken(data.refresh_token)
      }
    } catch (error) {
      updateCard(title, 'Network error', { message: error.message })
    } finally {
      setIsLoading(false)
    }
  }

  const handleRegister = async (event) => {
    event.preventDefault()
    await request('Регистрация', '/auth/register', {
      method: 'POST',
      body: JSON.stringify(registerForm),
    })
  }

  const handleLogin = async (event) => {
    event.preventDefault()
    await request('Логин', '/auth/login', {
      method: 'POST',
      body: JSON.stringify(loginForm),
    })
  }

  const handleRefresh = async (event) => {
    event.preventDefault()
    await request('Обновление токенов', '/auth/refresh', {
      method: 'POST',
      body: refreshToken ? JSON.stringify({ refresh_token: refreshToken }) : null,
    })
  }

  const openGoogleOAuth = () => {
    window.open(`${apiBaseUrl}/auth/google`, '_blank', 'noopener,noreferrer')
  }

  return (
    <div className="app-shell">
      <div className="ambient ambient-left" />
      <div className="ambient ambient-right" />

      <main className="app-frame">
        <section className="hero">
          <div>
            <span className="eyebrow">Auth Service Playground</span>
            <h1>Тестовый React-фронтенд для auth-service</h1>
            <p className="hero-copy">
              Здесь можно руками прогнать register, login, refresh и открыть
              Google OAuth flow, не трогая Postman.
            </p>
          </div>

          <label className="endpoint-card">
            <span className="field-label">API base URL</span>
            <input
              value={apiBaseUrl}
              onChange={(event) => setApiBaseUrl(event.target.value)}
              placeholder="http://localhost:8080"
            />
          </label>
        </section>

        <section className="grid">
          <form className="panel" onSubmit={handleRegister}>
            <div className="panel-head">
              <span className="panel-kicker">01</span>
              <h2>Регистрация</h2>
            </div>

            <label>
              <span className="field-label">Email</span>
              <input
                type="email"
                value={registerForm.email}
                onChange={(event) =>
                  setRegisterForm((prev) => ({ ...prev, email: event.target.value }))
                }
                placeholder="tester@example.com"
              />
            </label>

            <label>
              <span className="field-label">Password</span>
              <input
                type="password"
                value={registerForm.password}
                onChange={(event) =>
                  setRegisterForm((prev) => ({
                    ...prev,
                    password: event.target.value,
                  }))
                }
                placeholder="my-strong-password"
              />
            </label>

            <button className="primary-button" type="submit" disabled={isLoading}>
              Отправить register
            </button>
          </form>

          <form className="panel" onSubmit={handleLogin}>
            <div className="panel-head">
              <span className="panel-kicker">02</span>
              <h2>Логин</h2>
            </div>

            <label>
              <span className="field-label">Email</span>
              <input
                type="email"
                value={loginForm.email}
                onChange={(event) =>
                  setLoginForm((prev) => ({ ...prev, email: event.target.value }))
                }
                placeholder="tester@example.com"
              />
            </label>

            <label>
              <span className="field-label">Password</span>
              <input
                type="password"
                value={loginForm.password}
                onChange={(event) =>
                  setLoginForm((prev) => ({ ...prev, password: event.target.value }))
                }
                placeholder="my-strong-password"
              />
            </label>

            <button className="primary-button" type="submit" disabled={isLoading}>
              Отправить login
            </button>
          </form>

          <form className="panel" onSubmit={handleRefresh}>
            <div className="panel-head">
              <span className="panel-kicker">03</span>
              <h2>Refresh</h2>
            </div>

            <label>
              <span className="field-label">Refresh token</span>
              <textarea
                value={refreshToken}
                onChange={(event) => setRefreshToken(event.target.value)}
                placeholder="Если cookie уже стоит, поле можно оставить пустым"
                rows={5}
              />
            </label>

            <button className="primary-button" type="submit" disabled={isLoading}>
              Обновить токены
            </button>
          </form>

          <section className="panel panel-accent">
            <div className="panel-head">
              <span className="panel-kicker">04</span>
              <h2>Google OAuth</h2>
            </div>

            <p className="panel-copy">
              Откроет `/auth/google` в новой вкладке. Подходит для ручной проверки
              redirect flow и callback.
            </p>

            <button
              className="secondary-button"
              type="button"
              onClick={openGoogleOAuth}
            >
              Открыть Google OAuth
            </button>
          </section>
        </section>

        <section className="response-panel">
          <div className="response-head">
            <div>
              <span className="eyebrow">Inspector</span>
              <h2>{responseCard.title}</h2>
            </div>
            <span className="status-badge">{responseCard.status}</span>
          </div>

          <pre className="response-code">
            {JSON.stringify(responseCard.payload, null, 2) || 'Нет данных'}
          </pre>
        </section>
      </main>
    </div>
  )
}

export default App
