import { useState } from 'react'
import { Card } from '../components/Card'
import { Field } from '../components/Field'
import { Btn } from '../components/Btn'
import { ResponseBox } from '../components/ResponseBox'
import { authApi } from '../api'
import type { TokenState } from '../types'

interface Props {
  tokens: TokenState
  onTokens: (t: TokenState) => void
}

function useCall() {
  const [status, setStatus] = useState<number>()
  const [data, setData] = useState<unknown>()
  const [loading, setLoading] = useState(false)

  async function call(fn: () => Promise<{ status: number; data: unknown; isOk: boolean }>) {
    setLoading(true)
    try {
      const r = await fn()
      setStatus(r.status)
      setData(r.data)
      return r
    } finally {
      setLoading(false)
    }
  }
  return { status, data, loading, call }
}

export function AuthSection({ tokens, onTokens }: Props) {
  // Register
  const reg = useCall()
  const [regEmail, setRegEmail] = useState('')
  const [regPass, setRegPass] = useState('')
  const [regName, setRegName] = useState('')

  // Login
  const login = useCall()
  const [loginEmail, setLoginEmail] = useState('')
  const [loginPass, setLoginPass] = useState('')

  // Sessions
  const sessions = useCall()
  const revokeS = useCall()
  const [sessionId, setSessionId] = useState('')

  // Introspect
  const introspect = useCall()

  // Linked
  const linked = useCall()
  const unlink = useCall()
  const [provider, setProvider] = useState('google')

  return (
    <div className="flex flex-col gap-3">
      {/* Register */}
      <Card title="Создать аккаунт" method="POST" path="/auth/register">
        <div className="flex flex-col gap-3">
          <Field label="Email" value={regEmail} onChange={setRegEmail} type="email" placeholder="user@example.com" />
          <Field label="Password" value={regPass} onChange={setRegPass} type="password" placeholder="min 8 символов" />
          <Field label="Display Name" value={regName} onChange={setRegName} placeholder="min 4 символа" />
          <Btn loading={reg.loading} onClick={async () => {
            await reg.call(() => authApi.register({ email: regEmail, password: regPass, display_name: regName }))
          }}>
            Зарегистрироваться
          </Btn>
          <ResponseBox status={reg.status} data={reg.data} loading={reg.loading} />
        </div>
      </Card>

      {/* Login */}
      <Card title="Войти и получить токены" method="POST" path="/auth/login">
        <div className="flex flex-col gap-3">
          <Field label="Email" value={loginEmail} onChange={setLoginEmail} type="email" placeholder="user@example.com" />
          <Field label="Password" value={loginPass} onChange={setLoginPass} type="password" />
          <Btn loading={login.loading} onClick={async () => {
            const r = await login.call(() => authApi.login({ email: loginEmail, password: loginPass }))
            if (r?.isOk && r.data && typeof r.data === 'object') {
              const d = r.data as { access_token: string; refresh_token: string; account_id: string }
              onTokens({ accessToken: d.access_token, refreshToken: d.refresh_token, accountId: d.account_id })
            }
          }}>
            Войти
          </Btn>
          <ResponseBox status={login.status} data={login.data} loading={login.loading} />
        </div>
      </Card>

      {/* Refresh */}
      <Card title="Обновить токены" method="POST" path="/auth/refresh">
        <div className="flex flex-col gap-3">
          <div className="bg-[#1a1a24] rounded-lg px-3 py-2 text-xs font-mono text-slate-400 break-all">
            {tokens.refreshToken || <span className="text-slate-600">refresh token не установлен — сначала войдите</span>}
          </div>
          <RefreshCard tokens={tokens} onTokens={onTokens} />
        </div>
      </Card>

      {/* Logout */}
      <Card title="Выйти из сессии" method="POST" path="/auth/logout" requiresAuth>
        <LogoutCard tokens={tokens} onTokens={onTokens} />
      </Card>

      {/* Logout All */}
      <Card title="Выйти из всех сессий" method="POST" path="/auth/logout-all" requiresAuth>
        <LogoutAllCard tokens={tokens} onTokens={onTokens} />
      </Card>

      {/* Sessions */}
      <Card title="Список активных сессий" method="GET" path="/auth/sessions" requiresAuth>
        <div className="flex flex-col gap-3">
          <Btn loading={sessions.loading} onClick={() => sessions.call(() => authApi.sessions(tokens.accessToken))}>
            Получить сессии
          </Btn>
          <ResponseBox status={sessions.status} data={sessions.data} loading={sessions.loading} />
        </div>
      </Card>

      {/* Revoke session */}
      <Card title="Отозвать сессию" method="DELETE" path="/auth/sessions/:session_id" requiresAuth>
        <div className="flex flex-col gap-3">
          <Field label="Session ID" value={sessionId} onChange={setSessionId} mono placeholder="uuid" />
          <Btn variant="danger" loading={revokeS.loading} onClick={() =>
            revokeS.call(() => authApi.revokeSession(tokens.accessToken, sessionId))
          }>
            Отозвать
          </Btn>
          <ResponseBox status={revokeS.status} data={revokeS.data} loading={revokeS.loading} />
        </div>
      </Card>

      {/* Introspect */}
      <Card title="Проверить токен" method="POST" path="/auth/introspect">
        <div className="flex flex-col gap-3">
          <div className="text-xs text-slate-500">Использует текущий access token</div>
          <Btn loading={introspect.loading} variant="ghost" onClick={() =>
            introspect.call(() => authApi.introspect(tokens.accessToken))
          }>
            Introspect
          </Btn>
          <ResponseBox status={introspect.status} data={introspect.data} loading={introspect.loading} />
        </div>
      </Card>

      {/* Linked accounts */}
      <Card title="Привязанные аккаунты" method="GET" path="/auth/linked" requiresAuth>
        <div className="flex flex-col gap-3">
          <Btn loading={linked.loading} variant="ghost" onClick={() =>
            linked.call(() => authApi.linkedAccounts(tokens.accessToken))
          }>
            Получить
          </Btn>
          <ResponseBox status={linked.status} data={linked.data} loading={linked.loading} />
        </div>
      </Card>

      {/* Unlink */}
      <Card title="Отвязать провайдера" method="DELETE" path="/auth/linked/:provider" requiresAuth>
        <div className="flex flex-col gap-3">
          <Field label="Provider" value={provider} onChange={setProvider} placeholder="google" />
          <Btn variant="danger" loading={unlink.loading} onClick={() =>
            unlink.call(() => authApi.unlinkProvider(tokens.accessToken, provider))
          }>
            Отвязать
          </Btn>
          <ResponseBox status={unlink.status} data={unlink.data} loading={unlink.loading} />
        </div>
      </Card>

      {/* Google OAuth */}
      <Card title="Войти через Google" method="GET" path="/auth/google">
        <div className="flex flex-col gap-2">
          <p className="text-xs text-slate-500">Откроет редирект на Google OAuth</p>
          <a
            href="/auth/google"
            target="_blank"
            className="inline-flex items-center gap-2 px-4 py-2 bg-white text-gray-800 rounded-lg text-sm font-medium hover:bg-gray-100 transition-colors w-fit"
          >
            <svg width="16" height="16" viewBox="0 0 24 24">
              <path fill="#4285F4" d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"/>
              <path fill="#34A853" d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"/>
              <path fill="#FBBC05" d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"/>
              <path fill="#EA4335" d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"/>
            </svg>
            Google OAuth
          </a>
        </div>
      </Card>
    </div>
  )
}

