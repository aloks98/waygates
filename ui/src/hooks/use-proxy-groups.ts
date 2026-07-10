import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { HTTPError } from 'ky';
import { toast } from 'sonner';

import { api } from '../lib/api';
import type { ApiResponse } from '../types/api';
import type {
  AssignProxyGroupAclRequest,
  CreateProxyGroupRequest,
  ProxyGroup,
  ProxyGroupAclAssignment,
  ProxyGroupListResponse,
  UpdateProxyGroupAclRequest,
  UpdateProxyGroupRequest,
} from '../types/proxy-group';

const QUERY_KEY = ['proxy-groups'] as const;
// A proxy group mutation (settings or ACL) changes the effective config of
// every member proxy at once, so the proxies cache is stale too — without
// this the proxy grid/detail views keep showing pre-edit values.
const PROXIES_KEY = ['proxies'] as const;

function groupAclKey(groupId: number) {
  return [...QUERY_KEY, groupId, 'acl'] as const;
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

export function useProxyGroups() {
  const queryClient = useQueryClient();

  const query = useQuery({
    queryKey: QUERY_KEY,
    queryFn: async () => {
      const response = await api.get('proxy-groups').json<ApiResponse<ProxyGroupListResponse>>();
      return response.data;
    },
  });

  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: QUERY_KEY });
    void queryClient.invalidateQueries({ queryKey: PROXIES_KEY });
  };

  const create = useMutation({
    mutationFn: async (body: CreateProxyGroupRequest) => {
      const response = await api
        .post('proxy-groups', { json: body })
        .json<ApiResponse<ProxyGroup>>();
      return response.data;
    },
    onSuccess: (group) => {
      invalidate();
      toast.success(`Proxy group "${group?.name}" created successfully`);
    },
    onError: async (error) => {
      toast.error('Failed to create proxy group', { description: await handleApiError(error) });
    },
  });

  const update = useMutation({
    mutationFn: async ({ id, ...body }: UpdateProxyGroupRequest & { id: number }) => {
      const response = await api
        .put(`proxy-groups/${id}`, { json: body })
        .json<ApiResponse<ProxyGroup>>();
      return response.data;
    },
    onSuccess: (group) => {
      invalidate();
      toast.success(`Proxy group "${group?.name}" updated successfully`);
    },
    onError: async (error) => {
      toast.error('Failed to update proxy group', { description: await handleApiError(error) });
    },
  });

  const remove = useMutation({
    mutationFn: async (id: number) => {
      await api.delete(`proxy-groups/${id}`);
    },
    onSuccess: () => {
      invalidate();
      toast.success('Proxy group deleted successfully');
    },
    // A non-empty group deletes with 409, and the server message already
    // names the member count ("...: 7 member proxies; reassign or remove
    // them first"). Surface it verbatim instead of a generic failure toast —
    // the caller shouldn't need a second round trip just to say how many.
    onError: async (error) => {
      toast.error('Failed to delete proxy group', { description: await handleApiError(error) });
    },
  });

  return { ...query, create, update, remove };
}

export function useProxyGroup(id: number) {
  return useQuery({
    queryKey: [...QUERY_KEY, id],
    queryFn: async () => {
      const response = await api.get(`proxy-groups/${id}`).json<ApiResponse<ProxyGroup>>();
      return response.data;
    },
    enabled: id > 0,
  });
}

/**
 * ACL assignments for a proxy group. Task 5 built GET/POST/PUT/DELETE
 * /api/proxy-groups/:id/acl — without a caller here, "the group names one or
 * more ACL groups" (the highest-value inheritable field per the spec) is
 * unreachable from the UI.
 */
export function useProxyGroupAcl(groupId: number) {
  const queryClient = useQueryClient();
  const key = groupAclKey(groupId);

  const query = useQuery({
    queryKey: key,
    queryFn: async () => {
      const response = await api
        .get(`proxy-groups/${groupId}/acl`)
        .json<ApiResponse<ProxyGroupAclAssignment[]>>();
      return response.data ?? [];
    },
    enabled: groupId > 0,
  });

  // Changing a group's ACL changes what Caddy enforces for every member.
  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: key });
    void queryClient.invalidateQueries({ queryKey: PROXIES_KEY });
  };

  const assign = useMutation({
    mutationFn: async (body: AssignProxyGroupAclRequest) => {
      // The server returns the full updated assignment list, not just the
      // new row — this saves the caller a second round trip.
      const response = await api
        .post(`proxy-groups/${groupId}/acl`, { json: body })
        .json<ApiResponse<ProxyGroupAclAssignment[]>>();
      return response.data ?? [];
    },
    onSuccess: invalidate,
    onError: async (error) => {
      toast.error('Failed to assign access group', { description: await handleApiError(error) });
    },
  });

  const update = useMutation({
    mutationFn: async ({
      assignmentId,
      ...body
    }: UpdateProxyGroupAclRequest & { assignmentId: number }) => {
      await api.put(`proxy-groups/${groupId}/acl/${assignmentId}`, { json: body });
    },
    onSuccess: invalidate,
    onError: async (error) => {
      toast.error('Failed to update access group assignment', {
        description: await handleApiError(error),
      });
    },
  });

  const remove = useMutation({
    mutationFn: async (aclGroupId: number) => {
      await api.delete(`proxy-groups/${groupId}/acl/${aclGroupId}`);
    },
    onSuccess: invalidate,
    onError: async (error) => {
      toast.error('Failed to remove access group', { description: await handleApiError(error) });
    },
  });

  return { ...query, data: query.data ?? [], assign, update, remove };
}
