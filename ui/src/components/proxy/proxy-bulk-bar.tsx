import { Button } from '@e412/rnui-react';
import { Ban, Check, Trash2, X } from 'lucide-react';

interface ProxyBulkBarProps {
  count: number;
  onEnable: () => void;
  onDisable: () => void;
  onDelete: () => void;
  onClear: () => void;
  running: boolean;
}

export function ProxyBulkBar({
  count,
  onEnable,
  onDisable,
  onDelete,
  onClear,
  running,
}: ProxyBulkBarProps) {
  if (count === 0) return null;

  return (
    <div className="flex items-center gap-3 rounded-lg border bg-muted/50 px-4 py-2">
      <span className="text-sm font-medium text-muted-foreground">{count} selected</span>
      <div className="flex items-center gap-2 ml-2">
        <Button type="button" variant="outline" size="sm" onClick={onEnable} disabled={running}>
          <Check className="size-4" />
          Enable
        </Button>
        <Button type="button" variant="outline" size="sm" onClick={onDisable} disabled={running}>
          <Ban className="size-4" />
          Disable
        </Button>
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={onDelete}
          disabled={running}
          className="text-destructive hover:bg-destructive/10 hover:text-destructive border-destructive/30"
        >
          <Trash2 className="size-4" />
          Delete
        </Button>
        <Button type="button" variant="ghost" size="sm" onClick={onClear} disabled={running}>
          <X className="size-4" />
          Clear
        </Button>
      </div>
    </div>
  );
}
