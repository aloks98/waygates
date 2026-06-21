import {
  Button,
  Calendar,
  type Filter,
  type FilterFieldsConfig,
  Filters,
  Input,
  Popover,
  PopoverContent,
  PopoverTrigger,
  Tabs,
  TabsList,
  TabsTrigger,
} from '@e412/rnui-react';
import { format } from 'date-fns';
import { Calendar as CalendarIcon, Download, Search } from 'lucide-react';
import { useMemo } from 'react';

// Parse a yyyy-MM-dd string as a LOCAL date (avoids the UTC off-by-one that
// `new Date('2026-06-21')` causes in negative-offset timezones).
function parseLocalDate(s: string): Date {
  return new Date(`${s}T00:00:00`);
}

// Human label for the date-range trigger button.
function formatDateRangeLabel(dateFrom: string, dateTo: string): string {
  if (dateFrom && dateTo) {
    return `${format(parseLocalDate(dateFrom), 'MMM d')} – ${format(parseLocalDate(dateTo), 'MMM d, yyyy')}`;
  }
  if (dateFrom) {
    return `From ${format(parseLocalDate(dateFrom), 'MMM d, yyyy')}`;
  }
  if (dateTo) {
    return `Until ${format(parseLocalDate(dateTo), 'MMM d, yyyy')}`;
  }
  return 'Date range';
}

import { useAuditEventGroups } from '@/hooks';

export type ActivityView = 'timeline' | 'table';

const statusOptions = [
  { value: 'success', label: 'Success' },
  { value: 'failure', label: 'Failed' },
];
const resourceTypeOptions = [
  { value: 'proxy', label: 'Proxy' },
  { value: 'user', label: 'User' },
  { value: 'settings', label: 'Settings' },
  { value: 'system', label: 'System' },
  { value: 'acl', label: 'Access Control' },
];

export interface ActivityToolbarProps {
  search: string;
  onSearchChange: (s: string) => void;
  filters: Filter<string>[];
  onFiltersChange: (f: Filter<string>[]) => void;
  dateFrom: string;
  dateTo: string;
  onDateRangeChange: (from: string, to: string) => void;
  view: ActivityView;
  onViewChange: (v: ActivityView) => void;
  onExport: () => void;
  isExporting: boolean;
}

export function ActivityToolbar({
  search,
  onSearchChange,
  filters,
  onFiltersChange,
  dateFrom,
  dateTo,
  onDateRangeChange,
  view,
  onViewChange,
  onExport,
  isExporting,
}: ActivityToolbarProps) {
  const { eventGroups } = useAuditEventGroups();

  const actionOptions = useMemo(
    () =>
      eventGroups.flatMap((group) =>
        group.events.map((event) => ({
          // Config keys are `{group}_{verb}` (group may contain underscores);
          // the backend action is `{group}.{verb}` — replace only the LAST underscore.
          value: event.key.replace(/_([^_]+)$/, '.$1'),
          label: `${group.label.replace(' Events', '')} ${event.label}`,
        })),
      ),
    [eventGroups],
  );

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

  const selectedRange =
    dateFrom || dateTo
      ? {
          from: dateFrom ? parseLocalDate(dateFrom) : undefined,
          to: dateTo ? parseLocalDate(dateTo) : undefined,
        }
      : undefined;

  const dateRangeLabel = formatDateRangeLabel(dateFrom, dateTo);

  return (
    <div className="flex flex-wrap items-center gap-3">
      <div className="relative flex-1 min-w-[200px] max-w-sm">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 size-4 text-muted-foreground" />
        <Input
          value={search}
          onChange={(e) => onSearchChange(e.target.value)}
          placeholder="Search by action, user, or resource..."
          aria-label="Search activity"
          className="pl-9"
        />
      </div>

      <Filters
        filters={filters}
        fields={filterFields}
        onChange={onFiltersChange}
        addButtonText="Add Filter"
        variant="default"
        size="default"
      />

      <Popover>
        <PopoverTrigger
          render={
            <Button variant="outline" className="justify-start font-normal">
              <CalendarIcon className="size-4" />
              {dateRangeLabel}
            </Button>
          }
        />
        <PopoverContent className="w-auto p-0" align="start">
          <Calendar
            mode="range"
            numberOfMonths={2}
            selected={selectedRange}
            onSelect={(range: { from?: Date; to?: Date } | undefined) =>
              onDateRangeChange(
                range?.from ? format(range.from, 'yyyy-MM-dd') : '',
                range?.to ? format(range.to, 'yyyy-MM-dd') : '',
              )
            }
          />
          {(dateFrom || dateTo) && (
            <div className="border-t p-2">
              <Button
                variant="ghost"
                size="sm"
                className="w-full"
                onClick={() => onDateRangeChange('', '')}
              >
                Clear dates
              </Button>
            </div>
          )}
        </PopoverContent>
      </Popover>

      <Tabs value={view} onValueChange={(v) => onViewChange(v as ActivityView)}>
        <TabsList>
          <TabsTrigger value="timeline">Timeline</TabsTrigger>
          <TabsTrigger value="table">Table</TabsTrigger>
        </TabsList>
      </Tabs>

      <Button variant="outline" onClick={onExport} disabled={isExporting} className="ml-auto">
        <Download className="size-4" />
        {isExporting ? 'Exporting...' : 'Export CSV'}
      </Button>
    </div>
  );
}
