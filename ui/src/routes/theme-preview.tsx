/* ─── W Portal variants ─── */

function WP1({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 24 24" fill="none" className={className} aria-hidden="true">
      {/* Original — W arches + center waypoint dot */}
      <path
        d="M3 7l4 10 5-8 5 8 4-10"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <circle cx="12" cy="12" r="1.2" fill="currentColor" />
    </svg>
  );
}

function WP2({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 24 24" fill="none" className={className} aria-hidden="true">
      {/* Thicker strokes, larger dot — bolder presence */}
      <path
        d="M3 7l4 10 5-8 5 8 4-10"
        stroke="currentColor"
        strokeWidth="2.5"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <circle cx="12" cy="12" r="1.8" fill="currentColor" />
    </svg>
  );
}

function WP3({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 24 24" fill="none" className={className} aria-hidden="true">
      {/* Wider W + ring instead of filled dot — portal opening */}
      <path
        d="M2 8l4.5 10L12 9l5.5 9L22 8"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <circle cx="12" cy="12.5" r="1.5" stroke="currentColor" strokeWidth="1.2" />
    </svg>
  );
}

function WP4({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 24 24" fill="none" className={className} aria-hidden="true">
      {/* Taller W — more vertical, gate-like proportions */}
      <path
        d="M4 5l3.5 14L12 7l4.5 12L20 5"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <circle cx="12" cy="12" r="1.2" fill="currentColor" />
    </svg>
  );
}

function WP5({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 24 24" fill="none" className={className} aria-hidden="true">
      {/* Shallow W + glow fill behind the dot — warm portal energy */}
      <path
        d="M3 8l4 8 5-6 5 6 4-8"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <circle cx="12" cy="11.5" r="3" fill="currentColor" opacity="0.1" />
      <circle cx="12" cy="11.5" r="1.2" fill="currentColor" />
    </svg>
  );
}

function WP6({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 24 24" fill="none" className={className} aria-hidden="true">
      {/* W with flat base line — grounded, architectural */}
      <path
        d="M4 8l3.5 8L12 8l4.5 8L20 8"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <line
        x1="4"
        y1="18"
        x2="20"
        y2="18"
        stroke="currentColor"
        strokeWidth="1.5"
        strokeLinecap="round"
      />
      <circle cx="12" cy="13" r="1" fill="currentColor" />
    </svg>
  );
}

function WP7({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 24 24" fill="none" className={className} aria-hidden="true">
      {/* Rounded W — softer curves, friendlier */}
      <path
        d="M3 7c1.5 5 3 10 4.5 10S11 10 12 9c1 1 3 8 4.5 8S19.5 12 21 7"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
      />
      <circle cx="12" cy="12.5" r="1.2" fill="currentColor" />
    </svg>
  );
}

function WP8({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 24 24" fill="none" className={className} aria-hidden="true">
      {/* Double-stroke W — inner echo creates depth */}
      <path
        d="M3 7l4 10 5-8 5 8 4-10"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <path
        d="M6 9l2.5 6L12 9.5l3.5 5.5L18 9"
        stroke="currentColor"
        strokeWidth="1"
        strokeLinecap="round"
        strokeLinejoin="round"
        opacity="0.3"
      />
      <circle cx="12" cy="11.5" r="1" fill="currentColor" opacity="0.6" />
    </svg>
  );
}

function WP9({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 24 24" fill="none" className={className} aria-hidden="true">
      {/* Angular W + diamond dot — sharp, rune-like */}
      <path
        d="M3 7l4 10 5-8 5 8 4-10"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <path d="M12 10.5l1.2 1.5-1.2 1.5-1.2-1.5z" fill="currentColor" />
    </svg>
  );
}

