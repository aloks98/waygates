import { zodResolver } from '@hookform/resolvers/zod';
import { format } from 'date-fns';
import { ExternalLink, Globe, Pencil, Plus, Shield, Trash2 } from 'lucide-react';
import { useEffect, useState } from 'react';
import { useForm, useWatch } from 'react-hook-form';
import { z } from 'zod';

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
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardAction, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectGroup,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Skeleton } from '@/components/ui/skeleton';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { TagsInput } from '@/components/ui/tags-input';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';

const FORWARD_AUTH_HEADER_SUGGESTIONS = [
  'Remote-User',
  'Remote-Name',
  'Remote-Email',
  'Remote-Groups',
  'X-Forwarded-User',
  'X-Authentik-Username',
  'X-Authentik-Groups',
  'X-Authentik-Email',
];
import {
  useAddExternalProvider,
  useDeleteExternalProvider,
  useExternalProviders,
  useUpdateExternalProvider,
} from '@/hooks';
import { usePermissions } from '@/hooks/use-permissions';
import type { ACLExternalProvider, ProviderType } from '@/types/acl';

const externalProviderSchema = z.object({
  provider_type: z.enum(['authelia', 'authentik', 'custom']),
  name: z.string().min(1, 'Name is required').max(100, 'Name must be at most 100 characters'),
  verify_url: z.string().min(1, 'Verify URL is required').url('Must be a valid URL'),
  outpost_url: z.string().url('Must be a valid URL').optional().or(z.literal('')),
  headers_to_copy: z.array(z.string()).optional(),
});

type ExternalProviderFormValues = z.infer<typeof externalProviderSchema>;

const PROVIDER_TYPE_OPTIONS = [
  { value: 'authelia', label: 'Authelia' },
  { value: 'authentik', label: 'Authentik' },
  { value: 'custom', label: 'Custom Provider' },
] as const;

function getProviderTypeLabel(type: ProviderType): string {
  switch (type) {
    case 'authelia':
      return 'Authelia';
    case 'authentik':
      return 'Authentik';
    case 'custom':
      return 'Custom';
    default:
      return type;
  }
}

function getProviderTypeBadgeVariant(
  type: ProviderType,
): 'default' | 'secondary' | 'outline' | 'destructive' {
  switch (type) {
    case 'authelia':
      return 'default';
    case 'authentik':
      return 'secondary';
    case 'custom':
      return 'outline';
    default:
      return 'outline';
  }
}

interface ProviderFormModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  groupId: number;
  mode: 'create' | 'edit';
  initialData?: ACLExternalProvider | null;
}

