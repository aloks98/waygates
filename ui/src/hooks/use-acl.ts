import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { HTTPError } from 'ky';
import { toast } from 'sonner';

import { api } from '../lib/api';
import type {
  ACLBasicAuthUser,
  ACLBranding,
  ACLExternalProvider,
  ACLGroup,
  ACLGroupListParams,
  ACLGroupListResponse,
  ACLIPRule,
  ACLOAuthProviderRestriction,
  ACLWaygatesAuth,
  AssignACLRequest,
  ConfigureWaygatesAuthRequest,
  CreateACLGroupRequest,
  CreateBasicAuthUserRequest,
  CreateExternalProviderRequest,
  CreateIPRuleRequest,
  OAuthProvider,
  ProxyACLAssignment,
  SetOAuthProviderRestrictionRequest,
  UpdateACLBrandingRequest,
  UpdateACLGroupRequest,
  UpdateBasicAuthUserRequest,
  UpdateExternalProviderRequest,
  UpdateIPRuleRequest,
  UpdateProxyACLAssignmentRequest,
} from '../types/acl';
import type { ApiResponse } from '../types/api';

// Query Keys
const ACL_GROUPS_KEY = ['acl-groups'] as const;
const ACL_BRANDING_KEY = ['acl-branding'] as const;
const OAUTH_PROVIDERS_KEY = ['oauth-providers'] as const;

function aclGroupKey(id: number) {
  return [...ACL_GROUPS_KEY, id] as const;
}

function ipRulesKey(groupId: number) {
  return [...ACL_GROUPS_KEY, groupId, 'ip-rules'] as const;
}

function basicAuthUsersKey(groupId: number) {
  return [...ACL_GROUPS_KEY, groupId, 'basic-auth'] as const;
}

function externalProvidersKey(groupId: number) {
  return [...ACL_GROUPS_KEY, groupId, 'providers'] as const;
}

function waygatesAuthKey(groupId: number) {
  return [...ACL_GROUPS_KEY, groupId, 'waygates-auth'] as const;
}

function oauthProviderRestrictionsKey(groupId: number) {
  return [...ACL_GROUPS_KEY, groupId, 'oauth-restrictions'] as const;
}

function proxyAclKey(proxyId: number) {
  return ['proxies', proxyId, 'acl'] as const;
}

async function handleApiError(error: unknown): Promise<string> {
  if (error instanceof HTTPError) {
    try {
      const body = (await error.response.json()) as { error?: { message?: string } };
      return body?.error?.message || error.message;
    } catch {
      return error.message;
    }
  }
  return error instanceof Error ? error.message : 'An unknown error occurred';
}

// =============================================================================
// ACL Groups
// =============================================================================

export function useACLGroups(params: ACLGroupListParams = {}) {
  const { page = 1, limit = 20, ...filters } = params;

  const query = useQuery({
    queryKey: [...ACL_GROUPS_KEY, { page, limit, ...filters }],
    queryFn: async () => {
      const searchParams: Record<string, string> = {
        page: String(page),
        limit: String(limit),
      };

      if (filters.search) searchParams.search = filters.search;
      if (filters.sort) searchParams.sort = filters.sort;
      if (filters.order) searchParams.order = filters.order;

      const response = await api
        .get('acl/groups', { searchParams })
        .json<ApiResponse<ACLGroupListResponse>>();
      return response.data;
    },
  });

  return {
    groups: query.data?.items ?? [],
    total: query.data?.total ?? 0,
    page: query.data?.page ?? 1,
    limit: query.data?.limit ?? limit,
    totalPages: query.data?.total_pages ?? 1,
    isLoading: query.isLoading,
    isError: query.isError,
    error: query.error,
    refetch: query.refetch,
  };
}

export function useACLGroup(id: number) {
  const query = useQuery({
    queryKey: aclGroupKey(id),
    queryFn: async () => {
      const response = await api.get(`acl/groups/${id}`).json<ApiResponse<ACLGroup>>();
      return response.data;
    },
    enabled: id > 0,
  });

  return {
    group: query.data,
    isLoading: query.isLoading,
    isError: query.isError,
    error: query.error,
    refetch: query.refetch,
  };
}

