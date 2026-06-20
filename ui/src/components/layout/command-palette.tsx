import {
  Command,
  CommandDialog,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from '@e412/rnui-react';
import { useNavigate } from '@tanstack/react-router';
import {
  Activity,
  Globe,
  Home,
  LogOut,
  MoonStar,
  Network,
  Plus,
  RefreshCw,
  Settings,
  Shield,
} from 'lucide-react';
import { useTheme } from 'next-themes';
import { useEffect } from 'react';

import { useSyncStatus } from '@/hooks/use-dashboard';
import { useLogout } from '@/hooks/use-logout';
import { useCommandPalette } from '@/stores/command-palette';

export function CommandPalette() {
  const { open, setOpen, toggle } = useCommandPalette();
  const navigate = useNavigate();
  const { resolvedTheme, setTheme } = useTheme();
  const { triggerSync } = useSyncStatus();
  const logout = useLogout();

  useEffect(() => {
    const onKeyDown = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault();
        toggle();
      }
    };
    document.addEventListener('keydown', onKeyDown);
    return () => document.removeEventListener('keydown', onKeyDown);
  }, [toggle]);

  const run = (fn: () => void) => {
    setOpen(false);
    fn();
  };

  const go = (to: string) => run(() => navigate({ to }));

  return (
    <CommandDialog
      open={open}
      onOpenChange={setOpen}
      title="Command palette"
      description="Search for a page or action"
    >
      <Command>
        <CommandInput placeholder="Type a command or search…" />
        <CommandList>
          <CommandEmpty>No results found.</CommandEmpty>
          <CommandGroup heading="Navigate">
            <CommandItem onSelect={() => go('/dashboard')}>
              <Home className="size-4" />
              Dashboard
            </CommandItem>
            <CommandItem onSelect={() => go('/dashboard/proxies')}>
              <Globe className="size-4" />
              Proxies (HTTP)
            </CommandItem>
            <CommandItem onSelect={() => go('/dashboard/proxies/tcp-udp')}>
              <Network className="size-4" />
              Proxies (TCP/UDP)
            </CommandItem>
            <CommandItem onSelect={() => go('/dashboard/access')}>
              <Shield className="size-4" />
              Access
            </CommandItem>
            <CommandItem onSelect={() => go('/dashboard/activity')}>
              <Activity className="size-4" />
              Activity
            </CommandItem>
            <CommandItem onSelect={() => go('/dashboard/settings')}>
              <Settings className="size-4" />
              Settings
            </CommandItem>
          </CommandGroup>
          <CommandGroup heading="Actions">
            <CommandItem onSelect={() => go('/dashboard/proxies/new')}>
              <Plus className="size-4" />
              New HTTP proxy
            </CommandItem>
            <CommandItem onSelect={() => go('/dashboard/proxies/tcp-udp/new')}>
              <Plus className="size-4" />
              New TCP/UDP proxy
            </CommandItem>
            <CommandItem onSelect={() => run(() => triggerSync())}>
              <RefreshCw className="size-4" />
              Apply configuration now
            </CommandItem>
            <CommandItem
              onSelect={() => run(() => setTheme(resolvedTheme === 'dark' ? 'light' : 'dark'))}
            >
              <MoonStar className="size-4" />
              Toggle theme
            </CommandItem>
            <CommandItem onSelect={() => run(() => void logout())}>
              <LogOut className="size-4" />
              Sign out
            </CommandItem>
          </CommandGroup>
        </CommandList>
      </Command>
    </CommandDialog>
  );
}
