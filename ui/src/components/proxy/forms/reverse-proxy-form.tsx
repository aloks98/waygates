import {
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardHeading,
  CardTitle,
  CardToolbar,
  Field,
  FieldContent,
  FieldDescription,
  FieldError,
  FieldLabel,
  Input,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Switch,
} from '@e412/titanium';
import { useForm } from '@tanstack/react-form';
import { Plus, Trash2 } from 'lucide-react';
import { useEffect, useMemo, useState } from 'react';
import { z } from 'zod';
import type { CreateReverseProxyRequest, ProxyConfig } from '@/types/proxy';
import { type ACLAssignment, ACLSelector } from './acl-selector';

const upstreamSchema = z.object({
  host: z.string().min(1, 'Host is required'),
  port: z.number().min(1, 'Port must be at least 1').max(65535, 'Port must be at most 65535'),
  scheme: z.enum(['http', 'https']),
});

const reverseProxySchema = z.object({
  name: z.string().min(1, 'Name is required').max(255, 'Name must be at most 255 characters'),
  hostname: z
    .string()
    .min(1, 'Hostname is required')
    .max(253, 'Hostname must be at most 253 characters'),
  description: z.string().max(500, 'Description must be at most 500 characters').optional(),
  upstreams: z.array(upstreamSchema).min(1, 'Add at least one backend server'),
  ssl_enabled: z.boolean(),
  block_exploits: z.boolean(),
  tls_insecure_skip_verify: z.boolean(),
  lb_strategy: z.enum(['round_robin', 'least_conn', 'ip_hash', 'random']),
  health_check_enabled: z.boolean(),
  health_check_path: z.string(),
  health_check_interval: z.string(),
  health_check_timeout: z.string(),
});

type ReverseProxyFormValues = z.infer<typeof reverseProxySchema>;

interface ReverseProxyFormProps {
  initialData?: ProxyConfig | null;
  initialACLAssignments?: ACLAssignment[];
  onSubmit: (data: CreateReverseProxyRequest, aclAssignments?: ACLAssignment[]) => void;
  loading: boolean;
  onCancel: () => void;
}

// Helper to normalize upstreams from initial data
function normalizeUpstreams(
  data?: ProxyConfig | null,
): Array<{ host: string; port: number; scheme: 'http' | 'https' }> {
  if (!data?.upstreams?.length) {
    return [{ host: '', port: 8080, scheme: 'http' }];
  }
  return data.upstreams.map((u) => ({
    host: u.host || '',
    port: u.port || 8080,
    scheme: (String(u.scheme || '').toLowerCase() === 'https' ? 'https' : 'http') as
      | 'http'
      | 'https',
  }));
}

// Helper to get a valid lb_strategy value
function getValidLbStrategy(
  strategy: string | undefined,
): 'round_robin' | 'least_conn' | 'ip_hash' | 'random' {
  const validStrategies = ['round_robin', 'least_conn', 'ip_hash', 'random'] as const;
  if (strategy && validStrategies.includes(strategy as (typeof validStrategies)[number])) {
    return strategy as (typeof validStrategies)[number];
  }
  return 'round_robin';
}

