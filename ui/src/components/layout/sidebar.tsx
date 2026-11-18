import { Link, useLocation } from '@tanstack/react-router';
import {
  ChevronUp,
  ChevronDown,
  CircleHelp,
  Globe,
  Home,
  KeyRound,
  LogOut,
  Activity,
  Settings,
  Shield,
  User,
} from 'lucide-react';
import { type ReactNode, useEffect, useState } from 'react';

import { ChangePasswordDialog } from '@/components/layout/change-password-dialog';
import { CommandPalette } from '@/components/layout/command-palette';
import { TopBar } from '@/components/layout/top-bar';
import { WaygatesBrand } from '@/components/layout/waygate-logo';
import { Avatar } from '@/components/ui/avatar';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarMenuSub,
  SidebarMenuSubItem,
  SidebarMenuSubButton,
  SidebarHeader,
  SidebarInset,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarProvider,
  useSidebar,
} from '@/components/ui/sidebar';
import { useLogout } from '@/hooks/use-logout';
import { usePermissions } from '@/hooks/use-permissions';
import { cn } from '@/lib/utils';
import { useAuthStore } from '@/stores/auth';

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
            <Avatar className="size-16 rounded-md">
              <div className="flex size-full items-center justify-center rounded-md bg-primary text-primary-foreground text-xl font-medium">
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

function MobileNavigation() {
  const location = useLocation();
  const { setOpenMobile } = useSidebar();
  useEffect(() => {
    setOpenMobile(false);
  }, [location.href, setOpenMobile]);
  return null;
}

