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
  CardTitle,
  DataGrid,
  DataGridColumnHeader,
  DataGridContainer,
  DataGridPagination,
  DataGridTable,
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  Input,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Skeleton,
  Spinner,
  Switch,
} from '@e412/rnui-react';
import { zodResolver } from '@hookform/resolvers/zod';
import {
  type ColumnDef,
  getCoreRowModel,
  getPaginationRowModel,
  useReactTable,
} from '@tanstack/react-table';
import { format } from 'date-fns';
import { Eye, EyeOff, MoreHorizontal, Plus, Users } from 'lucide-react';
import { useEffect, useMemo, useState } from 'react';
import { useForm } from 'react-hook-form';
import { z } from 'zod';

import { useRegistrationSetting, useUsers } from '@/hooks/use-users';
import type { ManagedUser, Role } from '@/types/user';

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const ROLE_OPTIONS: { value: Role; label: string }[] = [
  { value: 'viewer', label: 'Viewer' },
  { value: 'operator', label: 'Operator' },
  { value: 'admin', label: 'Admin' },
];

const ROLE_LABELS = Object.fromEntries(ROLE_OPTIONS.map((o) => [o.value, o.label])) as Record<
  Role,
  string
>;

const ROLE_VARIANTS: Record<Role, 'default' | 'secondary' | 'outline'> = {
  admin: 'default',
  operator: 'secondary',
  viewer: 'outline',
};

function formatLastLogin(value: string | null): string {
  if (!value) return 'Never';
  try {
    return format(new Date(value), 'MMM d, yyyy');
  } catch {
    return 'Never';
  }
}

// ---------------------------------------------------------------------------
// Zod schemas
// ---------------------------------------------------------------------------

const createUserSchema = z.object({
  name: z.string().min(1, 'Name is required').max(100, 'Name must be at most 100 characters'),
  username: z
    .string()
    .min(1, 'Username is required')
    .max(50, 'Username must be at most 50 characters'),
  email: z.string().email('Must be a valid email'),
  role: z.enum(['admin', 'operator', 'viewer'] as const),
  password: z
    .string()
    .min(8, 'Password must be at least 8 characters')
    .max(128, 'Password must be at most 128 characters'),
  must_change_password: z.boolean(),
});

type CreateUserFormValues = z.infer<typeof createUserSchema>;

const editUserSchema = z.object({
  name: z.string().min(1, 'Name is required').max(100, 'Name must be at most 100 characters'),
  email: z.string().email('Must be a valid email'),
  role: z.enum(['admin', 'operator', 'viewer'] as const),
  active: z.boolean(),
});

type EditUserFormValues = z.infer<typeof editUserSchema>;

const resetPasswordSchema = z.object({
  password: z
    .string()
    .min(8, 'Password must be at least 8 characters')
    .max(128, 'Password must be at most 128 characters'),
  must_change_password: z.boolean(),
});

type ResetPasswordFormValues = z.infer<typeof resetPasswordSchema>;

// ---------------------------------------------------------------------------
// AddUserModal
// ---------------------------------------------------------------------------

interface AddUserModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

