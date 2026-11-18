import { ListFilter, X } from 'lucide-react';
import { useState } from 'react';

import { Button } from './button';
import { Checkbox } from './checkbox';
import { Field, FieldLabel } from './field';
import { Input } from './input';
import { Popover, PopoverContent, PopoverTrigger } from './popover';
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from './select';
export interface Filter<T extends string = string> {
  id: string;
  field: T;
  operator: string;
  values: string[];
}
export type FilterFieldsConfig<T extends string = string> = {
  key: T;
  label: string;
  type: 'select' | 'multiselect' | 'text';
  placeholder?: string;
  defaultOperator?: string;
  options?: { value: string; label: string }[];
  operators: { value: string; label: string; supportsMultiple?: boolean }[];
}[];
export function Filters<T extends string>({
  filters,
  fields,
  onChange,
}: {
  filters: Filter<T>[];
  fields: FilterFieldsConfig<T>;
  onChange: (filters: Filter<T>[]) => void;
  variant?: string;
  size?: string;
}) {
  const [open, setOpen] = useState(false);
  const [draft, setDraft] = useState<Filter<T>>({
    id: '',
    field: fields[0].key,
    operator: fields[0].operators[0].value,
    values: [],
  });
  const field = fields.find((f) => f.key === draft.field) ?? fields[0];
  const items = fields.map((f) => ({ value: f.key, label: f.label }));
  const changeField = (key: T | null) => {
    const f = fields.find((candidate) => candidate.key === key);
    if (f)
      setDraft({
        ...draft,
        field: f.key,
        operator: f.defaultOperator ?? f.operators[0].value,
        values: [],
      });
  };
  return (
    <div className="flex flex-wrap items-center gap-2">
      {filters.map((filter) => {
        const config = fields.find((f) => f.key === filter.field);
        return (
          <div
            key={filter.id}
            className="flex h-9.5 items-center gap-1 rounded-md border bg-card px-2 text-xs"
          >
            <button
              type="button"
              onClick={() => {
                setDraft(filter);
                setOpen(true);
              }}
              className="max-w-60 truncate"
            >
              {config?.label}: {config?.operators.find((o) => o.value === filter.operator)?.label}{' '}
              {filter.values
                .map((v) => config?.options?.find((o) => o.value === v)?.label ?? v)
                .join(', ')}
            </button>
            <Button
              variant="ghost"
              size="icon-xs"
              aria-label={`Remove ${config?.label} filter`}
              onClick={() => onChange(filters.filter((f) => f.id !== filter.id))}
            >
              <X />
            </Button>
          </div>
        );
      })}
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger
          render={
            <Button
              variant="outline"
              onClick={() => {
                const f =
                  fields.find(
                    (candidate) => !filters.some((filter) => filter.field === candidate.key),
                  ) ?? fields[0];
                setDraft({
                  id: crypto.randomUUID(),
                  field: f.key,
                  operator: f.defaultOperator ?? f.operators[0].value,
                  values: [],
                });
              }}
            />
          }
        >
          <ListFilter data-icon="inline-start" />
          Filter
        </PopoverTrigger>
        <PopoverContent align="start" className="flex w-80 flex-col gap-4">
          <Field>
            <FieldLabel>Field</FieldLabel>
            <Select items={items} value={draft.field} onValueChange={changeField}>
              <SelectTrigger className="w-full" aria-label="Filter field">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  {items.map((item) => (
                    <SelectItem key={item.value} value={item.value}>
                      {item.label}
                    </SelectItem>
                  ))}
                </SelectGroup>
              </SelectContent>
            </Select>
          </Field>
          <Select
            items={field.operators}
            value={draft.operator}
            onValueChange={(v) => v && setDraft({ ...draft, operator: v })}
          >
            <SelectTrigger className="w-full" aria-label="Filter operator">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                {field.operators.map((o) => (
                  <SelectItem key={o.value} value={o.value}>
                    {o.label}
                  </SelectItem>
                ))}
              </SelectGroup>
            </SelectContent>
          </Select>
          {field.type === 'text' ? (
            <Input
              aria-label="Filter value"
              placeholder={field.placeholder}
              value={draft.values[0] ?? ''}
              onChange={(e) => setDraft({ ...draft, values: [e.target.value] })}
            />
          ) : field.type === 'multiselect' ? (
            <div className="flex max-h-52 flex-col gap-3 overflow-auto">
              {field.options?.map((o) => (
                <label key={o.value} className="flex items-center gap-2 text-sm">
                  <Checkbox
                    checked={draft.values.includes(o.value)}
                    onCheckedChange={(checked) =>
                      setDraft({
                        ...draft,
                        values: checked
                          ? [...draft.values, o.value]
                          : draft.values.filter((v) => v !== o.value),
                      })
                    }
                  />
                  {o.label}
                </label>
              ))}
            </div>
          ) : (
            <Select
              items={field.options}
              value={draft.values[0] ?? null}
              onValueChange={(v) => v && setDraft({ ...draft, values: [v] })}
            >
              <SelectTrigger className="w-full" aria-label="Filter value">
                <SelectValue placeholder="Choose a value" />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  {field.options?.map((o) => (
                    <SelectItem key={o.value} value={o.value}>
                      {o.label}
                    </SelectItem>
                  ))}
                </SelectGroup>
              </SelectContent>
            </Select>
          )}
          <Button
            disabled={!draft.values.some((v) => v.trim())}
            onClick={() => {
              onChange([
                ...filters.filter((f) => f.field !== draft.field && f.id !== draft.id),
                draft,
              ]);
              setOpen(false);
            }}
          >
            Apply
          </Button>
        </PopoverContent>
      </Popover>
    </div>
  );
}
