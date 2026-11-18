import { useNavigate, useParams } from '@tanstack/react-router';
import { format } from 'date-fns';
import { ArrowLeft, ArrowRight, ChevronRight, Pencil, Shield, Trash2 } from 'lucide-react';
import { useState } from 'react';

import { getModeLabel } from '@/components/acl/access-labels';
import { ACLGroupFormModal } from '@/components/acl/acl-group-form-modal';
import { BasicAuthTab } from '@/components/acl/basic-auth-tab';
import { ExternalProvidersTab } from '@/components/acl/external-providers-tab';
import { GroupUsageTab } from '@/components/acl/group-usage-tab';
import { IPRulesTab } from '@/components/acl/ip-rules-tab';
import { OAuthSSOTab } from '@/components/acl/oauth-sso-tab';
import { WaygatesAccountTab } from '@/components/acl/waygates-account-tab';
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
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { DetailRow } from '@/components/ui/detail-row';
import { Skeleton } from '@/components/ui/skeleton';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { useACLGroup, useDeleteACLGroup } from '@/hooks';
import { usePermissions } from '@/hooks/use-permissions';

function OverviewTab({
  groupId,
  onTabChange,
}: {
  groupId: number;
  onTabChange: (tab: string) => void;
}) {
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

  const methods = [
    { key: 'ip-rules', label: 'IP rules', count: group.ip_rules?.length ?? 0 },
    { key: 'basic-auth', label: 'Basic auth', count: group.basic_auth_users?.length ?? 0 },
    {
      key: 'oauth-sso',
      label: 'OAuth / SSO',
      count: group.waygates_auth?.allowed_providers?.length ?? 0,
    },
    {
      key: 'account',
      label: 'Waygates account',
      count: group.waygates_auth?.enabled ? 'Enabled' : 'Disabled',
    },
    { key: 'forward-auth', label: 'Forward auth', count: group.external_providers?.length ?? 0 },
  ];
  return (
    <div className="grid items-start gap-6 lg:grid-cols-2">
      <Card>
        <CardHeader>
          <CardTitle>Group details</CardTitle>
        </CardHeader>
        <CardContent>
          <dl className="divide-y">
            <DetailRow label="Name">{group.name}</DetailRow>
            {group.description && <DetailRow label="Description">{group.description}</DetailRow>}
            <DetailRow label="How methods combine">
              {getModeLabel(group.combination_mode)}
            </DetailRow>
            <DetailRow label="Updated">{format(new Date(group.updated_at), 'PP p')}</DetailRow>
            <DetailRow label="Created">{format(new Date(group.created_at), 'PP p')}</DetailRow>
            {group.created_by_user && (
              <DetailRow label="Created by">{group.created_by_user.username}</DetailRow>
            )}
          </dl>
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle>Authentication methods</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-3">
          {methods.map((method) => (
            <Button
              key={method.key}
              variant="outline"
              className="h-11 w-full justify-between"
              onClick={() => onTabChange(method.key)}
            >
              <span>{method.label}</span>
              <span className="flex items-center gap-3 text-muted-foreground">
                <span className="text-xs">{method.count}</span>
                <ChevronRight className="size-4" />
              </span>
            </Button>
          ))}
          <Button variant="ghost" className="justify-start" onClick={() => onTabChange('usage')}>
            View assigned proxies
            <ArrowRight data-icon="inline-end" />
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}

export function ACLGroupDetailPage() {
  const params = useParams({ strict: false });
  const groupId = parseInt(params.groupId, 10);
  const navigate = useNavigate();

  const { group, isLoading } = useACLGroup(groupId);
  const { deleteGroup, isDeleting } = useDeleteACLGroup();
  const { canUpdateAccess, canDeleteAccess } = usePermissions();

  const [editModalOpen, setEditModalOpen] = useState(false);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [activeTab, setActiveTab] = useState('overview');

  const handleDelete = async () => {
    await deleteGroup(groupId);
    setDeleteDialogOpen(false);
    navigate({ to: '/access' });
  };

  if (isLoading) {
    return (
      <div className="flex flex-col gap-6">
        <div className="flex min-w-0 items-center gap-4">
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
          The access control group you're looking for doesn't exist or has been deleted.
        </p>
        <Button onClick={() => navigate({ to: '/access' })}>
          <ArrowLeft className="size-4" />
          Back to Groups
        </Button>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex min-w-0 items-center gap-4">
          <Button variant="ghost" size="icon" onClick={() => navigate({ to: '/access' })}>
            <ArrowLeft className="size-4" />
            <span className="sr-only">Back</span>
          </Button>
          <div className="flex items-center gap-3">
            <div className="flex items-center justify-center size-10 rounded bg-primary/10">
              <Shield className="size-5 text-primary" />
            </div>
            <div>
              <h1 className="text-[28px] font-semibold">{group.name}</h1>
              {group.description && (
                <p className="text-sm text-muted-foreground">{group.description}</p>
              )}
            </div>
          </div>
        </div>
        {(canUpdateAccess || canDeleteAccess) && (
          <div className="flex items-center gap-2">
            {canUpdateAccess && (
              <Button variant="outline" onClick={() => setEditModalOpen(true)}>
                <Pencil className="size-4" />
                Edit
              </Button>
            )}
            {canDeleteAccess && (
              <Button
                variant="outline"
                className="text-destructive hover:text-destructive"
                onClick={() => setDeleteDialogOpen(true)}
              >
                <Trash2 className="size-4" />
                Delete
              </Button>
            )}
          </div>
        )}
      </div>

      <Tabs value={activeTab} onValueChange={setActiveTab}>
        <div className="max-w-full overflow-x-auto pb-1">
          <TabsList variant="line">
            <TabsTrigger value="overview">Overview</TabsTrigger>
            <TabsTrigger value="ip-rules">IP Rules</TabsTrigger>
            <TabsTrigger value="basic-auth">Basic Auth</TabsTrigger>
            <TabsTrigger value="oauth-sso">OAuth / SSO</TabsTrigger>
            <TabsTrigger value="account">Waygates Account</TabsTrigger>
            <TabsTrigger value="forward-auth">Forward Auth</TabsTrigger>
            <TabsTrigger value="usage">Usage</TabsTrigger>
          </TabsList>
        </div>

        <TabsContent value="overview" className="mt-6">
          <OverviewTab groupId={groupId} onTabChange={setActiveTab} />
        </TabsContent>

        <TabsContent value="ip-rules" className="mt-6">
          <IPRulesTab groupId={groupId} />
        </TabsContent>

        <TabsContent value="basic-auth" className="mt-6">
          <BasicAuthTab groupId={groupId} />
        </TabsContent>

        <TabsContent value="oauth-sso" className="mt-6">
          <OAuthSSOTab groupId={groupId} />
        </TabsContent>

        <TabsContent value="account" className="mt-6">
          <WaygatesAccountTab groupId={groupId} />
        </TabsContent>

        <TabsContent value="forward-auth" className="mt-6">
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
            <AlertDialogTitle>Delete Access Group</AlertDialogTitle>
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