export function AppSidebar({
  children,
  fillViewport = false,
}: {
  children: ReactNode;
  fillViewport?: boolean;
}) {
  const location = useLocation();
  const { user } = useAuthStore();
  const handleLogout = useLogout();
  const permissions = usePermissions();
  const [profileOpen, setProfileOpen] = useState(false);
  const [passwordOpen, setPasswordOpen] = useState(false);

  const active = (path: string) =>
    path === '/' ? location.pathname === '/' : location.pathname.startsWith(path);
  const [proxiesOpen, setProxiesOpen] = useState(true);
  const [operationsOpen, setOperationsOpen] = useState(true);
  const logsSource = (location.search as { source?: string }).source ?? 'runtime';
  const navLink = (
    label: string,
    path: string,
    icon: ReactNode,
    selected = active(path),
    source?: string,
  ) => (
    <SidebarMenuItem key={label}>
      <SidebarMenuButton
        render={<Link to={path} search={source ? { source } : {}} />}
        isActive={selected}
      >
        {icon}
        <span>{label}</span>
      </SidebarMenuButton>
    </SidebarMenuItem>
  );

  return (
    <SidebarProvider className={cn(fillViewport && 'h-full min-h-0')}>
      <MobileNavigation />
      <Sidebar>
        <SidebarHeader className="h-16 justify-center px-5">
          <Link to="/" aria-label="Waygates overview" className="text-white">
            <WaygatesBrand />
          </Link>
        </SidebarHeader>
        <SidebarContent className="px-2">
          <SidebarGroup>
            <SidebarGroupLabel>Workspace</SidebarGroupLabel>
            <SidebarGroupContent>
              <SidebarMenu>
                {navLink('Overview', '/', <Home />)}
                {(permissions.canReadProxies || permissions.canReadL4Proxies) && (
                  <Collapsible
                    open={proxiesOpen}
                    onOpenChange={setProxiesOpen}
                    render={<SidebarMenuItem />}
                  >
                    <CollapsibleTrigger render={<SidebarMenuButton />}>
                      <Globe />
                      <span>Proxies</span>
                      <ChevronDown
                        className={`ml-auto transition-transform ${proxiesOpen ? '' : '-rotate-90'}`}
                      />
                    </CollapsibleTrigger>
                    <CollapsibleContent>
                      <SidebarMenuSub>
                        {permissions.canReadProxies && (
                          <SidebarMenuSubItem>
                            <SidebarMenuSubButton
                              render={<Link to="/proxies" />}
                              isActive={active('/proxies') && !active('/proxies/tcp-udp')}
                            >
                              HTTP proxies
                            </SidebarMenuSubButton>
                          </SidebarMenuSubItem>
                        )}
                        {permissions.canReadL4Proxies && (
                          <SidebarMenuSubItem>
                            <SidebarMenuSubButton
                              render={<Link to="/proxies/tcp-udp" />}
                              isActive={active('/proxies/tcp-udp')}
                            >
                              TCP / UDP
                            </SidebarMenuSubButton>
                          </SidebarMenuSubItem>
                        )}
                      </SidebarMenuSub>
                    </CollapsibleContent>
                  </Collapsible>
                )}
                {permissions.canReadAccess && navLink('Access', '/access', <Shield />)}
                {(permissions.canReadAuditLogs || permissions.canReadCaddyLogs) && (
                  <Collapsible
                    open={operationsOpen}
                    onOpenChange={setOperationsOpen}
                    render={<SidebarMenuItem />}
                  >
                    <CollapsibleTrigger render={<SidebarMenuButton />}>
                      <Activity />
                      <span>Operations</span>
                      <ChevronDown
                        className={`ml-auto transition-transform ${operationsOpen ? '' : '-rotate-90'}`}
                      />
                    </CollapsibleTrigger>
                    <CollapsibleContent>
                      <SidebarMenuSub>
                        {permissions.canReadAuditLogs && (
                          <SidebarMenuSubItem>
                            <SidebarMenuSubButton
                              render={<Link to="/activity" />}
                              isActive={active('/activity')}
                            >
                              Activity
                            </SidebarMenuSubButton>
                          </SidebarMenuSubItem>
                        )}
                        {permissions.canReadCaddyLogs && (
                          <>
                            <SidebarMenuSubItem>
                              <SidebarMenuSubButton
                                render={<Link to="/caddy-logs" search={{ source: 'access' }} />}
                                isActive={active('/caddy-logs') && logsSource === 'access'}
                              >
                                Access logs
                              </SidebarMenuSubButton>
                            </SidebarMenuSubItem>
                            <SidebarMenuSubItem>
                              <SidebarMenuSubButton
                                render={<Link to="/caddy-logs" search={{ source: 'runtime' }} />}
                                isActive={active('/caddy-logs') && logsSource === 'runtime'}
                              >
                                Runtime logs
                              </SidebarMenuSubButton>
                            </SidebarMenuSubItem>
                          </>
                        )}
                      </SidebarMenuSub>
                    </CollapsibleContent>
                  </Collapsible>
                )}
                {permissions.canReadSettings &&
                  navLink(
                    'Settings',
                    '/settings',
                    <Settings />,
                    active('/settings') || active('/caddy-config'),
                  )}
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>
          <SidebarGroup>
            <SidebarGroupLabel>Resources</SidebarGroupLabel>
            <SidebarMenu>
              <SidebarMenuItem>
                <SidebarMenuButton
                  render={
                    <a
                      href="https://github.com/aloks98/waygates#readme"
                      target="_blank"
                      rel="noopener noreferrer"
                    />
                  }
                >
                  <CircleHelp />
                  <span>Help &amp; docs</span>
                </SidebarMenuButton>
              </SidebarMenuItem>
            </SidebarMenu>
          </SidebarGroup>
        </SidebarContent>

        <SidebarFooter className="border-t border-sidebar-border p-3">
          <SidebarMenu>
            <SidebarMenuItem>
              <DropdownMenu>
                <DropdownMenuTrigger render={<SidebarMenuButton size="lg" />}>
                  <div className="flex items-center gap-3 flex-1">
                    <Avatar className="size-8 rounded-md">
                      <div className="flex size-full items-center justify-center rounded-md bg-primary text-primary-foreground text-sm font-medium">
                        {user?.name?.charAt(0).toUpperCase() || 'U'}
                      </div>
                    </Avatar>
                    <div className="flex-1 truncate text-left">
                      <p className="truncate text-sm font-medium">{user?.name || 'User'}</p>
                      <p className="truncate text-xs text-sidebar-muted-foreground">
                        {user?.email}
                      </p>
                    </div>
                    <ChevronUp className="size-4 text-sidebar-foreground" />
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
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarFooter>
      </Sidebar>

      <SidebarInset className={cn('min-w-0', fillViewport && 'min-h-0')}>
        <TopBar />
        <div
          id="main-content"
          className={cn(
            'min-w-0 flex-1 p-4 md:p-8',
            fillViewport && 'flex min-h-0 flex-col overflow-hidden',
          )}
        >
          {children}
        </div>
      </SidebarInset>

      <ProfileDialog open={profileOpen} onOpenChange={setProfileOpen} />
      <ChangePasswordDialog open={passwordOpen} onOpenChange={setPasswordOpen} />
      <CommandPalette />
    </SidebarProvider>
  );
}
