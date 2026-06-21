import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@e412/rnui-react';
import { ChevronDown } from 'lucide-react';
import type { ReactNode } from 'react';

interface FormSectionProps {
  title: string;
  description?: string;
  hasError?: boolean;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  children: ReactNode;
}

export function FormSection({
  title,
  description,
  hasError,
  open,
  onOpenChange,
  children,
}: FormSectionProps) {
  return (
    <Collapsible open={open} onOpenChange={onOpenChange} className="rounded-lg border">
      <CollapsibleTrigger className="flex w-full items-center justify-between gap-2 px-4 py-3 text-left">
        <div className="flex items-center gap-2">
          <span className="font-medium">{title}</span>
          {hasError && (
            <span
              aria-label="Section has errors"
              className="inline-block size-2 rounded-full bg-destructive"
            />
          )}
        </div>
        <ChevronDown
          className={`size-4 text-muted-foreground transition-transform ${open ? 'rotate-180' : ''}`}
        />
      </CollapsibleTrigger>
      <CollapsibleContent>
        <div className="space-y-4 px-4 pb-4">
          {description && <p className="text-sm text-muted-foreground">{description}</p>}
          {children}
        </div>
      </CollapsibleContent>
    </Collapsible>
  );
}
