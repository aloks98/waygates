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
} from '@e412/titanium';
import { useNavigate, useParams } from '@tanstack/react-router';
import { ArrowLeft, Globe, Trash2 } from 'lucide-react';
import { useCallback, useMemo, useState } from 'react';

import {
  type ACLAssignment,
  getProxyTypeLabel,
  RedirectForm,
  ReverseProxyForm,
  StaticForm,
} from '@/components/proxy';
import { getProxyTypeIcon } from '@/components/proxy/cells';
import { useAssignACL, useProxyACL, useRemoveACL } from '@/hooks';
import { useProxies, useProxy } from '@/hooks/use-proxies';
import type { CreateProxyRequest, ProxyConfig } from '@/types/proxy';

export function ProxyDetailPage() {
  const params = useParams({ from: '/dashboard/proxies/$proxyId' });
  const proxyId = parseInt(params.proxyId, 10);
  const navigate = useNavigate();

  const { proxy, isLoading } = useProxy(proxyId);
  const { update, remove, isUpdating, isDeleting } = useProxies();
  const { assignACL } = useAssignACL();
  const { removeACL } = useRemoveACL();
  const { assignments: proxyACL } = useProxyACL(proxyId);

  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);

  const aclAssignments: ACLAssignment[] = useMemo(() => {
    return proxyACL.map((a) => ({
      acl_group_id: a.acl_group_id,
      acl_group_name: a.acl_group?.name,
      path_pattern: a.path_pattern,
      priority: a.priority,
      enabled: a.enabled,
    }));
  }, [proxyACL]);

  const hasProxyDataChanged = useCallback(
    (newData: CreateProxyRequest, originalProxy: ProxyConfig): boolean => {
      if (newData.name !== originalProxy.name) return true;
      if (newData.hostname !== originalProxy.hostname) return true;
      if (newData.type !== originalProxy.type) return true;
      if (newData.ssl_enabled !== originalProxy.ssl_enabled) return true;
      if (newData.block_exploits !== originalProxy.block_exploits) return true;
      if (newData.tls_insecure_skip_verify !== originalProxy.tls_insecure_skip_verify) return true;

      if (originalProxy.type === 'reverse_proxy') {
        const oldUpstreams = originalProxy.upstreams || [];
        const newUpstreams = newData.upstreams || [];
        if (JSON.stringify(oldUpstreams) !== JSON.stringify(newUpstreams)) return true;
        const oldHeaders = originalProxy.custom_headers || [];
        const newHeaders = newData.custom_headers || [];
        if (JSON.stringify(oldHeaders) !== JSON.stringify(newHeaders)) return true;
        if (newData.health_check_path !== originalProxy.health_check_path) return true;
        if (newData.load_balancing !== originalProxy.load_balancing) return true;
      }

      if (originalProxy.type === 'redirect') {
        if (newData.redirect_url !== originalProxy.redirect_url) return true;
        if (newData.redirect_code !== originalProxy.redirect_code) return true;
      }

      if (originalProxy.type === 'static') {
        if (newData.root_path !== originalProxy.root_path) return true;
        if (
          JSON.stringify(newData.try_files || []) !== JSON.stringify(originalProxy.try_files || [])
        )
          return true;
        if (newData.browse !== originalProxy.browse) return true;
        if (newData.index !== originalProxy.index) return true;
      }

      return false;
    },
    [],
  );

  const handleUpdate = async (data: CreateProxyRequest, newACLAssignments?: ACLAssignment[]) => {
    if (!proxy) return;

    if (hasProxyDataChanged(data, proxy)) {
      await update({ id: proxy.id, data });
    }

    const assignments = newACLAssignments ?? [];
    const oldGroupIds = proxyACL.map((a) => a.acl_group_id);
    const newGroupIds = assignments.map((a) => a.acl_group_id);

    for (const oldAssignment of proxyACL) {
      if (!newGroupIds.includes(oldAssignment.acl_group_id)) {
        await removeACL({ proxyId: proxy.id, groupId: oldAssignment.acl_group_id });
      }
    }

    for (const newAssignment of assignments) {
      if (!oldGroupIds.includes(newAssignment.acl_group_id)) {
        await assignACL({
          proxyId: proxy.id,
          data: {
            acl_group_id: newAssignment.acl_group_id,
            path_pattern: newAssignment.path_pattern,
            priority: newAssignment.priority,
            enabled: newAssignment.enabled,
          },
        });
      }
    }
  };

  const handleDelete = async () => {
    await remove(proxyId);
    setDeleteDialogOpen(false);
    navigate({ to: '/dashboard/proxies' });
  };

  const handleCancel = () => {
    navigate({ to: '/dashboard/proxies' });
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
        <Globe className="size-12 text-muted-foreground" />
        <h2 className="text-xl font-semibold">Proxy Not Found</h2>
        <p className="text-muted-foreground">
          The proxy you're looking for doesn't exist or has been deleted.
        </p>
        <Button onClick={() => navigate({ to: '/dashboard/proxies' })}>
          <ArrowLeft className="size-4" />
          Back to Proxies
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
            onClick={() => navigate({ to: '/dashboard/proxies' })}
          >
            <ArrowLeft className="size-4" />
            <span className="sr-only">Back</span>
          </Button>
          <div className="flex items-center gap-3">
            <div className="flex items-center justify-center size-10 rounded-lg bg-primary/10">
              {getProxyTypeIcon(proxy.type)}
            </div>
            <div>
              <div className="flex items-center gap-2">
                <h1 className="text-2xl font-bold">{proxy.name}</h1>
                <Badge variant={proxy.is_active ? 'default' : 'secondary'}>
                  {proxy.is_active ? 'Active' : 'Inactive'}
                </Badge>
              </div>
              <p className="text-sm text-muted-foreground">
                {getProxyTypeLabel(proxy.type)} &middot; {proxy.hostname}
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
      {proxy.type === 'reverse_proxy' && (
        <ReverseProxyForm
          initialData={proxy}
          initialACLAssignments={aclAssignments}
          onSubmit={handleUpdate}
          loading={isUpdating}
          onCancel={handleCancel}
        />
      )}
      {proxy.type === 'redirect' && (
        <RedirectForm
          initialData={proxy}
          initialACLAssignments={aclAssignments}
          onSubmit={handleUpdate}
          loading={isUpdating}
          onCancel={handleCancel}
        />
      )}
      {proxy.type === 'static' && (
        <StaticForm
          initialData={proxy}
          initialACLAssignments={aclAssignments}
          onSubmit={handleUpdate}
          loading={isUpdating}
          onCancel={handleCancel}
        />
      )}

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
