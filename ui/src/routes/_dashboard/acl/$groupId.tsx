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
  CardDescription,
  CardHeader,
  CardHeading,
  CardTitle,
  Skeleton,
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from '@e412/titanium';
import { useNavigate, useParams } from '@tanstack/react-router';
import { format } from 'date-fns';
import {
  ArrowLeft,
  Calendar,
  Globe,
  Key,
  Network,
  Pencil,
  Shield,
  Trash2,
  User,
  Users,
} from 'lucide-react';
import { useState } from 'react';
import { ACLGroupFormModal } from '@/components/acl/acl-group-form-modal';
import { BasicAuthTab } from '@/components/acl/basic-auth-tab';
import { ExternalProvidersTab } from '@/components/acl/external-providers-tab';
import { GroupUsageTab } from '@/components/acl/group-usage-tab';
import { IPRulesTab } from '@/components/acl/ip-rules-tab';
import { WaygatesAuthTab } from '@/components/acl/waygates-auth-tab';
import { useACLGroup, useDeleteACLGroup } from '@/hooks';
import type { CombinationMode } from '@/types/acl';

function getCombinationModeLabel(mode: CombinationMode): string {
  switch (mode) {
    case 'any':
      return 'Any Match';
    case 'all':
      return 'All Required';
    case 'ip_bypass':
      return 'IP Bypass';
    default:
      return mode;
  }
}

function getCombinationModeDescription(mode: CombinationMode): string {
  switch (mode) {
    case 'any':
      return 'Access is granted if any authentication method succeeds';
    case 'all':
      return 'All configured authentication methods must succeed';
    case 'ip_bypass':
      return 'If IP matches allow rules, other auth methods are skipped';
    default:
      return '';
  }
}

