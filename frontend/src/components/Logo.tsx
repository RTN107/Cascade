function Mark({ size }: { size: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 32 32" aria-hidden="true">
      <rect x="3" y="5" width="26" height="6" rx="2.5" fill="var(--accent)" />
      <rect x="8" y="13" width="19" height="6" rx="2.5" fill="var(--accent)" opacity="0.75" />
      <rect x="13" y="21" width="12" height="6" rx="2.5" fill="var(--accent)" opacity="0.5" />
    </svg>
  )
}

export default function Logo({ size = 'default' }: { size?: 'default' | 'large' }) {
  const markSize = size === 'large' ? 34 : 24
  const textSize = size === 'large' ? '2rem' : '1.25rem'

  return (
    <span style={{ display: 'inline-flex', alignItems: 'center', gap: '0.5rem' }}>
      <Mark size={markSize} />
      <span
        style={{
          fontFamily: 'var(--font-display)',
          fontSize: textSize,
          fontWeight: 600,
          letterSpacing: '-0.01em',
          color: 'var(--ink)',
        }}
      >
        Cascade
      </span>
    </span>
  )
}
