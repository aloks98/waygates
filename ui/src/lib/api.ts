import ky, { type KyInstance } from 'ky';

import { useAuthStore } from '../stores/auth';
import type { ApiResponse, TokenPair } from '../types/api';

const API_BASE_URL = '/api';

let isRefreshing = false;
let refreshPromise: Promise<TokenPair | null> | null = null;

async function refreshTokens(): Promise<TokenPair | null> {
  const { refreshToken, setTokens, logout } = useAuthStore.getState();

  if (!refreshToken) {
    logout();
    return null;
  }

  try {
    const response = await ky
      .post(`${API_BASE_URL}/auth/refresh`, {
        json: { refresh_token: refreshToken },
      })
      .json<ApiResponse<TokenPair>>();

    if (response.success && response.data) {
      setTokens(response.data);
      return response.data;
    }

    logout();
    return null;
  } catch {
    logout();
    return null;
  }
}

async function handleTokenRefresh(): Promise<TokenPair | null> {
  if (isRefreshing) {
    return refreshPromise;
  }

  isRefreshing = true;
  refreshPromise = refreshTokens().finally(() => {
    isRefreshing = false;
    refreshPromise = null;
  });

  return refreshPromise;
}

export const api: KyInstance = ky.create({
  prefixUrl: API_BASE_URL,
  hooks: {
    beforeRequest: [
      (request) => {
        const { accessToken } = useAuthStore.getState();
        if (accessToken) {
          request.headers.set('Authorization', `Bearer ${accessToken}`);
        }
      },
    ],
    afterResponse: [
      async (request, _options, response) => {
        // Refresh-and-retry on 401, EXCEPT for the credential/refresh endpoints
        // themselves (those must not loop). Authenticated /auth/* calls such as
        // /auth/me DO refresh-and-retry on an expired access token — otherwise the
        // user (and their permissions) never load on a stale-token page load.
        const skipRefresh = ['/auth/login', '/auth/register', '/auth/refresh'].some((p) =>
          request.url.includes(p),
        );
        if (response.status === 401 && !skipRefresh) {
          const tokens = await handleTokenRefresh();

          if (tokens) {
            // Use native fetch for retry to avoid ky's prefixUrl handling
            // request.url is already the full absolute URL
            const headers = new Headers(request.headers);
            headers.set('Authorization', `Bearer ${tokens.access_token}`);

            return fetch(request.url, {
              method: request.method,
              headers,
              body: request.body,
              credentials: request.credentials,
            });
          }

          // Refresh failed - ensure user is logged out
          const { logout } = useAuthStore.getState();
          logout();
        }
        return response;
      },
    ],
  },
});

export const publicApi: KyInstance = ky.create({
  prefixUrl: API_BASE_URL,
});
