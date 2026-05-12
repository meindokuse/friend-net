interface Props {
  status?: number
  data?: unknown
  loading?: boolean
}

function statusColor(code: number) {
  if (code >= 200 && code < 300) return 'text-emerald-400'
  if (code >= 400 && code < 500) return 'text-amber-400'
  return 'text-red-400'
}

export function ResponseBox({ status, data, loading }: Props) {
  if (loading) {
    return (
      <div className="bg-[#1a1a24] border border-[#2e2e3e] rounded-xl p-4 mt-3 flex items-center gap-2 text-slate-400">
        <span className="animate-spin text-lg">⏳</span> Ожидание...
      </div>
    )
  }

  if (status === undefined) return null

  const json = typeof data === 'object' ? JSON.stringify(data, null, 2) : String(data)

  return (
    <div className="bg-[#1a1a24] border border-[#2e2e3e] rounded-xl mt-3 overflow-hidden">
      <div className="flex items-center gap-2 px-4 py-2 border-b border-[#2e2e3e] bg-[#14141e]">
        <span className={`font-mono font-bold text-sm ${statusColor(status)}`}>{status}</span>
        <span className="text-slate-500 text-xs">Response</span>
      </div>
      <pre className="p-4 text-xs text-slate-300 overflow-auto max-h-64 leading-relaxed">{json}</pre>
    </div>
  )
}