function ProviderFormModal({
  open,
  onOpenChange,
  groupId,
  mode,
  initialData,
}: ProviderFormModalProps) {
  const { addProvider, isAdding } = useAddExternalProvider();
  const { updateProvider, isUpdating } = useUpdateExternalProvider();

  const isLoading = isAdding || isUpdating;
  const isEditMode = mode === 'edit' && initialData;

  const form = useForm<ExternalProviderFormValues>({
    resolver: zodResolver(externalProviderSchema),
    defaultValues: {
      provider_type: 'authelia' as ProviderType,
      name: '',
      verify_url: '',
      outpost_url: '',
      headers_to_copy: [],
    },
  });

  useEffect(() => {
    if (!open) return;
    if (isEditMode) {
      form.reset({
        provider_type: initialData.provider_type,
        name: initialData.name,
        verify_url: initialData.verify_url,
        outpost_url: initialData.outpost_url || '',
        headers_to_copy: initialData.headers_to_copy || [],
      });
    } else {
      form.reset({
        provider_type: 'authelia',
        name: '',
        verify_url: '',
        outpost_url: '',
        headers_to_copy: [],
      });
    }
  }, [open, isEditMode, initialData, form]);

  const isAuthentik = useWatch({ control: form.control, name: 'provider_type' }) === 'authentik';

  const handleOpenChange = (isOpen: boolean) => {
    if (!isOpen) {
      form.reset();
    }
    onOpenChange(isOpen);
  };

  const onSubmit = async (value: ExternalProviderFormValues) => {
    const data = {
      provider_type: value.provider_type,
      name: value.name,
      verify_url: value.verify_url,
      outpost_url: value.provider_type === 'authentik' ? value.outpost_url || undefined : undefined,
      headers_to_copy: value.headers_to_copy?.length ? value.headers_to_copy : undefined,
    };

    if (isEditMode) {
      await updateProvider({
        id: initialData.id,
        groupId,
        data,
      });
    } else {
      await addProvider({
        groupId,
        data,
      });
    }
    onOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Globe className="size-5" />
            {isEditMode ? 'Edit Forward Auth Provider' : 'Add Forward Auth Provider'}
          </DialogTitle>
        </DialogHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)}>
            <div className="grid gap-4 py-2">
              <div className="grid gap-4">
                <FormField
                  control={form.control}
                  name="provider_type"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Provider Type</FormLabel>
                      <Select
                        items={PROVIDER_TYPE_OPTIONS}
                        value={field.value}
                        onValueChange={field.onChange}
                      >
                        <FormControl>
                          <SelectTrigger>
                            <SelectValue />
                          </SelectTrigger>
                        </FormControl>
                        <SelectContent>
                          <SelectGroup>
                            {PROVIDER_TYPE_OPTIONS.map((option) => (
                              <SelectItem key={option.value} value={option.value}>
                                {option.label}
                              </SelectItem>
                            ))}
                          </SelectGroup>
                        </SelectContent>
                      </Select>

                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name="name"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Name</FormLabel>
                      <FormControl>
                        <Input placeholder="e.g., Corporate SSO" {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name="verify_url"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Verify URL</FormLabel>
                      <FormControl>
                        <Input
                          type="url"
                          placeholder="https://auth.example.com/api/verify"
                          {...field}
                        />
                      </FormControl>

                      <FormDescription>
                        Full HTTP(S) verification URL, including path and query. For Authentik, use
                        /outpost.goauthentik.io/auth/caddy. The provider controls login redirects,
                        cookies and access-denied responses.
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                {isAuthentik ? (
                  <FormField
                    control={form.control}
                    name="outpost_url"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Outpost URL (optional)</FormLabel>
                        <FormControl>
                          <Input type="url" placeholder="https://id.nexus.e412.in" {...field} />
                        </FormControl>
                        <FormDescription>
                          Origin only. Defaults to the Verify URL origin. Receives
                          /outpost.goauthentik.io/ callbacks and logout on the application host.
                        </FormDescription>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                ) : null}

                <FormField
                  control={form.control}
                  name="headers_to_copy"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Headers to Copy (optional)</FormLabel>
                      <FormControl>
                        <TagsInput
                          value={field.value ?? []}
                          onValueChange={field.onChange}
                          placeholder="e.g., Remote-User, Remote-Groups, Remote-Email"
                          delimiters={['Enter', ',']}
                          suggestions={FORWARD_AUTH_HEADER_SUGGESTIONS}
                        />
                      </FormControl>

                      <FormDescription>
                        Verified response headers sent to the application. Empty uses provider
                        defaults. Client-supplied identity headers are removed, including on IP
                        exceptions.
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </div>
            </div>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => handleOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={isLoading}>
                {isLoading ? 'Saving...' : isEditMode ? 'Save Changes' : 'Add Provider'}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
}

interface ExternalProvidersTabProps {
  groupId: number;
}

