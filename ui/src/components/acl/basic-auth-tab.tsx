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
  Card,
  CardContent,
  CardDescription,
  CardAction,
  CardHeader,
  CardTitle,
  Dialog,
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
  Skeleton,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@e412/rnui-react';
import { useForm } from '@tanstack/react-form';
import { format } from 'date-fns';
import { Eye, EyeOff, Key, Plus, Trash2, User } from 'lucide-react';
import { useState } from 'react';
import { z } from 'zod';

import { useAddBasicAuthUser, useBasicAuthUsers, useDeleteBasicAuthUser } from '@/hooks';
import type { ACLBasicAuthUser } from '@/types/acl';

const basicAuthUserSchema = z.object({
  username: z
    .string()
    .min(1, 'Username is required')
    .max(50, 'Username must be at most 50 characters')
    .regex(
      /^[a-zA-Z0-9_-]+$/,
      'Username can only contain letters, numbers, underscores, and hyphens',
    ),
  password: z
    .string()
    .min(8, 'Password must be at least 8 characters')
    .max(128, 'Password must be at most 128 characters'),
});

type BasicAuthUserFormValues = z.infer<typeof basicAuthUserSchema>;

interface AddUserModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  groupId: number;
}

function AddUserModal({ open, onOpenChange, groupId }: AddUserModalProps) {
  const { addUser, isAdding } = useAddBasicAuthUser();
  const [showPassword, setShowPassword] = useState(false);

  const form = useForm({
    defaultValues: {
      username: '',
      password: '',
    } as BasicAuthUserFormValues,
    validators: {
      onSubmit: basicAuthUserSchema,
    },
    onSubmit: async ({ value }) => {
      await addUser({
        groupId,
        data: {
          username: value.username,
          password: value.password,
        },
      });
      onOpenChange(false);
    },
  });

  const handleOpenChange = (isOpen: boolean) => {
    if (!isOpen) {
      form.reset();
      setShowPassword(false);
    }
    onOpenChange(isOpen);
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Key className="size-5" />
            Add Basic Auth User
          </DialogTitle>
        </DialogHeader>
        <form
          onSubmit={(e) => {
            e.preventDefault();
            e.stopPropagation();
            form.handleSubmit();
          }}
        >
          <div className="grid gap-4 py-2">
            <FieldGroup>
              <form.Field name="username">
                {(field) => {
                  const hasError = field.state.meta.isTouched && field.state.meta.errors.length > 0;
                  return (
                    <Field data-invalid={hasError}>
                      <FieldLabel htmlFor={field.name}>Username</FieldLabel>
                      <Input
                        id={field.name}
                        placeholder="e.g., john_doe"
                        value={field.state.value}
                        onChange={(e) => field.handleChange(e.target.value)}
                        onBlur={field.handleBlur}
                        aria-invalid={hasError}
                        autoComplete="off"
                      />
                      <FieldDescription>
                        Letters, numbers, underscores, and hyphens only
                      </FieldDescription>
                      {hasError && <FieldError errors={field.state.meta.errors} />}
                    </Field>
                  );
                }}
              </form.Field>

              <form.Field name="password">
                {(field) => {
                  const hasError = field.state.meta.isTouched && field.state.meta.errors.length > 0;
                  return (
                    <Field data-invalid={hasError}>
                      <FieldLabel htmlFor={field.name}>Password</FieldLabel>
                      <div className="relative">
                        <Input
                          id={field.name}
                          type={showPassword ? 'text' : 'password'}
                          placeholder="Enter a secure password"
                          value={field.state.value}
                          onChange={(e) => field.handleChange(e.target.value)}
                          onBlur={field.handleBlur}
                          aria-invalid={hasError}
                          autoComplete="new-password"
                          className="pr-10"
                        />
                        <Button
                          type="button"
                          variant="ghost"
                          size="sm"
                          className="absolute right-1 top-1/2 -translate-y-1/2 size-8 p-0"
                          onClick={() => setShowPassword(!showPassword)}
                        >
                          {showPassword ? (
                            <EyeOff className="size-4" />
                          ) : (
                            <Eye className="size-4" />
                          )}
                          <span className="sr-only">
                            {showPassword ? 'Hide password' : 'Show password'}
                          </span>
                        </Button>
                      </div>
                      <FieldDescription>Minimum 8 characters</FieldDescription>
                      {hasError && <FieldError errors={field.state.meta.errors} />}
                    </Field>
                  );
                }}
              </form.Field>
            </FieldGroup>
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => handleOpenChange(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={isAdding}>
              {isAdding ? 'Adding...' : 'Add User'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

interface BasicAuthTabProps {
  groupId: number;
}

export function BasicAuthTab({ groupId }: BasicAuthTabProps) {
  const { users, isLoading } = useBasicAuthUsers(groupId);
  const { deleteUser, isDeleting } = useDeleteBasicAuthUser();

  const [addModalOpen, setAddModalOpen] = useState(false);
  const [deletingUser, setDeletingUser] = useState<ACLBasicAuthUser | null>(null);

  const handleDelete = async () => {
    if (!deletingUser) return;
    await deleteUser({ id: deletingUser.id, groupId });
    setDeletingUser(null);
  };

  return (
    <>
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Key className="size-5" />
            Basic Authentication Users
          </CardTitle>
          <CardDescription>
            Manage users who can access protected resources using HTTP Basic Auth. Passwords are
            securely hashed and cannot be retrieved.
          </CardDescription>
          <CardAction>
            <Button onClick={() => setAddModalOpen(true)}>
              <Plus className="size-4" />
              Add User
            </Button>
          </CardAction>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="space-y-3">
              {[1, 2, 3].map((i) => (
                <div key={i} className="flex items-center gap-4">
                  <Skeleton className="size-8 rounded-full" />
                  <Skeleton className="h-5 w-32" />
                  <Skeleton className="h-5 w-24 flex-1" />
                  <Skeleton className="h-8 w-16" />
                </div>
              ))}
            </div>
          ) : users.length === 0 ? (
            <div className="text-center py-8 text-muted-foreground">
              <User className="size-12 mx-auto mb-4 opacity-50" />
              <p>No users configured</p>
              <p className="text-sm mt-1">Add users to enable HTTP Basic Authentication</p>
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Username</TableHead>
                  <TableHead>Created</TableHead>
                  <TableHead>Last Updated</TableHead>
                  <TableHead className="w-20 text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {users.map((user) => (
                  <TableRow key={user.id}>
                    <TableCell>
                      <div className="flex items-center gap-2">
                        <div className="flex items-center justify-center size-8 rounded-full bg-muted">
                          <User className="size-4 text-muted-foreground" />
                        </div>
                        <span className="font-medium">{user.username}</span>
                      </div>
                    </TableCell>
                    <TableCell>
                      <Tooltip>
                        <TooltipTrigger
                          render={<span className="text-sm text-muted-foreground cursor-default" />}
                        >
                          {format(new Date(user.created_at), 'MMM d, yyyy')}
                        </TooltipTrigger>
                        <TooltipContent>{format(new Date(user.created_at), 'PPpp')}</TooltipContent>
                      </Tooltip>
                    </TableCell>
                    <TableCell>
                      <Tooltip>
                        <TooltipTrigger
                          render={<span className="text-sm text-muted-foreground cursor-default" />}
                        >
                          {format(new Date(user.updated_at), 'MMM d, yyyy')}
                        </TooltipTrigger>
                        <TooltipContent>{format(new Date(user.updated_at), 'PPpp')}</TooltipContent>
                      </Tooltip>
                    </TableCell>
                    <TableCell>
                      <div className="flex justify-end">
                        <Tooltip>
                          <TooltipTrigger
                            render={
                              <Button
                                variant="ghost"
                                size="sm"
                                className="size-8 p-0 text-destructive hover:text-destructive"
                                onClick={() => setDeletingUser(user)}
                              />
                            }
                          >
                            <Trash2 className="size-4" />
                            <span className="sr-only">Delete</span>
                          </TooltipTrigger>
                          <TooltipContent>Delete user</TooltipContent>
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

      <AddUserModal open={addModalOpen} onOpenChange={setAddModalOpen} groupId={groupId} />

      <AlertDialog open={!!deletingUser} onOpenChange={(open) => !open && setDeletingUser(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete User</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete user <strong>{deletingUser?.username}</strong>? They
              will no longer be able to authenticate using Basic Auth. This action cannot be undone.
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
