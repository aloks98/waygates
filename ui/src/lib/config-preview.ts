import { HTTPError } from 'ky';

import type { ProxyACLAssignment } from '@/types/acl';
import type { ApiResponse } from '@/types/api';
import type { CaddyConfig } from '@/types/caddy-config';
import type { CreateL4ProxyRequest } from '@/types/l4-proxy';
import type { CreateProxyRequest } from '@/types/proxy';

import { api } from './api';

export type ConfigPreviewRequest =
  | (CreateProxyRequest & { ssl_forced: boolean })
  | CreateL4ProxyRequest;

class ConfigPreviewUnavailableError extends Error {}

async function loadSavedL4Config(request: CreateL4ProxyRequest, signal: AbortSignal) {
  if (!request.is_active) {
    throw new ConfigPreviewUnavailableError(
      'This proxy is inactive and has no generated config. Update the Waygates backend to preview inactive proxies.',
    );
  }

  // Older backends expose the generated config but do not have the draft endpoint.
  // Read the actual server entry; never substitute saved config for an unsaved draft.
  const response = await api.get('caddy-config', { signal }).json<
    ApiResponse<{
      apps?: { layer4?: { servers?: Record<string, CaddyConfig> }; tls?: unknown };
    }>
  >();
  const apps = response.data?.apps;
  const serverName = `${request.name}_${request.protocol}_${request.listen_port}`;
  const server = apps?.layer4?.servers?.[serverName];
  if (!server) {
    throw new ConfigPreviewUnavailableError(
      'This proxy is missing from the generated config. Check its routes and the system sync status.',
    );
  }
  return {
    apps: {
      layer4: { servers: { [serverName]: server } },
      // TLS termination can reference shared certificate and automation settings.
      ...(request.routes?.some((route) => route.tls_terminate) && apps?.tls
        ? { tls: apps.tls }
        : {}),
    },
  };
}

export async function loadConfigPreview({
  request,
  assignments,
  saved = false,
  signal,
}: {
  request: ConfigPreviewRequest;
  assignments: Pick<ProxyACLAssignment, 'acl_group_id' | 'path_pattern' | 'priority' | 'enabled'>[];
  saved?: boolean;
  signal: AbortSignal;
}): Promise<CaddyConfig> {
  try {
    const response = await api
      .post('caddy-config/preview', {
        json: 'protocol' in request ? { l4_proxy: request } : { proxy: request, assignments },
        signal,
      })
      .json<ApiResponse<CaddyConfig>>();
    if (!response.success || !response.data) throw new Error('Missing config preview');
    return response.data;
  } catch (error) {
    if (error instanceof HTTPError && [404, 405].includes(error.response.status)) {
      if (saved && 'protocol' in request) return loadSavedL4Config(request, signal);
      throw new ConfigPreviewUnavailableError(
        'This server does not support draft config previews yet. Update the Waygates backend to enable them.',
      );
    }
    throw error;
  }
}

export function configPreviewErrorMessage(error: unknown, isL4: boolean) {
  if (error instanceof ConfigPreviewUnavailableError) return error.message;
  if (error instanceof HTTPError && error.response.status === 403) {
    return 'You do not have permission to view generated config.';
  }
  if (error instanceof HTTPError && error.response.status === 400) {
    return isL4
      ? 'Couldn’t generate the TCP/UDP config. Check the listener and routing settings.'
      : 'Couldn’t generate the config. Check the proxy and access rules.';
  }
  return 'Couldn’t load the generated config. Check the server connection and retry.';
}
