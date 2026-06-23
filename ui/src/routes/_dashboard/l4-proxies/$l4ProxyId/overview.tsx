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
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
  Skeleton,
} from '@e412/rnui-react';
import { Link, useNavigate, useParams } from '@tanstack/react-router';
import { ArrowLeft, Copy, MoreHorizontal, Network, Pencil, Trash2 } from 'lucide-react';
import { useState } from 'react';

import { L4ProxyConfigCard } from '@/components/l4-proxy/overview/l4-proxy-config-card';
import { DetailRow } from '@/components/ui/detail-row';
import { useL4Proxies, useL4Proxy } from '@/hooks/use-l4-proxies';
import { usePermissions } from '@/hooks/use-permissions';

export function L4ProxyOverviewPage() {
  const params = useParams({ strict: false });
  const l4ProxyId = parseInt(params.l4ProxyId, 10);
  const navigate = useNavigate();

  const { proxy, isLoading } = useL4Proxy(l4ProxyId);
  const { remove, isDeleting } = useL4Proxies();
  const { canCreateProxies, canDeleteProxies, canUpdateProxies } = usePermissions();

  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);

  const handleDelete = async () => {
    await remove(l4ProxyId);
    setDeleteDialogOpen(false);
    navigate({ to: '/proxies/tcp-udp' });
  };

  if (isLoading) {
    return (
      <div className="space-y-6 max-w-5xl">
        <div className="flex items-center gap-4">
          <Skeleton className="size-8 rounded" />
          <Skeleton className="size-10 rounded-lg" />
          <div className="space-y-2">
            <Skeleton className="h-7 w-48" />
            <Skeleton className="h-4 w-32" />
          </div>
        </div>
        <Skeleton className="h-48 rounded-lg" />
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
        <Button onClick={() => navigate({ to: '/proxies/tcp-udp' })}>
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
          <Button variant="ghost" size="icon" onClick={() => navigate({ to: '/proxies/tcp-udp' })}>
            <ArrowLeft className="size-4" />
            <span className="sr-only">Back</span>
          </Button>
          <div className="flex items-center gap-3">
            <div className="flex items-center justify-center size-10 rounded bg-primary/10">
              <Network className="size-5 text-primary" />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <h1 className="text-2xl font-bold">{proxy.name}</h1>
                <Badge variant="outline">{proxy.protocol.toUpperCase()}</Badge>
                <Badge variant={proxy.is_active ? 'default' : 'secondary'}>
                  {proxy.is_active ? 'Active' : 'Inactive'}
                </Badge>
              </div>
              <p className="text-sm text-muted-foreground">Port {proxy.listen_port}</p>
            </div>
          </div>
        </div>
        <div className="flex items-center gap-2">
          {canUpdateProxies && (
            <Button
              onClick={() =>
                navigate({
                  to: '/proxies/tcp-udp/$l4ProxyId/edit',
                  params: { l4ProxyId: String(proxy.id) },
                })
              }
            >
              <Pencil className="size-4" />
              Edit
            </Button>
          )}
          {(canCreateProxies || canDeleteProxies) && (
            <DropdownMenu>
              <DropdownMenuTrigger
                render={<Button variant="ghost" size="icon" className="size-9" />}
              >
                <MoreHorizontal className="size-4" />
                <span className="sr-only">More actions</span>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-40">
                {canCreateProxies && (
                  <DropdownMenuItem
                    onClick={() =>
                      navigate({
                        to: '/proxies/tcp-udp/new',
                        search: { duplicate: proxy.id },
                      })
                    }
                  >
                    <Copy className="size-4" />
                    Duplicate
                  </DropdownMenuItem>
                )}
                {canDeleteProxies && (
                  <>
                    {canCreateProxies && <DropdownMenuSeparator />}
                    <DropdownMenuItem
                      onClick={() => setDeleteDialogOpen(true)}
                      className="text-destructive focus:text-destructive"
                    >
                      <Trash2 className="size-4" />
                      Delete
                    </DropdownMenuItem>
                  </>
                )}
              </DropdownMenuContent>
            </DropdownMenu>
          )}
        </div>
      </div>

      {/* Config card */}
      <L4ProxyConfigCard proxy={proxy} />

      {/* Details card */}
      <Card>
        <CardHeader>
          <CardTitle>Details</CardTitle>
        </CardHeader>
        <CardContent className="divide-y">
          <DetailRow label="Description">{proxy.description || '—'}</DetailRow>
          <DetailRow label="Generated config">
            <Link
              to="/caddy-config"
              className="text-sm text-primary underline-offset-4 hover:underline"
            >
              View full config
            </Link>
          </DetailRow>
          <DetailRow label="Created">{new Date(proxy.created_at).toLocaleString()}</DetailRow>
          <DetailRow label="Updated">{new Date(proxy.updated_at).toLocaleString()}</DetailRow>
        </CardContent>
      </Card>

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
