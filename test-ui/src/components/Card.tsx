import { useState } from 'react'
import type { ReactNode } from 'react'

interface Props {
  title: string
  method: string
  path: string
  children: ReactNode
  requiresAuth?: boolean
}

const methodColors: Record<string, string> = {
  GET: 'bg-blue-500/20 text-blue-400 border-blue-500/30',
  POST: 'bg-emerald-500/20 text-emerald-400 border-emerald-500/30',
  PATCH: 'bg-amber-500/20 text-amber-400 border-amber-500/30',
  DELETE: 'bg-red-500/20 text-red-400 border-red-500/30',
}

export function Card({ title, method, path, children, requiresAuth }: Props) {
  const [open, setOpen] = useState(false)
  const color = methodColors[method] || 'bg-slate-500/20 text-slate-400 border-slate-500/30'

  return (
    <div className="bg-[#14141e] border border-[#2e2e3e] rounded-xl overflow-hidden">
      <button
        onClick={() => setOpen(o => !o)}
        className="w-full flex items-center gap-3 px-4 py-3 hover:bg-[#1a1a24] transition-colors text-left"
      >
        <span className={`text-xs font-mono font-bold px-2 py-0.5 rounded border ${color}`}>{method}</span>
        <code className="text-sm text-slate-300 font-mono flex-1">{path}</code>
        {requiresAuth && (
          <span className="text-xs text-violet-400 bg-violet-500/10 border border-violet-500/20 px-2 py-0.5 rounded">
            🔑 auth
          </span>
        )}
        <span className="text-slate-500 text-xs ml-auto">{open ? '▲' : '▼'}</span>
      </button>
      {open && (
        <div className="px-4 pb-4 border-t border-[#2e2e3e] pt-4">
          <h3 className="text-sm font-semibold text-slate-200 mb-3">{title}</h3>
          {children}
        </div>
      )}
    </div>
  )
}
