import type { ReactNode } from 'react'

interface Props {
  children: ReactNode
  onClick: () => void
  variant?: 'primary' | 'danger' | 'ghost'
  loading?: boolean
  className?: string
}

export function Btn({ children, onClick, variant = 'primary', loading, className = '' }: Props) {
  const base = 'px-4 py-2 rounded-lg text-sm font-medium transition-all flex items-center gap-2 disabled:opacity-50'
  const variants = {
    primary: 'bg-violet-600 hover:bg-violet-500 text-white',
    danger: 'bg-red-600/80 hover:bg-red-500 text-white',
    ghost: 'bg-[#2e2e3e] hover:bg-[#3a3a50] text-slate-300',
  }
  return (
    <button
      onClick={onClick}
      disabled={loading}
      className={`${base} ${variants[variant]} ${className}`}
    >
      {loading && <span className="animate-spin text-xs">⏳</span>}
      {children}
    </button>
  )
}
