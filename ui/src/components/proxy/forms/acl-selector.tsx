import { Link } from '@tanstack/react-router';
import { Plus, SlidersHorizontal, Trash2 } from 'lucide-react';
import { useState } from 'react';

import { Button } from '@/components/ui/button';
import { Field, FieldLabel } from '@/components/ui/field';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Sheet, SheetContent, SheetFooter, SheetHeader, SheetTitle } from '@/components/ui/sheet';
import { Switch } from '@/components/ui/switch';
import { useACLGroups } from '@/hooks';
import { usePermissions } from '@/hooks/use-permissions';

export interface ACLAssignment {
  acl_group_id: number;
  path_pattern: string;
  priority: number;
  enabled: boolean;
}
interface ACLSelectorProps {
  value: ACLAssignment[];
  onChange: (value: ACLAssignment[]) => void;
  disabled?: boolean;
}

export function ACLSelector({ value, onChange, disabled }: ACLSelectorProps) {
  const { canReadAccess, canUpdateAccess, canDeleteAccess } = usePermissions();
  const { groups, isLoading, isError } = useACLGroups({ limit: 100 }, canReadAccess);
  disabled = disabled || !canUpdateAccess || isError || (value.length > 0 && !canDeleteAccess);
  const [open, setOpen] = useState(false);
  const [draft, setDraft] = useState<(ACLAssignment & { rowId: string })[]>([]);
  const defaults = value.filter((a) => a.path_pattern === '/*');
  const selected =
    defaults.length === 0
      ? 'public'
      : defaults.length === 1 && defaults[0].enabled
        ? String(defaults[0].acl_group_id)
        : 'custom';
  const items = [
    { value: 'public', label: 'Public · no sign in' },
    ...groups.map((g) => ({ value: String(g.id), label: g.name })),
    ...(selected === 'custom' ? [{ value: 'custom', label: 'Custom access rules' }] : []),
  ];
  if (selected !== 'public' && selected !== 'custom' && !items.some((i) => i.value === selected))
    items.push({ value: selected, label: `Access group ${selected}` });
  const update = (index: number, patch: Partial<ACLAssignment>) =>
    setDraft(draft.map((a, i) => (i === index ? { ...a, ...patch } : a)));
  const valid =
    new Set(draft.map((a) => a.acl_group_id)).size === draft.length &&
    draft.every(
      (a) =>
        a.acl_group_id > 0 &&
        a.path_pattern.startsWith('/') &&
        Number.isInteger(a.priority) &&
        a.priority >= 0 &&
        a.priority <= 1000,
    );
  if (!canReadAccess) return null;
  return (
    <Field orientation="horizontal" className="flex flex-wrap items-center justify-between gap-3">
      <FieldLabel htmlFor="proxy-access">Access</FieldLabel>
      <div className="flex flex-1 flex-wrap justify-end gap-2">
        <Select
          items={items}
          value={selected}
          disabled={disabled || isLoading}
          onValueChange={(v) => {
            if (!v || v === 'custom') return;
            const paths = value.filter(
              (a) => a.path_pattern !== '/*' && (v === 'public' || a.acl_group_id !== Number(v)),
            );
            onChange(
              v === 'public'
                ? paths
                : [
                    { acl_group_id: Number(v), path_pattern: '/*', priority: 100, enabled: true },
                    ...paths,
                  ],
            );
          }}
        >
          <SelectTrigger id="proxy-access" className="w-60">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectGroup>
              {items.map((i) => (
                <SelectItem key={i.value} value={i.value}>
                  {i.label}
                </SelectItem>
              ))}
            </SelectGroup>
          </SelectContent>
        </Select>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          disabled={disabled}
          onClick={() => {
            setDraft(value.map((assignment) => ({ ...assignment, rowId: crypto.randomUUID() })));
            setOpen(true);
          }}
        >
          <SlidersHorizontal />
          <span className="sr-only">Path rules</span>
        </Button>
      </div>
      <Sheet open={open} onOpenChange={setOpen}>
        <SheetContent className="sm:max-w-xl" aria-describedby={undefined}>
          <SheetHeader>
            <SheetTitle>Access rules</SheetTitle>
          </SheetHeader>
          <div className="flex min-h-0 flex-1 flex-col gap-5 overflow-auto px-6 py-2">
            {draft.map((a, i) => (
              <div key={a.rowId} className="flex flex-col gap-4 rounded-lg border p-4">
                <Field>
                  <FieldLabel htmlFor={`rule-group-${i}`}>Access group</FieldLabel>
                  <Select
                    items={groups.map((g) => ({ value: String(g.id), label: g.name }))}
                    value={String(a.acl_group_id)}
                    onValueChange={(v) => v && update(i, { acl_group_id: Number(v) })}
                  >
                    <SelectTrigger className="w-full" id={`rule-group-${i}`}>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectGroup>
                        {groups.map((g) => (
                          <SelectItem key={g.id} value={String(g.id)}>
                            {g.name}
                          </SelectItem>
                        ))}
                      </SelectGroup>
                    </SelectContent>
                  </Select>
                </Field>
                <div className="grid grid-cols-2 gap-4">
                  <Field>
                    <FieldLabel htmlFor={`rule-path-${i}`}>Path</FieldLabel>
                    <Input
                      id={`rule-path-${i}`}
                      className="font-mono"
                      value={a.path_pattern}
                      onChange={(e) => update(i, { path_pattern: e.target.value })}
                      placeholder="/admin/*"
                    />
                  </Field>
                  <Field>
                    <FieldLabel htmlFor={`rule-priority-${i}`}>Priority</FieldLabel>
                    <Input
                      id={`rule-priority-${i}`}
                      type="number"
                      min={0}
                      max={1000}
                      value={Number.isNaN(a.priority) ? '' : a.priority}
                      onChange={(e) => update(i, { priority: e.target.valueAsNumber })}
                    />
                  </Field>
                </div>
                <div className="flex items-center justify-between">
                  <label className="flex items-center gap-2 text-sm">
                    <Switch
                      checked={a.enabled}
                      onCheckedChange={(enabled) => update(i, { enabled })}
                    />
                    Enabled
                  </label>
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon-sm"
                    aria-label="Remove rule"
                    onClick={() => setDraft(draft.filter((_, index) => index !== i))}
                  >
                    <Trash2 />
                  </Button>
                </div>
              </div>
            ))}
            {groups.length > 0 ? (
              <Button
                type="button"
                variant="outline"
                onClick={() =>
                  setDraft([
                    ...draft,
                    {
                      rowId: crypto.randomUUID(),
                      acl_group_id: (
                        groups.find(
                          (group) => !draft.some((rule) => rule.acl_group_id === group.id),
                        ) ?? groups[0]
                      ).id,
                      path_pattern: '/*',
                      priority: 100,
                      enabled: true,
                    },
                  ])
                }
              >
                <Plus data-icon="inline-start" />
                Add rule
              </Button>
            ) : (
              <Link to="/access" className="text-sm text-accent-foreground underline">
                Create an access group
              </Link>
            )}
            {!valid && (
              <p role="alert" className="text-sm text-destructive">
                Each rule needs a unique group, a path starting with /, and a priority from 0 to
                1000.
              </p>
            )}
          </div>
          <SheetFooter className="flex-row justify-end border-t">
            <Button type="button" variant="outline" onClick={() => setOpen(false)}>
              Cancel
            </Button>
            <Button
              type="button"
              disabled={!valid || disabled}
              onClick={() => {
                onChange(draft.map(({ rowId: _rowId, ...assignment }) => assignment));
                setOpen(false);
              }}
            >
              Apply
            </Button>
          </SheetFooter>
        </SheetContent>
      </Sheet>
    </Field>
  );
}
