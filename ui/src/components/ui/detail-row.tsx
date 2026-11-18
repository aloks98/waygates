import type { ReactNode } from 'react';

interface DetailRowProps {
  label: string;
  children: ReactNode;
}

export function DetailRow({ label, children }: DetailRowProps) {
  return (
    <div className="grid grid-cols-[minmax(100px,1fr)_minmax(0,2fr)] gap-4 py-3 text-sm">
      <dt className="font-medium text-muted-foreground">{label}</dt>
      <dd className="break-words text-foreground">{children}</dd>
    </div>
  );
}
