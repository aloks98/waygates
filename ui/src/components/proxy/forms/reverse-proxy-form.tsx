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
import { useEffect, useState } from 'react';
import { z } from 'zod';
import type { CreateReverseProxyRequest, ProxyConfig } from '@/types/proxy';

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
  upstreams: z.array(upstreamSchema).min(1, 'At least one upstream is required'),
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
  onSubmit: (data: CreateReverseProxyRequest) => void;
  loading: boolean;
  onCancel: () => void;
}

export function ReverseProxyForm({
  initialData,
  onSubmit,
  loading,
  onCancel,
}: ReverseProxyFormProps) {
  const [upstreams, setUpstreams] = useState<
    Array<{ host: string; port: number; scheme: 'http' | 'https' }>
  >([{ host: '', port: 8080, scheme: 'http' }]);

  const form = useForm({
    defaultValues: {
      name: '',
      hostname: '',
      description: '',
      upstreams: [{ host: '', port: 8080, scheme: 'http' as const }],
      block_exploits: true,
      tls_insecure_skip_verify: false,
      lb_strategy: 'round_robin' as const,
      health_check_enabled: false,
      health_check_path: '/health',
      health_check_interval: '30s',
      health_check_timeout: '5s',
    } as ReverseProxyFormValues,
    validators: {
      onSubmit: reverseProxySchema,
    },
    onSubmit: async ({ value }) => {
      const data: CreateReverseProxyRequest = {
        type: 'reverse_proxy',
        name: value.name,
        hostname: value.hostname,
        description: value.description || undefined,
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

      onSubmit(data);
    },
  });

  useEffect(() => {
    if (initialData) {
      const upstreamData = initialData.upstreams?.length
        ? initialData.upstreams
        : [{ host: '', port: 8080, scheme: 'http' as const }];

      setUpstreams(upstreamData);

      form.setFieldValue('name', initialData.name || '');
      form.setFieldValue('hostname', initialData.hostname || '');
      form.setFieldValue('description', initialData.description || '');
      form.setFieldValue('upstreams', upstreamData);
      form.setFieldValue('block_exploits', initialData.block_exploits ?? true);
      form.setFieldValue('tls_insecure_skip_verify', initialData.tls_insecure_skip_verify ?? false);
      form.setFieldValue('lb_strategy', initialData.load_balancing?.strategy || 'round_robin');
      form.setFieldValue(
        'health_check_enabled',
        initialData.load_balancing?.health_checks?.enabled ?? false,
      );
      form.setFieldValue(
        'health_check_path',
        initialData.load_balancing?.health_checks?.path || '/health',
      );
      form.setFieldValue(
        'health_check_interval',
        initialData.load_balancing?.health_checks?.interval || '30s',
      );
      form.setFieldValue(
        'health_check_timeout',
        initialData.load_balancing?.health_checks?.timeout || '5s',
      );
    }
  }, [initialData, form.setFieldValue]);

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

      <Card>
        <CardHeader>
          <CardHeading>
            <CardTitle>Upstream Servers</CardTitle>
            <CardDescription>Backend servers that will handle incoming requests</CardDescription>
          </CardHeading>
          <CardToolbar>
            <Button type="button" variant="outline" size="sm" onClick={addUpstream}>
              <Plus className="mr-1 size-4" />
              Add Upstream
            </Button>
          </CardToolbar>
        </CardHeader>
        <CardContent className="space-y-3">
          {upstreams.map((upstream, index) => (
            <div key={index} className="flex items-start gap-2">
              <div className="w-24">
                <Select
                  value={upstream.scheme}
                  onValueChange={(value) => updateUpstream(index, 'scheme', value)}
                >
                  <SelectTrigger>
                    <SelectValue />
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
                  type="number"
                  placeholder="8080"
                  value={upstream.port}
                  onChange={(e) => updateUpstream(index, 'port', parseInt(e.target.value, 10) || 0)}
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

      {upstreams.length > 1 && (
        <Card>
          <CardHeader>
            <CardHeading>
              <CardTitle>Load Balancing</CardTitle>
              <CardDescription>Distribute traffic across multiple upstream servers</CardDescription>
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
                      <SelectItem value="round_robin">Round Robin</SelectItem>
                      <SelectItem value="least_conn">Least Connections</SelectItem>
                      <SelectItem value="ip_hash">IP Hash (Sticky)</SelectItem>
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
                    <FieldDescription>Monitor upstream server availability</FieldDescription>
                  </FieldContent>
                  <Switch checked={field.state.value} onCheckedChange={field.handleChange} />
                </Field>
              )}
            </form.Field>

            <form.Subscribe selector={(state) => state.values.health_check_enabled}>
              {(healthCheckEnabled) =>
                healthCheckEnabled && (
                  <div className="grid gap-4 sm:grid-cols-3">
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
                    <form.Field name="health_check_interval">
                      {(field) => (
                        <Field>
                          <FieldLabel>Interval</FieldLabel>
                          <Input
                            placeholder="30s"
                            value={field.state.value}
                            onChange={(e) => field.handleChange(e.target.value)}
                          />
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
                        </Field>
                      )}
                    </form.Field>
                  </div>
                )
              }
            </form.Subscribe>
          </CardContent>
        </Card>
      )}

      <Card>
        <CardHeader>
          <CardHeading>
            <CardTitle>Security Options</CardTitle>
            <CardDescription>Configure security settings for upstream connections</CardDescription>
          </CardHeading>
        </CardHeader>
        <CardContent className="space-y-4">
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
                  <FieldLabel>Skip TLS Verification</FieldLabel>
                  <FieldDescription>Allow self-signed certificates on upstream</FieldDescription>
                </FieldContent>
                <Switch checked={field.state.value} onCheckedChange={field.handleChange} />
              </Field>
            )}
          </form.Field>
        </CardContent>
      </Card>

      <div className="flex justify-end gap-2">
        <Button type="button" variant="outline" onClick={onCancel}>
          Cancel
        </Button>
        <Button type="submit" disabled={loading}>
          {loading ? 'Saving...' : 'Save'}
        </Button>
      </div>
    </form>
  );
}
