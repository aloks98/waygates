import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  Button,
} from '@e412/rnui-react';
import { useNavigate } from '@tanstack/react-router';
import { Plus } from 'lucide-react';
import { useState } from 'react';

import { ProxyGroupDataGrid } from '@/components/proxy-group';
import { usePermissions } from '@/hooks/use-permissions';
import { useProxyGroups } from '@/hooks/use-proxy-groups';
import type { ProxyGroup } from '@/types/proxy-group';

export function ProxyGroupsListPage() {
  const navigate = useNavigate();
  const { canCreateProxyGroups, canUpdateProxyGroups, canDeleteProxyGroups } = usePermissions();
  const { data, isLoading, remove } = useProxyGroups();

  const [deletingGroup, setDeletingGroup] = useState<ProxyGroup | null>(null);

  const handleEdit = (group: ProxyGroup) => {
    navigate({ to: '/proxy-groups/$groupId/edit', params: { groupId: String(group.id) } });
  };

  const handleDelete = async () => {
    if (!deletingGroup) return;
    await remove.mutateAsync(deletingGroup.id);
    setDeletingGroup(null);
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Proxy Groups</h1>
          <p className="text-sm text-muted-foreground">
            Share a base domain, SSL/security settings, and access control across a set of proxies.
          </p>
        </div>
        {canCreateProxyGroups && (
          <Button onClick={() => navigate({ to: '/proxy-groups/new' })}>
            <Plus className="size-4" />
            New Proxy Group
          </Button>
        )}
      </div>

      <ProxyGroupDataGrid
        data={data?.items ?? []}
        isLoading={isLoading}
        canUpdate={canUpdateProxyGroups}
        canDelete={canDeleteProxyGroups}
        onEdit={handleEdit}
        onDelete={setDeletingGroup}
        onRowClick={canUpdateProxyGroups ? handleEdit : undefined}
      />

      <AlertDialog open={!!deletingGroup} onOpenChange={(open) => !open && setDeletingGroup(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Proxy Group</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete <strong>{deletingGroup?.name}</strong>? This action
              cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDelete}
              disabled={remove.isPending}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {remove.isPending ? 'Deleting...' : 'Delete'}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
