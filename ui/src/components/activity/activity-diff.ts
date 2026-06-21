export interface FieldChange {
  field: string;
  oldValue: unknown;
  newValue: unknown;
}

function isOldNew(v: unknown): v is { old: unknown; new: unknown } {
  return (
    typeof v === 'object' &&
    v !== null &&
    !Array.isArray(v) &&
    'old' in (v as Record<string, unknown>) &&
    'new' in (v as Record<string, unknown>)
  );
}

export function extractFieldChanges(
  details: Record<string, unknown> | null | undefined,
): FieldChange[] {
  // Every update event nests its per-field {old,new} map under `details.changes`
  // (audit_service.go); the rest of details is metadata. Read only that.
  const changes = details?.changes;
  if (!changes || typeof changes !== 'object' || Array.isArray(changes)) return [];
  const out: FieldChange[] = [];
  for (const [field, value] of Object.entries(changes as Record<string, unknown>)) {
    if (isOldNew(value)) {
      out.push({ field, oldValue: value.old, newValue: value.new });
    }
  }
  return out;
}

export function formatDiffValue(v: unknown): string {
  if (v === null || v === undefined) return '—';
  if (typeof v === 'string') return v;
  if (typeof v === 'boolean' || typeof v === 'number') return String(v);
  try {
    return JSON.stringify(v);
  } catch {
    return String(v);
  }
}
