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
        // Skip refresh logic for auth endpoints to prevent loops
        if (response.status === 401 && !request.url.includes('/auth/')) {
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