function AddUserModal({ open, onOpenChange }: AddUserModalProps) {
  const { createUser, isCreating } = useUsers();
  const [showPassword, setShowPassword] = useState(false);

  const form = useForm<CreateUserFormValues>({
    resolver: zodResolver(createUserSchema),
    defaultValues: {
      name: '',
      username: '',
      email: '',
      role: 'viewer',
      password: '',
      must_change_password: false,
    },
  });

  useEffect(() => {
    if (open) {
      form.reset({
        name: '',
        username: '',
        email: '',
        role: 'viewer',
        password: '',
        must_change_password: false,
      });
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

  const onSubmit = async (values: CreateUserFormValues) => {
    await createUser(values);
    onOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Users className="size-5" />
            Add User
          </DialogTitle>
        </DialogHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)}>
            <div className="grid gap-4 py-2">
              <FormField
                control={form.control}
                name="name"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Full Name</FormLabel>
                    <FormControl>
                      <Input placeholder="Jane Smith" autoComplete="off" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="username"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Username</FormLabel>
                    <FormControl>
                      <Input placeholder="jsmith" autoComplete="off" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="email"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Email</FormLabel>
                    <FormControl>
                      <Input
                        type="email"
                        placeholder="jane@example.com"
                        autoComplete="off"
                        {...field}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="role"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Role</FormLabel>
                    <Select items={ROLE_OPTIONS} value={field.value} onValueChange={field.onChange}>
                      <FormControl>
                        <SelectTrigger>
                          <SelectValue placeholder="Select role" />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent>
                        {ROLE_OPTIONS.map((o) => (
                          <SelectItem key={o.value} value={o.value}>
                            {o.label}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
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

              <FormField
                control={form.control}
                name="must_change_password"
                render={({ field }) => (
                  <FormItem className="flex items-center justify-between rounded-lg border p-3">
                    <div className="space-y-0.5">
                      <FormLabel>Must change on first login</FormLabel>
                      <FormDescription>
                        Require the user to set a new password when they first log in.
                      </FormDescription>
                    </div>
                    <FormControl>
                      <Switch checked={field.value} onCheckedChange={field.onChange} />
                    </FormControl>
                  </FormItem>
                )}
              />
            </div>

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => handleOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={isCreating}>
                {isCreating ? (
                  <>
                    <Spinner variant="circle" />
                    Creating...
                  </>
                ) : (
                  'Create User'
                )}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
}

// ---------------------------------------------------------------------------
// EditUserModal
// ---------------------------------------------------------------------------

interface EditUserModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  user: ManagedUser | null;
}

function EditUserModal({ open, onOpenChange, user }: EditUserModalProps) {
  const { updateUser, isUpdating } = useUsers();

  const form = useForm<EditUserFormValues>({
    resolver: zodResolver(editUserSchema),
    defaultValues: {
      name: '',
      email: '',
      role: 'viewer',
      active: true,
    },
  });

  useEffect(() => {
    if (open && user) {
      form.reset({
        name: user.name,
        email: user.email,
        role: user.role,
        active: user.active,
      });
    }
  }, [open, user, form]);

  const handleOpenChange = (isOpen: boolean) => {
    if (!isOpen) form.reset();
    onOpenChange(isOpen);
  };

  const onSubmit = async (values: EditUserFormValues) => {
    if (!user) return;
    await updateUser({ id: user.id, data: values });
    onOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Users className="size-5" />
            Edit User
          </DialogTitle>
        </DialogHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)}>
            <div className="grid gap-4 py-2">
              {/* Username — read-only display */}
              <div className="space-y-1">
                <p className="text-sm font-medium">Username</p>
                <p className="rounded-md border bg-muted/50 px-3 py-2 text-sm font-mono text-muted-foreground">
                  {user?.username}
                </p>
              </div>

              <FormField
                control={form.control}
                name="name"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Full Name</FormLabel>
                    <FormControl>
                      <Input placeholder="Jane Smith" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="email"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Email</FormLabel>
                    <FormControl>
                      <Input type="email" placeholder="jane@example.com" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="role"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Role</FormLabel>
                    <Select items={ROLE_OPTIONS} value={field.value} onValueChange={field.onChange}>
                      <FormControl>
                        <SelectTrigger>
                          <SelectValue placeholder="Select role" />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent>
                        {ROLE_OPTIONS.map((o) => (
                          <SelectItem key={o.value} value={o.value}>
                            {o.label}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="active"
                render={({ field }) => (
                  <FormItem className="flex items-center justify-between rounded-lg border p-3">
                    <div className="space-y-0.5">
                      <FormLabel>Active</FormLabel>
                      <FormDescription>Inactive users cannot log in.</FormDescription>
                    </div>
                    <FormControl>
                      <Switch checked={field.value} onCheckedChange={field.onChange} />
                    </FormControl>
                  </FormItem>
                )}
              />
            </div>

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => handleOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={isUpdating}>
                {isUpdating ? (
                  <>
                    <Spinner variant="circle" />
                    Saving...
                  </>
                ) : (
                  'Save Changes'
                )}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
}

// ---------------------------------------------------------------------------
// ResetPasswordModal
// ---------------------------------------------------------------------------

interface ResetPasswordModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  user: ManagedUser | null;
}

function ResetPasswordModal({ open, onOpenChange, user }: ResetPasswordModalProps) {
  const { resetPassword, isResettingPassword } = useUsers();
  const [showPassword, setShowPassword] = useState(false);

  const form = useForm<ResetPasswordFormValues>({
    resolver: zodResolver(resetPasswordSchema),
    defaultValues: {
      password: '',
      must_change_password: true,
    },
  });

  useEffect(() => {
    if (open) {
      form.reset({ password: '', must_change_password: true });
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

  const onSubmit = async (values: ResetPasswordFormValues) => {
    if (!user) return;
    await resetPassword({ id: user.id, data: values });
    onOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>Reset Password — {user?.username}</DialogTitle>
        </DialogHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)}>
            <div className="grid gap-4 py-2">
              <FormField
                control={form.control}
                name="password"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>New Password</FormLabel>
                    <FormControl>
                      <div className="relative">
                        <Input
                          type={showPassword ? 'text' : 'password'}
                          placeholder="Enter new password"
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

              <FormField
                control={form.control}
                name="must_change_password"
                render={({ field }) => (
                  <FormItem className="flex items-center justify-between rounded-lg border p-3">
                    <div className="space-y-0.5">
                      <FormLabel>Must change on next login</FormLabel>
                      <FormDescription>
                        Require the user to set a new password when they next log in.
                      </FormDescription>
                    </div>
                    <FormControl>
                      <Switch checked={field.value} onCheckedChange={field.onChange} />
                    </FormControl>
                  </FormItem>
                )}
              />
            </div>

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => handleOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={isResettingPassword}>
                {isResettingPassword ? (
                  <>
                    <Spinner variant="circle" />
                    Resetting...
                  </>
                ) : (
                  'Reset Password'
                )}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
}

// ---------------------------------------------------------------------------
// Open Registration Card
// ---------------------------------------------------------------------------

function OpenRegistrationCard() {
  const { registrationOpen, isLoading, setRegistrationOpen, isUpdating } = useRegistrationSetting();

  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <Skeleton className="h-6 w-48" />
          <Skeleton className="h-4 w-80" />
        </CardHeader>
        <CardContent>
          <Skeleton className="h-12 w-full" />
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Open Registration</CardTitle>
        <CardDescription>
          Allow anyone to create an account without an invitation. Disable this in production
          environments to restrict access.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div className="flex items-center justify-between rounded-lg border p-4">
          <div className="space-y-0.5">
            <p className="text-sm font-medium">Enable open registration</p>
            <p className="text-sm text-muted-foreground">
              {registrationOpen
                ? 'Anyone can sign up for an account.'
                : 'New accounts can only be created by administrators.'}
            </p>
          </div>
          <Switch
            checked={registrationOpen}
            onCheckedChange={(checked) => setRegistrationOpen(checked)}
            disabled={isUpdating}
          />
        </div>
      </CardContent>
    </Card>
  );
}

// ---------------------------------------------------------------------------
// User Actions Cell
// ---------------------------------------------------------------------------

interface UserActionsCellProps {
  user: ManagedUser;
  onEdit: (user: ManagedUser) => void;
  onResetPassword: (user: ManagedUser) => void;
  onToggleActive: (user: ManagedUser) => void;
  onDelete: (user: ManagedUser) => void;
}

function UserActionsCell({
  user,
  onEdit,
  onResetPassword,
  onToggleActive,
  onDelete,
}: UserActionsCellProps) {
  return (
    <div className="flex justify-end">
      <DropdownMenu>
        <DropdownMenuTrigger render={<Button variant="ghost" size="sm" className="size-8 p-0" />}>
          <MoreHorizontal className="size-4" />
          <span className="sr-only">Open actions menu</span>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-44">
          <DropdownMenuItem onClick={() => onEdit(user)}>Edit</DropdownMenuItem>
          <DropdownMenuItem onClick={() => onResetPassword(user)}>Reset Password</DropdownMenuItem>
          <DropdownMenuItem onClick={() => onToggleActive(user)}>
            {user.active ? 'Deactivate' : 'Activate'}
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem
            onClick={() => onDelete(user)}
            className="text-destructive focus:text-destructive"
          >
            Delete
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}

// ---------------------------------------------------------------------------
// UserManagement (main page component)
// ---------------------------------------------------------------------------

export function UserManagement() {
  const { users, isLoading, updateUser, deleteUser } = useUsers();

  const [addOpen, setAddOpen] = useState(false);
  const [editUser, setEditUser] = useState<ManagedUser | null>(null);
  const [resetPasswordUser, setResetPasswordUser] = useState<ManagedUser | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<ManagedUser | null>(null);

  const handleToggleActive = async (user: ManagedUser) => {
    await updateUser({
      id: user.id,
      data: { name: user.name, email: user.email, role: user.role, active: !user.active },
    });
  };

  const handleDelete = async () => {
    if (!deleteTarget) return;
    await deleteUser(deleteTarget.id);
    setDeleteTarget(null);
  };

  const columns = useMemo<ColumnDef<ManagedUser>[]>(
    () => [
      {
        accessorKey: 'name',
        header: ({ column }) => <DataGridColumnHeader column={column} title="Name" />,
        cell: ({ row }) => (
          <span className="font-medium truncate block">{row.getValue('name')}</span>
        ),
        minSize: 120,
        maxSize: 180,
        meta: { skeleton: <Skeleton className="h-5 w-32" /> },
      },
      {
        accessorKey: 'username',
        header: ({ column }) => <DataGridColumnHeader column={column} title="Username" />,
        cell: ({ row }) => (
          <span className="font-mono text-sm truncate block">{row.getValue('username')}</span>
        ),
        minSize: 100,
        maxSize: 150,
        meta: { skeleton: <Skeleton className="h-5 w-24" /> },
      },
      {
        accessorKey: 'email',
        header: ({ column }) => <DataGridColumnHeader column={column} title="Email" />,
        cell: ({ row }) => (
          <span className="text-sm text-muted-foreground truncate block">
            {row.getValue('email')}
          </span>
        ),
        minSize: 160,
        maxSize: 260,
        meta: { skeleton: <Skeleton className="h-5 w-40" /> },
      },
      {
        accessorKey: 'role',
        header: 'Role',
        cell: ({ row }) => {
          const role = row.getValue('role') as Role;
          return <Badge variant={ROLE_VARIANTS[role]}>{ROLE_LABELS[role]}</Badge>;
        },
        enableSorting: false,
        minSize: 80,
        maxSize: 110,
        meta: { skeleton: <Skeleton className="h-6 w-20 rounded" /> },
      },
      {
        accessorKey: 'active',
        header: 'Status',
        cell: ({ row }) => {
          const active = row.getValue('active') as boolean;
          return (
            <Badge variant={active ? 'default' : 'secondary'}>
              {active ? 'Active' : 'Inactive'}
            </Badge>
          );
        },
        enableSorting: false,
        minSize: 70,
        maxSize: 100,
        meta: { skeleton: <Skeleton className="h-6 w-16 rounded" /> },
      },
      {
        accessorKey: 'last_login_at',
        header: 'Last Login',
        cell: ({ row }) => (
          <span className="text-sm text-muted-foreground">
            {formatLastLogin(row.getValue('last_login_at'))}
          </span>
        ),
        enableSorting: false,
        minSize: 90,
        maxSize: 130,
        meta: { skeleton: <Skeleton className="h-5 w-24" /> },
      },
      {
        accessorKey: 'created_at',
        header: ({ column }) => <DataGridColumnHeader column={column} title="Created" />,
        cell: ({ row }) => {
          const val = row.getValue('created_at') as string;
          return (
            <span className="text-sm text-muted-foreground">
              {val ? format(new Date(val), 'MMM d, yyyy') : '—'}
            </span>
          );
        },
        minSize: 90,
        maxSize: 130,
        meta: { skeleton: <Skeleton className="h-5 w-24" /> },
      },
      {
        id: 'actions',
        header: '',
        cell: ({ row }) => (
          <UserActionsCell
            user={row.original}
            onEdit={setEditUser}
            onResetPassword={setResetPasswordUser}
            onToggleActive={handleToggleActive}
            onDelete={setDeleteTarget}
          />
        ),
        enableSorting: false,
        minSize: 56,
        maxSize: 80,
        meta: {
          skeleton: (
            <div className="flex justify-end">
              <Skeleton className="size-8" />
            </div>
          ),
        },
      },
    ],
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [],
  );

  const table = useReactTable({
    data: users,
    columns,
    getCoreRowModel: getCoreRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    getRowId: (row) => String(row.id),
  });

  return (
    <div className="space-y-6">
      <OpenRegistrationCard />

      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <div className="space-y-1">
            <h2 className="flex items-center gap-2 text-lg font-semibold">
              <Users className="size-5" />
              Users
            </h2>
            <p className="text-sm text-muted-foreground">
              Manage user accounts and their access levels.
            </p>
          </div>
          <Button onClick={() => setAddOpen(true)}>
            <Plus className="size-4" />
            Add User
          </Button>
        </div>

        <DataGrid
          table={table}
          recordCount={users.length}
          isLoading={isLoading}
          loadingMode="skeleton"
          emptyMessage={
            <div className="flex flex-col items-center justify-center py-12 text-center">
              <div className="rounded bg-muted p-4">
                <Users className="size-8 text-muted-foreground" />
              </div>
              <h3 className="mt-4 text-base font-medium">No users yet</h3>
              <p className="mt-1.5 text-sm text-muted-foreground max-w-[260px]">
                Add the first user to grant access to this instance.
              </p>
            </div>
          }
        >
          <DataGridContainer>
            <DataGridTable />
            <div className="border-t px-4 py-2">
              <DataGridPagination
                sizes={[10, 20, 50]}
                className="[&_[data-slot=select-trigger]]:w-fit"
              />
            </div>
          </DataGridContainer>
        </DataGrid>
      </div>

      {/* Dialogs */}
      <AddUserModal open={addOpen} onOpenChange={setAddOpen} />

      <EditUserModal
        open={!!editUser}
        onOpenChange={(open) => !open && setEditUser(null)}
        user={editUser}
      />

      <ResetPasswordModal
        open={!!resetPasswordUser}
        onOpenChange={(open) => !open && setResetPasswordUser(null)}
        user={resetPasswordUser}
      />

      <AlertDialog open={!!deleteTarget} onOpenChange={(open) => !open && setDeleteTarget(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete User</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete{' '}
              <strong>{deleteTarget?.name ?? deleteTarget?.username}</strong>? This action cannot be
              undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDelete}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