function WP10({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 24 24" fill="none" className={className} aria-hidden="true">
      {/* Symmetrical W + upward arrow from center — ascent/travel */}
      <path
        d="M3 8l4 10 5-8 5 8 4-10"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <line
        x1="12"
        y1="13"
        x2="12"
        y2="6"
        stroke="currentColor"
        strokeWidth="1.5"
        strokeLinecap="round"
      />
      <path
        d="M10 8l2-2 2 2"
        stroke="currentColor"
        strokeWidth="1.5"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}

/* ─── Preview ─── */

function LogoContext({
  name,
  Logo,
}: {
  name: string;
  Logo: React.ComponentType<{ className?: string }>;
}) {
  return (
    <div className="border rounded bg-card text-card-foreground overflow-hidden">
      <div className="px-4 py-2 border-b bg-muted/30 text-xs font-medium text-muted-foreground">
        {name}
      </div>
      <div className="p-4 space-y-4">
        <div className="flex items-end gap-4">
          <div className="flex flex-col items-center gap-1">
            <div className="flex size-8 items-center justify-center rounded bg-primary text-primary-foreground">
              <Logo className="size-5" />
            </div>
            <span className="text-[10px] text-muted-foreground">32</span>
          </div>
          <div className="flex flex-col items-center gap-1">
            <div className="flex size-10 items-center justify-center rounded bg-primary text-primary-foreground">
              <Logo className="size-6" />
            </div>
            <span className="text-[10px] text-muted-foreground">40</span>
          </div>
          <div className="flex flex-col items-center gap-1">
            <div className="flex size-14 items-center justify-center rounded bg-primary text-primary-foreground">
              <Logo className="size-8" />
            </div>
            <span className="text-[10px] text-muted-foreground">56</span>
          </div>
          <div className="flex flex-col items-center gap-1">
            <div className="flex size-10 items-center justify-center rounded border border-primary text-primary">
              <Logo className="size-6" />
            </div>
            <span className="text-[10px] text-muted-foreground">outline</span>
          </div>
          <div className="flex flex-col items-center gap-1">
            <div className="flex size-10 items-center justify-center rounded bg-muted text-foreground">
              <Logo className="size-6" />
            </div>
            <span className="text-[10px] text-muted-foreground">muted</span>
          </div>
        </div>
        {/* Sidebar mock */}
        <div className="flex items-center gap-2.5 p-2 rounded border bg-card">
          <div className="flex size-8 items-center justify-center rounded bg-primary text-primary-foreground">
            <Logo className="size-5" />
          </div>
          <span
            className="text-lg font-semibold tracking-tight"
            style={{ fontFamily: 'var(--font-display)' }}
          >
            Waygates
          </span>
        </div>
      </div>
    </div>
  );
}

export function ThemePreviewPage() {
  const variants = [
    { name: 'Original — clean W + dot', Logo: WP1 },
    { name: 'Bold — thicker stroke + larger dot', Logo: WP2 },
    { name: 'Ring — portal opening instead of dot', Logo: WP3 },
    { name: 'Tall — more vertical, gate proportions', Logo: WP4 },
    { name: 'Glow — soft aura behind the dot', Logo: WP5 },
    { name: 'Grounded — W + base line', Logo: WP6 },
    { name: 'Rounded — softer curves, friendly', Logo: WP7 },
    { name: 'Echo — double stroke, depth effect', Logo: WP8 },
    { name: 'Diamond — angular dot, rune-like', Logo: WP9 },
    { name: 'Arrow — upward travel from center', Logo: WP10 },
  ];

  return (
    <div className="min-h-screen bg-background text-foreground">
      <div className="max-w-5xl mx-auto px-6 py-8 space-y-8">
        <div>
          <h1 className="text-2xl font-bold">W Portal — Variants</h1>
          <p className="text-muted-foreground mt-1">
            10 variants of the W Portal logo. Same base shape, different character.
          </p>
        </div>

        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {variants.map(({ name, Logo }) => (
            <LogoContext key={name} name={name} Logo={Logo} />
          ))}
        </div>

        <p className="text-sm text-muted-foreground text-center">
          Pick one and I'll apply it across the app.
        </p>
      </div>
    </div>
  );
}
