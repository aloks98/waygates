import { Search } from 'lucide-react';

import { Breadcrumbs } from '@/components/layout/breadcrumbs';
import { SyncStatus } from '@/components/layout/sync-status';
import { ThemeToggle } from '@/components/layout/theme-toggle';
import { Button } from '@/components/ui/button';
import { Kbd } from '@/components/ui/kbd';
import { SidebarTrigger } from '@/components/ui/sidebar';
import { useCommandPalette } from '@/stores/command-palette';

export function TopBar() {
  const setOpen = useCommandPalette((s) => s.setOpen);

  return (
    <header className="flex h-16 shrink-0 items-center gap-3 border-b border-border bg-card px-4 md:px-8">
      <SidebarTrigger />
      <div className="min-w-0 flex-1 overflow-hidden">
        <Breadcrumbs />
      </div>
      <div className="ml-auto flex items-center gap-1.5">
        <SyncStatus />
        <Button
          aria-label="Search"
          variant="outline"
          size="sm"
          className="text-muted-foreground gap-2"
          onClick={() => setOpen(true)}
        >
          <Search data-icon="inline-start" />
          <span className="hidden sm:inline">Search</span>
          <Kbd className="hidden sm:inline-flex">⌘K</Kbd>
        </Button>
        <ThemeToggle />
      </div>
    </header>
  );
}
