import {
  Badge,
  Button,
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  Textarea,
} from '@e412/rnui-react';
import { Upload } from 'lucide-react';
import { useEffect, useRef, useState } from 'react';
import { toast } from 'sonner';

import { useProxyImport } from '@/hooks/use-proxy-import';
import type { ImportItemResult, ImportReport } from '@/lib/proxy-import';
import { parseImportJson } from '@/lib/proxy-import';

interface ProxyImportDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

type Stage = 'input' | 'preview' | 'report';

function statusBadgeVariant(
  status: ImportItemResult['status'],
): 'default' | 'secondary' | 'destructive' | 'outline' {
  switch (status) {
    case 'valid':
    case 'created':
      return 'default';
    case 'conflict':
    case 'skipped_conflict':
      return 'secondary';
    case 'invalid':
    case 'failed':
      return 'destructive';
    default:
      return 'outline';
  }
}

function statusLabel(status: ImportItemResult['status']): string {
  switch (status) {
    case 'valid':
      return 'Import';
    case 'conflict':
      return 'Skip';
    case 'invalid':
      return 'Invalid';
    case 'created':
      return 'Created';
    case 'skipped_conflict':
      return 'Skipped';
    case 'failed':
      return 'Failed';
    default:
      return status;
  }
}

export function ProxyImportDialog({ open, onOpenChange }: ProxyImportDialogProps) {
  const { previewImport, applyImport, isPreviewing, isApplying } = useProxyImport();

  const [stage, setStage] = useState<Stage>('input');
  const [pasteText, setPasteText] = useState('');
  const [parseError, setParseError] = useState('');
  const [pendingItems, setPendingItems] = useState<unknown[]>([]);
  const [previewReport, setPreviewReport] = useState<ImportReport | null>(null);
  const [applyReport, setApplyReport] = useState<ImportReport | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (open) {
      setStage('input');
      setPasteText('');
      setParseError('');
      setPendingItems([]);
      setPreviewReport(null);
      setApplyReport(null);
    }
  }, [open]);

  const handlePreview = async (text: string) => {
    setParseError('');
    const result = parseImportJson(text);
    if (!result.ok) {
      setParseError(result.error);
      return;
    }
    try {
      const report = await previewImport(result.items);
      setPendingItems(result.items);
      setPreviewReport(report);
      setStage('preview');
    } catch {
      setParseError('Failed to preview import. Check your connection and try again.');
    }
  };

  const handleFileChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    const text = await file.text();
    await handlePreview(text);
    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
  };

  const handleApply = async () => {
    if (!previewReport) return;
    try {
      const report = await applyImport(pendingItems);
      setApplyReport(report);
      setStage('report');
      const s = report.summary;
      toast.success(
        `Import complete: ${s.created} created · ${s.conflicts} skipped · ${s.failed} failed`,
      );
    } catch {
      toast.error('Import failed. Please try again.');
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        {stage === 'input' && (
          <>
            <DialogHeader>
              <DialogTitle>Import Proxies</DialogTitle>
            </DialogHeader>
            <div className="space-y-4 py-2">
              <div>
                <input
                  ref={fileInputRef}
                  type="file"
                  accept="application/json,.json"
                  className="hidden"
                  onChange={handleFileChange}
                />
                <Button
                  variant="outline"
                  className="w-full"
                  onClick={() => fileInputRef.current?.click()}
                  disabled={isPreviewing}
                >
                  <Upload className="size-4" />
                  {isPreviewing ? 'Loading…' : 'Choose JSON file'}
                </Button>
              </div>
              <div className="relative flex items-center gap-2">
                <div className="h-px flex-1 bg-border" />
                <span className="text-xs text-muted-foreground">or paste JSON</span>
                <div className="h-px flex-1 bg-border" />
              </div>
              <Textarea
                value={pasteText}
                onChange={(e) => setPasteText(e.target.value)}
                placeholder='[{"name": "my-proxy", ...}]'
                className="min-h-[120px] font-mono text-xs"
              />
              {parseError && <p className="text-sm text-destructive">{parseError}</p>}
            </div>
            <DialogFooter>
              <Button variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button
                onClick={() => handlePreview(pasteText)}
                disabled={!pasteText.trim() || isPreviewing}
              >
                {isPreviewing ? 'Previewing…' : 'Preview'}
              </Button>
            </DialogFooter>
          </>
        )}

        {stage === 'preview' && previewReport && (
          <>
            <DialogHeader>
              <DialogTitle>Preview Import</DialogTitle>
            </DialogHeader>
            <div className="space-y-3 py-2">
              <p className="text-sm text-muted-foreground">
                {previewReport.summary.importable} importable · {previewReport.summary.conflicts}{' '}
                conflicts · {previewReport.summary.invalid} invalid
              </p>
              <div className="max-h-72 overflow-y-auto rounded-md border divide-y">
                {previewReport.items.map((item) => (
                  <div
                    key={item.index}
                    className="flex items-start justify-between gap-2 px-3 py-2"
                  >
                    <div className="min-w-0">
                      <p className="text-sm font-medium truncate">{item.name}</p>
                      <p className="text-xs text-muted-foreground truncate">{item.hostname}</p>
                      {item.reason && (
                        <p className="text-xs text-muted-foreground mt-0.5">{item.reason}</p>
                      )}
                    </div>
                    <Badge variant={statusBadgeVariant(item.status)} className="shrink-0 text-xs">
                      {statusLabel(item.status)}
                    </Badge>
                  </div>
                ))}
              </div>
            </div>
            <DialogFooter>
              <Button variant="outline" onClick={() => setStage('input')}>
                Back
              </Button>
              <Button
                onClick={handleApply}
                disabled={previewReport.summary.importable === 0 || isApplying}
              >
                {isApplying
                  ? 'Importing…'
                  : `Import ${previewReport.summary.importable} ${previewReport.summary.importable === 1 ? 'proxy' : 'proxies'}`}
              </Button>
            </DialogFooter>
          </>
        )}

        {stage === 'report' && applyReport && (
          <>
            <DialogHeader>
              <DialogTitle>Import Complete</DialogTitle>
            </DialogHeader>
            <div className="space-y-3 py-2">
              <p className="text-sm text-muted-foreground">
                Created {applyReport.summary.created} · Skipped {applyReport.summary.conflicts} ·
                Failed {applyReport.summary.failed}
              </p>
              <div className="max-h-72 overflow-y-auto rounded-md border divide-y">
                {applyReport.items.map((item) => (
                  <div
                    key={item.index}
                    className="flex items-start justify-between gap-2 px-3 py-2"
                  >
                    <div className="min-w-0">
                      <p className="text-sm font-medium truncate">{item.name}</p>
                      <p className="text-xs text-muted-foreground truncate">{item.hostname}</p>
                      {item.reason && (
                        <p className="text-xs text-muted-foreground mt-0.5">{item.reason}</p>
                      )}
                    </div>
                    <Badge variant={statusBadgeVariant(item.status)} className="shrink-0 text-xs">
                      {statusLabel(item.status)}
                    </Badge>
                  </div>
                ))}
              </div>
            </div>
            <DialogFooter>
              <Button onClick={() => onOpenChange(false)}>Done</Button>
            </DialogFooter>
          </>
        )}
      </DialogContent>
    </Dialog>
  );
}