function RefreshCard({ tokens, onTokens }: Props) {
  const r = useCall()
  return (
    <>
      <Btn loading={r.loading} variant="ghost" onClick={async () => {
        const res = await r.call(() => authApi.refresh(tokens.refreshToken))
        if (res?.isOk && res.data && typeof res.data === 'object') {
          const d = res.data as { access_token: string; refresh_token: string; account_id: string }
          onTokens({ accessToken: d.access_token, refreshToken: d.refresh_token, accountId: d.account_id })
        }
      }}>
        Обновить токены
      </Btn>
      <ResponseBox status={r.status} data={r.data} loading={r.loading} />
    </>
  )
}

function LogoutCard({ tokens, onTokens }: Props) {
  const r = useCall()
  return (
    <>
      <Btn variant="danger" loading={r.loading} onClick={async () => {
        await r.call(() => authApi.logout(tokens.accessToken, tokens.refreshToken))
        onTokens({ accessToken: '', refreshToken: '', accountId: '' })
      }}>
        Выйти
      </Btn>
      <ResponseBox status={r.status} data={r.data} loading={r.loading} />
    </>
  )
}

function LogoutAllCard({ tokens, onTokens }: Props) {
  const r = useCall()
  return (
    <>
      <Btn variant="danger" loading={r.loading} onClick={async () => {
        await r.call(() => authApi.logoutAll(tokens.accessToken))
        onTokens({ accessToken: '', refreshToken: '', accountId: '' })
      }}>
        Выйти из всех устройств
      </Btn>
      <ResponseBox status={r.status} data={r.data} loading={r.loading} />
    </>
  )
}