export function useCreateACLGroup() {
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async (data: CreateACLGroupRequest) => {
      return await api.post('acl/groups', { json: data }).json<ApiResponse<ACLGroup>>();
    },
    onSuccess: (response) => {
      queryClient.invalidateQueries({ queryKey: ACL_GROUPS_KEY });
      toast.success(`ACL group "${response.data?.name}" created successfully`);
    },
    onError: async (error) => {
      const message = await handleApiError(error);
      toast.error('Failed to create ACL group', { description: message });
    },
  });

  return {
    createGroup: mutation.mutateAsync,
    isCreating: mutation.isPending,
  };
}

export function useUpdateACLGroup() {
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async ({ id, data }: { id: number; data: UpdateACLGroupRequest }) => {
      return await api.put(`acl/groups/${id}`, { json: data }).json<ApiResponse<ACLGroup>>();
    },
    onSuccess: (response, variables) => {
      queryClient.invalidateQueries({ queryKey: ACL_GROUPS_KEY });
      queryClient.invalidateQueries({ queryKey: aclGroupKey(variables.id) });
      toast.success(`ACL group "${response.data?.name}" updated successfully`);
    },
    onError: async (error) => {
      const message = await handleApiError(error);
      toast.error('Failed to update ACL group', { description: message });
    },
  });

  return {
    updateGroup: mutation.mutateAsync,
    isUpdating: mutation.isPending,
  };
}

export function useDeleteACLGroup() {
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async (id: number) => {
      await api.delete(`acl/groups/${id}`);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ACL_GROUPS_KEY });
      toast.success('ACL group deleted successfully');
    },
    onError: async (error) => {
      const message = await handleApiError(error);
      toast.error('Failed to delete ACL group', { description: message });
    },
  });

  return {
    deleteGroup: mutation.mutateAsync,
    isDeleting: mutation.isPending,
  };
}

// =============================================================================
// IP Rules
// =============================================================================

export function useIPRules(groupId: number) {
  const query = useQuery({
    queryKey: ipRulesKey(groupId),
    queryFn: async () => {
      const response = await api
        .get(`acl/groups/${groupId}/ip-rules`)
        .json<ApiResponse<ACLIPRule[]>>();
      return response.data;
    },
    enabled: groupId > 0,
  });

  return {
    rules: query.data ?? [],
    isLoading: query.isLoading,
    isError: query.isError,
    error: query.error,
    refetch: query.refetch,
  };
}

export function useAddIPRule() {
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async ({ groupId, data }: { groupId: number; data: CreateIPRuleRequest }) => {
      return await api
        .post(`acl/groups/${groupId}/ip-rules`, { json: data })
        .json<ApiResponse<ACLIPRule>>();
    },
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: ipRulesKey(variables.groupId) });
      queryClient.invalidateQueries({ queryKey: aclGroupKey(variables.groupId) });
      toast.success('IP rule added successfully');
    },
    onError: async (error) => {
      const message = await handleApiError(error);
      toast.error('Failed to add IP rule', { description: message });
    },
  });

  return {
    addRule: mutation.mutateAsync,
    isAdding: mutation.isPending,
  };
}

export function useUpdateIPRule() {
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async ({
      id,
      data,
    }: {
      id: number;
      groupId: number;
      data: UpdateIPRuleRequest;
    }) => {
      return await api.put(`acl/ip-rules/${id}`, { json: data }).json<ApiResponse<ACLIPRule>>();
    },
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: ipRulesKey(variables.groupId) });
      queryClient.invalidateQueries({ queryKey: aclGroupKey(variables.groupId) });
      toast.success('IP rule updated successfully');
    },
    onError: async (error) => {
      const message = await handleApiError(error);
      toast.error('Failed to update IP rule', { description: message });
    },
  });

  return {
    updateRule: mutation.mutateAsync,
    isUpdating: mutation.isPending,
  };
}

export function useDeleteIPRule() {
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async ({ id }: { id: number; groupId: number }) => {
      await api.delete(`acl/ip-rules/${id}`);
    },
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: ipRulesKey(variables.groupId) });
      queryClient.invalidateQueries({ queryKey: aclGroupKey(variables.groupId) });
      toast.success('IP rule deleted successfully');
    },
    onError: async (error) => {
      const message = await handleApiError(error);
      toast.error('Failed to delete IP rule', { description: message });
    },
  });

  return {
    deleteRule: mutation.mutateAsync,
    isDeleting: mutation.isPending,
  };
}

// =============================================================================
// Basic Auth Users
// =============================================================================

