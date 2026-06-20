import { Button, Kbd, SidebarTrigger } from '@e412/rnui-react';
import { Search } from 'lucide-react';

import { Breadcrumbs } from '@/components/layout/breadcrumbs';
import { SyncStatus } from '@/components/layout/sync-status';
import { ThemeToggle } from '@/components/layout/theme-toggle';
import { useCommandPalette } from '@/stores/command-palette';

export function TopBar() {
  const setOpen = useCommandPalette((s) => s.setOpen);

  return (
    <header className="flex h-14 items-center gap-3 border-b border-border px-4">
      <SidebarTrigger />
      <Breadcrumbs />
      <div className="ml-auto flex items-center gap-1.5">
        <Button
          variant="outline"
          size="sm"
          className="text-muted-foreground gap-2"
          onClick={() => setOpen(true)}
        >
          <Search className="size-4" />
          <span className="hidden sm:inline">Search</span>
          <Kbd>⌘K</Kbd>
        </Button>
        <SyncStatus />
        <ThemeToggle />
      </div>
    </header>
  );
}
