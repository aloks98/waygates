import type { PaginatedResponse } from './api';
import type { CustomHeaders } from './proxy';

/**
 * ProxyGroup is a config-inheritance parent that member HTTP proxies inherit
 * settings from. It is NOT an ACLGroup (an auth grouping) — see acl.ts.
 *
 * null on the four settings booleans means "the group says nothing"; members
 * fall through to their own value or the system default. Never coerce null
 * to false — that silently turns "inherit" into "explicitly disabled".
 */
export interface ProxyGroup {
  id: number;
  name: string;
  description?: string;
  base_domain?: string;
  ssl_enabled: boolean | null;
  ssl_forced: boolean | null;
  tls_insecure_skip_verify: boolean | null;
  block_exploits: boolean | null;
  custom_headers?: CustomHeaders;
  member_count: number;
  created_at: string;
  updated_at: string;
}

export type ProxyGroupListResponse = PaginatedResponse<ProxyGroup>;

export type CreateProxyGroupRequest = Omit<
  ProxyGroup,
  'id' | 'member_count' | 'created_at' | 'updated_at'
>;
export type UpdateProxyGroupRequest = CreateProxyGroupRequest;

/**
 * A row from GET /api/proxy-groups/:id/acl. Mirrors ProxyACLAssignment
 * (types/acl.ts) column-for-column, scoped to a proxy group instead of a
 * single proxy.
 */
export interface ProxyGroupAclAssignment {
  id: number;
  proxy_group_id: number;
  acl_group_id: number;
  path_pattern: string;
  priority: number;
  enabled: boolean;
  acl_group?: { id: number; name: string };
}

export interface AssignProxyGroupAclRequest {
  acl_group_id: number;
  path_pattern: string;
  priority: number;
  // Omitted means "enabled" (server default). An explicit false is the
  // documented way a member proxy's group opts out of an ACL in the same
  // save that adds it — see diffAclAssignments in lib/diff-acl-assignments.ts.
  enabled?: boolean;
}

export interface UpdateProxyGroupAclRequest {
  path_pattern: string;
  priority: number;
  enabled: boolean;
}