export function useBasicAuthUsers(groupId: number) {
  const query = useQuery({
    queryKey: basicAuthUsersKey(groupId),
    queryFn: async () => {
      const response = await api
        .get(`acl/groups/${groupId}/basic-auth`)
        .json<ApiResponse<ACLBasicAuthUser[]>>();
      return response.data;
    },
    enabled: groupId > 0,
  });

  return {
    users: query.data ?? [],
    isLoading: query.isLoading,
    isError: query.isError,
    error: query.error,
    refetch: query.refetch,
  };
}

export function useAddBasicAuthUser() {
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async ({
      groupId,
      data,
    }: {
      groupId: number;
      data: CreateBasicAuthUserRequest;
    }) => {
      return await api
        .post(`acl/groups/${groupId}/basic-auth`, { json: data })
        .json<ApiResponse<ACLBasicAuthUser>>();
    },
    onSuccess: (response, variables) => {
      queryClient.invalidateQueries({ queryKey: basicAuthUsersKey(variables.groupId) });
      queryClient.invalidateQueries({ queryKey: aclGroupKey(variables.groupId) });
      toast.success(`User "${response.data?.username}" added successfully`);
    },
    onError: async (error) => {
      const message = await handleApiError(error);
      toast.error('Failed to add basic auth user', { description: message });
    },
  });

  return {
    addUser: mutation.mutateAsync,
    isAdding: mutation.isPending,
  };
}

export function useUpdateBasicAuthUser() {
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async ({
      id,
      data,
    }: {
      id: number;
      groupId: number;
      data: UpdateBasicAuthUserRequest;
    }) => {
      return await api
        .put(`acl/basic-auth/${id}`, { json: data })
        .json<ApiResponse<ACLBasicAuthUser>>();
    },
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: basicAuthUsersKey(variables.groupId) });
      queryClient.invalidateQueries({ queryKey: aclGroupKey(variables.groupId) });
      toast.success('Basic auth user updated successfully');
    },
    onError: async (error) => {
      const message = await handleApiError(error);
      toast.error('Failed to update basic auth user', { description: message });
    },
  });

  return {
    updateUser: mutation.mutateAsync,
    isUpdating: mutation.isPending,
  };
}

export function useDeleteBasicAuthUser() {
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async ({ id }: { id: number; groupId: number }) => {
      await api.delete(`acl/basic-auth/${id}`);
    },
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: basicAuthUsersKey(variables.groupId) });
      queryClient.invalidateQueries({ queryKey: aclGroupKey(variables.groupId) });
      toast.success('Basic auth user deleted successfully');
    },
    onError: async (error) => {
      const message = await handleApiError(error);
      toast.error('Failed to delete basic auth user', { description: message });
    },
  });

  return {
    deleteUser: mutation.mutateAsync,
    isDeleting: mutation.isPending,
  };
}

// =============================================================================
// External Providers
// =============================================================================

export function useExternalProviders(groupId: number) {
  const query = useQuery({
    queryKey: externalProvidersKey(groupId),
    queryFn: async () => {
      const response = await api
        .get(`acl/groups/${groupId}/providers`)
        .json<ApiResponse<ACLExternalProvider[]>>();
      return response.data;
    },
    enabled: groupId > 0,
  });

  return {
    providers: query.data ?? [],
    isLoading: query.isLoading,
    isError: query.isError,
    error: query.error,
    refetch: query.refetch,
  };
}

export function useAddExternalProvider() {
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async ({
      groupId,
      data,
    }: {
      groupId: number;
      data: CreateExternalProviderRequest;
    }) => {
      return await api
        .post(`acl/groups/${groupId}/providers`, { json: data })
        .json<ApiResponse<ACLExternalProvider>>();
    },
    onSuccess: (response, variables) => {
      queryClient.invalidateQueries({ queryKey: externalProvidersKey(variables.groupId) });
      queryClient.invalidateQueries({ queryKey: aclGroupKey(variables.groupId) });
      toast.success(`Provider "${response.data?.name}" added successfully`);
    },
    onError: async (error) => {
      const message = await handleApiError(error);
      toast.error('Failed to add external provider', { description: message });
    },
  });

  return {
    addProvider: mutation.mutateAsync,
    isAdding: mutation.isPending,
  };
}

