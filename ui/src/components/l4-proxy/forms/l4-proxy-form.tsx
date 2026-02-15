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
import { ChevronDown, ChevronUp, Plus, Trash2 } from 'lucide-react';
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

// Matcher type display labels
const MATCHER_TYPE_LABELS: Record<L4MatcherType, string> = {
  any: 'Any (Match All)',
  tls: 'TLS/SNI',
  ssh: 'SSH',
  postgres: 'PostgreSQL',
  http: 'HTTP',
  rdp: 'RDP',
  socks5: 'SOCKS5',
  remote_ip: 'Remote IP',
  regexp: 'Regular Expression',
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
      setRoutes(normalizeRoutes(initialData));
      form.reset(defaultValues);
    }
  }, [initialData, form, defaultValues]);

  // Route management functions
  const addRoute = () => {
    const newRoutes = [...routes, createEmptyRoute()];
    setRoutes(newRoutes);
    setExpandedRoutes(new Set([...expandedRoutes, newRoutes.length - 1]));
  };

  const removeRoute = (index: number) => {
    const newRoutes = routes.filter((_, i) => i !== index);
    setRoutes(newRoutes.length > 0 ? newRoutes : [createEmptyRoute()]);
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
                  <FieldDescription>Transport protocol</FieldDescription>
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
              Configure routing rules and upstream servers for this proxy
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
                      {MATCHER_TYPE_LABELS[route.matcher_type]}
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
                  {/* Route Basic Settings */}
                  <div className="grid gap-4 sm:grid-cols-3">
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
                      />
                      <FieldDescription>Lower values match first</FieldDescription>
                    </Field>

                    <Field>
                      <FieldLabel>Matcher Type</FieldLabel>
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
                              {MATCHER_TYPE_LABELS[type]}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                      <FieldDescription>How to match incoming connections</FieldDescription>
                    </Field>

                    <Field>
                      <FieldLabel>Load Balancing</FieldLabel>
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
                    </Field>
                  </div>

                  {/* Matcher-specific fields */}
                  {route.matcher_type === 'tls' && (
                    <div className="space-y-3">
                      <div className="flex items-center justify-between">
                        <FieldLabel>SNI Hostnames</FieldLabel>
                        <Button
                          type="button"
                          variant="outline"
                          size="sm"
                          onClick={() => addSniHostname(routeIndex)}
                        >
                          <Plus className="mr-1 size-3" />
                          Add Hostname
                        </Button>
                      </div>
                      {(route.sni_hostnames?.length ?? 0) === 0 && (
                        <p className="text-sm text-muted-foreground">
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
                    <div className="space-y-3">
                      <div className="flex items-center justify-between">
                        <FieldLabel>Allowed IP Ranges</FieldLabel>
                        <Button
                          type="button"
                          variant="outline"
                          size="sm"
                          onClick={() => addIpRange(routeIndex)}
                        >
                          <Plus className="mr-1 size-3" />
                          Add IP Range
                        </Button>
                      </div>
                      {(route.allowed_ip_ranges?.length ?? 0) === 0 && (
                        <p className="text-sm text-muted-foreground">
                          Add at least one IP range for remote_ip matching
                        </p>
                      )}
                      {route.allowed_ip_ranges?.map((ipRange, ipIndex) => (
                        <div key={ipIndex} className="flex items-center gap-2">
                          <Input
                            placeholder="192.168.1.0/24"
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
                    <Field>
                      <FieldLabel>Regex Pattern</FieldLabel>
                      <Input
                        placeholder="^GET /api/.*"
                        value={route.regex_pattern || ''}
                        onChange={(e) => updateRoute(routeIndex, 'regex_pattern', e.target.value)}
                      />
                      <FieldDescription>Regular expression to match against data</FieldDescription>
                    </Field>
                  )}

                  {/* TLS Settings */}
                  <div className="grid gap-4 sm:grid-cols-3">
                    <Field orientation="horizontal">
                      <FieldContent>
                        <FieldLabel>TLS Terminate</FieldLabel>
                        <FieldDescription>Terminate TLS at the proxy</FieldDescription>
                      </FieldContent>
                      <Switch
                        checked={route.tls_terminate}
                        onCheckedChange={(checked) => {
                          updateRoute(routeIndex, 'tls_terminate', checked);
                          if (checked) {
                            updateRoute(routeIndex, 'tls_passthrough', false);
                          }
                        }}
                      />
                    </Field>

                    <Field orientation="horizontal">
                      <FieldContent>
                        <FieldLabel>TLS Passthrough</FieldLabel>
                        <FieldDescription>Pass TLS to upstream</FieldDescription>
                      </FieldContent>
                      <Switch
                        checked={route.tls_passthrough}
                        onCheckedChange={(checked) => {
                          updateRoute(routeIndex, 'tls_passthrough', checked);
                          if (checked) {
                            updateRoute(routeIndex, 'tls_terminate', false);
                          }
                        }}
                      />
                    </Field>

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
                        <SelectTrigger>
                          <SelectValue placeholder="None" />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="none">None</SelectItem>
                          {L4_PROXY_PROTOCOL_VERSIONS.map((version) => (
                            <SelectItem key={version} value={version}>
                              {version.toUpperCase()}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </Field>
                  </div>

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
                    {route.upstreams.map((upstream, upstreamIndex) => (
                      <div key={upstreamIndex} className="flex items-start gap-2">
                        <div className="flex-1">
                          <Input
                            placeholder="192.168.1.100"
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
                            placeholder="3306"
                            value={upstream.port || ''}
                            onChange={(e) => {
                              const value = e.target.value.replace(/\D/g, '');
                              const port = value ? Math.min(Number.parseInt(value, 10), 65535) : 0;
                              updateUpstream(routeIndex, upstreamIndex, 'port', port);
                            }}
                          />
                        </div>
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
                        {route.upstreams.length > 1 && (
                          <Button
                            type="button"
                            variant="ghost"
                            size="icon"
                            onClick={() => removeUpstream(routeIndex, upstreamIndex)}
                          >
                            <Trash2 className="size-4 text-destructive" />
                          </Button>
                        )}
                      </div>
                    ))}
                    <p className="text-xs text-muted-foreground">
                      Format: Host | Port | Weight (optional)
                    </p>
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