function OverviewTab({ groupId }: { groupId: number }) {
  const { group, isLoading } = useACLGroup(groupId);

  if (isLoading) {
    return (
      <div className="grid gap-6 md:grid-cols-2">
        <Card>
          <CardHeader>
            <Skeleton className="h-6 w-32" />
          </CardHeader>
          <CardContent className="space-y-4">
            <Skeleton className="h-4 w-full" />
            <Skeleton className="h-4 w-3/4" />
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <Skeleton className="h-6 w-32" />
          </CardHeader>
          <CardContent className="space-y-4">
            <Skeleton className="h-4 w-full" />
            <Skeleton className="h-4 w-3/4" />
          </CardContent>
        </Card>
      </div>
    );
  }

  if (!group) {
    return <div className="text-center py-8 text-muted-foreground">Group not found</div>;
  }

  return (
    <div className="grid gap-6 md:grid-cols-2">
      <Card>
        <CardHeader>
          <CardHeading>
            <CardTitle className="flex items-center gap-2">
              <Shield className="size-4" />
              Group Details
            </CardTitle>
          </CardHeading>
        </CardHeader>
        <CardContent className="space-y-4">
          <div>
            <p className="text-xs text-muted-foreground uppercase tracking-wide mb-1">Name</p>
            <p className="font-medium">{group.name}</p>
          </div>
          {group.description && (
            <div>
              <p className="text-xs text-muted-foreground uppercase tracking-wide mb-1">
                Description
              </p>
              <p className="text-sm text-muted-foreground">{group.description}</p>
            </div>
          )}
          <div>
            <p className="text-xs text-muted-foreground uppercase tracking-wide mb-1">
              Combination Mode
            </p>
            <div className="flex items-center gap-2">
              <Badge variant="secondary">{getCombinationModeLabel(group.combination_mode)}</Badge>
            </div>
            <p className="text-xs text-muted-foreground mt-1">
              {getCombinationModeDescription(group.combination_mode)}
            </p>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardHeading>
            <CardTitle className="flex items-center gap-2">
              <Calendar className="size-4" />
              Metadata
            </CardTitle>
          </CardHeading>
        </CardHeader>
        <CardContent className="space-y-4">
          <div>
            <p className="text-xs text-muted-foreground uppercase tracking-wide mb-1">Created</p>
            <p className="text-sm">{format(new Date(group.created_at), 'PPpp')}</p>
          </div>
          <div>
            <p className="text-xs text-muted-foreground uppercase tracking-wide mb-1">
              Last Updated
            </p>
            <p className="text-sm">{format(new Date(group.updated_at), 'PPpp')}</p>
          </div>
          {group.created_by_user && (
            <div>
              <p className="text-xs text-muted-foreground uppercase tracking-wide mb-1">
                Created By
              </p>
              <div className="flex items-center gap-2">
                <User className="size-4 text-muted-foreground" />
                <span className="text-sm">{group.created_by_user.username}</span>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      <Card className="md:col-span-2">
        <CardHeader>
          <CardHeading>
            <CardTitle>Configuration Summary</CardTitle>
            <CardDescription>
              Overview of all authentication methods configured for this group
            </CardDescription>
          </CardHeading>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <div className="flex items-center gap-3 p-3 rounded-lg border bg-muted/30">
              <div className="flex items-center justify-center size-10 rounded-full bg-blue-500/10">
                <Network className="size-5 text-blue-500" />
              </div>
              <div>
                <p className="text-2xl font-bold">{group.ip_rules?.length ?? 0}</p>
                <p className="text-xs text-muted-foreground">IP Rules</p>
              </div>
            </div>
            <div className="flex items-center gap-3 p-3 rounded-lg border bg-muted/30">
              <div className="flex items-center justify-center size-10 rounded-full bg-green-500/10">
                <Key className="size-5 text-green-500" />
              </div>
              <div>
                <p className="text-2xl font-bold">{group.basic_auth_users?.length ?? 0}</p>
                <p className="text-xs text-muted-foreground">Basic Auth Users</p>
              </div>
            </div>
            <div className="flex items-center gap-3 p-3 rounded-lg border bg-muted/30">
              <div className="flex items-center justify-center size-10 rounded-full bg-purple-500/10">
                <Users className="size-5 text-purple-500" />
              </div>
              <div>
                <p className="text-2xl font-bold">{group.waygates_auth?.enabled ? 'Yes' : 'No'}</p>
                <p className="text-xs text-muted-foreground">Waygates Auth</p>
              </div>
            </div>
            <div className="flex items-center gap-3 p-3 rounded-lg border bg-muted/30">
              <div className="flex items-center justify-center size-10 rounded-full bg-orange-500/10">
                <Globe className="size-5 text-orange-500" />
              </div>
              <div>
                <p className="text-2xl font-bold">{group.external_providers?.length ?? 0}</p>
                <p className="text-xs text-muted-foreground">External Providers</p>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

export function ACLGroupDetailPage() {
  const params = useParams({ from: '/dashboard/acl/$groupId' });
  const groupId = parseInt(params.groupId, 10);
  const navigate = useNavigate();

  const { group, isLoading } = useACLGroup(groupId);
  const { deleteGroup, isDeleting } = useDeleteACLGroup();

  const [editModalOpen, setEditModalOpen] = useState(false);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);

  const handleDelete = async () => {
    await deleteGroup(groupId);
    setDeleteDialogOpen(false);
    navigate({ to: '/dashboard/acl' });
  };

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div className="flex items-center gap-4">
          <Skeleton className="size-8" />
          <Skeleton className="h-8 w-48" />
        </div>
        <div className="grid gap-4 md:grid-cols-2">
          <Skeleton className="h-48" />
          <Skeleton className="h-48" />
        </div>
      </div>
    );
  }

  if (!group) {
    return (
      <div className="flex flex-col items-center justify-center py-12 space-y-4">
        <Shield className="size-12 text-muted-foreground" />
        <h2 className="text-xl font-semibold">Group Not Found</h2>
        <p className="text-muted-foreground">
          The ACL group you're looking for doesn't exist or has been deleted.
        </p>
        <Button onClick={() => navigate({ to: '/dashboard/acl' })}>
          <ArrowLeft className="size-4" />
          Back to Groups
        </Button>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" onClick={() => navigate({ to: '/dashboard/acl' })}>
            <ArrowLeft className="size-4" />
            <span className="sr-only">Back</span>
          </Button>
          <div className="flex items-center gap-3">
            <div className="flex items-center justify-center size-10 rounded-lg bg-primary/10">
              <Shield className="size-5 text-primary" />
            </div>
            <div>
              <h1 className="text-2xl font-bold">{group.name}</h1>
              {group.description && (
                <p className="text-sm text-muted-foreground">{group.description}</p>
              )}
            </div>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" onClick={() => setEditModalOpen(true)}>
            <Pencil className="size-4" />
            Edit
          </Button>
          <Button
            variant="outline"
            className="text-destructive hover:text-destructive"
            onClick={() => setDeleteDialogOpen(true)}
          >
            <Trash2 className="size-4" />
            Delete
          </Button>
        </div>
      </div>

      <Tabs defaultValue="overview">
        <TabsList variant="line">
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="ip-rules">IP Rules</TabsTrigger>
          <TabsTrigger value="basic-auth">Basic Auth</TabsTrigger>
          <TabsTrigger value="waygates-auth">Waygates Auth</TabsTrigger>
          <TabsTrigger value="external">External Providers</TabsTrigger>
          <TabsTrigger value="usage">Usage</TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="mt-6">
          <OverviewTab groupId={groupId} />
        </TabsContent>

        <TabsContent value="ip-rules" className="mt-6">
          <IPRulesTab groupId={groupId} />
        </TabsContent>

        <TabsContent value="basic-auth" className="mt-6">
          <BasicAuthTab groupId={groupId} />
        </TabsContent>

        <TabsContent value="waygates-auth" className="mt-6">
          <WaygatesAuthTab groupId={groupId} />
        </TabsContent>

        <TabsContent value="external" className="mt-6">
          <ExternalProvidersTab groupId={groupId} />
        </TabsContent>

        <TabsContent value="usage" className="mt-6">
          <GroupUsageTab groupId={groupId} />
        </TabsContent>
      </Tabs>

      <ACLGroupFormModal
        open={editModalOpen}
        onOpenChange={setEditModalOpen}
        mode="edit"
        initialData={group}
      />

      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete ACL Group</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete <strong>{group.name}</strong>? This will remove all
              associated IP rules, authentication settings, and provider configurations. Proxies
              using this group will no longer be protected. This action cannot be undone.
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
