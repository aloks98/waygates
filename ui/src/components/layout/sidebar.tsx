import {
  Alert,
  AlertDescription,
  Avatar,
  Badge,
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
  FieldGroup,
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
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
} from '@e412/rnui-react';
import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation } from '@tanstack/react-query';
import { Link, useLocation } from '@tanstack/react-router';
import {
  CheckCircle2,
  ChevronUp,
  ClipboardList,
  FileCode2,
  Globe,
  Home,
  KeyRound,
  LogOut,
  ScrollText,
  Settings,
  Shield,
  User,
  XCircle,
} from 'lucide-react';
import { type ReactNode, useState } from 'react';
import { useForm } from 'react-hook-form';
import { z } from 'zod';

import { CommandPalette } from '@/components/layout/command-palette';
import { TopBar } from '@/components/layout/top-bar';
import { WaygateLogo } from '@/components/layout/waygate-logo';
import { useLogout } from '@/hooks/use-logout';
import { api } from '@/lib/api';
import { useAuthStore } from '@/stores/auth';

interface NavItem {
  label: string;
  path: string;
  icon: ReactNode;
}

const navItems: NavItem[] = [
  { label: 'Dashboard', path: '/', icon: <Home className="size-4" /> },
  { label: 'Proxies', path: '/proxies', icon: <Globe className="size-4" /> },
  { label: 'Access', path: '/access', icon: <Shield className="size-4" /> },
  { label: 'Activity', path: '/activity', icon: <ClipboardList className="size-4" /> },
  { label: 'Caddy Logs', path: '/caddy-logs', icon: <ScrollText className="size-4" /> },
  {
    label: 'Caddy Config',
    path: '/caddy-config',
    icon: <FileCode2 className="size-4" />,
  },
  { label: 'Settings', path: '/settings', icon: <Settings className="size-4" /> },
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
type PasswordValues = z.infer<typeof passwordSchema>;

function ChangePasswordDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const [status, setStatus] = useState<{ type: 'success' | 'error'; message: string } | null>(null);

  const form = useForm<PasswordValues>({
    resolver: zodResolver(passwordSchema),
    mode: 'onTouched',
    defaultValues: {
      current_password: '',
      new_password: '',
      confirm_password: '',
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

  const onSubmit = (value: PasswordValues) => {
    setStatus(null);
    mutation.mutate({
      current_password: value.current_password,
      new_password: value.new_password,
    });
  };

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
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)}>
            <div className="grid gap-4 py-2">
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

                <FormField
                  control={form.control}
                  name="current_password"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Current Password</FormLabel>
                      <FormControl>
                        <Input type="password" autoComplete="current-password" {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name="new_password"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>New Password</FormLabel>
                      <FormControl>
                        <Input type="password" autoComplete="new-password" {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name="confirm_password"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Confirm New Password</FormLabel>
                      <FormControl>
                        <Input type="password" autoComplete="new-password" {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </FieldGroup>
            </div>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => handleOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={mutation.isPending || status?.type === 'success'}>
                {mutation.isPending ? 'Changing...' : 'Change Password'}
              </Button>
            </DialogFooter>
          </form>
        </Form>
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
        <div className="grid gap-4 py-2 space-y-4">
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

          <div className="space-y-3 rounded border p-4">
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
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Close
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export function AppSidebar({ children }: { children: ReactNode }) {
  const location = useLocation();
  const { user } = useAuthStore();
  const handleLogout = useLogout();
  const [profileOpen, setProfileOpen] = useState(false);
  const [passwordOpen, setPasswordOpen] = useState(false);

  return (
    <SidebarProvider>
      <Sidebar>
        <SidebarHeader>
          <div className="flex items-center gap-2 px-2 py-2">
            <div className="flex size-8 items-center justify-center rounded bg-primary text-primary-foreground">
              <WaygateLogo className="size-5" />
            </div>
            <span
              className="text-lg font-semibold tracking-tight"
              style={{ fontFamily: '"Bricolage Grotesque", system-ui, sans-serif' }}
            >
              Waygates
            </span>
          </div>
        </SidebarHeader>

        <SidebarContent>
          <SidebarGroup>
            <SidebarGroupContent>
              <SidebarMenu>
                {navItems.map((item) => {
                  const isActive =
                    item.path === '/'
                      ? location.pathname === '/'
                      : location.pathname === item.path ||
                        location.pathname.startsWith(`${item.path}/`);
                  return (
                    <SidebarMenuItem key={item.path}>
                      <SidebarMenuButton render={<Link to={item.path} />} isActive={isActive}>
                        {item.icon}
                        <span>{item.label}</span>
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
            <DropdownMenuTrigger
              render={<Button variant="ghost" className="w-full justify-start h-auto py-2 px-2" />}
            >
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
            </DropdownMenuTrigger>
            <DropdownMenuContent align="start" className="w-56">
              <DropdownMenuGroup>
                <DropdownMenuLabel>My Account</DropdownMenuLabel>
              </DropdownMenuGroup>
              <DropdownMenuSeparator />
              <DropdownMenuItem onClick={() => setProfileOpen(true)}>
                <User className="size-4" />
                Profile
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => setPasswordOpen(true)}>
                <KeyRound className="size-4" />
                Change Password
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem
                onClick={handleLogout}
                className="text-destructive focus:text-destructive"
              >
                <LogOut className="size-4" />
                Sign out
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </SidebarFooter>
      </Sidebar>

      <SidebarInset>
        <TopBar />
        <main className="flex-1 overflow-auto p-6">{children}</main>
      </SidebarInset>

      <ProfileDialog open={profileOpen} onOpenChange={setProfileOpen} />
      <ChangePasswordDialog open={passwordOpen} onOpenChange={setPasswordOpen} />
      <CommandPalette />
    </SidebarProvider>
  );
}
