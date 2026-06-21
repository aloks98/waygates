import type { Filter } from '@e412/rnui-react';
import { useNavigate, useSearch } from '@tanstack/react-router';
import type { PaginationState } from '@tanstack/react-table';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import {
  ActivityDetailSheet,
  ActivityTable,
  ActivityTimeline,
  ActivityToolbar,
  type ActivityView,
} from '@/components/activity';
import { useAuditLogs, useExportAuditLogs } from '@/hooks';
import type { AuditLogListParams } from '@/types/audit';

const operatorMap: Record<string, string> = {
  is: '',
  is_any_of: 'in',
  is_not_any_of: 'not_in',
  is_not: 'not',
  contains: 'contains',
  starts_with: 'starts_with',
  ends_with: 'ends_with',
};

export function ActivityPage() {
  const navigate = useNavigate();
  const search = useSearch({ strict: false }) as { view?: string };
  const view: ActivityView = search.view === 'table' ? 'table' : 'timeline';

  const [pagination, setPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 20 });
  const [searchText, setSearchText] = useState('');
  const [debouncedSearch, setDebouncedSearch] = useState('');
  const [filters, setFilters] = useState<Filter<string>[]>([]);
  const [debouncedFilters, setDebouncedFilters] = useState<Filter<string>[]>([]);
  const [dateFrom, setDateFrom] = useState('');
  const [dateTo, setDateTo] = useState('');
  const [selectedLogId, setSelectedLogId] = useState<number | null>(null);
  const [sheetOpen, setSheetOpen] = useState(false);
  const searchDebounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const filtersDebounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    if (searchDebounceRef.current) clearTimeout(searchDebounceRef.current);
    searchDebounceRef.current = setTimeout(() => {
      setDebouncedSearch(searchText);
      setPagination((p) => ({ ...p, pageIndex: 0 }));
    }, 300);
    return () => {
      if (searchDebounceRef.current) clearTimeout(searchDebounceRef.current);
    };
  }, [searchText]);

  useEffect(() => {
    if (filtersDebounceRef.current) clearTimeout(filtersDebounceRef.current);
    filtersDebounceRef.current = setTimeout(() => {
      setDebouncedFilters(filters);
      setPagination((p) => ({ ...p, pageIndex: 0 }));
    }, 500);
    return () => {
      if (filtersDebounceRef.current) clearTimeout(filtersDebounceRef.current);
    };
  }, [filters]);

  // Reset to page 1 when the date range changes.
  useEffect(() => {
    setPagination((p) => ({ ...p, pageIndex: 0 }));
  }, [dateFrom, dateTo]);

  const buildFilterValue = useCallback((operator: string, values: string[]): string => {
    const backendOp = operatorMap[operator] || '';
    const value = values.join(',');
    return backendOp ? `${backendOp}:${value}` : value;
  }, []);

  const apiParams = useMemo<AuditLogListParams>(() => {
    const params: AuditLogListParams = {
      page: pagination.pageIndex + 1,
      limit: pagination.pageSize,
    };
    if (debouncedSearch) params.search = debouncedSearch;
    if (dateFrom) params.date_from = dateFrom;
    if (dateTo) params.date_to = dateTo;
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
  }, [pagination, debouncedSearch, debouncedFilters, dateFrom, dateTo, buildFilterValue]);

  const { logs, total, totalPages, isLoading } = useAuditLogs(apiParams);
  const { exportLogs, isExporting } = useExportAuditLogs();

  const handleExport = () => {
    const { page: _p, limit: _l, ...exportParams } = apiParams;
    exportLogs(exportParams);
  };

  const handleSelect = useCallback((id: number) => {
    setSelectedLogId(id);
    setSheetOpen(true);
  }, []);

  const handleViewChange = useCallback(
    (v: ActivityView) => {
      navigate({ to: '/dashboard/activity', search: { view: v }, replace: true });
    },
    [navigate],
  );

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Activity</h1>

      <ActivityToolbar
        search={searchText}
        onSearchChange={setSearchText}
        filters={filters}
        onFiltersChange={setFilters}
        dateFrom={dateFrom}
        dateTo={dateTo}
        onDateRangeChange={(from, to) => {
          setDateFrom(from);
          setDateTo(to);
        }}
        view={view}
        onViewChange={handleViewChange}
        onExport={handleExport}
        isExporting={isExporting}
      />

      {view === 'timeline' ? (
        <ActivityTimeline logs={logs} isLoading={isLoading} onSelect={handleSelect} />
      ) : (
        <ActivityTable
          logs={logs}
          isLoading={isLoading}
          total={total}
          pageCount={totalPages}
          pagination={pagination}
          onPaginationChange={setPagination}
          onSelect={handleSelect}
        />
      )}

      <ActivityDetailSheet logId={selectedLogId} open={sheetOpen} onOpenChange={setSheetOpen} />
    </div>
  );
}
