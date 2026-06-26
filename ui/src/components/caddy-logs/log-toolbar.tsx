import {
  Button,
  Input,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@e412/rnui-react';
import { Pause, Play, Search, Trash2 } from 'lucide-react';

import type { CaddyLogSource } from '@/types/caddy-logs';

const RUNTIME_LEVELS = [
  { value: 'all', label: 'All Levels' },
  { value: 'debug', label: 'Debug' },
  { value: 'info', label: 'Info' },
  { value: 'warn', label: 'Warn' },
  { value: 'error', label: 'Error' },
] as const;

const ACCESS_STATUSES = [
  { value: 'all', label: 'All Statuses' },
  { value: '2xx', label: '2xx' },
  { value: '3xx', label: '3xx' },
  { value: '4xx', label: '4xx' },
  { value: '5xx', label: '5xx' },
] as const;

interface LogToolbarProps {
  source: CaddyLogSource;
  isStreaming: boolean;
  search: string;
  onSearchChange: (v: string) => void;
  levelFilter: string;
  onLevelFilterChange: (v: string) => void;
  statusFilter: string;
  onStatusFilterChange: (v: string) => void;
  onPause: () => void;
  onResume: () => void;
  onClear: () => void;
}

export function LogToolbar({
  source,
  isStreaming,
  search,
  onSearchChange,
  levelFilter,
  onLevelFilterChange,
  statusFilter,
  onStatusFilterChange,
  onPause,
  onResume,
  onClear,
}: LogToolbarProps) {
  return (
    <div className="flex flex-wrap items-center gap-3">
      <div className="relative flex-1 min-w-[200px] max-w-sm">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 size-4 text-muted-foreground" />
        <Input
          value={search}
          onChange={(e) => onSearchChange(e.target.value)}
          placeholder="Search logs..."
          aria-label="Search logs"
          className="pl-9"
        />
      </div>

      {source === 'runtime' && (
        <Select items={RUNTIME_LEVELS} value={levelFilter} onValueChange={onLevelFilterChange}>
          <SelectTrigger className="w-[120px]">
            <SelectValue placeholder="Level" />
          </SelectTrigger>
          <SelectContent>
            {RUNTIME_LEVELS.map((level) => (
              <SelectItem key={level.value} value={level.value}>
                {level.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      )}

      {source === 'access' && (
        <Select items={ACCESS_STATUSES} value={statusFilter} onValueChange={onStatusFilterChange}>
          <SelectTrigger className="w-[130px]">
            <SelectValue placeholder="Status" />
          </SelectTrigger>
          <SelectContent>
            {ACCESS_STATUSES.map((status) => (
              <SelectItem key={status.value} value={status.value}>
                {status.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      )}

      <div className="ml-auto flex items-center gap-2">
        {isStreaming ? (
          <Button variant="outline" size="sm" onClick={onPause}>
            <Pause className="size-4" />
            Pause
          </Button>
        ) : (
          <Button variant="outline" size="sm" onClick={onResume}>
            <Play className="size-4" />
            Resume
          </Button>
        )}
        <Button variant="outline" size="sm" onClick={onClear}>
          <Trash2 className="size-4" />
          Clear
        </Button>
      </div>
    </div>
  );
}
