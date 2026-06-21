import {
  Button,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@e412/rnui-react';
import { Copy, MoreHorizontal, Pencil, Trash2 } from 'lucide-react';

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
    <div className="flex justify-end">
      <DropdownMenu>
        <DropdownMenuTrigger render={<Button variant="ghost" size="sm" className="size-8 p-0" />}>
          <MoreHorizontal className="size-4" />
          <span className="sr-only">Open actions menu</span>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-40">
          {canEdit && onEdit && (
            <DropdownMenuItem onClick={() => onEdit(proxy)}>
              <Pencil className="size-4" />
              Edit
            </DropdownMenuItem>
          )}
          {onDuplicate && (
            <DropdownMenuItem onClick={() => onDuplicate(proxy)}>
              <Copy className="size-4" />
              Duplicate
            </DropdownMenuItem>
          )}
          {canDelete && onDelete && (
            <>
              {(canEdit || onDuplicate) && <DropdownMenuSeparator />}
              <DropdownMenuItem
                onClick={() => onDelete(proxy)}
                className="text-destructive focus:text-destructive"
              >
                <Trash2 className="size-4" />
                Delete
              </DropdownMenuItem>
            </>
          )}
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}
