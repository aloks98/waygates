import {
  Badge,
  Button,
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
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
} from '@e412/rnui-react';
import { useForm } from '@tanstack/react-form';
import { ChevronDown, ChevronUp, HelpCircle, Plus, Trash2 } from 'lucide-react';
import { useEffect, useMemo, useState } from 'react';

import {
  type L4ProxyFormValues,
  type L4RouteFormValues,
  l4ProxySchema,
} from '@/lib/form-validation';
import type {
  CreateL4ProxyRequest,
  L4LoadBalancingPolicy,
  L4MatcherType,
  L4Protocol,
  L4Proxy,
  L4ProxyProtocolVersion,
} from '@/types/l4-proxy';
import {
  L4_LOAD_BALANCING_POLICIES,
  L4_MATCHER_TYPES,
  L4_PROTOCOLS,
  L4_PROXY_PROTOCOL_VERSIONS,
} from '@/types/l4-proxy';

interface L4ProxyFormProps {
  initialData?: L4Proxy | null;
  onSubmit: (data: CreateL4ProxyRequest) => void;
  loading: boolean;
  onCancel: () => void;
}

// Default empty route for adding new routes
const createEmptyRoute = (): L4RouteFormValues => ({
  priority: 0,
  matcher_type: 'any',
  upstreams: [{ host: '', port: 8080 }],
  load_balancing_policy: 'round_robin',
  tls_terminate: false,
  tls_passthrough: false,
  sni_hostnames: [],
  allowed_ip_ranges: [],
  regex_pattern: '',
});

// Helper to normalize routes from initial data
function normalizeRoutes(data?: L4Proxy | null): L4RouteFormValues[] {
  if (!data?.routes?.length) {
    return [createEmptyRoute()];
  }
  return data.routes.map((route) => ({
    priority: route.priority ?? 0,
    matcher_type: route.matcher_type ?? 'any',
    upstreams: route.upstreams?.map((u) => ({
      host: u.host || '',
      port: u.port || 8080,
      weight: u.weight ?? undefined,
    })) || [{ host: '', port: 8080 }],
    load_balancing_policy: route.load_balancing_policy ?? 'round_robin',
    tls_terminate: route.tls_terminate ?? false,
    tls_passthrough: route.tls_passthrough ?? false,
    sni_hostnames: route.sni_hostnames || [],
    allowed_ip_ranges: route.allowed_ip_ranges || [],
    regex_pattern: route.regex_pattern || '',
    proxy_protocol_version: route.proxy_protocol_version ?? undefined,
  }));
}

// Matcher type display labels and descriptions
const MATCHER_TYPE_CONFIG: Record<L4MatcherType, { label: string; description: string }> = {
  any: {
    label: 'Any (Match All)',
    description: 'Matches all incoming connections without any filtering',
  },
  tls: {
    label: 'TLS/SNI',
    description: 'Match TLS connections by Server Name Indication (hostname)',
  },
  ssh: {
    label: 'SSH',
    description: 'Detect and match SSH protocol connections',
  },
  postgres: {
    label: 'PostgreSQL',
    description: 'Detect and match PostgreSQL database connections',
  },
  http: {
    label: 'HTTP',
    description: 'Detect and match HTTP protocol at Layer 4',
  },
  rdp: {
    label: 'RDP',
    description: 'Detect and match Remote Desktop Protocol connections',
  },
  socks5: {
    label: 'SOCKS5',
    description: 'Detect and match SOCKS5 proxy protocol connections',
  },
  remote_ip: {
    label: 'Remote IP',
    description: 'Match connections from specific IP addresses or CIDR ranges',
  },
  regexp: {
    label: 'Regular Expression',
    description: 'Match connections using a regex pattern on initial data',
  },
};

// Load balancing policy display labels
const LB_POLICY_LABELS: Record<L4LoadBalancingPolicy, string> = {
  round_robin: 'Round Robin — each server takes turns',
  least_conn: 'Least Connections — prefer less busy servers',
  random: 'Random',
  first: 'First Available — use the first server that responds',
  ip_hash: 'Sticky — same client always reaches same server',
};

