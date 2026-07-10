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
  CardDescription,
  CardHeader,
  CardTitle,
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  Input,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@e412/rnui-react';
import { zodResolver } from '@hookform/resolvers/zod';
import { useEffect, useState } from 'react';
import { type Control, useForm } from 'react-hook-form';

import { type ProxyGroupFormData, proxyGroupSchema } from '@/lib/form-validation';
import type { CreateProxyGroupRequest, ProxyGroup } from '@/types/proxy-group';

interface ProxyGroupFormProps {
  mode: 'create' | 'edit';
  initialData?: ProxyGroup | null;
  onSubmit: (data: CreateProxyGroupRequest) => void;
  loading: boolean;
  onCancel: () => void;
}

const DEFAULTS: ProxyGroupFormData = {
  name: '',
  description: '',
  base_domain: '',
  ssl_enabled: null,
  ssl_forced: null,
  tls_insecure_skip_verify: null,
  block_exploits: null,
};

function toFormDefaults(group: ProxyGroup): ProxyGroupFormData {
  return {
    name: group.name,
    description: group.description ?? '',
    base_domain: group.base_domain ?? '',
    ssl_enabled: group.ssl_enabled,
    ssl_forced: group.ssl_forced,
    tls_insecure_skip_verify: group.tls_insecure_skip_verify,
    block_exploits: group.block_exploits,
  };
}

// Tri-state: 'inherit' <-> null. Never collapse null to false — a group that
// says nothing about a setting is not a group that disables it.
const TRI_STATE_ITEMS = [
  { value: 'inherit', label: 'Inherit (system default)' },
  { value: 'true', label: 'Enabled' },
  { value: 'false', label: 'Disabled' },
];

function toSelectValue(value: boolean | null): string {
  return value === null ? 'inherit' : String(value);
}

function fromSelectValue(value: string): boolean | null {
  return value === 'inherit' ? null : value === 'true';
}

type TriStateFieldName =
  | 'ssl_enabled'
  | 'ssl_forced'
  | 'tls_insecure_skip_verify'
  | 'block_exploits';

function TriStateField({
  control,
  name,
  label,
  description,
}: {
  control: Control<ProxyGroupFormData>;
  name: TriStateFieldName;
  label: string;
  description: string;
}) {
  return (
    <FormField
      control={control}
      name={name}
      render={({ field }) => (
        <FormItem>
          <FormLabel>{label}</FormLabel>
          <Select
            items={TRI_STATE_ITEMS}
            value={toSelectValue(field.value)}
            onValueChange={(value) => field.onChange(fromSelectValue(value))}
          >
            <FormControl>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
            </FormControl>
            <SelectContent>
              {TRI_STATE_ITEMS.map((item) => (
                <SelectItem key={item.value} value={item.value}>
                  {item.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <FormDescription>{description}</FormDescription>
          <FormMessage />
        </FormItem>
      )}
    />
  );
}

export function ProxyGroupForm({
  mode,
  initialData,
  onSubmit,
  loading,
  onCancel,
}: ProxyGroupFormProps) {
  const form = useForm<ProxyGroupFormData>({
    resolver: zodResolver(proxyGroupSchema),
    mode: 'onTouched',
    defaultValues: initialData ? toFormDefaults(initialData) : DEFAULTS,
  });

  // Group data arrives async on edit — reset once it lands.
  useEffect(() => {
    if (initialData) form.reset(toFormDefaults(initialData));
  }, [initialData, form]);

  // Base-domain change on a group that already has one and has members
  // re-homes every member. That's confirmed before submit, not silently done.
  const [pendingValues, setPendingValues] = useState<ProxyGroupFormData | null>(null);

  const buildRequest = (values: ProxyGroupFormData): CreateProxyGroupRequest => ({
    name: values.name,
    description: values.description || undefined,
    base_domain: values.base_domain || undefined,
    ssl_enabled: values.ssl_enabled,
    ssl_forced: values.ssl_forced,
    tls_insecure_skip_verify: values.tls_insecure_skip_verify,
    block_exploits: values.block_exploits,
    // No custom-headers editor in this form (out of scope here). PUT
    // replaces the whole record, so pass the existing value through
    // untouched rather than silently clearing it when other fields change.
    custom_headers: initialData?.custom_headers,
  });

  const rehomesMembers = (values: ProxyGroupFormData): boolean =>
    mode === 'edit' &&
    !!initialData?.base_domain &&
    (initialData?.member_count ?? 0) > 0 &&
    (values.base_domain || undefined) !== initialData.base_domain;

  const handleValid = (values: ProxyGroupFormData) => {
    if (rehomesMembers(values)) {
      setPendingValues(values);
      return;
    }
    onSubmit(buildRequest(values));
  };

  const confirmRehome = () => {
    if (pendingValues) onSubmit(buildRequest(pendingValues));
    setPendingValues(null);
  };

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(handleValid)} className="space-y-6">
        <Card>
          <CardHeader>
            <CardTitle>Basics</CardTitle>
            <CardDescription>
              Name this group and (optionally) give its members a shared base domain.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <FormField
              control={form.control}
              name="name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Name</FormLabel>
                  <FormControl>
                    <Input autoFocus {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="description"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Description</FormLabel>
                  <FormControl>
                    <Input {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="base_domain"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Base Domain</FormLabel>
                  <FormControl>
                    <Input placeholder="group.acme.in" {...field} />
                  </FormControl>
                  <FormDescription>
                    Label-addressed member proxies are hosted under this domain. Changing it on a
                    group that already has members re-homes them.
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Inherited Settings</CardTitle>
            <CardDescription>
              Members that don't set their own value fall through to what's chosen here. "Inherit"
              leaves the choice to the system default — it is not the same as "Disabled".
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <TriStateField
              control={form.control}
              name="ssl_enabled"
              label="Enable HTTPS"
              description="Automatically obtain and manage SSL/TLS certificates"
            />
            <TriStateField
              control={form.control}
              name="ssl_forced"
              label="Force HTTPS"
              description="Redirect plain HTTP requests to HTTPS"
            />
            <TriStateField
              control={form.control}
              name="block_exploits"
              label="Block Common Exploits"
              description="Block SQL injection, XSS, and other common attacks"
            />
            <TriStateField
              control={form.control}
              name="tls_insecure_skip_verify"
              label="Allow Self-Signed Certificates"
              description="Trust backend servers even if their certificate isn't from a public authority"
            />
          </CardContent>
        </Card>

        <div className="flex justify-end gap-2">
          <Button type="button" variant="outline" onClick={onCancel}>
            Cancel
          </Button>
          <Button type="submit" disabled={loading}>
            {loading ? 'Saving…' : mode === 'create' ? 'Create Proxy Group' : 'Save Changes'}
          </Button>
        </div>
      </form>

      <AlertDialog open={!!pendingValues} onOpenChange={(open) => !open && setPendingValues(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Change base domain?</AlertDialogTitle>
            <AlertDialogDescription>
              Changing the base domain re-homes all <strong>{initialData?.member_count}</strong>{' '}
              member proxies. Their old hostnames stop resolving immediately and Caddy will request
              new certificates for every new hostname. Continue?
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction onClick={confirmRehome}>Continue</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </Form>
  );
}
