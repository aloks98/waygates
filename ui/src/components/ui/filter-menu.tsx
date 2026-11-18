import { Plus, X } from 'lucide-react';

import { useIsMobile } from '@/hooks/use-mobile';

import { Button } from './button';
import { ButtonGroup } from './button-group';
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from './dropdown-menu';
import type { Filter } from './filters';

export interface MenuFilterField<T extends string = string> {
  key: T;
  label: string;
  options: { value: string; label: string }[];
  anyLabel?: string;
  allowExclusion?: boolean;
}

function FilterChoices<T extends string>({
  field,
  filter,
  onChange,
}: {
  field: MenuFilterField<T>;
  filter?: Filter<T>;
  onChange: (value: string, excluded?: boolean) => void;
}) {
  const value = filter?.values[0] ?? '';
  return (
    <>
      <DropdownMenuGroup>
        <DropdownMenuLabel>{field.label}</DropdownMenuLabel>
        <DropdownMenuRadioGroup value={value} onValueChange={(next: string) => onChange(next)}>
          <DropdownMenuRadioItem value="" closeOnClick className="min-h-9">
            {field.anyLabel ?? `Any ${field.label.toLowerCase()}`}
          </DropdownMenuRadioItem>
          {field.options.map((option) => (
            <DropdownMenuRadioItem
              key={option.value}
              value={option.value}
              closeOnClick
              className="min-h-9"
            >
              {option.label}
            </DropdownMenuRadioItem>
          ))}
        </DropdownMenuRadioGroup>
      </DropdownMenuGroup>
      {field.allowExclusion && (
        <>
          <DropdownMenuSeparator />
          <DropdownMenuGroup>
            <DropdownMenuCheckboxItem
              checked={filter?.operator === 'is_not'}
              disabled={!value}
              onCheckedChange={(excluded) => onChange(value, excluded)}
              className="min-h-9"
            >
              Exclude selected {field.label.toLowerCase()}
            </DropdownMenuCheckboxItem>
          </DropdownMenuGroup>
        </>
      )}
    </>
  );
}

export function FilterMenu<T extends string>({
  filters,
  fields,
  onChange,
}: {
  filters: Filter<T>[];
  fields: MenuFilterField<T>[];
  onChange: (filters: Filter<T>[]) => void;
}) {
  const isMobile = useIsMobile();

  function update(field: MenuFilterField<T>, value: string, excluded?: boolean) {
    const current = filters.find((filter) => filter.field === field.key);
    if (!value) {
      onChange(filters.filter((filter) => filter.field !== field.key));
      return;
    }
    const next: Filter<T> = {
      id: current?.id ?? crypto.randomUUID(),
      field: field.key,
      operator:
        field.allowExclusion && (excluded ?? current?.operator === 'is_not') ? 'is_not' : 'is',
      values: [value],
    };
    onChange(
      current
        ? filters.map((filter) => (filter.field === field.key ? next : filter))
        : [...filters, next],
    );
  }

  return (
    <div className="flex min-w-0 flex-wrap items-center gap-2">
      <DropdownMenu>
        <DropdownMenuTrigger render={<Button variant="outline" />}>
          <Plus data-icon="inline-start" />
          Add filter
        </DropdownMenuTrigger>
        <DropdownMenuContent className="w-48">
          <DropdownMenuGroup>
            <DropdownMenuLabel>Filter by</DropdownMenuLabel>
            {fields.map((field) => (
              <DropdownMenuSub key={field.key}>
                <DropdownMenuSubTrigger className="min-h-9">{field.label}</DropdownMenuSubTrigger>
                <DropdownMenuSubContent
                  className="w-60"
                  side={isMobile ? 'bottom' : 'right'}
                  sideOffset={4}
                  alignOffset={-4}
                >
                  <FilterChoices
                    field={field}
                    filter={filters.find((filter) => filter.field === field.key)}
                    onChange={(value, excluded) => update(field, value, excluded)}
                  />
                </DropdownMenuSubContent>
              </DropdownMenuSub>
            ))}
          </DropdownMenuGroup>
        </DropdownMenuContent>
      </DropdownMenu>

      {filters.map((filter) => {
        const field = fields.find((candidate) => candidate.key === filter.field);
        if (!field) return null;
        const valueLabel =
          field.options.find((option) => option.value === filter.values[0])?.label ??
          filter.values[0];
        const label = `${field.label}: ${filter.operator === 'is_not' ? 'Not ' : ''}${valueLabel}`;
        return (
          <ButtonGroup key={filter.id} aria-label={`${field.label} filter`} className="max-w-full">
            <DropdownMenu>
              <DropdownMenuTrigger
                render={<Button variant="outline" className="min-w-0" />}
                aria-label={`Edit ${field.label} filter`}
              >
                <span className="max-w-56 truncate" title={label}>
                  {label}
                </span>
              </DropdownMenuTrigger>
              <DropdownMenuContent className="w-60">
                <FilterChoices
                  field={field}
                  filter={filter}
                  onChange={(value, excluded) => update(field, value, excluded)}
                />
              </DropdownMenuContent>
            </DropdownMenu>
            <Button
              variant="outline"
              size="icon"
              className="w-8"
              aria-label={`Remove ${field.label} filter`}
              onClick={() => update(field, '')}
            >
              <X />
            </Button>
          </ButtonGroup>
        );
      })}

      {filters.length > 0 && (
        <Button variant="ghost" size="sm" onClick={() => onChange([])}>
          Clear all
        </Button>
      )}
    </div>
  );
}