export function L4ProxyForm({ initialData, onSubmit, loading, onCancel }: L4ProxyFormProps) {
  const [expandedRoutes, setExpandedRoutes] = useState<Set<number>>(() => new Set([0]));

  // Compute default values based on initialData
  const defaultValues = useMemo<L4ProxyFormValues>(() => {
    if (initialData) {
      return {
        name: initialData.name || '',
        description: initialData.description ?? '',
        listen_port: initialData.listen_port || 1234,
        protocol: initialData.protocol || 'tcp',
        is_active: initialData.is_active ?? true,
        routes: normalizeRoutes(initialData),
      };
    }
    return {
      name: '',
      description: '',
      listen_port: 1234,
      protocol: 'tcp' as L4Protocol,
      is_active: true,
      routes: [createEmptyRoute()],
    };
  }, [initialData]);

  const form = useForm({
    defaultValues,
    validators: {
      onSubmit: l4ProxySchema,
    },
    onSubmit: async ({ value }) => {
      const routes = value.routes || [];
      const data: CreateL4ProxyRequest = {
        name: value.name,
        description: value.description || undefined,
        listen_port: value.listen_port,
        protocol: value.protocol,
        is_active: value.is_active,
        routes: routes.map((route) => ({
          priority: route.priority,
          matcher_type: route.matcher_type,
          upstreams: route.upstreams.map((u) => ({
            host: u.host,
            port: u.port,
            weight: u.weight ?? undefined,
          })),
          load_balancing_policy: route.load_balancing_policy,
          tls_terminate: route.tls_terminate,
          tls_passthrough: route.tls_passthrough,
          sni_hostnames:
            route.matcher_type === 'tls' ? route.sni_hostnames?.filter(Boolean) : undefined,
          allowed_ip_ranges:
            route.matcher_type === 'remote_ip'
              ? route.allowed_ip_ranges?.filter(Boolean)
              : undefined,
          regex_pattern:
            route.matcher_type === 'regexp' ? route.regex_pattern || undefined : undefined,
          proxy_protocol_version: route.proxy_protocol_version ?? undefined,
        })),
      };

      await onSubmit(data);
    },
  });

  // Reset form when initialData changes (for edit mode)
  useEffect(() => {
    if (initialData) {
      form.reset(defaultValues);
    }
  }, [initialData, form, defaultValues]);

  const toggleRouteExpansion = (index: number) => {
    const newExpanded = new Set(expandedRoutes);
    if (newExpanded.has(index)) {
      newExpanded.delete(index);
    } else {
      newExpanded.add(index);
    }
    setExpandedRoutes(newExpanded);
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
          <CardTitle>Basic Information</CardTitle>
          <CardDescription>General settings for this TCP/UDP proxy</CardDescription>
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
                      placeholder="MySQL Proxy"
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

            <form.Field name="description">
              {(field) => {
                const hasError = field.state.meta.isTouched && field.state.meta.errors.length > 0;
                return (
                  <Field data-invalid={hasError}>
                    <FieldLabel htmlFor={field.name}>Description (optional)</FieldLabel>
                    <Input
                      id={field.name}
                      placeholder="Proxy for database connections"
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

          <div className="grid gap-4 sm:grid-cols-3">
            <form.Field name="listen_port">
              {(field) => {
                const hasError = field.state.meta.isTouched && field.state.meta.errors.length > 0;
                return (
                  <Field data-invalid={hasError}>
                    <FieldLabel htmlFor={field.name}>Listen Port</FieldLabel>
                    <Input
                      id={field.name}
                      type="text"
                      inputMode="numeric"
                      pattern="[0-9]*"
                      placeholder="3306"
                      value={field.state.value || ''}
                      onChange={(e) => {
                        const value = e.target.value.replace(/\D/g, '');
                        const port = value ? Math.min(Number.parseInt(value, 10), 65535) : 0;
                        field.handleChange(port);
                      }}
                      onBlur={field.handleBlur}
                      aria-invalid={hasError}
                    />
                    <FieldDescription>Port to listen on (1-65535)</FieldDescription>
                    {hasError && <FieldError errors={field.state.meta.errors} />}
                  </Field>
                );
              }}
            </form.Field>

            <form.Field name="protocol">
              {(field) => (
                <Field>
                  <FieldLabel>Protocol</FieldLabel>
                  <Select
                    value={field.state.value}
                    onValueChange={(val) => field.handleChange(val as L4Protocol)}
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {L4_PROTOCOLS.map((protocol) => (
                        <SelectItem key={protocol} value={protocol}>
                          {protocol.toUpperCase()}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <FieldDescription>TCP or UDP</FieldDescription>
                </Field>
              )}
            </form.Field>

            <form.Field name="is_active">
              {(field) => (
                <Field orientation="horizontal" className="sm:pt-6">
                  <FieldContent>
                    <FieldLabel>Active</FieldLabel>
                    <FieldDescription>Enable this proxy</FieldDescription>
                  </FieldContent>
                  <Switch checked={field.state.value} onCheckedChange={field.handleChange} />
                </Field>
              )}
            </form.Field>
          </div>
        </CardContent>
      </Card>

      {/* Routes */}
      <Card>
        <form.Field name="routes" mode="array">
          {(routesField) => {
            const routes = routesField.state.value ?? [];
            return (
              <>
                <CardHeader>
                  <CardTitle>Routes</CardTitle>
                  <CardDescription>
                    Define how incoming connections are matched and forwarded to your backend
                    servers. You can add multiple routes with different matching rules.
                  </CardDescription>
                  <CardAction>
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      onClick={() => {
                        routesField.pushValue(createEmptyRoute());
                        setExpandedRoutes((prev) => new Set([...prev, routes.length]));
                      }}
                    >
                      <Plus className="mr-1 size-4" />
                      Add Route
                    </Button>
                  </CardAction>
                </CardHeader>
                <CardContent className="space-y-4">
                  {routes.map((route, routeIndex) => (
                    <Card key={routeIndex} className="border-dashed">
                      <CardHeader className="pb-2">
                        <button
                          type="button"
                          className="flex items-center gap-2 text-left"
                          onClick={() => toggleRouteExpansion(routeIndex)}
                        >
                          {expandedRoutes.has(routeIndex) ? (
                            <ChevronUp className="size-4" />
                          ) : (
                            <ChevronDown className="size-4" />
                          )}
                          <CardTitle className="text-base">Route {routeIndex + 1}</CardTitle>
                          <Badge variant="outline" className="ml-2">
                            {MATCHER_TYPE_CONFIG[route.matcher_type].label}
                          </Badge>
                          {route.upstreams.length > 0 && route.upstreams[0].host && (
                            <Badge variant="secondary" className="ml-1">
                              {route.upstreams.length} upstream
                              {route.upstreams.length > 1 ? 's' : ''}
                            </Badge>
                          )}
                        </button>
                        <CardAction>
                          {routes.length > 1 && (
                            <Button
                              type="button"
                              variant="ghost"
                              size="icon"
                              onClick={() => {
                                routesField.removeValue(routeIndex);
                                setExpandedRoutes((prev) => {
                                  const adjusted = new Set<number>();
                                  for (const idx of prev) {
                                    if (idx < routeIndex) adjusted.add(idx);
                                    else if (idx > routeIndex) adjusted.add(idx - 1);
                                  }
                                  return adjusted;
                                });
                              }}
                            >
                              <Trash2 className="size-4 text-destructive" />
                            </Button>
                          )}
                        </CardAction>
                      </CardHeader>

                      {expandedRoutes.has(routeIndex) && (
                        <CardContent className="space-y-4 pt-2">
                          {/* Matcher Type Selection */}
                          <form.Field name={`routes[${routeIndex}].matcher_type`}>
                            {(matcherField) => (
                              <Field>
                                <FieldLabel className="flex items-center gap-1">
                                  Connection Matcher
                                  <HelpCircle className="size-3.5 text-muted-foreground" />
                                </FieldLabel>
                                <Select
                                  value={matcherField.state.value}
                                  onValueChange={(val) =>
                                    matcherField.handleChange(val as L4MatcherType)
                                  }
                                >
                                  <SelectTrigger>
                                    <SelectValue />
                                  </SelectTrigger>
                                  <SelectContent>
                                    {L4_MATCHER_TYPES.map((type) => (
                                      <SelectItem key={type} value={type}>
                                        <div className="flex flex-col">
                                          <span>{MATCHER_TYPE_CONFIG[type].label}</span>
                                        </div>
                                      </SelectItem>
                                    ))}
                                  </SelectContent>
                                </Select>
                                <FieldDescription>
                                  {MATCHER_TYPE_CONFIG[matcherField.state.value].description}
                                </FieldDescription>
                              </Field>
                            )}
                          </form.Field>

                          {/* Matcher-specific fields */}
                          {route.matcher_type === 'tls' && (
                            <form.Field name={`routes[${routeIndex}].sni_hostnames`} mode="array">
                              {(sniField) => {
                                const hasError = sniField.state.meta.errors.length > 0;
                                const hostnames = sniField.state.value ?? [];
                                return (
                                  <div className="space-y-3 rounded-lg border p-4 bg-muted/30">
                                    <div className="flex items-center justify-between">
                                      <FieldLabel>SNI Hostnames</FieldLabel>
                                      <Button
                                        type="button"
                                        variant="outline"
                                        size="sm"
                                        onClick={() => sniField.pushValue('')}
                                      >
                                        <Plus className="mr-1 size-3" />
                                        Add
                                      </Button>
                                    </div>
                                    <FieldDescription className="mt-0">
                                      Match TLS connections by the hostname in the TLS handshake
                                      (Server Name Indication)
                                    </FieldDescription>
                                    {hasError && <FieldError errors={sniField.state.meta.errors} />}
                                    {hostnames.map((_, sniIndex) => (
                                      <div key={sniIndex} className="flex items-center gap-2">
                                        <form.Field
                                          name={`routes[${routeIndex}].sni_hostnames[${sniIndex}]`}
                                        >
                                          {(hostnameField) => (
                                            <Input
                                              placeholder="example.com"
                                              value={hostnameField.state.value}
                                              onChange={(e) =>
                                                hostnameField.handleChange(e.target.value)
                                              }
                                              className="flex-1"
                                            />
                                          )}
                                        </form.Field>
                                        <Button
                                          type="button"
                                          variant="ghost"
                                          size="icon"
                                          onClick={() => sniField.removeValue(sniIndex)}
                                        >
                                          <Trash2 className="size-4 text-destructive" />
                                        </Button>
                                      </div>
                                    ))}
                                  </div>
                                );
                              }}
                            </form.Field>
                          )}

                          {route.matcher_type === 'remote_ip' && (
                            <form.Field
                              name={`routes[${routeIndex}].allowed_ip_ranges`}
                              mode="array"
                            >
                              {(ipField) => {
                                const hasError = ipField.state.meta.errors.length > 0;
                                const ranges = ipField.state.value ?? [];
                                return (
                                  <div className="space-y-3 rounded-lg border p-4 bg-muted/30">
                                    <div className="flex items-center justify-between">
                                      <FieldLabel>Allowed IP Ranges</FieldLabel>
                                      <Button
                                        type="button"
                                        variant="outline"
                                        size="sm"
                                        onClick={() => ipField.pushValue('')}
                                      >
                                        <Plus className="mr-1 size-3" />
                                        Add
                                      </Button>
                                    </div>
                                    <FieldDescription className="mt-0">
                                      Only allow connections from these IP addresses or CIDR ranges
                                    </FieldDescription>
                                    {hasError && <FieldError errors={ipField.state.meta.errors} />}
                                    {ranges.map((_, ipIndex) => (
                                      <div key={ipIndex} className="flex items-center gap-2">
                                        <form.Field
                                          name={`routes[${routeIndex}].allowed_ip_ranges[${ipIndex}]`}
                                        >
                                          {(ipItemField) => (
                                            <Input
                                              placeholder="192.168.1.0/24 or 10.0.0.1"
                                              value={ipItemField.state.value}
                                              onChange={(e) =>
                                                ipItemField.handleChange(e.target.value)
                                              }
                                              className="flex-1"
                                            />
                                          )}
                                        </form.Field>
                                        <Button
                                          type="button"
                                          variant="ghost"
                                          size="icon"
                                          onClick={() => ipField.removeValue(ipIndex)}
                                        >
                                          <Trash2 className="size-4 text-destructive" />
                                        </Button>
                                      </div>
                                    ))}
                                  </div>
                                );
                              }}
                            </form.Field>
                          )}

                          {route.matcher_type === 'regexp' && (
                            <div className="rounded-lg border p-4 bg-muted/30">
                              <form.Field name={`routes[${routeIndex}].regex_pattern`}>
                                {(regexField) => {
                                  const hasError = regexField.state.meta.errors.length > 0;
                                  return (
                                    <Field data-invalid={hasError}>
                                      <FieldLabel>Regex Pattern</FieldLabel>
                                      <Input
                                        placeholder="^GET /api/.*"
                                        value={regexField.state.value || ''}
                                        onChange={(e) => regexField.handleChange(e.target.value)}
                                        aria-invalid={hasError}
                                      />
                                      <FieldDescription>
                                        Regular expression to match against the first bytes of data
                                      </FieldDescription>
                                      {hasError && (
                                        <FieldError errors={regexField.state.meta.errors} />
                                      )}
                                    </Field>
                                  );
                                }}
                              </form.Field>
                            </div>
                          )}

                          {/* Upstreams */}
                          <form.Field name={`routes[${routeIndex}].upstreams`} mode="array">
                            {(upstreamsField) => {
                              const upstreams = upstreamsField.state.value;
                              return (
                                <div className="space-y-3">
                                  <div className="flex items-center justify-between">
                                    <FieldLabel>Backend Servers</FieldLabel>
                                    <Button
                                      type="button"
                                      variant="outline"
                                      size="sm"
                                      onClick={() =>
                                        upstreamsField.pushValue({ host: '', port: 8080 })
                                      }
                                    >
                                      <Plus className="mr-1 size-3" />
                                      Add Server
                                    </Button>
                                  </div>
                                  <FieldDescription className="mt-0">
                                    Where to forward matched connections
                                  </FieldDescription>
                                  {upstreams.map((_, upstreamIndex) => (
                                    <div key={upstreamIndex} className="flex items-start gap-2">
                                      <form.Field
                                        name={`routes[${routeIndex}].upstreams[${upstreamIndex}].host`}
                                      >
                                        {(hostField) => {
                                          const hasError = hostField.state.meta.errors.length > 0;
                                          return (
                                            <div className="flex-1">
                                              <Input
                                                placeholder="Host (e.g., 192.168.1.100)"
                                                value={hostField.state.value}
                                                onChange={(e) =>
                                                  hostField.handleChange(e.target.value)
                                                }
                                                onBlur={hostField.handleBlur}
                                                aria-invalid={hasError}
                                              />
                                              {hasError && (
                                                <FieldError errors={hostField.state.meta.errors} />
                                              )}
                                            </div>
                                          );
                                        }}
                                      </form.Field>
                                      <form.Field
                                        name={`routes[${routeIndex}].upstreams[${upstreamIndex}].port`}
                                      >
                                        {(portField) => {
                                          const hasError = portField.state.meta.errors.length > 0;
                                          return (
                                            <div className="w-24">
                                              <Input
                                                type="text"
                                                inputMode="numeric"
                                                pattern="[0-9]*"
                                                placeholder="Port"
                                                value={portField.state.value || ''}
                                                onChange={(e) => {
                                                  const value = e.target.value.replace(/\D/g, '');
                                                  const port = value
                                                    ? Math.min(Number.parseInt(value, 10), 65535)
                                                    : 0;
                                                  portField.handleChange(port);
                                                }}
                                                onBlur={portField.handleBlur}
                                                aria-invalid={hasError}
                                              />
                                              {hasError && (
                                                <FieldError errors={portField.state.meta.errors} />
                                              )}
                                            </div>
                                          );
                                        }}
                                      </form.Field>
                                      {upstreams.length > 1 && (
                                        <>
                                          <form.Field
                                            name={`routes[${routeIndex}].upstreams[${upstreamIndex}].weight`}
                                          >
                                            {(weightField) => (
                                              <div className="w-20">
                                                <Input
                                                  type="text"
                                                  inputMode="numeric"
                                                  pattern="[0-9]*"
                                                  placeholder="Weight"
                                                  value={weightField.state.value || ''}
                                                  onChange={(e) => {
                                                    const value = e.target.value.replace(/\D/g, '');
                                                    weightField.handleChange(
                                                      value
                                                        ? Number.parseInt(value, 10)
                                                        : undefined,
                                                    );
                                                  }}
                                                />
                                              </div>
                                            )}
                                          </form.Field>
                                          <Button
                                            type="button"
                                            variant="ghost"
                                            size="icon"
                                            onClick={() =>
                                              upstreamsField.removeValue(upstreamIndex)
                                            }
                                          >
                                            <Trash2 className="size-4 text-destructive" />
                                          </Button>
                                        </>
                                      )}
                                    </div>
                                  ))}
                                </div>
                              );
                            }}
                          </form.Field>

                          {/* Load Balancing - only show when multiple upstreams */}
                          {route.upstreams.length > 1 && (
                            <form.Field name={`routes[${routeIndex}].load_balancing_policy`}>
                              {(lbField) => (
                                <Field>
                                  <FieldLabel>Load Balancing Strategy</FieldLabel>
                                  <Select
                                    value={lbField.state.value}
                                    onValueChange={(val) =>
                                      lbField.handleChange(val as L4LoadBalancingPolicy)
                                    }
                                  >
                                    <SelectTrigger>
                                      <SelectValue />
                                    </SelectTrigger>
                                    <SelectContent>
                                      {L4_LOAD_BALANCING_POLICIES.map((policy) => (
                                        <SelectItem key={policy} value={policy}>
                                          {LB_POLICY_LABELS[policy]}
                                        </SelectItem>
                                      ))}
                                    </SelectContent>
                                  </Select>
                                  <FieldDescription>
                                    How to distribute connections across your servers
                                  </FieldDescription>
                                </Field>
                              )}
                            </form.Field>
                          )}

                          {/* Priority - only show when multiple routes */}
                          {routes.length > 1 && (
                            <form.Field name={`routes[${routeIndex}].priority`}>
                              {(priorityField) => (
                                <Field>
                                  <FieldLabel>Priority</FieldLabel>
                                  <Input
                                    type="text"
                                    inputMode="numeric"
                                    pattern="[0-9]*"
                                    placeholder="0"
                                    value={priorityField.state.value || ''}
                                    onChange={(e) => {
                                      const value = e.target.value.replace(/\D/g, '');
                                      priorityField.handleChange(
                                        value ? Number.parseInt(value, 10) : 0,
                                      );
                                    }}
                                    className="w-24"
                                  />
                                  <FieldDescription>
                                    Lower values are matched first (default: 0)
                                  </FieldDescription>
                                </Field>
                              )}
                            </form.Field>
                          )}

                          {/* TLS Settings */}
                          <div className="space-y-4 rounded-lg border p-4">
                            <FieldLabel className="text-sm font-medium">TLS Settings</FieldLabel>
                            <form.Field name={`routes[${routeIndex}].tls_terminate`}>
                              {(tlsTermField) => {
                                const hasError = tlsTermField.state.meta.errors.length > 0;
                                return (
                                  <>
                                    {hasError && (
                                      <FieldError errors={tlsTermField.state.meta.errors} />
                                    )}
                                    <div className="grid gap-4 sm:grid-cols-2">
                                      <Field orientation="horizontal">
                                        <FieldContent>
                                          <FieldLabel>TLS Termination</FieldLabel>
                                          <FieldDescription>
                                            Decrypt traffic here, forward unencrypted to your server
                                          </FieldDescription>
                                        </FieldContent>
                                        <Switch
                                          checked={tlsTermField.state.value === true}
                                          onCheckedChange={(checked: boolean) => {
                                            tlsTermField.handleChange(checked);
                                            if (checked) {
                                              form.setFieldValue(
                                                `routes[${routeIndex}].tls_passthrough`,
                                                false,
                                              );
                                            }
                                          }}
                                        />
                                      </Field>

                                      <form.Field name={`routes[${routeIndex}].tls_passthrough`}>
                                        {(tlsPassField) => (
                                          <Field orientation="horizontal">
                                            <FieldContent>
                                              <FieldLabel>TLS Passthrough</FieldLabel>
                                              <FieldDescription>
                                                Pass encrypted traffic through to your server as-is
                                              </FieldDescription>
                                            </FieldContent>
                                            <Switch
                                              checked={tlsPassField.state.value === true}
                                              onCheckedChange={(checked: boolean) => {
                                                tlsPassField.handleChange(checked);
                                                if (checked) {
                                                  form.setFieldValue(
                                                    `routes[${routeIndex}].tls_terminate`,
                                                    false,
                                                  );
                                                }
                                              }}
                                            />
                                          </Field>
                                        )}
                                      </form.Field>
                                    </div>
                                  </>
                                );
                              }}
                            </form.Field>

                            <form.Field name={`routes[${routeIndex}].proxy_protocol_version`}>
                              {(ppField) => (
                                <Field>
                                  <FieldLabel>Proxy Protocol</FieldLabel>
                                  <Select
                                    value={ppField.state.value || 'none'}
                                    onValueChange={(val) =>
                                      ppField.handleChange(
                                        val === 'none'
                                          ? undefined
                                          : (val as L4ProxyProtocolVersion),
                                      )
                                    }
                                  >
                                    <SelectTrigger className="w-48">
                                      <SelectValue placeholder="None" />
                                    </SelectTrigger>
                                    <SelectContent>
                                      <SelectItem value="none">None</SelectItem>
                                      {L4_PROXY_PROTOCOL_VERSIONS.map((version) => (
                                        <SelectItem key={version} value={version}>
                                          Version {version.replace('v', '')}
                                        </SelectItem>
                                      ))}
                                    </SelectContent>
                                  </Select>
                                  <FieldDescription>
                                    Forward the original client IP to your server (needed by some
                                    services)
                                  </FieldDescription>
                                </Field>
                              )}
                            </form.Field>
                          </div>
                        </CardContent>
                      )}
                    </Card>
                  ))}
                </CardContent>
              </>
            );
          }}
        </form.Field>
      </Card>

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
