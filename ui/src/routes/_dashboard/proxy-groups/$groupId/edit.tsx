import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  Alert,
  AlertDescription,
  Badge,
  Button,
  Skeleton,
} from '@e412/rnui-react';
import { useNavigate, useParams } from '@tanstack/react-router';
import { ArrowLeft, Info, Layers, Trash2 } from 'lucide-react';
import { useMemo, useState } from 'react';

import { ProxyGroupForm } from '@/components/proxy-group';
import { type ACLAssignment, ACLSelector } from '@/components/proxy/forms/acl-selector';
import { useProxyGroup, useProxyGroupAcl, useProxyGroups } from '@/hooks/use-proxy-groups';
import type { CreateProxyGroupRequest } from '@/types/proxy-group';

export function ProxyGroupEditPage() {
  const params = useParams({ strict: false });
  const groupId = parseInt(params.groupId, 10);
  const navigate = useNavigate();

  const { data: group, isLoading } = useProxyGroup(groupId);
  const { update, remove } = useProxyGroups();
  const {
    data: groupAcl,
    assign,
    update: updateAcl,
    remove: removeAcl,
  } = useProxyGroupAcl(groupId);

  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);

  const aclAssignments: ACLAssignment[] = useMemo(
    () =>
      groupAcl.map((a) => ({
        acl_group_id: a.acl_group_id,
        path_pattern: a.path_pattern,
        priority: a.priority,
        enabled: a.enabled,
      })),
    [groupAcl],
  );

  // ACLSelector is the same component the proxy edit page uses — only the
  // mutation target differs. There, onChange buffers into local state until
  // the proxy form submits; here, the group ACL endpoints invalidate on
  // their own, so onChange diffs against server state and persists
  // immediately: assign for group ids that are new, remove for group ids
  // that dropped out, and update (PUT) for a group id that's still present
  // but had its enabled/priority/path_pattern edited — without that last
  // case, toggling an existing assignment would silently not persist.
  const handleAclChange = async (next: ACLAssignment[]) => {
    const nextIds = next.map((a) => a.acl_group_id);

    for (const prev of groupAcl) {
      if (!nextIds.includes(prev.acl_group_id)) {
        await removeAcl.mutateAsync(prev.acl_group_id);
      }
    }

    for (const assignment of next) {
      const prev = groupAcl.find((a) => a.acl_group_id === assignment.acl_group_id);
      if (!prev) {
        await assign.mutateAsync({
          acl_group_id: assignment.acl_group_id,
          path_pattern: assignment.path_pattern,
          priority: assignment.priority,
        });
      } else if (
        prev.enabled !== assignment.enabled ||
        prev.priority !== assignment.priority ||
        prev.path_pattern !== assignment.path_pattern
      ) {
        await updateAcl.mutateAsync({
          assignmentId: prev.id,
          path_pattern: assignment.path_pattern,
          priority: assignment.priority,
          enabled: assignment.enabled,
        });
      }
    }
  };

  const handleUpdate = async (data: CreateProxyGroupRequest) => {
    if (!group) return;
    await update.mutateAsync({ id: group.id, ...data });
  };

  const handleDelete = async () => {
    await remove.mutateAsync(groupId);
    setDeleteDialogOpen(false);
    navigate({ to: '/proxy-groups' });
  };

  const handleCancel = () => {
    navigate({ to: '/proxy-groups' });
  };

  if (isLoading) {
    return (
      <div className="space-y-6 max-w-3xl">
        <div className="flex items-center gap-4">
          <Skeleton className="size-8 rounded" />
          <Skeleton className="h-7 w-48" />
        </div>
        <Skeleton className="h-64 rounded-lg" />
        <Skeleton className="h-48 rounded-lg" />
      </div>
    );
  }

  if (!group) {
    return (
      <div className="flex flex-col items-center justify-center py-12 space-y-4">
        <Layers className="size-12 text-muted-foreground" />
        <h2 className="text-xl font-semibold">Proxy Group Not Found</h2>
        <p className="text-muted-foreground">
          The proxy group you're looking for doesn't exist or has been deleted.
        </p>
        <Button onClick={() => navigate({ to: '/proxy-groups' })}>
          <ArrowLeft className="size-4" />
          Back to Proxy Groups
        </Button>
      </div>
    );
  }

  return (
    <div className="space-y-6 max-w-3xl">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" onClick={() => navigate({ to: '/proxy-groups' })}>
            <ArrowLeft className="size-4" />
            <span className="sr-only">Back</span>
          </Button>
          <div className="flex items-center gap-3">
            <div className="flex items-center justify-center size-10 rounded bg-primary/10">
              <Layers className="size-5 text-primary" />
            </div>
            <div>
              <h1 className="text-2xl font-bold">{group.name}</h1>
              <p className="text-sm text-muted-foreground">
                <Badge variant="outline" className="text-xs">
                  {group.member_count} {group.member_count === 1 ? 'member' : 'members'}
                </Badge>
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

      <ProxyGroupForm
        mode="edit"
        initialData={group}
        onSubmit={handleUpdate}
        loading={update.isPending}
        onCancel={handleCancel}
      />

      <Alert variant="info">
        <Info className="size-4" />
        <AlertDescription>
          These access rules apply to every member proxy that has not set its own overriding ACL
          assignments. A proxy's own assignments always take priority over what's configured here —
          check the member proxy's edit page if you're not sure what applies.
        </AlertDescription>
      </Alert>

      <ACLSelector value={aclAssignments} onChange={handleAclChange} disabled={update.isPending} />

      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Proxy Group</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete <strong>{group.name}</strong>? This action cannot be
              undone. Groups with member proxies can't be deleted — reassign or remove the members
              first.
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
