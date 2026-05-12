interface FieldProps {
  label: string
  value: string
  onChange: (v: string) => void
  type?: string
  placeholder?: string
  mono?: boolean
}

export function Field({ label, value, onChange, type = 'text', placeholder, mono }: FieldProps) {
  return (
    <label className="flex flex-col gap-1">
      <span className="text-xs text-slate-400 font-medium uppercase tracking-wider">{label}</span>
      <input
        type={type}
        value={value}
        onChange={e => onChange(e.target.value)}
        placeholder={placeholder}
        className={`bg-[#1a1a24] border border-[#2e2e3e] rounded-lg px-3 py-2 text-sm text-slate-200 placeholder:text-slate-600 focus:outline-none focus:border-violet-500 transition-colors ${mono ? 'font-mono' : ''}`}
      />
    </label>
  )
}

interface SelectProps {
  label: string
  value: string
  onChange: (v: string) => void
  options: string[]
}

export function SelectField({ label, value, onChange, options }: SelectProps) {
  return (
    <label className="flex flex-col gap-1">
      <span className="text-xs text-slate-400 font-medium uppercase tracking-wider">{label}</span>
      <select
        value={value}
        onChange={e => onChange(e.target.value)}
        className="bg-[#1a1a24] border border-[#2e2e3e] rounded-lg px-3 py-2 text-sm text-slate-200 focus:outline-none focus:border-violet-500 transition-colors"
      >
        {options.map(o => <option key={o} value={o}>{o}</option>)}
      </select>
    </label>
  )
}