export function useUpdateExternalProvider() {
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async ({
      id,
      data,
    }: {
      id: number;
      groupId: number;
      data: UpdateExternalProviderRequest;
    }) => {
      return await api
        .put(`acl/providers/${id}`, { json: data })
        .json<ApiResponse<ACLExternalProvider>>();
    },
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: externalProvidersKey(variables.groupId) });
      queryClient.invalidateQueries({ queryKey: aclGroupKey(variables.groupId) });
      toast.success('External provider updated successfully');
    },
    onError: async (error) => {
      const message = await handleApiError(error);
      toast.error('Failed to update external provider', { description: message });
    },
  });

  return {
    updateProvider: mutation.mutateAsync,
    isUpdating: mutation.isPending,
  };
}

export function useDeleteExternalProvider() {
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async ({ id }: { id: number; groupId: number }) => {
      await api.delete(`acl/providers/${id}`);
    },
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: externalProvidersKey(variables.groupId) });
      queryClient.invalidateQueries({ queryKey: aclGroupKey(variables.groupId) });
      toast.success('External provider deleted successfully');
    },
    onError: async (error) => {
      const message = await handleApiError(error);
      toast.error('Failed to delete external provider', { description: message });
    },
  });

  return {
    deleteProvider: mutation.mutateAsync,
    isDeleting: mutation.isPending,
  };
}

// =============================================================================
// Waygates Auth
// =============================================================================

export function useWaygatesAuth(groupId: number) {
  const query = useQuery({
    queryKey: waygatesAuthKey(groupId),
    queryFn: async () => {
      const response = await api
        .get(`acl/groups/${groupId}/waygates-auth`)
        .json<ApiResponse<ACLWaygatesAuth>>();
      return response.data;
    },
    enabled: groupId > 0,
  });

  return {
    config: query.data,
    isLoading: query.isLoading,
    isError: query.isError,
    error: query.error,
    refetch: query.refetch,
  };
}

export function useConfigureWaygatesAuth() {
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async ({
      groupId,
      data,
    }: {
      groupId: number;
      data: ConfigureWaygatesAuthRequest;
    }) => {
      return await api
        .put(`acl/groups/${groupId}/waygates-auth`, { json: data })
        .json<ApiResponse<ACLWaygatesAuth>>();
    },
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: waygatesAuthKey(variables.groupId) });
      queryClient.invalidateQueries({ queryKey: aclGroupKey(variables.groupId) });
      toast.success('Waygates authentication configured successfully');
    },
    onError: async (error) => {
      const message = await handleApiError(error);
      toast.error('Failed to configure Waygates authentication', { description: message });
    },
  });

  return {
    configureAuth: mutation.mutateAsync,
    isConfiguring: mutation.isPending,
  };
}

// =============================================================================
// OAuth Provider Restrictions
// =============================================================================

export function useOAuthProviderRestrictions(groupId: number) {
  const query = useQuery({
    queryKey: oauthProviderRestrictionsKey(groupId),
    queryFn: async () => {
      const response = await api
        .get(`acl/groups/${groupId}/oauth-restrictions`)
        .json<ApiResponse<ACLOAuthProviderRestriction[]>>();
      return response.data;
    },
    enabled: groupId > 0,
  });

  return {
    restrictions: query.data ?? [],
    isLoading: query.isLoading,
    isError: query.isError,
    error: query.error,
    refetch: query.refetch,
  };
}

export function useSetOAuthProviderRestriction() {
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async ({
      groupId,
      provider,
      data,
    }: {
      groupId: number;
      provider: string;
      data: SetOAuthProviderRestrictionRequest;
    }) => {
      return await api
        .put(`acl/groups/${groupId}/oauth-restrictions/${provider}`, { json: data })
        .json<ApiResponse<ACLOAuthProviderRestriction>>();
    },
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: oauthProviderRestrictionsKey(variables.groupId) });
      queryClient.invalidateQueries({ queryKey: aclGroupKey(variables.groupId) });
      toast.success(`OAuth restriction for ${variables.provider} updated successfully`);
    },
    onError: async (error) => {
      const message = await handleApiError(error);
      toast.error('Failed to update OAuth provider restriction', { description: message });
    },
  });

  return {
    setRestriction: mutation.mutateAsync,
    isSetting: mutation.isPending,
  };
}

export function useDeleteOAuthProviderRestriction() {
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async ({ groupId, provider }: { groupId: number; provider: string }) => {
      await api.delete(`acl/groups/${groupId}/oauth-restrictions/${provider}`);
    },
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: oauthProviderRestrictionsKey(variables.groupId) });
      queryClient.invalidateQueries({ queryKey: aclGroupKey(variables.groupId) });
      toast.success(`OAuth restriction for ${variables.provider} removed successfully`);
    },
    onError: async (error) => {
      const message = await handleApiError(error);
      toast.error('Failed to remove OAuth provider restriction', { description: message });
    },
  });

  return {
    deleteRestriction: mutation.mutateAsync,
    isDeleting: mutation.isPending,
  };
}

