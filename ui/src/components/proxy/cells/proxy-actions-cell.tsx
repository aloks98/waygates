import { Button, Tooltip, TooltipContent, TooltipTrigger } from '@e412/rnui-react';
import { Copy, Pencil, Trash2 } from 'lucide-react';

import type { ProxyConfig } from '@/types/proxy';

interface ProxyActionsCellProps {
  proxy: ProxyConfig;
  canEdit?: boolean;
  canDelete?: boolean;
  onEdit?: (proxy: ProxyConfig) => void;
  onDelete?: (proxy: ProxyConfig) => void;
  onDuplicate?: (proxy: ProxyConfig) => void;
}

export function ProxyActionsCell({
  proxy,
  canEdit = false,
  canDelete = false,
  onEdit,
  onDelete,
  onDuplicate,
}: ProxyActionsCellProps) {
  if (!canEdit && !canDelete && !onDuplicate) {
    return null;
  }

  return (
    <div className="flex justify-end gap-1">
      {canEdit && onEdit && (
        <Tooltip>
          <TooltipTrigger
            render={<Button variant="ghost" size="sm" onClick={() => onEdit(proxy)} />}
          >
            <Pencil className="size-4" />
            <span className="sr-only">Edit proxy</span>
          </TooltipTrigger>
          <TooltipContent>Edit proxy</TooltipContent>
        </Tooltip>
      )}
      {onDuplicate && (
        <Tooltip>
          <TooltipTrigger
            render={<Button variant="ghost" size="sm" onClick={() => onDuplicate(proxy)} />}
          >
            <Copy className="size-4" />
            <span className="sr-only">Duplicate proxy</span>
          </TooltipTrigger>
          <TooltipContent>Duplicate</TooltipContent>
        </Tooltip>
      )}
      {canDelete && onDelete && (
        <Tooltip>
          <TooltipTrigger
            render={<Button variant="ghost" size="sm" onClick={() => onDelete(proxy)} />}
          >
            <Trash2 className="size-4 text-destructive" />
            <span className="sr-only">Delete proxy</span>
          </TooltipTrigger>
          <TooltipContent>Delete proxy</TooltipContent>
        </Tooltip>
      )}
    </div>
  );
}
