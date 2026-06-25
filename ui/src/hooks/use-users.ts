import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { HTTPError } from 'ky';
import { toast } from 'sonner';

import { api, publicApi } from '../lib/api';
import type { ApiResponse } from '../types/api';
import type {
  CreateUserRequest,
  ManagedUser,
  ResetPasswordRequest,
  UpdateUserRequest,
} from '../types/user';

const USERS_QUERY_KEY = ['users'] as const;
const REGISTRATION_QUERY_KEY = ['settings', 'registration-status'] as const;

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

export function useUsers() {
  const queryClient = useQueryClient();

  const query = useQuery({
    queryKey: USERS_QUERY_KEY,
    queryFn: async () => {
      const response = await api.get('users').json<ApiResponse<ManagedUser[]>>();
      return response.data ?? [];
    },
  });

  const createUserMutation = useMutation({
    mutationFn: async (data: CreateUserRequest) => {
      return await api.post('users', { json: data }).json<ApiResponse<ManagedUser>>();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: USERS_QUERY_KEY });
      toast.success('User created successfully');
    },
    onError: async (error) => {
      const message = await handleApiError(error);
      toast.error('Failed to create user', { description: message });
    },
  });

  const updateUserMutation = useMutation({
    mutationFn: async ({ id, data }: { id: number; data: UpdateUserRequest }) => {
      return await api.put(`users/${id}`, { json: data }).json<ApiResponse<ManagedUser>>();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: USERS_QUERY_KEY });
      toast.success('User updated successfully');
    },
    onError: async (error) => {
      const message = await handleApiError(error);
      toast.error('Failed to update user', { description: message });
    },
  });

  const resetPasswordMutation = useMutation({
    mutationFn: async ({ id, data }: { id: number; data: ResetPasswordRequest }) => {
      return await api.post(`users/${id}/password`, { json: data }).json<ApiResponse<void>>();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: USERS_QUERY_KEY });
      toast.success('Password reset successfully');
    },
    onError: async (error) => {
      const message = await handleApiError(error);
      toast.error('Failed to reset password', { description: message });
    },
  });

  const deleteUserMutation = useMutation({
    mutationFn: async (id: number) => {
      return await api.delete(`users/${id}`).json<ApiResponse<void>>();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: USERS_QUERY_KEY });
      toast.success('User deleted successfully');
    },
    onError: async (error) => {
      const message = await handleApiError(error);
      toast.error('Failed to delete user', { description: message });
    },
  });

  return {
    users: query.data ?? [],
    isLoading: query.isLoading,
    isError: query.isError,
    error: query.error,
    createUser: createUserMutation.mutateAsync,
    isCreating: createUserMutation.isPending,
    updateUser: updateUserMutation.mutateAsync,
    isUpdating: updateUserMutation.isPending,
    resetPassword: resetPasswordMutation.mutateAsync,
    isResettingPassword: resetPasswordMutation.isPending,
    deleteUser: deleteUserMutation.mutateAsync,
    isDeleting: deleteUserMutation.isPending,
  };
}

export function useRegistrationSetting() {
  const queryClient = useQueryClient();

  const query = useQuery({
    queryKey: REGISTRATION_QUERY_KEY,
    queryFn: async () => {
      const response = await publicApi
        .get('auth/registration-status')
        .json<ApiResponse<{ open: boolean }>>();
      return response.data ?? { open: false };
    },
  });

  const mutation = useMutation({
    mutationFn: async (open: boolean) => {
      return await api
        .put('settings/auth.open_registration', { json: { value: open ? 'true' : 'false' } })
        .json<ApiResponse<{ key: string; value: string }>>();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: REGISTRATION_QUERY_KEY });
      toast.success('Registration setting updated');
    },
    onError: async (error) => {
      const message = await handleApiError(error);
      toast.error('Failed to update registration setting', { description: message });
    },
  });

  return {
    registrationOpen: query.data?.open ?? false,
    isLoading: query.isLoading,
    setRegistrationOpen: mutation.mutateAsync,
    isUpdating: mutation.isPending,
  };
}