// =============================================================================
// Proxy ACL
// =============================================================================

export function useProxyACL(proxyId: number) {
  const query = useQuery({
    queryKey: proxyAclKey(proxyId),
    queryFn: async () => {
      const response = await api
        .get(`proxies/${proxyId}/acl`)
        .json<ApiResponse<ProxyACLAssignment[]>>();
      return response.data;
    },
    enabled: proxyId > 0,
  });

  return {
    assignments: query.data ?? [],
    isLoading: query.isLoading,
    isError: query.isError,
    error: query.error,
    refetch: query.refetch,
  };
}

export function useAssignACL() {
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async ({ proxyId, data }: { proxyId: number; data: AssignACLRequest }) => {
      return await api
        .post(`proxies/${proxyId}/acl`, { json: data })
        .json<ApiResponse<ProxyACLAssignment>>();
    },
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: proxyAclKey(variables.proxyId) });
      toast.success('ACL assigned to proxy successfully');
    },
    onError: async (error) => {
      const message = await handleApiError(error);
      toast.error('Failed to assign ACL to proxy', { description: message });
    },
  });

  return {
    assignACL: mutation.mutateAsync,
    isAssigning: mutation.isPending,
  };
}

export function useUpdateProxyACLAssignment() {
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async ({
      proxyId,
      assignmentId,
      data,
    }: {
      proxyId: number;
      assignmentId: number;
      data: UpdateProxyACLAssignmentRequest;
    }) => {
      return await api
        .put(`proxies/${proxyId}/acl/${assignmentId}`, { json: data })
        .json<ApiResponse<ProxyACLAssignment>>();
    },
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: proxyAclKey(variables.proxyId) });
      toast.success('ACL assignment updated successfully');
    },
    onError: async (error) => {
      const message = await handleApiError(error);
      toast.error('Failed to update ACL assignment', { description: message });
    },
  });

  return {
    updateAssignment: mutation.mutateAsync,
    isUpdating: mutation.isPending,
  };
}

export function useRemoveACL() {
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async ({ proxyId, groupId }: { proxyId: number; groupId: number }) => {
      await api.delete(`proxies/${proxyId}/acl/${groupId}`);
    },
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: proxyAclKey(variables.proxyId) });
      toast.success('ACL removed from proxy successfully');
    },
    onError: async (error) => {
      const message = await handleApiError(error);
      toast.error('Failed to remove ACL from proxy', { description: message });
    },
  });

  return {
    removeACL: mutation.mutateAsync,
    isRemoving: mutation.isPending,
  };
}

// =============================================================================
// Branding
// =============================================================================

export function useACLBranding() {
  const query = useQuery({
    queryKey: ACL_BRANDING_KEY,
    queryFn: async () => {
      const response = await api.get('acl/branding').json<ApiResponse<ACLBranding>>();
      return response.data;
    },
  });

  return {
    branding: query.data,
    isLoading: query.isLoading,
    isError: query.isError,
    error: query.error,
    refetch: query.refetch,
  };
}

export function useUpdateACLBranding() {
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async (data: UpdateACLBrandingRequest) => {
      return await api.put('acl/branding', { json: data }).json<ApiResponse<ACLBranding>>();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ACL_BRANDING_KEY });
      toast.success('ACL branding updated successfully');
    },
    onError: async (error) => {
      const message = await handleApiError(error);
      toast.error('Failed to update ACL branding', { description: message });
    },
  });

  return {
    updateBranding: mutation.mutateAsync,
    isUpdating: mutation.isPending,
  };
}

// =============================================================================
// OAuth Providers
// =============================================================================

// Returns available OAuth providers (those with env vars configured on backend)
export function useOAuthProviders() {
  const query = useQuery({
    queryKey: OAUTH_PROVIDERS_KEY,
    queryFn: async () => {
      const response = await api.get('auth/oauth/providers').json<ApiResponse<OAuthProvider[]>>();
      return response.data;
    },
    staleTime: 5 * 60 * 1000, // OAuth providers don't change frequently
  });

  return {
    providers: query.data ?? [],
    isLoading: query.isLoading,
    isError: query.isError,
    error: query.error,
    refetch: query.refetch,
  };
}
