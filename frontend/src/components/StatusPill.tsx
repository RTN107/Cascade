type Tone = 'pending' | 'approved' | 'rejected' | 'flag'

const styles: Record<Tone, React.CSSProperties> = {
  pending: { background: 'transparent', color: 'var(--ink-60)', border: '1px solid var(--ink-20)' },
  approved: { background: 'var(--accent-10)', color: 'var(--accent)', border: '1px solid transparent' },
  rejected: { background: 'var(--ink-05)', color: 'var(--ink-40)', border: '1px solid transparent' },
  flag: { background: 'var(--accent)', color: 'var(--surface)', border: '1px solid transparent' },
}

export default function StatusPill({ tone, children }: { tone: Tone; children: React.ReactNode }) {
  return (
    <span
      style={{
        ...styles[tone],
        display: 'inline-block',
        padding: '0.15rem 0.6rem',
        borderRadius: '999px',
        fontSize: '0.78rem',
        fontWeight: 500,
        lineHeight: 1.4,
      }}
    >
      {children}
    </span>
  )
}
