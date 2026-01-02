import { Link, useLocation } from '@tanstack/react-router';
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarProvider,
  SidebarInset,
  SidebarTrigger,
  Button,
  Avatar,
  Separator,
} from '@e412/titanium';
import { Home, Globe, LogOut } from 'lucide-react';
import { useAuthStore } from '../stores/auth';
import { api } from '../lib/api';

interface NavItem {
  label: string;
  path: string;
  icon: React.ReactNode;
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
];

export function AppSidebar({ children }: { children: React.ReactNode }) {
  const location = useLocation();
  const { user, logout } = useAuthStore();

  const handleLogout = async () => {
    try {
      await api.post('auth/logout');
    } catch {
      // Ignore errors, logout locally anyway
    }
    logout();
  };

  return (
    <SidebarProvider>
      <Sidebar>
        <SidebarHeader className="border-b border-border p-4">
          <div className="flex items-center gap-2">
            <Globe className="size-6 text-primary" />
            <span className="text-lg font-semibold">Homelab Proxy</span>
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

        <SidebarFooter className="border-t border-border p-4">
          <div className="flex items-center gap-3">
            <Avatar className="size-8">
              <div className="flex size-full items-center justify-center bg-primary text-primary-foreground text-sm font-medium">
                {user?.name?.charAt(0).toUpperCase() || 'U'}
              </div>
            </Avatar>
            <div className="flex-1 truncate">
              <p className="truncate text-sm font-medium">{user?.name || 'User'}</p>
              <p className="truncate text-xs text-muted-foreground">{user?.email}</p>
            </div>
          </div>
          <Separator className="my-3" />
          <Button
            variant="ghost"
            className="w-full justify-start"
            onClick={handleLogout}
          >
            <LogOut className="mr-2 size-4" />
            Sign out
          </Button>
        </SidebarFooter>
      </Sidebar>

      <SidebarInset>
        <header className="flex h-14 items-center gap-4 border-b border-border px-4">
          <SidebarTrigger />
        </header>
        <main className="flex-1 overflow-auto p-6">
          {children}
        </main>
      </SidebarInset>
    </SidebarProvider>
  );
}
