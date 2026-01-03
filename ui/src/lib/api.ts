import ky, { type KyInstance, type Options } from 'ky';
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
      async (request, options, response) => {
        if (response.status === 401 && !request.url.includes('/auth/refresh')) {
          const tokens = await handleTokenRefresh();

          if (tokens) {
            const retryOptions: Options = {
              ...options,
              headers: {
                ...Object.fromEntries(request.headers.entries()),
                Authorization: `Bearer ${tokens.access_token}`,
              },
            };
            return ky(request.url, retryOptions);
          }
        }
        return response;
      },
    ],
  },
});

export const publicApi: KyInstance = ky.create({
  prefixUrl: API_BASE_URL,
});
