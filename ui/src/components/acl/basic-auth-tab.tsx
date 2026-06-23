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
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
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
import { zodResolver } from '@hookform/resolvers/zod';
import { format } from 'date-fns';
import { Eye, EyeOff, Key, Plus, Trash2, User } from 'lucide-react';
import { useEffect, useState } from 'react';
import { useForm } from 'react-hook-form';
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

  const form = useForm<BasicAuthUserFormValues>({
    resolver: zodResolver(basicAuthUserSchema),
    defaultValues: {
      username: '',
      password: '',
    },
  });

  useEffect(() => {
    if (open) {
      form.reset({ username: '', password: '' });
      setShowPassword(false);
    }
  }, [open, form]);

  const handleOpenChange = (isOpen: boolean) => {
    if (!isOpen) {
      form.reset();
      setShowPassword(false);
    }
    onOpenChange(isOpen);
  };

  const onSubmit = async (value: BasicAuthUserFormValues) => {
    await addUser({
      groupId,
      data: {
        username: value.username,
        password: value.password,
      },
    });
    onOpenChange(false);
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
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)}>
            <div className="grid gap-4 py-2">
              <div className="grid gap-4">
                <FormField
                  control={form.control}
                  name="username"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Username</FormLabel>
                      <FormControl>
                        <Input placeholder="e.g., john_doe" autoComplete="off" {...field} />
                      </FormControl>
                      <FormDescription>
                        Letters, numbers, underscores, and hyphens only
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name="password"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Password</FormLabel>
                      <FormControl>
                        <div className="relative">
                          <Input
                            type={showPassword ? 'text' : 'password'}
                            placeholder="Enter a secure password"
                            autoComplete="new-password"
                            className="pr-10"
                            {...field}
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
                      </FormControl>
                      <FormDescription>Minimum 8 characters</FormDescription>
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
              <Button type="submit" disabled={isAdding}>
                {isAdding ? 'Adding...' : 'Add User'}
              </Button>
            </DialogFooter>
          </form>
        </Form>
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
                  <Skeleton className="size-8 rounded-none" />
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
                        <div className="flex items-center justify-center size-8 rounded-none bg-muted">
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
