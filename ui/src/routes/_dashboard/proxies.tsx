import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  Button,
  Card,
  CardContent,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
  Input,
} from '@e412/titanium';
import { ArrowRight, ChevronDown, FolderOpen, Globe, Plus } from 'lucide-react';
import { useState } from 'react';
import { getProxyTypeLabel, ProxyDataGrid, ProxyFormModal } from '@/components/proxy';
import { usePermissions } from '@/hooks/use-permissions';
import { useProxies } from '@/hooks/use-proxies';
import type { Proxy, ProxyType } from '@/types/proxy';

export function ProxiesPage() {
  const { canCreateProxies, canUpdateProxies, canDeleteProxies } = usePermissions();
  const {
    proxies,
    isLoading,
    create,
    update,
    remove,
    toggle,
    isCreating,
    isUpdating,
    isDeleting,
    isToggling,
  } = useProxies();

  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [createProxyType, setCreateProxyType] = useState<ProxyType>('reverse_proxy');
  const [editingProxy, setEditingProxy] = useState<Proxy | null>(null);
  const [deletingProxy, setDeletingProxy] = useState<Proxy | null>(null);
  const [search, setSearch] = useState('');

  const handleCreateProxy = (type: ProxyType) => {
    setCreateProxyType(type);
    setCreateModalOpen(true);
  };

  const handleCreate = async (data: Parameters<typeof create>[0]) => {
    await create(data);
    setCreateModalOpen(false);
  };

  const handleUpdate = async (data: Parameters<typeof create>[0]) => {
    if (!editingProxy) return;
    await update({ id: editingProxy.id, data });
    setEditingProxy(null);
  };

  const handleDelete = async () => {
    if (!deletingProxy) return;
    await remove(deletingProxy.id);
    setDeletingProxy(null);
  };

  const filteredProxies = search
    ? proxies.filter(
        (proxy) =>
          proxy.name.toLowerCase().includes(search.toLowerCase()) ||
          proxy.hostname.toLowerCase().includes(search.toLowerCase()),
      )
    : proxies;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Proxies</h1>
        {canCreateProxies && (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button>
                <Plus className="size-4" />
                Add Proxy
                <ChevronDown className="ml-1 size-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onClick={() => handleCreateProxy('reverse_proxy')}>
                <Globe className="size-4" />
                Reverse Proxy
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => handleCreateProxy('redirect')}>
                <ArrowRight className="size-4" />
                Redirect
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => handleCreateProxy('static')}>
                <FolderOpen className="size-4" />
                Static File Server
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        )}
      </div>

      <div>
        <Input
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder="Search proxies..."
          className="max-w-sm"
        />
      </div>

      <Card>
        <CardContent className="p-0">
          <ProxyDataGrid
            data={filteredProxies}
            isLoading={isLoading}
            canUpdateProxies={canUpdateProxies}
            canDeleteProxies={canDeleteProxies}
            onEdit={setEditingProxy}
            onDelete={setDeletingProxy}
            onToggleStatus={(id, enable) => toggle({ id, enable })}
            isToggling={isToggling}
          />
        </CardContent>
      </Card>

      <ProxyFormModal
        open={createModalOpen}
        onOpenChange={setCreateModalOpen}
        onSubmit={handleCreate}
        proxyType={createProxyType}
        title={`Create ${getProxyTypeLabel(createProxyType)}`}
        loading={isCreating}
      />

      {editingProxy && (
        <ProxyFormModal
          open={!!editingProxy}
          onOpenChange={(open) => !open && setEditingProxy(null)}
          onSubmit={handleUpdate}
          initialData={editingProxy}
          proxyType={editingProxy.type}
          title={`Edit ${getProxyTypeLabel(editingProxy.type)}`}
          loading={isUpdating}
        />
      )}

      <AlertDialog open={!!deletingProxy} onOpenChange={(open) => !open && setDeletingProxy(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Proxy</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete{' '}
              <strong>{deletingProxy?.name || deletingProxy?.hostname}</strong>? This action cannot
              be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDelete}
              disabled={isDeleting}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {isDeleting ? 'Deleting...' : 'Delete'}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
