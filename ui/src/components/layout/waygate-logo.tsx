import markUrl from '@/assets/waygates-mark.svg';

/**
 * Waygates logo — synthwave pixel portal (multi-color, transparent).
 * NOTE: the mark is fixed multi-color; `className` controls size only.
 * The previous monochrome `currentColor` contract no longer applies.
 */
export function WaygateLogo({ className }: { className?: string }) {
  return <img src={markUrl} alt="" aria-hidden="true" className={className} />;
}
