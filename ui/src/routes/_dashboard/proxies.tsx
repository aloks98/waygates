import { useState, useEffect } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useForm } from 'react-hook-form';
import {
  Button,
  Input,
  Card,
  CardContent,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  Badge,
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogBody,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Form,
  FormField,
  FormItem,
  FormLabel,
  FormControl,
  FormMessage,
} from '@e412/titanium';
import { Plus, Pencil, Trash2 } from 'lucide-react';
import { api } from '../../lib/api';
import type { ApiResponse, PaginatedResponse } from '../../types/api';
import type { Proxy, CreateProxyRequest, ProxyType } from '../../types/proxy';

interface ProxyFormValues {
  hostname: string;
  target_url: string;
  proxy_type: ProxyType;
}

function ProxyFormModal({
  open,
  onOpenChange,
  onSubmit,
  initialData,
  title,
  loading,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSubmit: (data: ProxyFormValues) => void;
  initialData?: Proxy | null;
  title: string;
  loading: boolean;
}) {
  const form = useForm<ProxyFormValues>({
    defaultValues: {
      hostname: '',
      target_url: '',
      proxy_type: 'reverse_proxy',
    },
  });

  useEffect(() => {
    if (open) {
      form.reset({
        hostname: initialData?.hostname || '',
        target_url: initialData?.target_url || '',
        proxy_type: initialData?.proxy_type || 'reverse_proxy',
      });
    }
  }, [open, initialData, form]);

  const handleSubmit = (values: ProxyFormValues) => {
    onSubmit(values);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
        </DialogHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(handleSubmit)}>
            <DialogBody className="space-y-4">
              <FormField
                control={form.control}
                name="hostname"
                rules={{ required: 'Hostname is required' }}
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Hostname</FormLabel>
                    <FormControl>
                      <Input placeholder="e.g., app.example.com" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="target_url"
                rules={{ required: 'Target URL is required' }}
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Target URL</FormLabel>
                    <FormControl>
                      <Input placeholder="e.g., http://localhost:3000" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="proxy_type"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Proxy Type</FormLabel>
                    <Select value={field.value} onValueChange={field.onChange}>
                      <FormControl>
                        <SelectTrigger>
                          <SelectValue placeholder="Select type" />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent>
                        <SelectItem value="reverse_proxy">Reverse Proxy</SelectItem>
                        <SelectItem value="redirect">Redirect</SelectItem>
                        <SelectItem value="static">Static</SelectItem>
                      </SelectContent>
                    </Select>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </DialogBody>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={loading}>
                {loading ? 'Saving...' : 'Save'}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
}

function DeleteConfirmModal({
  open,
  onOpenChange,
  onConfirm,
  proxyName,
  loading,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: () => void;
  proxyName: string;
  loading: boolean;
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Delete Proxy</DialogTitle>
        </DialogHeader>
        <DialogBody>
          <p className="text-muted-foreground">
            Are you sure you want to delete <strong>{proxyName}</strong>? This action cannot be undone.
          </p>
        </DialogBody>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button variant="destructive" onClick={onConfirm} disabled={loading}>
            {loading ? 'Deleting...' : 'Delete'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export function ProxiesPage() {
  const queryClient = useQueryClient();
  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [editingProxy, setEditingProxy] = useState<Proxy | null>(null);
  const [deletingProxy, setDeletingProxy] = useState<Proxy | null>(null);
  const [search, setSearch] = useState('');

  const { data, isLoading } = useQuery({
    queryKey: ['proxies', search],
    queryFn: async () => {
      const params: Record<string, string> = { limit: '100' };
      if (search) params.search = search;
      const response = await api
        .get('proxies', { searchParams: params })
        .json<ApiResponse<PaginatedResponse<Proxy>>>();
      return response.data;
    },
  });

  const createMutation = useMutation({
    mutationFn: async (data: CreateProxyRequest) => {
      const response = await api
        .post('proxies', { json: data })
        .json<ApiResponse<Proxy>>();
      return response;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['proxies'] });
      setCreateModalOpen(false);
    },
  });

  const updateMutation = useMutation({
    mutationFn: async ({ id, data }: { id: number; data: Partial<Proxy> }) => {
      const response = await api
        .put(`proxies/${id}`, { json: data })
        .json<ApiResponse<Proxy>>();
      return response;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['proxies'] });
      setEditingProxy(null);
    },
  });

  const deleteMutation = useMutation({
    mutationFn: async (id: number) => {
      await api.delete(`proxies/${id}`);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['proxies'] });
      setDeletingProxy(null);
    },
  });

  const toggleMutation = useMutation({
    mutationFn: async ({ id, enable }: { id: number; enable: boolean }) => {
      await api.post(`proxies/${id}/${enable ? 'enable' : 'disable'}`);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['proxies'] });
    },
  });

  const proxies = data?.items || [];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Proxies</h1>
        <Button onClick={() => setCreateModalOpen(true)}>
          <Plus className="size-4" />
          Add Proxy
        </Button>
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
          {isLoading ? (
            <div className="flex h-32 items-center justify-center text-muted-foreground">
              Loading...
            </div>
          ) : proxies.length === 0 ? (
            <div className="flex h-32 items-center justify-center text-muted-foreground">
              No proxies found. Create your first proxy to get started.
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Hostname</TableHead>
                  <TableHead>Target</TableHead>
                  <TableHead>Type</TableHead>
                  <TableHead>SSL</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {proxies.map((proxy) => (
                  <TableRow key={proxy.id}>
                    <TableCell className="font-medium">{proxy.hostname}</TableCell>
                    <TableCell className="text-muted-foreground">{proxy.target_url}</TableCell>
                    <TableCell>
                      <Badge variant="secondary">{proxy.proxy_type}</Badge>
                    </TableCell>
                    <TableCell>
                      {proxy.ssl_enabled ? (
                        <span className="text-green-600">Enabled</span>
                      ) : (
                        <span className="text-muted-foreground">Disabled</span>
                      )}
                    </TableCell>
                    <TableCell>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => toggleMutation.mutate({ id: proxy.id, enable: !proxy.is_active })}
                        disabled={toggleMutation.isPending}
                      >
                        <Badge variant={proxy.is_active ? 'default' : 'secondary'}>
                          {proxy.is_active ? 'Active' : 'Inactive'}
                        </Badge>
                      </Button>
                    </TableCell>
                    <TableCell className="text-right">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => setEditingProxy(proxy)}
                      >
                        <Pencil className="size-4" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => setDeletingProxy(proxy)}
                      >
                        <Trash2 className="size-4 text-destructive" />
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      <ProxyFormModal
        open={createModalOpen}
        onOpenChange={setCreateModalOpen}
        onSubmit={(data) => createMutation.mutate(data)}
        title="Create Proxy"
        loading={createMutation.isPending}
      />

      <ProxyFormModal
        open={!!editingProxy}
        onOpenChange={(open) => !open && setEditingProxy(null)}
        onSubmit={(data) => editingProxy && updateMutation.mutate({ id: editingProxy.id, data })}
        initialData={editingProxy}
        title="Edit Proxy"
        loading={updateMutation.isPending}
      />

      <DeleteConfirmModal
        open={!!deletingProxy}
        onOpenChange={(open) => !open && setDeletingProxy(null)}
        onConfirm={() => deletingProxy && deleteMutation.mutate(deletingProxy.id)}
        proxyName={deletingProxy?.hostname || ''}
        loading={deleteMutation.isPending}
      />
    </div>
  );
}
