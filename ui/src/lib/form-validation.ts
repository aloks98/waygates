import { type ZodSchema, z } from 'zod';

import {
  L4_LOAD_BALANCING_POLICIES,
  L4_MATCHER_TYPES,
  L4_PROTOCOLS,
  L4_PROXY_PROTOCOL_VERSIONS,
} from '../types/l4-proxy';

/**
 * Creates a TanStack Form validator from a Zod schema.
 * This is needed because @tanstack/zod-form-adapter doesn't support Zod 4.x yet.
 */
export function zodValidator<T>(schema: ZodSchema<T>) {
  return {
    validate: ({ value }: { value: T }) => {
      const result = schema.safeParse(value);
      if (result.success) {
        return undefined;
      }
      // Zod 4 uses 'issues' instead of 'errors'
      return result.error.issues.map((issue) => issue.message).join(', ');
    },
    validateAsync: async ({ value }: { value: T }) => {
      const result = await schema.safeParseAsync(value);
      if (result.success) {
        return undefined;
      }
      return result.error.issues.map((issue) => issue.message).join(', ');
    },
  };
}

// ============================================================================
// L4 Proxy Validation Schemas
// ============================================================================

/**
 * Schema for L4 upstream server configuration
 */
export const l4UpstreamSchema = z.object({
  host: z.string().min(1, 'Host is required'),
  port: z.number().min(1, 'Port must be at least 1').max(65535, 'Port must be at most 65535'),
  weight: z.number().min(1, 'Weight must be at least 1').optional(),
});

export type L4UpstreamFormValues = z.infer<typeof l4UpstreamSchema>;

/**
 * Schema for L4 route configuration
 */
export const l4RouteSchema = z
  .object({
    priority: z.number().min(0, 'Priority must be at least 0'),
    matcher_type: z.enum(L4_MATCHER_TYPES, {
      errorMap: () => ({ message: 'Invalid matcher type' }),
    }),
    sni_hostnames: z.array(z.string()).optional(),
    allowed_ip_ranges: z.array(z.string()).optional(),
    regex_pattern: z.string().optional(),
    upstreams: z.array(l4UpstreamSchema).min(1, 'At least one upstream is required'),
    load_balancing_policy: z.enum(L4_LOAD_BALANCING_POLICIES, {
      errorMap: () => ({ message: 'Invalid load balancing policy' }),
    }),
    tls_terminate: z.boolean(),
    tls_passthrough: z.boolean(),
    proxy_protocol_version: z.enum(L4_PROXY_PROTOCOL_VERSIONS).optional(),
  })
  .superRefine((data, ctx) => {
    // Validate sni_hostnames required for tls matcher
    if (data.matcher_type === 'tls') {
      if (!data.sni_hostnames || data.sni_hostnames.length === 0) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          message: 'SNI hostnames are required for TLS matcher',
          path: ['sni_hostnames'],
        });
      }
    }
    // Validate regex_pattern required for regexp matcher
    if (data.matcher_type === 'regexp') {
      if (!data.regex_pattern || data.regex_pattern.trim() === '') {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          message: 'Regex pattern is required for regexp matcher',
          path: ['regex_pattern'],
        });
      }
    }
    // Validate allowed_ip_ranges for remote_ip matcher
    if (data.matcher_type === 'remote_ip') {
      if (!data.allowed_ip_ranges || data.allowed_ip_ranges.length === 0) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          message: 'IP ranges are required for remote_ip matcher',
          path: ['allowed_ip_ranges'],
        });
      }
    }
    // Validate TLS settings are mutually exclusive
    if (data.tls_terminate && data.tls_passthrough) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        message: 'TLS terminate and passthrough cannot both be enabled',
        path: ['tls_terminate'],
      });
    }
  });

export type L4RouteFormValues = z.infer<typeof l4RouteSchema>;

/**
 * Schema for L4 proxy configuration
 */
export const l4ProxySchema = z.object({
  name: z.string().min(1, 'Name is required').max(255, 'Name must be at most 255 characters'),
  description: z.string().max(500, 'Description must be at most 500 characters').optional(),
  listen_port: z
    .number()
    .min(1, 'Port must be at least 1')
    .max(65535, 'Port must be at most 65535'),
  protocol: z.enum(L4_PROTOCOLS, {
    errorMap: () => ({ message: 'Protocol must be tcp or udp' }),
  }),
  is_active: z.boolean(),
  routes: z.array(l4RouteSchema).optional(),
});

export type L4ProxyFormValues = z.infer<typeof l4ProxySchema>;
