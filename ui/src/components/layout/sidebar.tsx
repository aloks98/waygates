import {
  Alert,
  AlertDescription,
  Avatar,
  Badge,
  Button,
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
  Field,
  FieldError,
  FieldGroup,
  FieldLabel,
  Input,
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarHeader,
  SidebarInset,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarProvider,
  SidebarTrigger,
} from '@e412/titanium';
import { useForm } from '@tanstack/react-form';
import { useMutation } from '@tanstack/react-query';
import { Link, useLocation, useRouter } from '@tanstack/react-router';
import {
  CheckCircle2,
  ChevronUp,
  ClipboardList,
  Globe,
  Home,
  KeyRound,
  LogOut,
  Settings,
  Shield,
  User,
  XCircle,
} from 'lucide-react';
import { type ReactNode, useState } from 'react';
import { z } from 'zod';
import { api } from '@/lib/api';
import { useAuthStore } from '@/stores/auth';

interface NavItem {
  label: string;
  path: string;
  icon: ReactNode;
}

const navItems: NavItem[] = [
  {
    label: 'Dashboard',
    path: '/dashboard',
    icon: <Home className="size-4" />,
  },
  {
    label: 'Proxies',
    path: '/dashboard/proxies',
    icon: <Globe className="size-4" />,
  },
  {
    label: 'Audit Logs',
    path: '/dashboard/audit-logs',
    icon: <ClipboardList className="size-4" />,
  },
  {
    label: 'Access Control',
    path: '/dashboard/acl',
    icon: <Shield className="size-4" />,
  },
  {
    label: 'Settings',
    path: '/dashboard/settings',
    icon: <Settings className="size-4" />,
  },
];

const passwordSchema = z
  .object({
    current_password: z.string().min(1, 'Current password is required'),
    new_password: z.string().min(8, 'Password must be at least 8 characters'),
    confirm_password: z.string().min(1, 'Please confirm your password'),
  })
  .refine((data) => data.new_password === data.confirm_password, {
    message: "Passwords don't match",
    path: ['confirm_password'],
  });

function ChangePasswordDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const [status, setStatus] = useState<{ type: 'success' | 'error'; message: string } | null>(null);

  const form = useForm({
    defaultValues: {
      current_password: '',
      new_password: '',
      confirm_password: '',
    },
    validators: {
      onChange: passwordSchema,
    },
    onSubmit: async ({ value }) => {
      setStatus(null);
      mutation.mutate({
        current_password: value.current_password,
        new_password: value.new_password,
      });
    },
  });

  const mutation = useMutation({
    mutationFn: async (data: { current_password: string; new_password: string }) => {
      return await api.post('auth/change-password', { json: data }).json();
    },
    onSuccess: () => {
      setStatus({ type: 'success', message: 'Password changed successfully!' });
      form.reset();
      setTimeout(() => {
        onOpenChange(false);
        setStatus(null);
      }, 1500);
    },
    onError: (error: Error) => {
      setStatus({ type: 'error', message: error.message || 'Failed to change password' });
    },
  });

  const handleOpenChange = (isOpen: boolean) => {
    if (!isOpen) {
      setStatus(null);
      form.reset();
    }
    onOpenChange(isOpen);
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Change Password</DialogTitle>
          <DialogDescription>Enter your current password and choose a new one.</DialogDescription>
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
              {status && (
                <Alert variant={status.type === 'error' ? 'destructive' : 'success'}>
                  {status.type === 'success' ? (
                    <CheckCircle2 className="size-4" />
                  ) : (
                    <XCircle className="size-4" />
                  )}
                  <AlertDescription>{status.message}</AlertDescription>
                </Alert>
              )}

              <form.Field name="current_password">
                {(field) => {
                  const hasError = field.state.meta.isTouched && field.state.meta.errors.length > 0;
                  return (
                    <Field data-invalid={hasError}>
                      <FieldLabel htmlFor={field.name}>Current Password</FieldLabel>
                      <Input
                        id={field.name}
                        type="password"
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

              <form.Field name="new_password">
                {(field) => {
                  const hasError = field.state.meta.isTouched && field.state.meta.errors.length > 0;
                  return (
                    <Field data-invalid={hasError}>
                      <FieldLabel htmlFor={field.name}>New Password</FieldLabel>
                      <Input
                        id={field.name}
                        type="password"
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

              <form.Field name="confirm_password">
                {(field) => {
                  const hasError = field.state.meta.isTouched && field.state.meta.errors.length > 0;
                  return (
                    <Field data-invalid={hasError}>
                      <FieldLabel htmlFor={field.name}>Confirm New Password</FieldLabel>
                      <Input
                        id={field.name}
                        type="password"
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
            </FieldGroup>
          </DialogBody>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => handleOpenChange(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={mutation.isPending || status?.type === 'success'}>
              {mutation.isPending ? 'Changing...' : 'Change Password'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

function ProfileDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const { user } = useAuthStore();

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>Profile</DialogTitle>
        </DialogHeader>
        <DialogBody className="space-y-4">
          <div className="flex items-center gap-4">
            <Avatar className="size-16">
              <div className="flex size-full items-center justify-center bg-primary text-primary-foreground text-xl font-medium">
                {user?.name?.charAt(0).toUpperCase() || 'U'}
              </div>
            </Avatar>
            <div>
              <h3 className="text-lg font-semibold">{user?.name || 'User'}</h3>
              <p className="text-sm text-muted-foreground">@{user?.username}</p>
              {user?.role && (
                <Badge variant="secondary" className="mt-1 capitalize">
                  {user.role}
                </Badge>
              )}
            </div>
          </div>

          <div className="space-y-3 rounded-lg border p-4">
            <div>
              <p className="text-xs text-muted-foreground uppercase tracking-wide">Email</p>
              <p className="text-sm">{user?.email}</p>
            </div>
            <div>
              <p className="text-xs text-muted-foreground uppercase tracking-wide">Username</p>
              <p className="text-sm">{user?.username}</p>
            </div>
          </div>

          {user?.permissions && user.permissions.length > 0 && (
            <div className="space-y-2">
              <p className="text-xs text-muted-foreground uppercase tracking-wide">Permissions</p>
              <div className="flex flex-wrap gap-1.5">
                {user.permissions.map((permission) => (
                  <Badge key={permission} variant="outline" className="text-xs font-normal">
                    {permission}
                  </Badge>
                ))}
              </div>
            </div>
          )}
        </DialogBody>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Close
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export function AppSidebar({ children }: { children: React.ReactNode }) {
  const location = useLocation();
  const router = useRouter();
  const { user, logout } = useAuthStore();
  const [profileOpen, setProfileOpen] = useState(false);
  const [passwordOpen, setPasswordOpen] = useState(false);

  const handleLogout = async () => {
    try {
      await api.post('auth/logout');
    } catch {
      // Ignore errors, logout locally anyway
    }
    logout();
    router.navigate({ to: '/login' });
  };

  return (
    <SidebarProvider>
      <Sidebar>
        <SidebarHeader>
          <div className="flex items-center gap-2 px-2 py-2">
            <div className="flex size-8 items-center justify-center rounded-lg bg-primary text-primary-foreground">
              <Globe className="size-5" />
            </div>
            <span className="font-semibold">Waygates</span>
          </div>
        </SidebarHeader>

        <SidebarContent>
          <SidebarGroup>
            <SidebarGroupContent>
              <SidebarMenu>
                {navItems.map((item) => {
                  const isActive = location.pathname === item.path;
                  return (
                    <SidebarMenuItem key={item.path}>
                      <SidebarMenuButton asChild isActive={isActive}>
                        <Link to={item.path}>
                          {item.icon}
                          <span>{item.label}</span>
                        </Link>
                      </SidebarMenuButton>
                    </SidebarMenuItem>
                  );
                })}
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>
        </SidebarContent>

        <SidebarFooter className="border-t border-border p-2">
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" className="w-full justify-start h-auto py-2 px-2">
                <div className="flex items-center gap-3 flex-1">
                  <Avatar className="size-8">
                    <div className="flex size-full items-center justify-center bg-primary text-primary-foreground text-sm font-medium">
                      {user?.name?.charAt(0).toUpperCase() || 'U'}
                    </div>
                  </Avatar>
                  <div className="flex-1 truncate text-left">
                    <p className="truncate text-sm font-medium">{user?.name || 'User'}</p>
                    <p className="truncate text-xs text-muted-foreground">{user?.email}</p>
                  </div>
                  <ChevronUp className="size-4 text-muted-foreground" />
                </div>
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="start" className="w-56">
              <DropdownMenuLabel>My Account</DropdownMenuLabel>
              <DropdownMenuSeparator />
              <DropdownMenuItem onClick={() => setProfileOpen(true)}>
                <User className=" size-4" />
                Profile
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => setPasswordOpen(true)}>
                <KeyRound className=" size-4" />
                Change Password
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem
                onClick={handleLogout}
                className="text-destructive focus:text-destructive"
              >
                <LogOut className=" size-4" />
                Sign out
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </SidebarFooter>
      </Sidebar>

      <SidebarInset>
        <header className="flex h-14 items-center gap-4 border-b border-border px-4">
          <SidebarTrigger />
        </header>
        <main className="flex-1 overflow-auto p-6">{children}</main>
      </SidebarInset>

      <ProfileDialog open={profileOpen} onOpenChange={setProfileOpen} />
      <ChangePasswordDialog open={passwordOpen} onOpenChange={setPasswordOpen} />
    </SidebarProvider>
  );
}
