import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  Badge,
  Button,
  Skeleton,
} from '@e412/rnui-react';
import { useNavigate, useParams } from '@tanstack/react-router';
import { ArrowLeft, Network, Trash2 } from 'lucide-react';
import { useState } from 'react';

import { L4ProxyForm } from '@/components/l4-proxy';
import { useL4Proxies, useL4Proxy } from '@/hooks/use-l4-proxies';
import type { CreateL4ProxyRequest } from '@/types/l4-proxy';

function getProtocolIcon(_protocol: string) {
  return <Network className="size-5 text-primary" />;
}

function getProtocolLabel(protocol: string): string {
  return protocol.toUpperCase();
}

export function L4ProxyEditPage() {
  const params = useParams({ from: '/dashboard/proxies/tcp-udp/$l4ProxyId/edit' });
  const l4ProxyId = parseInt(params.l4ProxyId, 10);
  const navigate = useNavigate();

  const { proxy, isLoading } = useL4Proxy(l4ProxyId);
  const { update, remove, isUpdating, isDeleting } = useL4Proxies();

  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);

  const handleUpdate = async (data: CreateL4ProxyRequest) => {
    if (!proxy) return;
    await update({ id: proxy.id, data });
  };

  const handleDelete = async () => {
    await remove(l4ProxyId);
    setDeleteDialogOpen(false);
    navigate({ to: '/dashboard/proxies/tcp-udp' });
  };

  const handleCancel = () => {
    navigate({
      to: '/dashboard/proxies/tcp-udp/$l4ProxyId',
      params: { l4ProxyId: String(l4ProxyId) },
    });
  };

  if (isLoading) {
    return (
      <div className="space-y-6 max-w-5xl">
        <div className="flex items-center gap-4">
          <Skeleton className="size-8 rounded" />
          <div className="flex items-center gap-3">
            <Skeleton className="size-10 rounded-lg" />
            <div className="space-y-2">
              <Skeleton className="h-7 w-48" />
              <Skeleton className="h-4 w-32" />
            </div>
          </div>
        </div>
        <Skeleton className="h-48 rounded-lg" />
        <div className="grid gap-6 lg:grid-cols-2">
          <Skeleton className="h-64 rounded-lg" />
          <Skeleton className="h-64 rounded-lg" />
        </div>
        <Skeleton className="h-32 rounded-lg" />
      </div>
    );
  }

  if (!proxy) {
    return (
      <div className="flex flex-col items-center justify-center py-12 space-y-4">
        <Network className="size-12 text-muted-foreground" />
        <h2 className="text-xl font-semibold">Proxy Not Found</h2>
        <p className="text-muted-foreground">
          The TCP/UDP proxy you're looking for doesn't exist or has been deleted.
        </p>
        <Button onClick={() => navigate({ to: '/dashboard/proxies/tcp-udp' })}>
          <ArrowLeft className="size-4" />
          Back to TCP/UDP Proxies
        </Button>
      </div>
    );
  }

  return (
    <div className="space-y-6 max-w-5xl">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button
            variant="ghost"
            size="icon"
            onClick={() =>
              navigate({
                to: '/dashboard/proxies/tcp-udp/$l4ProxyId',
                params: { l4ProxyId: String(l4ProxyId) },
              })
            }
          >
            <ArrowLeft className="size-4" />
            <span className="sr-only">Back</span>
          </Button>
          <div className="flex items-center gap-3">
            <div className="flex items-center justify-center size-10 rounded bg-primary/10">
              {getProtocolIcon(proxy.protocol)}
            </div>
            <div>
              <div className="flex items-center gap-2">
                <h1 className="text-2xl font-bold">{proxy.name}</h1>
                <Badge variant="outline">{getProtocolLabel(proxy.protocol)}</Badge>
                <Badge variant={proxy.is_active ? 'default' : 'secondary'}>
                  {proxy.is_active ? 'Active' : 'Inactive'}
                </Badge>
              </div>
              <p className="text-sm text-muted-foreground">
                Port {proxy.listen_port}
                {proxy.description && ` · ${proxy.description}`}
              </p>
            </div>
          </div>
        </div>
        <Button
          variant="outline"
          className="text-destructive hover:text-destructive"
          onClick={() => setDeleteDialogOpen(true)}
        >
          <Trash2 className="size-4" />
          Delete
        </Button>
      </div>

      {/* Form */}
      <L4ProxyForm
        mode="edit"
        initialData={proxy}
        onSubmit={handleUpdate}
        loading={isUpdating}
        onCancel={handleCancel}
      />

      {/* Delete Dialog */}
      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Proxy</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete <strong>{proxy.name}</strong>? This action cannot be
              undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDelete}
              disabled={isDeleting}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {isDeleting ? 'Deleting...' : 'Delete'}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
