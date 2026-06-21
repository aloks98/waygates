export type ImportItemStatus =
  | 'valid'
  | 'conflict'
  | 'invalid'
  | 'created'
  | 'skipped_conflict'
  | 'failed';

export interface ImportItemResult {
  index: number;
  name: string;
  hostname: string;
  type: string;
  status: ImportItemStatus;
  reason?: string;
}

export interface ImportSummary {
  total: number;
  importable: number;
  conflicts: number;
  invalid: number;
  created: number;
  failed: number;
}

export interface ImportReport {
  summary: ImportSummary;
  items: ImportItemResult[];
}

export function parseImportJson(
  text: string,
): { ok: true; items: unknown[] } | { ok: false; error: string } {
  let parsed: unknown;
  try {
    parsed = JSON.parse(text);
  } catch {
    return { ok: false, error: 'Not valid JSON.' };
  }
  if (!Array.isArray(parsed)) {
    return { ok: false, error: 'Expected a JSON array of proxies.' };
  }
  if (parsed.length === 0) {
    return { ok: false, error: 'The file contains no proxies.' };
  }
  return { ok: true, items: parsed };
}
