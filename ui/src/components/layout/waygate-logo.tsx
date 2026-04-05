/**
 * Waygates Logo — W Portal (Bold).
 * A stylized W forming two arches with a center waypoint dot.
 * Inspired by Elden Ring's waygate portals.
 */
export function WaygateLogo({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 24 24" fill="none" className={className} aria-hidden="true">
      <path
        d="M2 6l4.5 12L12 7l5.5 11L22 6"
        stroke="currentColor"
        strokeWidth="2.8"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <circle cx="12" cy="11.5" r="2" fill="currentColor" />
    </svg>
  );
}
