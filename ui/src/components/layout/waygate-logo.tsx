import markUrl from '@/assets/brand/portal.svg';
import wordmarkUrl from '@/assets/brand/wordmark.svg';
import { cn } from '@/lib/utils';

export function WaygateLogo({ className }: { className?: string }) {
  return (
    <img
      src={markUrl}
      width={28}
      height={28}
      alt=""
      aria-hidden="true"
      className={cn('shrink-0', className)}
    />
  );
}

export function WaygatesWordmark({ className }: { className?: string }) {
  return (
    <span
      role="img"
      aria-label="Waygates"
      className={cn('waygates-wordmark', className)}
      style={{ maskImage: `url(${wordmarkUrl})` }}
    />
  );
}

export function WaygatesBrand({ className }: { className?: string }) {
  return (
    <div className={cn('flex items-center gap-2', className)}>
      <WaygateLogo />
      <WaygatesWordmark />
    </div>
  );
}
