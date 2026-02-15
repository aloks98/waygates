import {
  Badge,
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
import { ChevronDown, ChevronUp, HelpCircle, Plus, Trash2 } from 'lucide-react';
import { useEffect, useMemo, useState } from 'react';
import {
  type L4ProxyFormValues,
  type L4RouteFormValues,
  type L4UpstreamFormValues,
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
      weight: u.weight,
    })) || [{ host: '', port: 8080 }],
    load_balancing_policy: route.load_balancing_policy ?? 'round_robin',
    tls_terminate: route.tls_terminate ?? false,
    tls_passthrough: route.tls_passthrough ?? false,
    sni_hostnames: route.sni_hostnames || [],
    allowed_ip_ranges: route.allowed_ip_ranges || [],
    regex_pattern: route.regex_pattern || '',
    proxy_protocol_version: route.proxy_protocol_version,
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
  round_robin: 'Round Robin',
  least_conn: 'Least Connections',
  random: 'Random',
  first: 'First Available',
  ip_hash: 'IP Hash (Sticky)',
};

export function L4ProxyForm({ initialData, onSubmit, loading, onCancel }: L4ProxyFormProps) {
  const [routes, setRoutes] = useState<L4RouteFormValues[]>(() => normalizeRoutes(initialData));
  const [expandedRoutes, setExpandedRoutes] = useState<Set<number>>(() => new Set([0]));

  // Compute default values based on initialData
  const defaultValues = useMemo<L4ProxyFormValues>(() => {
    if (initialData) {
      return {
        name: initialData.name || '',
        description: initialData.description || '',
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
            weight: u.weight,
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
          proxy_protocol_version: route.proxy_protocol_version,
        })),
      };

      onSubmit(data);
    },
  });

  // Reset form when initialData changes (for edit mode)
  useEffect(() => {
    if (initialData) {
      const normalizedRoutes = normalizeRoutes(initialData);
      setRoutes(normalizedRoutes);
      form.reset(defaultValues);
    }
  }, [initialData, form, defaultValues]);

  // Route management functions - sync to form state immediately
  const addRoute = () => {
    const newRoutes = [...routes, createEmptyRoute()];
    setRoutes(newRoutes);
    form.setFieldValue('routes', newRoutes);
    setExpandedRoutes(new Set([...expandedRoutes, newRoutes.length - 1]));
  };

  const removeRoute = (index: number) => {
    const newRoutes = routes.filter((_, i) => i !== index);
    const finalRoutes = newRoutes.length > 0 ? newRoutes : [createEmptyRoute()];
    setRoutes(finalRoutes);
    form.setFieldValue('routes', finalRoutes);
    const newExpanded = new Set(expandedRoutes);
    newExpanded.delete(index);
    setExpandedRoutes(newExpanded);
  };

  const updateRoute = <K extends keyof L4RouteFormValues>(
    index: number,
    key: K,
    value: L4RouteFormValues[K],
  ) => {
    const newRoutes = [...routes];
    newRoutes[index] = { ...newRoutes[index], [key]: value };
    setRoutes(newRoutes);
    form.setFieldValue('routes', newRoutes);
  };

  const toggleRouteExpansion = (index: number) => {
    const newExpanded = new Set(expandedRoutes);
    if (newExpanded.has(index)) {
      newExpanded.delete(index);
    } else {
      newExpanded.add(index);
    }
    setExpandedRoutes(newExpanded);
  };

  // Upstream management functions
  const addUpstream = (routeIndex: number) => {
    const newUpstreams = [...routes[routeIndex].upstreams, { host: '', port: 8080 }];
    updateRoute(routeIndex, 'upstreams', newUpstreams);
  };

  const removeUpstream = (routeIndex: number, upstreamIndex: number) => {
    const newUpstreams = routes[routeIndex].upstreams.filter((_, i) => i !== upstreamIndex);
    updateRoute(
      routeIndex,
      'upstreams',
      newUpstreams.length > 0 ? newUpstreams : [{ host: '', port: 8080 }],
    );
  };

  const updateUpstream = <K extends keyof L4UpstreamFormValues>(
    routeIndex: number,
    upstreamIndex: number,
    key: K,
    value: L4UpstreamFormValues[K],
  ) => {
    const newUpstreams = [...routes[routeIndex].upstreams];
    newUpstreams[upstreamIndex] = { ...newUpstreams[upstreamIndex], [key]: value };
    updateRoute(routeIndex, 'upstreams', newUpstreams);
  };

  // SNI hostnames management
  const addSniHostname = (routeIndex: number) => {
    const current = routes[routeIndex].sni_hostnames || [];
    updateRoute(routeIndex, 'sni_hostnames', [...current, '']);
  };

  const removeSniHostname = (routeIndex: number, sniIndex: number) => {
    const current = routes[routeIndex].sni_hostnames || [];
    updateRoute(
      routeIndex,
      'sni_hostnames',
      current.filter((_, i) => i !== sniIndex),
    );
  };

  const updateSniHostname = (routeIndex: number, sniIndex: number, value: string) => {
    const current = [...(routes[routeIndex].sni_hostnames || [])];
    current[sniIndex] = value;
    updateRoute(routeIndex, 'sni_hostnames', current);
  };

  // IP ranges management
  const addIpRange = (routeIndex: number) => {
    const current = routes[routeIndex].allowed_ip_ranges || [];
    updateRoute(routeIndex, 'allowed_ip_ranges', [...current, '']);
  };

  const removeIpRange = (routeIndex: number, ipIndex: number) => {
    const current = routes[routeIndex].allowed_ip_ranges || [];
    updateRoute(
      routeIndex,
      'allowed_ip_ranges',
      current.filter((_, i) => i !== ipIndex),
    );
  };

  const updateIpRange = (routeIndex: number, ipIndex: number, value: string) => {
    const current = [...(routes[routeIndex].allowed_ip_ranges || [])];
    current[ipIndex] = value;
    updateRoute(routeIndex, 'allowed_ip_ranges', current);
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
            <CardDescription>General settings for this L4 proxy</CardDescription>
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
        <CardHeader>
          <CardHeading>
            <CardTitle>Routes</CardTitle>
            <CardDescription>
              Define how incoming connections are matched and forwarded to upstream servers. You can
              add multiple routes with different matching rules.
            </CardDescription>
          </CardHeading>
          <CardToolbar>
            <Button type="button" variant="outline" size="sm" onClick={addRoute}>
              <Plus className="mr-1 size-4" />
              Add Route
            </Button>
          </CardToolbar>
        </CardHeader>
        <CardContent className="space-y-4">
          {routes.map((route, routeIndex) => (
            <Card key={routeIndex} className="border-dashed">
              <CardHeader className="pb-2">
                <CardHeading>
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
                        {route.upstreams.length} upstream{route.upstreams.length > 1 ? 's' : ''}
                      </Badge>
                    )}
                  </button>
                </CardHeading>
                <CardToolbar>
                  {routes.length > 1 && (
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      onClick={() => removeRoute(routeIndex)}
                    >
                      <Trash2 className="size-4 text-destructive" />
                    </Button>
                  )}
                </CardToolbar>
              </CardHeader>

              {expandedRoutes.has(routeIndex) && (
                <CardContent className="space-y-4 pt-2">
                  {/* Matcher Type Selection */}
                  <Field>
                    <FieldLabel className="flex items-center gap-1">
                      Connection Matcher
                      <HelpCircle className="size-3.5 text-muted-foreground" />
                    </FieldLabel>
                    <Select
                      value={route.matcher_type}
                      onValueChange={(val) =>
                        updateRoute(routeIndex, 'matcher_type', val as L4MatcherType)
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
                      {MATCHER_TYPE_CONFIG[route.matcher_type].description}
                    </FieldDescription>
                  </Field>

                  {/* Matcher-specific fields */}
                  {route.matcher_type === 'tls' && (
                    <div className="space-y-3 rounded-lg border p-4 bg-muted/30">
                      <div className="flex items-center justify-between">
                        <FieldLabel>SNI Hostnames</FieldLabel>
                        <Button
                          type="button"
                          variant="outline"
                          size="sm"
                          onClick={() => addSniHostname(routeIndex)}
                        >
                          <Plus className="mr-1 size-3" />
                          Add
                        </Button>
                      </div>
                      <FieldDescription className="mt-0">
                        Match TLS connections by the hostname in the TLS handshake (Server Name
                        Indication)
                      </FieldDescription>
                      {(route.sni_hostnames?.length ?? 0) === 0 && (
                        <p className="text-sm text-amber-600">
                          Add at least one SNI hostname for TLS matching
                        </p>
                      )}
                      {route.sni_hostnames?.map((hostname, sniIndex) => (
                        <div key={sniIndex} className="flex items-center gap-2">
                          <Input
                            placeholder="example.com"
                            value={hostname}
                            onChange={(e) =>
                              updateSniHostname(routeIndex, sniIndex, e.target.value)
                            }
                            className="flex-1"
                          />
                          <Button
                            type="button"
                            variant="ghost"
                            size="icon"
                            onClick={() => removeSniHostname(routeIndex, sniIndex)}
                          >
                            <Trash2 className="size-4 text-destructive" />
                          </Button>
                        </div>
                      ))}
                    </div>
                  )}

                  {route.matcher_type === 'remote_ip' && (
                    <div className="space-y-3 rounded-lg border p-4 bg-muted/30">
                      <div className="flex items-center justify-between">
                        <FieldLabel>Allowed IP Ranges</FieldLabel>
                        <Button
                          type="button"
                          variant="outline"
                          size="sm"
                          onClick={() => addIpRange(routeIndex)}
                        >
                          <Plus className="mr-1 size-3" />
                          Add
                        </Button>
                      </div>
                      <FieldDescription className="mt-0">
                        Only allow connections from these IP addresses or CIDR ranges
                      </FieldDescription>
                      {(route.allowed_ip_ranges?.length ?? 0) === 0 && (
                        <p className="text-sm text-amber-600">
                          Add at least one IP range for remote_ip matching
                        </p>
                      )}
                      {route.allowed_ip_ranges?.map((ipRange, ipIndex) => (
                        <div key={ipIndex} className="flex items-center gap-2">
                          <Input
                            placeholder="192.168.1.0/24 or 10.0.0.1"
                            value={ipRange}
                            onChange={(e) => updateIpRange(routeIndex, ipIndex, e.target.value)}
                            className="flex-1"
                          />
                          <Button
                            type="button"
                            variant="ghost"
                            size="icon"
                            onClick={() => removeIpRange(routeIndex, ipIndex)}
                          >
                            <Trash2 className="size-4 text-destructive" />
                          </Button>
                        </div>
                      ))}
                    </div>
                  )}

                  {route.matcher_type === 'regexp' && (
                    <div className="rounded-lg border p-4 bg-muted/30">
                      <Field>
                        <FieldLabel>Regex Pattern</FieldLabel>
                        <Input
                          placeholder="^GET /api/.*"
                          value={route.regex_pattern || ''}
                          onChange={(e) => updateRoute(routeIndex, 'regex_pattern', e.target.value)}
                        />
                        <FieldDescription>
                          Regular expression to match against the first bytes of data
                        </FieldDescription>
                      </Field>
                    </div>
                  )}

                  {/* Upstreams */}
                  <div className="space-y-3">
                    <div className="flex items-center justify-between">
                      <FieldLabel>Upstream Servers</FieldLabel>
                      <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        onClick={() => addUpstream(routeIndex)}
                      >
                        <Plus className="mr-1 size-3" />
                        Add Upstream
                      </Button>
                    </div>
                    <FieldDescription className="mt-0">
                      Servers to forward matched connections to
                    </FieldDescription>
                    {route.upstreams.map((upstream, upstreamIndex) => (
                      <div key={upstreamIndex} className="flex items-start gap-2">
                        <div className="flex-1">
                          <Input
                            placeholder="Host (e.g., 192.168.1.100)"
                            value={upstream.host}
                            onChange={(e) =>
                              updateUpstream(routeIndex, upstreamIndex, 'host', e.target.value)
                            }
                          />
                        </div>
                        <div className="w-24">
                          <Input
                            type="text"
                            inputMode="numeric"
                            pattern="[0-9]*"
                            placeholder="Port"
                            value={upstream.port || ''}
                            onChange={(e) => {
                              const value = e.target.value.replace(/\D/g, '');
                              const port = value ? Math.min(Number.parseInt(value, 10), 65535) : 0;
                              updateUpstream(routeIndex, upstreamIndex, 'port', port);
                            }}
                          />
                        </div>
                        {route.upstreams.length > 1 && (
                          <>
                            <div className="w-20">
                              <Input
                                type="text"
                                inputMode="numeric"
                                pattern="[0-9]*"
                                placeholder="Weight"
                                value={upstream.weight || ''}
                                onChange={(e) => {
                                  const value = e.target.value.replace(/\D/g, '');
                                  updateUpstream(
                                    routeIndex,
                                    upstreamIndex,
                                    'weight',
                                    value ? Number.parseInt(value, 10) : undefined,
                                  );
                                }}
                              />
                            </div>
                            <Button
                              type="button"
                              variant="ghost"
                              size="icon"
                              onClick={() => removeUpstream(routeIndex, upstreamIndex)}
                            >
                              <Trash2 className="size-4 text-destructive" />
                            </Button>
                          </>
                        )}
                      </div>
                    ))}
                  </div>

                  {/* Load Balancing - only show when multiple upstreams */}
                  {route.upstreams.length > 1 && (
                    <Field>
                      <FieldLabel>Load Balancing Strategy</FieldLabel>
                      <Select
                        value={route.load_balancing_policy}
                        onValueChange={(val) =>
                          updateRoute(
                            routeIndex,
                            'load_balancing_policy',
                            val as L4LoadBalancingPolicy,
                          )
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
                        How to distribute connections across multiple upstreams
                      </FieldDescription>
                    </Field>
                  )}

                  {/* Priority - only show when multiple routes */}
                  {routes.length > 1 && (
                    <Field>
                      <FieldLabel>Priority</FieldLabel>
                      <Input
                        type="text"
                        inputMode="numeric"
                        pattern="[0-9]*"
                        placeholder="0"
                        value={route.priority || ''}
                        onChange={(e) => {
                          const value = e.target.value.replace(/\D/g, '');
                          updateRoute(
                            routeIndex,
                            'priority',
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

                  {/* TLS Settings */}
                  <div className="space-y-4 rounded-lg border p-4">
                    <FieldLabel className="text-sm font-medium">TLS Settings</FieldLabel>
                    <div className="grid gap-4 sm:grid-cols-2">
                      <Field orientation="horizontal">
                        <FieldContent>
                          <FieldLabel>TLS Termination</FieldLabel>
                          <FieldDescription>
                            Decrypt TLS at this proxy and forward plain traffic
                          </FieldDescription>
                        </FieldContent>
                        <Switch
                          checked={route.tls_terminate === true}
                          onCheckedChange={(checked: boolean) => {
                            const newRoutes = [...routes];
                            newRoutes[routeIndex] = {
                              ...newRoutes[routeIndex],
                              tls_terminate: checked,
                              tls_passthrough: checked
                                ? false
                                : newRoutes[routeIndex].tls_passthrough,
                            };
                            setRoutes(newRoutes);
                            form.setFieldValue('routes', newRoutes);
                          }}
                        />
                      </Field>

                      <Field orientation="horizontal">
                        <FieldContent>
                          <FieldLabel>TLS Passthrough</FieldLabel>
                          <FieldDescription>
                            Forward encrypted TLS traffic directly to upstream
                          </FieldDescription>
                        </FieldContent>
                        <Switch
                          checked={route.tls_passthrough === true}
                          onCheckedChange={(checked: boolean) => {
                            const newRoutes = [...routes];
                            newRoutes[routeIndex] = {
                              ...newRoutes[routeIndex],
                              tls_passthrough: checked,
                              tls_terminate: checked ? false : newRoutes[routeIndex].tls_terminate,
                            };
                            setRoutes(newRoutes);
                            form.setFieldValue('routes', newRoutes);
                          }}
                        />
                      </Field>
                    </div>

                    <Field>
                      <FieldLabel>Proxy Protocol</FieldLabel>
                      <Select
                        value={route.proxy_protocol_version || 'none'}
                        onValueChange={(val) =>
                          updateRoute(
                            routeIndex,
                            'proxy_protocol_version',
                            val === 'none' ? undefined : (val as L4ProxyProtocolVersion),
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
                        Send client IP info to upstream using PROXY protocol (HAProxy standard)
                      </FieldDescription>
                    </Field>
                  </div>
                </CardContent>
              )}
            </Card>
          ))}
        </CardContent>
      </Card>

      {/* Actions */}
      <div className="flex justify-end gap-2 pt-4 border-t">
        <Button type="button" variant="outline" onClick={onCancel}>
          Cancel
        </Button>
        <Button type="submit" disabled={loading}>
          {loading ? 'Saving...' : initialData ? 'Save Changes' : 'Create L4 Proxy'}
        </Button>
      </div>
    </form>
  );
}
