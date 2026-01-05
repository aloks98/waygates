import { Button, type Filter, type FilterFieldsConfig, Filters, Input } from '@e412/titanium';
import type { PaginationState } from '@tanstack/react-table';
import { Download, Search } from 'lucide-react';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { AuditDataGrid } from '@/components/audit-logs';
import { useAuditEventGroups, useAuditLogs, useExportAuditLogs } from '@/hooks';
import type {
  AuditAction,
  AuditLogListParams,
  AuditResourceType,
  AuditStatus,
} from '@/types/audit';

const statusOptions = [
  { value: 'success', label: 'Success' },
  { value: 'failure', label: 'Failed' },
];

const resourceTypeOptions = [
  { value: 'proxy', label: 'Proxy' },
  { value: 'user', label: 'User' },
  { value: 'settings', label: 'Settings' },
  { value: 'system', label: 'System' },
];

export function AuditLogsPage() {
  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: 20,
  });
  const [search, setSearch] = useState('');
  const [debouncedSearch, setDebouncedSearch] = useState('');
  const [filters, setFilters] = useState<Filter<string>[]>([]);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const { eventGroups } = useAuditEventGroups();

  // Generate action options from event groups
  const actionOptions = useMemo(() => {
    return eventGroups.flatMap((group) =>
      group.events.map((event) => ({
        // Transform key from "proxy_create" to "proxy.create"
        value: event.key.replace('_', '.'),
        label: `${group.label.replace(' Events', '')} ${event.label}`,
      })),
    );
  }, [eventGroups]);

  const filterFields = useMemo<FilterFieldsConfig<string>>(
    () => [
      {
        key: 'action',
        label: 'Action',
        type: 'multiselect',
        options: actionOptions,
      },
      {
        key: 'status',
        label: 'Status',
        type: 'select',
        options: statusOptions,
      },
      {
        key: 'resource_type',
        label: 'Resource',
        type: 'multiselect',
        options: resourceTypeOptions,
      },
      {
        key: 'ip_address',
        label: 'IP Address',
        type: 'text',
        placeholder: 'Filter by IP...',
      },
    ],
    [actionOptions],
  );

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

  // Convert filters to API params
  const apiParams = useMemo<AuditLogListParams>(() => {
    const params: AuditLogListParams = {
      page: pagination.pageIndex + 1,
      limit: pagination.pageSize,
    };

    if (debouncedSearch) {
      params.search = debouncedSearch;
    }

    for (const filter of filters) {
      if (filter.values.length === 0) continue;

      switch (filter.field) {
        case 'action':
          // For multiselect, use first value (API only supports single action filter)
          params.action = filter.values[0] as AuditAction;
          break;
        case 'status':
          params.status = filter.values[0] as AuditStatus;
          break;
        case 'resource_type':
          params.resource_type = filter.values[0] as AuditResourceType;
          break;
        case 'ip_address':
          // IP filter goes into search
          if (!params.search) {
            params.search = filter.values[0];
          }
          break;
      }
    }

    return params;
  }, [pagination, debouncedSearch, filters]);

  const { logs, total, totalPages, isLoading } = useAuditLogs(apiParams);
  const { exportLogs, isExporting } = useExportAuditLogs();

  const handleExport = () => {
    const { page, limit, ...exportParams } = apiParams;
    exportLogs(exportParams);
  };

  const handleFiltersChange = useCallback((newFilters: Filter<string>[]) => {
    setFilters(newFilters);
    setPagination((prev) => ({ ...prev, pageIndex: 0 }));
  }, []);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Audit Logs</h1>
        <Button variant="outline" onClick={handleExport} disabled={isExporting}>
          <Download className="size-4" />
          {isExporting ? 'Exporting...' : 'Export CSV'}
        </Button>
      </div>

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

        <Filters
          filters={filters}
          fields={filterFields}
          onChange={handleFiltersChange}
          addButtonText="Add Filter"
          variant="outline"
          size="md"
        />
      </div>

      <AuditDataGrid
        data={logs}
        isLoading={isLoading}
        pageCount={totalPages}
        pagination={pagination}
        onPaginationChange={setPagination}
        total={total}
      />
    </div>
  );
}