export function ExternalProvidersTab({ groupId }: ExternalProvidersTabProps) {
  const { providers, isLoading } = useExternalProviders(groupId);
  const { deleteProvider, isDeleting } = useDeleteExternalProvider();
  const { canUpdateAccess, canDeleteAccess } = usePermissions();

  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [editingProvider, setEditingProvider] = useState<ACLExternalProvider | null>(null);
  const [deletingProvider, setDeletingProvider] = useState<ACLExternalProvider | null>(null);

  const handleDelete = async () => {
    if (!deletingProvider) return;
    await deleteProvider({ id: deletingProvider.id, groupId });
    setDeletingProvider(null);
  };

  return (
    <>
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Globe className="size-5" />
            Forward Auth
          </CardTitle>

          {canUpdateAccess && (
            <CardAction>
              <Button onClick={() => setCreateModalOpen(true)}>
                <Plus className="size-4" />
                Add Provider
              </Button>
            </CardAction>
          )}
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="space-y-3">
              {[1, 2].map((i) => (
                <div key={i} className="flex items-center gap-4">
                  <Skeleton className="size-10 rounded" />
                  <Skeleton className="h-5 w-32" />
                  <Skeleton className="h-6 w-20" />
                  <Skeleton className="h-5 w-48 flex-1" />
                  <Skeleton className="h-8 w-16" />
                </div>
              ))}
            </div>
          ) : providers.length === 0 ? (
            <div className="text-center py-8 text-muted-foreground">
              <Shield className="size-12 mx-auto mb-4 opacity-50" />
              <p>No forward auth providers configured</p>
              <p className="text-sm mt-1">
                Add providers like Authelia or Authentik for forward auth integration
              </p>
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead className="w-28">Type</TableHead>
                  <TableHead>Verify URL</TableHead>
                  <TableHead className="w-32">Created</TableHead>
                  {(canUpdateAccess || canDeleteAccess) && (
                    <TableHead className="w-20 text-right">Actions</TableHead>
                  )}
                </TableRow>
              </TableHeader>
              <TableBody>
                {providers.map((provider) => (
                  <TableRow key={provider.id}>
                    <TableCell>
                      <div className="flex items-center gap-2">
                        <div className="flex items-center justify-center size-8 rounded bg-muted">
                          <Globe className="size-4 text-muted-foreground" />
                        </div>
                        <span className="font-medium">{provider.name}</span>
                      </div>
                    </TableCell>
                    <TableCell>
                      <Badge variant={getProviderTypeBadgeVariant(provider.provider_type)}>
                        {getProviderTypeLabel(provider.provider_type)}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center gap-2 max-w-xs">
                        <code className="text-xs bg-muted px-2 py-0.5 rounded truncate">
                          {provider.verify_url}
                        </code>
                        <Tooltip>
                          <TooltipTrigger
                            render={
                              <a
                                href={provider.verify_url}
                                target="_blank"
                                rel="noopener noreferrer"
                                className="text-muted-foreground hover:text-foreground"
                              />
                            }
                          >
                            <ExternalLink className="size-3" />
                          </TooltipTrigger>
                          <TooltipContent>Open URL</TooltipContent>
                        </Tooltip>
                      </div>
                    </TableCell>
                    <TableCell>
                      <Tooltip>
                        <TooltipTrigger
                          render={<span className="text-sm text-muted-foreground cursor-default" />}
                        >
                          {format(new Date(provider.created_at), 'MMM d, yyyy')}
                        </TooltipTrigger>
                        <TooltipContent>
                          {format(new Date(provider.created_at), 'PPpp')}
                        </TooltipContent>
                      </Tooltip>
                    </TableCell>
                    {(canUpdateAccess || canDeleteAccess) && (
                      <TableCell>
                        <div className="flex justify-end gap-1">
                          {canUpdateAccess && (
                            <Tooltip>
                              <TooltipTrigger
                                render={
                                  <Button
                                    variant="ghost"
                                    size="sm"
                                    className="size-8 p-0"
                                    onClick={() => setEditingProvider(provider)}
                                  />
                                }
                              >
                                <Pencil className="size-4" />
                                <span className="sr-only">Edit</span>
                              </TooltipTrigger>
                              <TooltipContent>Edit provider</TooltipContent>
                            </Tooltip>
                          )}
                          {canDeleteAccess && (
                            <Tooltip>
                              <TooltipTrigger
                                render={
                                  <Button
                                    variant="ghost"
                                    size="sm"
                                    className="size-8 p-0 text-destructive hover:text-destructive"
                                    onClick={() => setDeletingProvider(provider)}
                                  />
                                }
                              >
                                <Trash2 className="size-4" />
                                <span className="sr-only">Delete</span>
                              </TooltipTrigger>
                              <TooltipContent>Delete provider</TooltipContent>
                            </Tooltip>
                          )}
                        </div>
                      </TableCell>
                    )}
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      <ProviderFormModal
        open={createModalOpen}
        onOpenChange={setCreateModalOpen}
        groupId={groupId}
        mode="create"
      />

      {editingProvider && (
        <ProviderFormModal
          open={!!editingProvider}
          onOpenChange={(open) => !open && setEditingProvider(null)}
          groupId={groupId}
          mode="edit"
          initialData={editingProvider}
        />
      )}

      <AlertDialog
        open={!!deletingProvider}
        onOpenChange={(open) => !open && setDeletingProvider(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Forward Auth Provider</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete <strong>{deletingProvider?.name}</strong>? Users
              authenticating through this forward auth provider will no longer have access. This
              action cannot be undone.
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
    </>
  );
}
