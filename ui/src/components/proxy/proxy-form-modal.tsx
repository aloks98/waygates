import { Dialog, DialogBody, DialogContent, DialogHeader, DialogTitle } from '@e412/titanium';
import type { CreateProxyRequest, ProxyConfig, ProxyType } from '@/types/proxy';
import { getProxyTypeIcon } from './cells';
import { type ACLAssignment, RedirectForm, ReverseProxyForm, StaticForm } from './forms';

interface ProxyFormModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSubmit: (data: CreateProxyRequest, aclAssignments?: ACLAssignment[]) => void;
  initialData?: ProxyConfig | null;
  initialACLAssignments?: ACLAssignment[];
  proxyType: ProxyType;
  title: string;
  loading: boolean;
}

export function ProxyFormModal({
  open,
  onOpenChange,
  onSubmit,
  initialData,
  initialACLAssignments,
  proxyType,
  title,
  loading,
}: ProxyFormModalProps) {
  const handleCancel = () => onOpenChange(false);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            {getProxyTypeIcon(proxyType)}
            {title}
          </DialogTitle>
        </DialogHeader>
        <DialogBody>
          {proxyType === 'reverse_proxy' && (
            <ReverseProxyForm
              initialData={initialData}
              initialACLAssignments={initialACLAssignments}
              onSubmit={onSubmit}
              loading={loading}
              onCancel={handleCancel}
            />
          )}
          {proxyType === 'redirect' && (
            <RedirectForm
              initialData={initialData}
              initialACLAssignments={initialACLAssignments}
              onSubmit={onSubmit}
              loading={loading}
              onCancel={handleCancel}
            />
          )}
          {proxyType === 'static' && (
            <StaticForm
              initialData={initialData}
              initialACLAssignments={initialACLAssignments}
              onSubmit={onSubmit}
              loading={loading}
              onCancel={handleCancel}
            />
          )}
        </DialogBody>
      </DialogContent>
    </Dialog>
  );
}
