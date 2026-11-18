import { Link, useNavigate, useParams } from '@tanstack/react-router';
import { ArrowLeft, Copy, Globe, MoreHorizontal, Pencil, Trash2 } from 'lucide-react';
import { useState } from 'react';

import { getProxyTypeLabel } from '@/components/proxy';
import { getProxyTypeIcon } from '@/components/proxy/cells';
import { ProxyConfigCard } from '@/components/proxy/overview/proxy-config-card';
import { ProxyConfigPreviewCard } from '@/components/proxy/overview/proxy-config-preview-card';
import { ProxyConnectionCard } from '@/components/proxy/overview/proxy-connection-card';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog';
import { Badge } from '@/components/ui/badge';
import { Button, buttonVariants } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { Skeleton } from '@/components/ui/skeleton';
import { usePermissions } from '@/hooks/use-permissions';
import { useProxies, useProxy } from '@/hooks/use-proxies';

export function ProxyOverviewPage() {
  const params = useParams({ strict: false });
  const proxyId = parseInt(params.proxyId, 10);
  const navigate = useNavigate();
  const { proxy, isLoading } = useProxy(proxyId);
  const { remove, isDeleting } = useProxies();
  const { canCreateProxies, canDeleteProxies, canUpdateProxies } = usePermissions();

  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);

  const handleDelete = async () => {
    await remove(proxyId);
    setDeleteDialogOpen(false);
    navigate({ to: '/proxies' });
  };

  if (isLoading) {
    return (
      <div className="flex max-w-[1440px] flex-col gap-6">
        <div className="flex min-w-0 flex-wrap items-center gap-4">
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
        <Globe className="size-12 text-muted-foreground" />
        <h2 className="text-xl font-semibold">Proxy Not Found</h2>
        <p className="text-muted-foreground">
          The proxy you're looking for doesn't exist or has been deleted.
        </p>
        <Button onClick={() => navigate({ to: '/proxies' })}>
          <ArrowLeft className="size-4" />
          Back to Proxies
        </Button>
      </div>
    );
  }

  return (
    <div className="flex max-w-[1440px] flex-col gap-6">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div className="flex min-w-0 flex-wrap items-center gap-4">
          <Button variant="ghost" size="icon" onClick={() => navigate({ to: '/proxies' })}>
            <ArrowLeft className="size-4" />
            <span className="sr-only">Back</span>
          </Button>
          <div className="flex min-w-0 items-center gap-3">
            <div className="flex items-center justify-center size-10 rounded bg-primary/10">
              {getProxyTypeIcon(proxy.type)}
            </div>
            <div>
              <div className="flex flex-wrap items-center gap-2">
                <h1 className="break-words text-[28px] font-semibold">{proxy.name}</h1>
                <Badge variant={proxy.is_active ? 'default' : 'secondary'}>
                  {proxy.is_active ? 'Active' : 'Inactive'}
                </Badge>
                {proxy.ssl_enabled && <Badge variant="outline">HTTPS</Badge>}
              </div>
              <p className="text-sm text-muted-foreground">
                {getProxyTypeLabel(proxy.type)} &middot; {proxy.hostname}
              </p>
            </div>
          </div>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          {canUpdateProxies && (
            <Button
              onClick={() =>
                navigate({
                  to: '/proxies/$proxyId/edit',
                  params: { proxyId: String(proxy.id) },
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
                        to: '/proxies/new',
                        search: { type: proxy.type, duplicate: proxy.id },
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

      <div className="grid items-start gap-6 xl:grid-cols-[minmax(0,1.8fr)_minmax(280px,1fr)]">
        <div className="flex min-w-0 flex-col gap-3">
          <ProxyConfigCard proxy={proxy} />
          <div className="flex flex-wrap items-center justify-between gap-x-4 gap-y-2 text-xs text-muted-foreground">
            <dl className="flex flex-wrap gap-x-4 gap-y-2">
              <div className="flex flex-wrap gap-x-1">
                <dt>Updated</dt>
                <dd>
                  <time dateTime={proxy.updated_at}>
                    {new Date(proxy.updated_at).toLocaleString()}
                  </time>
                </dd>
              </div>
              <div className="flex flex-wrap gap-x-1">
                <dt>Created</dt>
                <dd>
                  <time dateTime={proxy.created_at}>
                    {new Date(proxy.created_at).toLocaleString()}
                  </time>
                </dd>
              </div>
            </dl>
            <Link
              to="/activity"
              className={buttonVariants({ variant: 'link', size: 'xs', className: 'px-0' })}
            >
              View activity
            </Link>
          </div>
        </div>
        <aside className="flex min-w-0 flex-col gap-5">
          <ProxyConnectionCard proxy={proxy} />
          <ProxyConfigPreviewCard proxyId={proxy.id} />
        </aside>
      </div>

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