export function ReverseProxyForm({
  initialData,
  initialACLAssignments,
  onSubmit,
  loading,
  onCancel,
}: ReverseProxyFormProps) {
  const [upstreams, setUpstreams] = useState(() => normalizeUpstreams(initialData));
  const [aclAssignments, setAclAssignments] = useState<ACLAssignment[]>(
    initialACLAssignments ?? [],
  );

  // Compute default values based on initialData
  // This ensures the form is always initialized with the correct values
  const defaultValues = useMemo<ReverseProxyFormValues>(() => {
    if (initialData) {
      const upstreamData = normalizeUpstreams(initialData);
      return {
        name: initialData.name || '',
        hostname: initialData.hostname || '',
        description: initialData.description || '',
        upstreams: upstreamData,
        ssl_enabled: initialData.ssl_enabled ?? true,
        block_exploits: initialData.block_exploits ?? true,
        tls_insecure_skip_verify: initialData.tls_insecure_skip_verify ?? false,
        lb_strategy: getValidLbStrategy(initialData.load_balancing?.strategy),
        health_check_enabled: initialData.load_balancing?.health_checks?.enabled ?? false,
        health_check_path: initialData.load_balancing?.health_checks?.path || '/health',
        health_check_interval: initialData.load_balancing?.health_checks?.interval || '30s',
        health_check_timeout: initialData.load_balancing?.health_checks?.timeout || '5s',
      };
    }
    return {
      name: '',
      hostname: '',
      description: '',
      upstreams: [{ host: '', port: 8080, scheme: 'http' as const }],
      ssl_enabled: true,
      block_exploits: true,
      tls_insecure_skip_verify: false,
      lb_strategy: 'round_robin' as const,
      health_check_enabled: false,
      health_check_path: '/health',
      health_check_interval: '30s',
      health_check_timeout: '5s',
    };
  }, [initialData]);

  const form = useForm({
    defaultValues,
    validators: {
      onSubmit: reverseProxySchema,
    },
    onSubmit: async ({ value }) => {
      const data: CreateReverseProxyRequest = {
        type: 'reverse_proxy',
        name: value.name,
        hostname: value.hostname,
        description: value.description || undefined,
        ssl_enabled: value.ssl_enabled,
        upstreams: value.upstreams,
        block_exploits: value.block_exploits,
        tls_insecure_skip_verify: value.tls_insecure_skip_verify,
      };

      if (value.upstreams.length > 1) {
        data.load_balancing = {
          strategy: value.lb_strategy,
          health_checks: value.health_check_enabled
            ? {
                enabled: true,
                path: value.health_check_path,
                interval: value.health_check_interval,
                timeout: value.health_check_timeout,
                unhealthy_threshold: 3,
                healthy_threshold: 2,
              }
            : undefined,
        };
      }

      onSubmit(data, aclAssignments.length > 0 ? aclAssignments : undefined);
    },
  });

  // Reset form when initialData changes (for edit mode)
  // Use defaultValues from useMemo to avoid duplication
  useEffect(() => {
    if (initialData) {
      setUpstreams(normalizeUpstreams(initialData));
      form.reset(defaultValues);
    }
  }, [initialData, form, defaultValues]);

  // Update ACL assignments when initialACLAssignments changes (async load)
  useEffect(() => {
    if (initialACLAssignments) {
      setAclAssignments(initialACLAssignments);
    }
  }, [initialACLAssignments]);

  const addUpstream = () => {
    const newUpstreams = [...upstreams, { host: '', port: 8080, scheme: 'http' as const }];
    setUpstreams(newUpstreams);
    form.setFieldValue('upstreams', newUpstreams);
  };

  const removeUpstream = (index: number) => {
    const newUpstreams = upstreams.filter((_, i) => i !== index);
    setUpstreams(newUpstreams);
    form.setFieldValue('upstreams', newUpstreams);
  };

  const updateUpstream = (
    index: number,
    key: keyof (typeof upstreams)[0],
    value: string | number,
  ) => {
    const newUpstreams = [...upstreams];
    newUpstreams[index] = { ...newUpstreams[index], [key]: value };
    setUpstreams(newUpstreams);
    form.setFieldValue('upstreams', newUpstreams);
  };

  return (
    <form
      onSubmit={(e) => {
        e.preventDefault();
        e.stopPropagation();
        form.handleSubmit();
      }}
      className="space-y-6"
    >
      {/* Basic Information */}
      <Card>
        <CardHeader>
          <CardHeading>
            <CardTitle>Basic Information</CardTitle>
            <CardDescription>General settings for this reverse proxy</CardDescription>
          </CardHeading>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid gap-4 sm:grid-cols-2">
            <form.Field name="name">
              {(field) => {
                const hasError = field.state.meta.isTouched && field.state.meta.errors.length > 0;
                return (
                  <Field data-invalid={hasError}>
                    <FieldLabel htmlFor={field.name}>Name</FieldLabel>
                    <Input
                      id={field.name}
                      placeholder="My Backend API"
                      value={field.state.value}
                      onChange={(e) => field.handleChange(e.target.value)}
                      onBlur={field.handleBlur}
                      aria-invalid={hasError}
                    />
                    {hasError && <FieldError errors={field.state.meta.errors} />}
                  </Field>
                );
              }}
            </form.Field>

            <form.Field name="hostname">
              {(field) => {
                const hasError = field.state.meta.isTouched && field.state.meta.errors.length > 0;
                return (
                  <Field data-invalid={hasError}>
                    <FieldLabel htmlFor={field.name}>Hostname</FieldLabel>
                    <Input
                      id={field.name}
                      placeholder="api.example.com"
                      value={field.state.value}
                      onChange={(e) => field.handleChange(e.target.value)}
                      onBlur={field.handleBlur}
                      aria-invalid={hasError}
                    />
                    <FieldDescription>
                      The domain visitors will use to reach this service
                    </FieldDescription>
                    {hasError && <FieldError errors={field.state.meta.errors} />}
                  </Field>
                );
              }}
            </form.Field>
          </div>

          <form.Field name="description">
            {(field) => {
              const hasError = field.state.meta.isTouched && field.state.meta.errors.length > 0;
              return (
                <Field data-invalid={hasError}>
                  <FieldLabel htmlFor={field.name}>Description (optional)</FieldLabel>
                  <Input
                    id={field.name}
                    placeholder="A brief description of this proxy"
                    value={field.state.value}
                    onChange={(e) => field.handleChange(e.target.value)}
                    onBlur={field.handleBlur}
                    aria-invalid={hasError}
                  />
                  {hasError && <FieldError errors={field.state.meta.errors} />}
                </Field>
              );
            }}
          </form.Field>
        </CardContent>
      </Card>

      {/* Upstream Servers */}
      <Card>
        <CardHeader>
          <CardHeading>
            <CardTitle>Backend Servers</CardTitle>
            <CardDescription>
              Where to forward incoming traffic. Add the IP and port of your service.
            </CardDescription>
          </CardHeading>
          <CardToolbar>
            <Button type="button" variant="outline" size="sm" onClick={addUpstream}>
              <Plus className="mr-1 size-4" />
              Add Server
            </Button>
          </CardToolbar>
        </CardHeader>
        <CardContent className="space-y-3">
          {upstreams.length > 0 && (
            <div className="flex items-center gap-2 text-xs font-medium text-muted-foreground">
              <div className="w-24">Scheme</div>
              <div className="flex-1">Host</div>
              <div className="w-24">Port</div>
              {upstreams.length > 1 && <div className="w-9" />}
            </div>
          )}
          {upstreams.map((upstream, index) => (
            <div key={index} className="flex items-start gap-2">
              <div className="w-24">
                <Select
                  value={upstream.scheme}
                  onValueChange={(value: 'http' | 'https') =>
                    updateUpstream(index, 'scheme', value)
                  }
                >
                  <SelectTrigger>
                    <SelectValue placeholder="Scheme" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="http">HTTP</SelectItem>
                    <SelectItem value="https">HTTPS</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="flex-1">
                <Input
                  placeholder="192.168.1.100"
                  value={upstream.host}
                  onChange={(e) => updateUpstream(index, 'host', e.target.value)}
                />
              </div>
              <div className="w-24">
                <Input
                  type="text"
                  inputMode="numeric"
                  pattern="[0-9]*"
                  placeholder="8080"
                  value={upstream.port || ''}
                  onChange={(e) => {
                    const value = e.target.value.replace(/\D/g, '');
                    const port = value ? Math.min(parseInt(value, 10), 65535) : 0;
                    updateUpstream(index, 'port', port);
                  }}
                />
              </div>
              {upstreams.length > 1 && (
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  onClick={() => removeUpstream(index)}
                >
                  <Trash2 className="size-4 text-destructive" />
                </Button>
              )}
            </div>
          ))}
        </CardContent>
      </Card>

      {/* Load Balancing + Security side by side on large screens */}
      <div className="grid gap-6 lg:grid-cols-2">
        {upstreams.length > 1 && (
          <Card>
            <CardHeader>
              <CardHeading>
                <CardTitle>Load Balancing</CardTitle>
                <CardDescription>
                  How to distribute traffic across your backend servers
                </CardDescription>
              </CardHeading>
            </CardHeader>
            <CardContent className="space-y-4">
              <form.Field name="lb_strategy">
                {(field) => (
                  <Field>
                    <FieldLabel>Strategy</FieldLabel>
                    <Select
                      value={field.state.value}
                      onValueChange={(val) => field.handleChange(val as typeof field.state.value)}
                    >
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="round_robin">
                          Round Robin — each server takes turns
                        </SelectItem>
                        <SelectItem value="least_conn">
                          Least Connections — prefer less busy servers
                        </SelectItem>
                        <SelectItem value="ip_hash">
                          Sticky — same visitor always reaches same server
                        </SelectItem>
                        <SelectItem value="random">Random</SelectItem>
                      </SelectContent>
                    </Select>
                  </Field>
                )}
              </form.Field>

              <form.Field name="health_check_enabled">
                {(field) => (
                  <Field orientation="horizontal">
                    <FieldContent>
                      <FieldLabel>Health Checks</FieldLabel>
                      <FieldDescription>
                        Periodically check if your backend servers are reachable
                      </FieldDescription>
                    </FieldContent>
                    <Switch checked={field.state.value} onCheckedChange={field.handleChange} />
                  </Field>
                )}
              </form.Field>

              <form.Subscribe selector={(state) => state.values.health_check_enabled}>
                {(healthCheckEnabled) =>
                  healthCheckEnabled && (
                    <div className="space-y-4">
                      <form.Field name="health_check_path">
                        {(field) => (
                          <Field>
                            <FieldLabel>Path</FieldLabel>
                            <Input
                              placeholder="/health"
                              value={field.state.value}
                              onChange={(e) => field.handleChange(e.target.value)}
                            />
                          </Field>
                        )}
                      </form.Field>
                      <div className="grid gap-4 grid-cols-2">
                        <form.Field name="health_check_interval">
                          {(field) => (
                            <Field>
                              <FieldLabel>Check Every</FieldLabel>
                              <Input
                                placeholder="30s"
                                value={field.state.value}
                                onChange={(e) => field.handleChange(e.target.value)}
                              />
                              <FieldDescription>e.g., 30s, 1m, 5m</FieldDescription>
                            </Field>
                          )}
                        </form.Field>
                        <form.Field name="health_check_timeout">
                          {(field) => (
                            <Field>
                              <FieldLabel>Timeout</FieldLabel>
                              <Input
                                placeholder="5s"
                                value={field.state.value}
                                onChange={(e) => field.handleChange(e.target.value)}
                              />
                              <FieldDescription>How long to wait for a response</FieldDescription>
                            </Field>
                          )}
                        </form.Field>
                      </div>
                    </div>
                  )
                }
              </form.Subscribe>
            </CardContent>
          </Card>
        )}

        <Card className={upstreams.length <= 1 ? 'lg:col-span-2' : ''}>
          <CardHeader>
            <CardHeading>
              <CardTitle>Security</CardTitle>
              <CardDescription>HTTPS and connection security options</CardDescription>
            </CardHeading>
          </CardHeader>
          <CardContent className="space-y-4">
            <form.Field name="ssl_enabled">
              {(field) => (
                <Field orientation="horizontal">
                  <FieldContent>
                    <FieldLabel>Enable HTTPS</FieldLabel>
                    <FieldDescription>
                      Automatically obtain and manage SSL/TLS certificates
                    </FieldDescription>
                  </FieldContent>
                  <Switch checked={field.state.value} onCheckedChange={field.handleChange} />
                </Field>
              )}
            </form.Field>

            <form.Field name="block_exploits">
              {(field) => (
                <Field orientation="horizontal">
                  <FieldContent>
                    <FieldLabel>Block Common Exploits</FieldLabel>
                    <FieldDescription>
                      Block SQL injection, XSS, and other common attacks
                    </FieldDescription>
                  </FieldContent>
                  <Switch checked={field.state.value} onCheckedChange={field.handleChange} />
                </Field>
              )}
            </form.Field>

            <form.Field name="tls_insecure_skip_verify">
              {(field) => (
                <Field orientation="horizontal">
                  <FieldContent>
                    <FieldLabel>Allow Self-Signed Certificates</FieldLabel>
                    <FieldDescription>
                      Trust the backend server even if its certificate isn't from a public authority
                    </FieldDescription>
                  </FieldContent>
                  <Switch checked={field.state.value} onCheckedChange={field.handleChange} />
                </Field>
              )}
            </form.Field>
          </CardContent>
        </Card>
      </div>

      {/* Access Control */}
      <ACLSelector value={aclAssignments} onChange={setAclAssignments} disabled={loading} />

      {/* Actions */}
      <div className="flex justify-end gap-2 pt-4 border-t">
        <Button type="button" variant="outline" onClick={onCancel}>
          Cancel
        </Button>
        <Button type="submit" disabled={loading}>
          {loading ? 'Saving...' : initialData ? 'Save Changes' : 'Create Proxy'}
        </Button>
      </div>
    </form>
  );
}
