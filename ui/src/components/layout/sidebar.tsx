import {
  Avatar,
  Badge,
  Button,
  Dialog,
  DialogContent,
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
import { Link, useLocation } from '@tanstack/react-router';
import {
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
} from 'lucide-react';
import { type ReactNode, useState } from 'react';

import { ChangePasswordDialog } from '@/components/layout/change-password-dialog';
import { CommandPalette } from '@/components/layout/command-palette';
import { TopBar } from '@/components/layout/top-bar';
import { WaygateLogo } from '@/components/layout/waygate-logo';
import { useLogout } from '@/hooks/use-logout';
import { usePermissions } from '@/hooks/use-permissions';
import { useAuthStore } from '@/stores/auth';

interface NavItem {
  label: string;
  path: string;
  icon: ReactNode;
  permission?: string;
}

const navItems: NavItem[] = [
  { label: 'Dashboard', path: '/', icon: <Home className="size-4" /> },
  {
    label: 'Proxies',
    path: '/proxies',
    icon: <Globe className="size-4" />,
    permission: 'canReadProxies',
  },
  {
    label: 'Access',
    path: '/access',
    icon: <Shield className="size-4" />,
    permission: 'canReadAccess',
  },
  {
    label: 'Activity',
    path: '/activity',
    icon: <ClipboardList className="size-4" />,
    permission: 'canReadAuditLogs',
  },
  {
    label: 'Caddy Logs',
    path: '/caddy-logs',
    icon: <ScrollText className="size-4" />,
    permission: 'canReadCaddyLogs',
  },
  {
    label: 'Caddy Config',
    path: '/caddy-config',
    icon: <FileCode2 className="size-4" />,
    permission: 'canReadCaddyConfig',
  },
  {
    label: 'Settings',
    path: '/settings',
    icon: <Settings className="size-4" />,
    permission: 'canReadSettings',
  },
];

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
            <Avatar className="size-16 rounded-none">
              <div className="flex size-full items-center justify-center rounded-none bg-primary text-primary-foreground text-xl font-medium">
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
  const permissions = usePermissions();
  const [profileOpen, setProfileOpen] = useState(false);
  const [passwordOpen, setPasswordOpen] = useState(false);

  const visibleNavItems = navItems.filter((item) => {
    if (!item.permission) return true;
    return permissions[item.permission as keyof typeof permissions] === true;
  });

  return (
    <SidebarProvider>
      <Sidebar>
        <SidebarHeader>
          <div className="flex items-center gap-2.5 px-2 py-2">
            <WaygateLogo className="size-8" />
            <span
              className="text-3xl leading-none tracking-wide"
              style={{ fontFamily: 'var(--font-display)' }}
            >
              Waygates
            </span>
          </div>
        </SidebarHeader>

        <SidebarContent>
          <SidebarGroup>
            <SidebarGroupContent>
              <SidebarMenu>
                {visibleNavItems.map((item) => {
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
                <Avatar className="size-8 rounded-none">
                  <div className="flex size-full items-center justify-center rounded-none bg-primary text-primary-foreground text-sm font-medium">
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
