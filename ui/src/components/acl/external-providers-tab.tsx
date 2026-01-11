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
  CardToolbar,
  Dialog,
  DialogBody,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  Field,
  FieldDescription,
  FieldError,
  FieldGroup,
  FieldLabel,
  Input,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Skeleton,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  Textarea,
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@e412/titanium';
import { useForm } from '@tanstack/react-form';
import { format } from 'date-fns';
import { ExternalLink, Globe, Pencil, Plus, Shield, Trash2 } from 'lucide-react';
import { useEffect, useState } from 'react';
import { z } from 'zod';
import {
  useAddExternalProvider,
  useDeleteExternalProvider,
  useExternalProviders,
  useUpdateExternalProvider,
} from '@/hooks';
import type { ACLExternalProvider, ProviderType } from '@/types/acl';

const externalProviderSchema = z.object({
  provider_type: z.enum(['authelia', 'authentik', 'custom']),
  name: z.string().min(1, 'Name is required').max(100, 'Name must be at most 100 characters'),
  verify_url: z.string().min(1, 'Verify URL is required').url('Must be a valid URL'),
  auth_redirect_url: z.string().url('Must be a valid URL').optional().or(z.literal('')),
  headers_to_copy: z.array(z.string()).optional(),
});

type ExternalProviderFormValues = z.infer<typeof externalProviderSchema>;

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
): 'primary' | 'secondary' | 'outline' | 'destructive' {
  switch (type) {
    case 'authelia':
      return 'primary';
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

  const [headersInput, setHeadersInput] = useState('');

  const form = useForm({
    defaultValues: {
      provider_type: 'authelia' as ProviderType,
      name: '',
      verify_url: '',
      auth_redirect_url: '',
      headers_to_copy: [] as string[],
    } as ExternalProviderFormValues,
    validators: {
      onSubmit: externalProviderSchema,
    },
    onSubmit: async ({ value }) => {
      const data = {
        provider_type: value.provider_type,
        name: value.name,
        verify_url: value.verify_url,
        auth_redirect_url: value.auth_redirect_url || undefined,
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
    },
  });

  useEffect(() => {
    if (isEditMode) {
      form.setFieldValue('provider_type', initialData.provider_type);
      form.setFieldValue('name', initialData.name);
      form.setFieldValue('verify_url', initialData.verify_url);
      form.setFieldValue('auth_redirect_url', initialData.auth_redirect_url || '');
      form.setFieldValue('headers_to_copy', initialData.headers_to_copy || []);
      setHeadersInput((initialData.headers_to_copy || []).join(', '));
    } else {
      form.reset();
      setHeadersInput('');
    }
  }, [initialData, isEditMode, form.setFieldValue, form.reset]);

  const handleHeadersChange = (value: string) => {
    setHeadersInput(value);
    const headers = value
      .split(',')
      .map((h) => h.trim())
      .filter(Boolean);
    form.setFieldValue('headers_to_copy', headers);
  };

  const handleOpenChange = (isOpen: boolean) => {
    if (!isOpen) {
      form.reset();
      setHeadersInput('');
    }
    onOpenChange(isOpen);
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Globe className="size-5" />
            {isEditMode ? 'Edit External Provider' : 'Add External Provider'}
          </DialogTitle>
        </DialogHeader>
        <form
          onSubmit={(e) => {
            e.preventDefault();
            e.stopPropagation();
            form.handleSubmit();
          }}
        >
          <DialogBody>
            <FieldGroup>
              <form.Field name="provider_type">
                {(field) => (
                  <Field>
                    <FieldLabel>Provider Type</FieldLabel>
                    <Select
                      value={field.state.value}
                      onValueChange={(val) => field.handleChange(val as ProviderType)}
                    >
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="authelia">Authelia</SelectItem>
                        <SelectItem value="authentik">Authentik</SelectItem>
                        <SelectItem value="custom">Custom Provider</SelectItem>
                      </SelectContent>
                    </Select>
                    <FieldDescription>Select the type of authentication provider</FieldDescription>
                  </Field>
                )}
              </form.Field>

              <form.Field name="name">
                {(field) => {
                  const hasError = field.state.meta.isTouched && field.state.meta.errors.length > 0;
                  return (
                    <Field data-invalid={hasError}>
                      <FieldLabel htmlFor={field.name}>Name</FieldLabel>
                      <Input
                        id={field.name}
                        placeholder="e.g., Corporate SSO"
                        value={field.state.value}
                        onChange={(e) => field.handleChange(e.target.value)}
                        onBlur={field.handleBlur}
                        aria-invalid={hasError}
                      />
                      {hasError && <FieldError errors={field.state.meta.errors} />}
                    </Field>
                  );
                }}
              </form.Field>

              <form.Field name="verify_url">
                {(field) => {
                  const hasError = field.state.meta.isTouched && field.state.meta.errors.length > 0;
                  return (
                    <Field data-invalid={hasError}>
                      <FieldLabel htmlFor={field.name}>Verify URL</FieldLabel>
                      <Input
                        id={field.name}
                        type="url"
                        placeholder="https://auth.example.com/api/verify"
                        value={field.state.value}
                        onChange={(e) => field.handleChange(e.target.value)}
                        onBlur={field.handleBlur}
                        aria-invalid={hasError}
                      />
                      <FieldDescription>
                        URL to verify authentication tokens/sessions
                      </FieldDescription>
                      {hasError && <FieldError errors={field.state.meta.errors} />}
                    </Field>
                  );
                }}
              </form.Field>

              <form.Field name="auth_redirect_url">
                {(field) => {
                  const hasError = field.state.meta.isTouched && field.state.meta.errors.length > 0;
                  return (
                    <Field data-invalid={hasError}>
                      <FieldLabel htmlFor={field.name}>Auth Redirect URL (optional)</FieldLabel>
                      <Input
                        id={field.name}
                        type="url"
                        placeholder="https://auth.example.com/login"
                        value={field.state.value}
                        onChange={(e) => field.handleChange(e.target.value)}
                        onBlur={field.handleBlur}
                        aria-invalid={hasError}
                      />
                      <FieldDescription>
                        URL to redirect unauthenticated users for login
                      </FieldDescription>
                      {hasError && <FieldError errors={field.state.meta.errors} />}
                    </Field>
                  );
                }}
              </form.Field>

              <Field>
                <FieldLabel>Headers to Copy (optional)</FieldLabel>
                <Textarea
                  placeholder="e.g., Remote-User, Remote-Groups, Remote-Email"
                  value={headersInput}
                  onChange={(e) => handleHeadersChange(e.target.value)}
                  rows={2}
                />
                <FieldDescription>
                  Comma-separated list of headers to copy from auth response
                </FieldDescription>
                <form.Subscribe selector={(state) => state.values.headers_to_copy}>
                  {(headers) =>
                    headers &&
                    headers.length > 0 && (
                      <div className="flex flex-wrap gap-1 mt-2">
                        {headers.map((header) => (
                          <Badge key={header} variant="outline" className="font-mono text-xs">
                            {header}
                          </Badge>
                        ))}
                      </div>
                    )
                  }
                </form.Subscribe>
              </Field>
            </FieldGroup>
          </DialogBody>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => handleOpenChange(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={isLoading}>
              {isLoading ? 'Saving...' : isEditMode ? 'Save Changes' : 'Add Provider'}
            </Button>
          </DialogFooter>
        </form>
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
          <CardHeading>
            <CardTitle className="flex items-center gap-2">
              <Globe className="size-5" />
              External Authentication Providers
            </CardTitle>
            <CardDescription>
              Integrate with external authentication services like Authelia, Authentik, or custom
              forward auth providers.
            </CardDescription>
          </CardHeading>
          <CardToolbar>
            <Button onClick={() => setCreateModalOpen(true)}>
              <Plus className="size-4" />
              Add Provider
            </Button>
          </CardToolbar>
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
              <p>No external providers configured</p>
              <p className="text-sm mt-1">
                Add providers like Authelia or Authentik for SSO integration
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
                  <TableHead className="w-20 text-right">Actions</TableHead>
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
                          <TooltipTrigger asChild>
                            <a
                              href={provider.verify_url}
                              target="_blank"
                              rel="noopener noreferrer"
                              className="text-muted-foreground hover:text-foreground"
                            >
                              <ExternalLink className="size-3" />
                            </a>
                          </TooltipTrigger>
                          <TooltipContent>Open URL</TooltipContent>
                        </Tooltip>
                      </div>
                    </TableCell>
                    <TableCell>
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <span className="text-sm text-muted-foreground cursor-default">
                            {format(new Date(provider.created_at), 'MMM d, yyyy')}
                          </span>
                        </TooltipTrigger>
                        <TooltipContent>
                          {format(new Date(provider.created_at), 'PPpp')}
                        </TooltipContent>
                      </Tooltip>
                    </TableCell>
                    <TableCell>
                      <div className="flex justify-end gap-1">
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <Button
                              variant="ghost"
                              size="sm"
                              className="size-8 p-0"
                              onClick={() => setEditingProvider(provider)}
                            >
                              <Pencil className="size-4" />
                              <span className="sr-only">Edit</span>
                            </Button>
                          </TooltipTrigger>
                          <TooltipContent>Edit provider</TooltipContent>
                        </Tooltip>
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <Button
                              variant="ghost"
                              size="sm"
                              className="size-8 p-0 text-destructive hover:text-destructive"
                              onClick={() => setDeletingProvider(provider)}
                            >
                              <Trash2 className="size-4" />
                              <span className="sr-only">Delete</span>
                            </Button>
                          </TooltipTrigger>
                          <TooltipContent>Delete provider</TooltipContent>
                        </Tooltip>
                      </div>
                    </TableCell>
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
            <AlertDialogTitle>Delete External Provider</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete <strong>{deletingProvider?.name}</strong>? Users
              authenticating through this provider will no longer have access. This action cannot be
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
    </>
  );
}
