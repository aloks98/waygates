import { Button } from '@e412/titanium';
import { Pencil, Trash2 } from 'lucide-react';
import type { ProxyConfig } from '@/types/proxy';

interface ProxyActionsCellProps {
  proxy: ProxyConfig;
  canEdit?: boolean;
  canDelete?: boolean;
  onEdit?: (proxy: ProxyConfig) => void;
  onDelete?: (proxy: ProxyConfig) => void;
}

export function ProxyActionsCell({
  proxy,
  canEdit = false,
  canDelete = false,
  onEdit,
  onDelete,
}: ProxyActionsCellProps) {
  if (!canEdit && !canDelete) {
    return null;
  }

  return (
    <div className="flex justify-end gap-1">
      {canEdit && onEdit && (
        <Button variant="ghost" size="sm" onClick={() => onEdit(proxy)}>
          <Pencil className="size-4" />
        </Button>
      )}
      {canDelete && onDelete && (
        <Button variant="ghost" size="sm" onClick={() => onDelete(proxy)}>
          <Trash2 className="size-4 text-destructive" />
        </Button>
      )}
    </div>
  );
}
