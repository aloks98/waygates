import { flexRender, type Column, type Table as TanStackTable } from '@tanstack/react-table';
import { ArrowDown, ArrowUp, ArrowUpDown, ChevronLeft, ChevronRight } from 'lucide-react';
import { createContext, useContext, type ComponentProps, type ReactNode } from 'react';

import { cn } from '@/lib/utils';

import { Button } from './button';
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from './select';
import { Skeleton } from './skeleton';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from './table';

interface GridState<T> {
  table: TanStackTable<T>;
  recordCount: number;
  isLoading?: boolean;
  onRowClick?: (row: T) => void;
  emptyMessage?: ReactNode;
}
const GridContext = createContext<GridState<unknown> | null>(null);
function useGrid() {
  const state = useContext(GridContext);
  if (!state) throw new Error('Table must be inside DataGrid');
  return state;
}
export function DataGrid<T>({
  children,
  ...state
}: GridState<T> & { children: ReactNode; loadingMode?: string }) {
  return (
    <GridContext.Provider value={state as unknown as GridState<unknown>}>
      {children}
    </GridContext.Provider>
  );
}
export function DataGridContainer({ className, ...props }: ComponentProps<'div'>) {
  return (
    <div
      className={cn('min-w-0 overflow-hidden rounded-lg border bg-card', className)}
      {...props}
    />
  );
}
export function DataGridTable({ scrollable = false }: { scrollable?: boolean }) {
  const { table, isLoading, onRowClick, emptyMessage } = useGrid();
  const columns = table.getVisibleLeafColumns();
  const rows = table.getRowModel().rows;
  return (
    <Table
      containerClassName={cn(
        scrollable && 'min-h-0 flex-1 overflow-auto overscroll-contain scroll-pt-11',
      )}
    >
      <TableHeader className={cn(scrollable && 'sticky top-0 z-10 bg-card')}>
        {table.getHeaderGroups().map((group) => (
          <TableRow key={group.id} className="bg-muted/40 hover:bg-muted/40">
            {group.headers.map((header) => (
              <TableHead
                key={header.id}
                className="h-11 px-4 text-xs font-medium text-muted-foreground"
                style={
                  header.column.id === 'select'
                    ? { width: 48 }
                    : header.column.id === 'actions'
                      ? { width: 64 }
                      : undefined
                }
              >
                {header.isPlaceholder
                  ? null
                  : flexRender(header.column.columnDef.header, header.getContext())}
              </TableHead>
            ))}
          </TableRow>
        ))}
      </TableHeader>
      <TableBody>
        {isLoading ? (
          Array.from({ length: 5 }, (_, i) => (
            <TableRow key={i}>
              {columns.map((col) => (
                <TableCell className="h-18 px-4" key={col.id}>
                  {(col.columnDef.meta as { skeleton?: ReactNode })?.skeleton ?? (
                    <Skeleton className="h-4 w-24" />
                  )}
                </TableCell>
              ))}
            </TableRow>
          ))
        ) : rows.length ? (
          rows.map((row) => (
            <TableRow
              key={row.id}
              data-state={row.getIsSelected() ? 'selected' : undefined}
              className={
                onRowClick
                  ? 'cursor-pointer focus-visible:outline-2 focus-visible:outline-ring focus-visible:-outline-offset-2'
                  : undefined
              }
              tabIndex={onRowClick ? 0 : undefined}
              onClick={(e) => {
                // Menus render in a portal but their React events still bubble through the row.
                if (!e.currentTarget.contains(e.target as Node)) return;
                if (
                  !(e.target as HTMLElement).closest('a,button,input,[role=switch],[role=checkbox]')
                )
                  onRowClick?.(row.original);
              }}
              onKeyDown={(e) => {
                if (e.target === e.currentTarget && (e.key === 'Enter' || e.key === ' ')) {
                  e.preventDefault();
                  onRowClick?.(row.original);
                }
              }}
            >
              {row.getVisibleCells().map((cell) => (
                <TableCell key={cell.id} className="h-18 px-4 py-4">
                  {flexRender(cell.column.columnDef.cell, cell.getContext())}
                </TableCell>
              ))}
            </TableRow>
          ))
        ) : (
          <TableRow className="hover:bg-transparent">
            <TableCell
              colSpan={columns.length}
              className="h-40 whitespace-normal text-center text-sm text-foreground"
            >
              {emptyMessage ?? 'No results.'}
            </TableCell>
          </TableRow>
        )}
      </TableBody>
    </Table>
  );
}
export function DataGridColumnHeader<T>({
  column,
  title,
}: {
  column: Column<T, unknown>;
  title: string;
}) {
  if (!column.getCanSort()) return <span>{title}</span>;
  const sorted = column.getIsSorted();
  const Icon = sorted === 'asc' ? ArrowUp : sorted === 'desc' ? ArrowDown : ArrowUpDown;
  return (
    <Button
      variant="ghost"
      size="xs"
      className="-ml-2 text-xs"
      onClick={() => column.toggleSorting(sorted === 'asc')}
    >
      {title}
      <Icon className="size-3" data-icon="inline-end" />
    </Button>
  );
}
export function DataGridPagination({
  sizes = [10, 20, 50],
  className,
}: {
  sizes?: number[];
  className?: string;
}) {
  const { table, recordCount } = useGrid();
  const { pageIndex, pageSize } = table.getState().pagination;
  return (
    <div
      className={cn(
        'flex flex-wrap items-center justify-between gap-3 text-xs text-muted-foreground',
        className,
      )}
    >
      <span>
        {recordCount
          ? `${pageIndex * pageSize + 1}–${Math.min((pageIndex + 1) * pageSize, recordCount)} of ${recordCount.toLocaleString()}`
          : '0 results'}
      </span>
      <div className="flex items-center gap-2">
        <span className="hidden sm:inline">Rows per page</span>
        <Select value={String(pageSize)} onValueChange={(v) => v && table.setPageSize(Number(v))}>
          <SelectTrigger size="sm" aria-label="Rows per page">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectGroup>
              {sizes.map((size) => (
                <SelectItem key={size} value={String(size)}>
                  {size}
                </SelectItem>
              ))}
            </SelectGroup>
          </SelectContent>
        </Select>
        <Button
          variant="outline"
          size="icon-sm"
          aria-label="Previous page"
          disabled={!table.getCanPreviousPage()}
          onClick={() => table.previousPage()}
        >
          <ChevronLeft />
        </Button>
        <span>
          {pageIndex + 1} / {Math.max(1, table.getPageCount())}
        </span>
        <Button
          variant="outline"
          size="icon-sm"
          aria-label="Next page"
          disabled={!table.getCanNextPage()}
          onClick={() => table.nextPage()}
        >
          <ChevronRight />
        </Button>
      </div>
    </div>
  );
}
