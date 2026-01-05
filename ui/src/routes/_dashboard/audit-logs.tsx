import { Button, type Filter, type FilterFieldsConfig, Filters, Input } from '@e412/titanium';
import type { PaginationState } from '@tanstack/react-table';
import { Download, Search } from 'lucide-react';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { AuditDataGrid } from '@/components/audit-logs';
import { useAuditEventGroups, useAuditLogs, useExportAuditLogs } from '@/hooks';
import type { AuditLogListParams } from '@/types/audit';

// Map Titanium filter operators to backend operators
const operatorMap: Record<string, string> = {
  is: '', // defaults to eq
  is_any_of: 'in',
  is_not_any_of: 'not_in',
  is_not: 'not',
  contains: 'contains',
  starts_with: 'starts_with',
  ends_with: 'ends_with',
};

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
  const [debouncedFilters, setDebouncedFilters] = useState<Filter<string>[]>([]);
  const searchDebounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const filtersDebounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

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
          operators: [
            { value: 'is_any_of', label: 'is any of', supportsMultiple: true },
            { value: 'is_not_any_of', label: 'is not any of', supportsMultiple: true },
          ],
        },
        {
          key: 'status',
          label: 'Status',
          type: 'select',
          options: statusOptions,
          operators: [
            { value: 'is', label: 'is' },
            { value: 'is_not', label: 'is not' },
          ],
        },
        {
          key: 'resource_type',
          label: 'Resource',
          type: 'multiselect',
          options: resourceTypeOptions,
          operators: [
            { value: 'is_any_of', label: 'is any of', supportsMultiple: true },
            { value: 'is_not_any_of', label: 'is not any of', supportsMultiple: true },
          ],
        },
        {
          key: 'ip_address',
          label: 'IP Address',
          type: 'text',
          placeholder: 'Enter IP address...',
          defaultOperator: 'contains',
          operators: [
            { value: 'contains', label: 'contains' },
            { value: 'is', label: 'is' },
            { value: 'starts_with', label: 'starts with' },
            { value: 'ends_with', label: 'ends with' },
          ],
        },
      ],
      [actionOptions],
  );

  // Debounce search input
  useEffect(() => {
    if (searchDebounceRef.current) {
      clearTimeout(searchDebounceRef.current);
    }
    searchDebounceRef.current = setTimeout(() => {
      setDebouncedSearch(search);
      setPagination((prev) => ({ ...prev, pageIndex: 0 }));
    }, 300);

    return () => {
      if (searchDebounceRef.current) {
        clearTimeout(searchDebounceRef.current);
      }
    };
  }, [search]);

  // Debounce filters
  useEffect(() => {
    if (filtersDebounceRef.current) {
      clearTimeout(filtersDebounceRef.current);
    }
    filtersDebounceRef.current = setTimeout(() => {
      setDebouncedFilters(filters);
      setPagination((prev) => ({ ...prev, pageIndex: 0 }));
    }, 500);

    return () => {
      if (filtersDebounceRef.current) {
        clearTimeout(filtersDebounceRef.current);
      }
    };
  }, [filters]);

  // Helper to build filter value with operator prefix
  const buildFilterValue = (operator: string, values: string[]): string => {
    const backendOp = operatorMap[operator] || '';
    const value = values.join(',');
    return backendOp ? `${backendOp}:${value}` : value;
  };

  // Convert filters to API params using new format: field=operator:value
  const apiParams = useMemo<AuditLogListParams>(() => {
    const params: AuditLogListParams = {
      page: pagination.pageIndex + 1,
      limit: pagination.pageSize,
    };

    if (debouncedSearch) {
      params.search = debouncedSearch;
    }

    for (const filter of debouncedFilters) {
      if (filter.values.length === 0) continue;

      switch (filter.field) {
        case 'action':
          params.action = buildFilterValue(filter.operator, filter.values);
          break;
        case 'status':
          params.status = buildFilterValue(filter.operator, filter.values);
          break;
        case 'resource_type':
          params.resource_type = buildFilterValue(filter.operator, filter.values);
          break;
        case 'ip_address':
          params.ip_address = buildFilterValue(filter.operator, filter.values);
          break;
      }
    }

    return params;
  }, [pagination, debouncedSearch, debouncedFilters]);

  const { logs, total, totalPages, isLoading } = useAuditLogs(apiParams);
  const { exportLogs, isExporting } = useExportAuditLogs();

  const handleExport = () => {
    const { page, limit, ...exportParams } = apiParams;
    exportLogs(exportParams);
  };

  const handleFiltersChange = useCallback((newFilters: Filter<string>[]) => {
    setFilters(newFilters);
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
