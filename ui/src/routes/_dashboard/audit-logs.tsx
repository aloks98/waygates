import {
  Button,
  Input,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from '@e412/titanium';
import type { PaginationState } from '@tanstack/react-table';
import { Download, Search } from 'lucide-react';
import { useEffect, useRef, useState } from 'react';
import { AuditConfigPanel, AuditDataGrid } from '@/components/audit-logs';
import { useAuditLogs, useExportAuditLogs } from '@/hooks';
import type { AuditAction, AuditStatus } from '@/types/audit';

const actionOptions: { value: AuditAction | ''; label: string }[] = [
  { value: '', label: 'All Actions' },
  { value: 'proxy.create', label: 'Proxy Create' },
  { value: 'proxy.update', label: 'Proxy Update' },
  { value: 'proxy.delete', label: 'Proxy Delete' },
  { value: 'proxy.enable', label: 'Proxy Enable' },
  { value: 'proxy.disable', label: 'Proxy Disable' },
  { value: 'auth.login', label: 'Login' },
  { value: 'auth.logout', label: 'Logout' },
  { value: 'auth.register', label: 'Register' },
  { value: 'auth.password_change', label: 'Password Change' },
  { value: 'auth.login_failed', label: 'Failed Login' },
  { value: 'settings.update', label: 'Settings Update' },
  { value: 'sync.started', label: 'Sync Started' },
  { value: 'sync.completed', label: 'Sync Completed' },
  { value: 'sync.failed', label: 'Sync Failed' },
  { value: 'system.startup', label: 'System Startup' },
  { value: 'caddy.reload', label: 'Caddy Reload' },
];

const statusOptions: { value: AuditStatus | ''; label: string }[] = [
  { value: '', label: 'All Statuses' },
  { value: 'success', label: 'Success' },
  { value: 'failure', label: 'Failed' },
];

export function AuditLogsPage() {
  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: 20,
  });
  const [search, setSearch] = useState('');
  const [debouncedSearch, setDebouncedSearch] = useState('');
  const [actionFilter, setActionFilter] = useState<AuditAction | ''>('');
  const [statusFilter, setStatusFilter] = useState<AuditStatus | ''>('');
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Debounce search input
  useEffect(() => {
    if (debounceRef.current) {
      clearTimeout(debounceRef.current);
    }
    debounceRef.current = setTimeout(() => {
      setDebouncedSearch(search);
      setPagination((prev) => ({ ...prev, pageIndex: 0 }));
    }, 300);

    return () => {
      if (debounceRef.current) {
        clearTimeout(debounceRef.current);
      }
    };
  }, [search]);

  const { logs, total, totalPages, isLoading } = useAuditLogs({
    page: pagination.pageIndex + 1,
    limit: pagination.pageSize,
    search: debouncedSearch || undefined,
    action: actionFilter || undefined,
    status: statusFilter || undefined,
  });

  const { exportLogs, isExporting } = useExportAuditLogs();

  const handleExport = () => {
    exportLogs({
      search: debouncedSearch || undefined,
      action: actionFilter || undefined,
      status: statusFilter || undefined,
    });
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Audit Logs</h1>
        <Button variant="outline" onClick={handleExport} disabled={isExporting}>
          <Download className="size-4" />
          {isExporting ? 'Exporting...' : 'Export CSV'}
        </Button>
      </div>

      <Tabs defaultValue="logs" className="space-y-4">
        <TabsList>
          <TabsTrigger value="logs">Logs</TabsTrigger>
          <TabsTrigger value="config">Configuration</TabsTrigger>
        </TabsList>

        <TabsContent value="logs" className="space-y-4">
          <div className="flex flex-wrap items-center gap-4">
            <div className="relative flex-1 min-w-[200px] max-w-sm">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 size-4 text-muted-foreground" />
              <Input
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder="Search logs..."
                className="pl-9"
              />
            </div>

            <Select
              value={actionFilter}
              onValueChange={(value) => {
                setActionFilter(value as AuditAction | '');
                setPagination((prev) => ({ ...prev, pageIndex: 0 }));
              }}
            >
              <SelectTrigger className="w-[180px]">
                <SelectValue placeholder="All Actions" />
              </SelectTrigger>
              <SelectContent>
                {actionOptions.map((option) => (
                  <SelectItem key={option.value || 'all'} value={option.value}>
                    {option.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>

            <Select
              value={statusFilter}
              onValueChange={(value) => {
                setStatusFilter(value as AuditStatus | '');
                setPagination((prev) => ({ ...prev, pageIndex: 0 }));
              }}
            >
              <SelectTrigger className="w-[140px]">
                <SelectValue placeholder="All Statuses" />
              </SelectTrigger>
              <SelectContent>
                {statusOptions.map((option) => (
                  <SelectItem key={option.value || 'all'} value={option.value}>
                    {option.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <AuditDataGrid
            data={logs}
            isLoading={isLoading}
            pageCount={totalPages}
            pagination={pagination}
            onPaginationChange={setPagination}
            total={total}
          />
        </TabsContent>

        <TabsContent value="config">
          <AuditConfigPanel />
        </TabsContent>
      </Tabs>
    </div>
  );
}
