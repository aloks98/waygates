import type { User } from './auth';

// Enums/Constants
export type CombinationMode = 'any' | 'all' | 'ip_bypass';
export type IPRuleType = 'allow' | 'deny' | 'bypass';
export type ProviderType = 'authelia' | 'authentik' | 'custom';

// Core Types
export interface ACLGroup {
  id: number;
  name: string;
  description?: string;
  combination_mode: CombinationMode;
  created_at: string;
  updated_at: string;
  created_by_user?: User;
  ip_rules?: ACLIPRule[];
  basic_auth_users?: ACLBasicAuthUser[];
  external_providers?: ACLExternalProvider[];
  waygates_auth?: ACLWaygatesAuth;
}

export interface ACLIPRule {
  id: number;
  acl_group_id: number;
  rule_type: IPRuleType;
  cidr: string;
  description?: string;
  priority: number;
  created_at: string;
}

export interface ACLBasicAuthUser {
  id: number;
  acl_group_id: number;
  username: string;
  created_at: string;
  updated_at: string;
}

export interface ACLExternalProvider {
  id: number;
  acl_group_id: number;
  provider_type: ProviderType;
  name: string;
  verify_url: string;
  auth_redirect_url?: string;
  headers_to_copy?: string[];
  created_at: string;
  updated_at: string;
}

export interface ACLWaygatesAuth {
  id: number;
  acl_group_id: number;
  enabled: boolean;
  allowed_users?: number[];
  allowed_roles?: string[];
  allowed_email_patterns?: string[];
  require_2fa: boolean;
  session_ttl: number;
  created_at: string;
}

export interface ProxyACLAssignment {
  id: number;
  proxy_id: number;
  acl_group_id: number;
  path_pattern: string;
  priority: number;
  enabled: boolean;
  created_at: string;
  acl_group?: ACLGroup;
}

export interface ACLBranding {
  id: number;
  logo_url?: string;
  primary_color: string;
  background_color: string;
  title: string;
  subtitle?: string;
  footer_text?: string;
  custom_css?: string;
  updated_at: string;
}

export interface OAuthProvider {
  id: string;
  name: string;
  enabled: boolean;
}

// Request Types
export interface CreateACLGroupRequest {
  name: string;
  description?: string;
  combination_mode?: CombinationMode;
}

export interface UpdateACLGroupRequest {
  name?: string;
  description?: string;
  combination_mode?: CombinationMode;
}

export interface CreateIPRuleRequest {
  rule_type: IPRuleType;
  cidr: string;
  description?: string;
  priority?: number;
}

export interface UpdateIPRuleRequest {
  rule_type?: IPRuleType;
  cidr?: string;
  description?: string;
  priority?: number;
}

export interface CreateBasicAuthUserRequest {
  username: string;
  password: string;
}

export interface UpdateBasicAuthUserRequest {
  username?: string;
  password?: string;
}

export interface CreateExternalProviderRequest {
  provider_type: ProviderType;
  name: string;
  verify_url: string;
  auth_redirect_url?: string;
  headers_to_copy?: string[];
}

export interface UpdateExternalProviderRequest {
  provider_type?: ProviderType;
  name?: string;
  verify_url?: string;
  auth_redirect_url?: string;
  headers_to_copy?: string[];
}

export interface ConfigureWaygatesAuthRequest {
  enabled: boolean;
  allowed_users?: number[];
  allowed_roles?: string[];
  allowed_email_patterns?: string[];
  require_2fa?: boolean;
  session_ttl?: number;
}

export interface AssignACLRequest {
  acl_group_id: number;
  path_pattern: string;
  priority?: number;
  enabled?: boolean;
}

export interface UpdateProxyACLAssignmentRequest {
  path_pattern?: string;
  priority?: number;
  enabled?: boolean;
}

export interface UpdateACLBrandingRequest {
  logo_url?: string;
  primary_color?: string;
  background_color?: string;
  title?: string;
  subtitle?: string;
  footer_text?: string;
  custom_css?: string;
}

// Response Types
export interface ACLGroupListResponse {
  items: ACLGroup[];
  total: number;
  page: number;
  limit: number;
  total_pages: number;
}

// List Parameters
export interface ACLGroupListParams {
  page?: number;
  limit?: number;
  search?: string;
  sort?: 'name' | 'created_at' | 'updated_at';
  order?: 'asc' | 'desc';
}
