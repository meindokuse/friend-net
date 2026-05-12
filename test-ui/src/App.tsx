import { useState } from 'react'
import { AuthSection } from './sections/AuthSection'
import { UsersSection } from './sections/UsersSection'
import type { TokenState } from './types'

const TABS = ['Auth', 'Users'] as const
type Tab = typeof TABS[number]

const LS_KEY = 'fn_tokens'

function loadTokens(): TokenState {
  try {
    const raw = localStorage.getItem(LS_KEY)
    if (raw) return JSON.parse(raw)
  } catch {}
  return { accessToken: '', refreshToken: '', accountId: '' }
}

function saveTokens(t: TokenState) {
  localStorage.setItem(LS_KEY, JSON.stringify(t))
}

export default function App() {
  const [tab, setTab] = useState<Tab>('Auth')
  const [tokens, setTokens] = useState<TokenState>(loadTokens)

  function handleTokens(t: TokenState) {
    setTokens(t)
    saveTokens(t)
  }

  const hasToken = !!tokens.accessToken

  return (
    <div className="min-h-screen bg-[#0f0f13] flex flex-col">
      {/* Header */}
      <header className="border-b border-[#2e2e3e] bg-[#0c0c10] sticky top-0 z-10">
        <div className="max-w-5xl mx-auto px-4 py-3 flex items-center gap-4">
          <div className="flex items-center gap-2">
            <div className="w-7 h-7 rounded-lg bg-gradient-to-br from-violet-500 to-indigo-600 flex items-center justify-center text-white text-xs font-bold">
              FN
            </div>
            <span className="font-semibold text-slate-200 text-sm">friend-net</span>
            <span className="text-slate-600 text-xs">API Tester</span>
          </div>

          {/* Tabs */}
          <nav className="flex gap-1 ml-4">
            {TABS.map(t => (
              <button
                key={t}
                onClick={() => setTab(t)}
                className={`px-3 py-1.5 rounded-lg text-sm font-medium transition-colors ${
                  tab === t
                    ? 'bg-violet-600 text-white'
                    : 'text-slate-400 hover:text-slate-200 hover:bg-[#1a1a24]'
                }`}
              >
                {t === 'Auth' ? '🔐' : '👤'} {t}
              </button>
            ))}
          </nav>

          {/* Token status */}
          <div className="ml-auto flex items-center gap-2">
            {hasToken ? (
              <div className="flex items-center gap-2 text-xs">
                <span className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse" />
                <span className="text-slate-400">Авторизован</span>
                <code className="text-slate-500 bg-[#1a1a24] px-2 py-0.5 rounded font-mono">
                  {tokens.accountId.slice(0, 8)}...
                </code>
              </div>
            ) : (
              <div className="flex items-center gap-2 text-xs">
                <span className="w-2 h-2 rounded-full bg-slate-600" />
                <span className="text-slate-500">Не авторизован</span>
              </div>
            )}
          </div>
        </div>
      </header>

      <div className="max-w-5xl mx-auto w-full px-4 py-6 flex gap-6 flex-1">
        {/* Main content */}
        <main className="flex-1 min-w-0">
          {/* Token info bar */}
          {hasToken && (
            <div className="mb-4 bg-[#14141e] border border-violet-500/20 rounded-xl p-3 flex flex-col gap-1">
              <div className="flex items-center gap-2 text-xs text-slate-400">
                <span className="text-violet-400 font-medium">Access Token</span>
                <code className="text-slate-500 font-mono break-all flex-1">{tokens.accessToken.slice(0, 60)}...</code>
              </div>
              <div className="flex items-center gap-2 text-xs text-slate-400">
                <span className="text-violet-400 font-medium">Account ID</span>
                <code className="text-slate-500 font-mono">{tokens.accountId}</code>
              </div>
            </div>
          )}

          {tab === 'Auth' && <AuthSection tokens={tokens} onTokens={handleTokens} />}
          {tab === 'Users' && (
            <>
              {!tokens.accessToken && (
                <div className="mb-4 bg-amber-500/10 border border-amber-500/20 rounded-xl p-3 text-sm text-amber-400">
                  Для приватных роутов нужен access token. Войдите в секции Auth.
                </div>
              )}
              <UsersSection tokens={tokens} />
            </>
          )}
        </main>

        {/* Sidebar */}
        <aside className="w-60 shrink-0 hidden lg:block">
          <div className="sticky top-20 flex flex-col gap-3">
            <div className="bg-[#14141e] border border-[#2e2e3e] rounded-xl p-4">
              <h3 className="text-xs font-semibold text-slate-400 uppercase tracking-wider mb-3">Токены</h3>
              <div className="flex flex-col gap-2">
                <div>
                  <div className="text-xs text-slate-500 mb-1">Access Token</div>
                  <div className="text-xs font-mono text-slate-400 bg-[#0f0f13] rounded px-2 py-1.5 break-all leading-relaxed">
                    {tokens.accessToken || <span className="text-slate-600">—</span>}
                  </div>
                </div>
                <div>
                  <div className="text-xs text-slate-500 mb-1">Refresh Token</div>
                  <div className="text-xs font-mono text-slate-400 bg-[#0f0f13] rounded px-2 py-1.5 break-all leading-relaxed">
                    {tokens.refreshToken || <span className="text-slate-600">—</span>}
                  </div>
                </div>
                <div>
                  <div className="text-xs text-slate-500 mb-1">Account ID</div>
                  <div className="text-xs font-mono text-slate-400 bg-[#0f0f13] rounded px-2 py-1.5 break-all">
                    {tokens.accountId || <span className="text-slate-600">—</span>}
                  </div>
                </div>
                {hasToken && (
                  <button
                    onClick={() => handleTokens({ accessToken: '', refreshToken: '', accountId: '' })}
                    className="text-xs text-red-400 hover:text-red-300 text-left mt-1 transition-colors"
                  >
                    Очистить токены
                  </button>
                )}
              </div>
            </div>

            <div className="bg-[#14141e] border border-[#2e2e3e] rounded-xl p-4">
              <h3 className="text-xs font-semibold text-slate-400 uppercase tracking-wider mb-3">Сервисы</h3>
              <div className="flex flex-col gap-2 text-xs">
                <div className="flex items-center justify-between">
                  <span className="text-slate-400">auth-service</span>
                  <code className="text-slate-600">:8080</code>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-slate-400">user-service</span>
                  <code className="text-slate-600">:8081</code>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-slate-400">Traefik</span>
                  <code className="text-slate-600">:80</code>
                </div>
              </div>
            </div>

            <div className="bg-[#14141e] border border-[#2e2e3e] rounded-xl p-4">
              <h3 className="text-xs font-semibold text-slate-400 uppercase tracking-wider mb-3">Легенда</h3>
              <div className="flex flex-col gap-1.5 text-xs">
                {[
                  ['GET', 'bg-blue-500/20 text-blue-400'],
                  ['POST', 'bg-emerald-500/20 text-emerald-400'],
                  ['PATCH', 'bg-amber-500/20 text-amber-400'],
                  ['DELETE', 'bg-red-500/20 text-red-400'],
                ].map(([m, cls]) => (
                  <div key={m} className="flex items-center gap-2">
                    <span className={`px-1.5 py-0.5 rounded font-mono font-bold text-xs ${cls}`}>{m}</span>
                    <span className="text-slate-500">{
                      m === 'GET' ? 'Получить данные' :
                      m === 'POST' ? 'Создать / действие' :
                      m === 'PATCH' ? 'Обновить' : 'Удалить'
                    }</span>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </aside>
      </div>
    </div>
  )
}
