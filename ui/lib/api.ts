import { SuccessResponse } from '@/types/api';

const BASE_URL = `${process.env.NEXT_PUBLIC_BACKEND_URL}/api`;

type ApiOptions<T = never> = {
    method?: 'GET' | 'POST' | 'PUT' | 'DELETE';
    headers?: Record<string, string>;
    body?: T;
    authenticate?: boolean; // New option to control authentication
};

export async function api<T = never, B = never>(
    endpoint: string,
    options: ApiOptions<B> = {}
): Promise<SuccessResponse<T>> {
    const { method = 'GET', headers = {}, body, authenticate = true } = options;

    const config: RequestInit = {
        method,
        headers: {
            'Content-Type': 'application/json',
            ...headers,
        },
    };

    if (authenticate) {
        const authToken = localStorage.getItem('authToken');
        if (authToken) {
            config.headers = {
                ...config.headers,
                'Authorization': `Bearer ${authToken}`,
            };
        }
    }

    if (body) {
        config.body = JSON.stringify(body);
    }

    try {
        const response = await fetch(`${BASE_URL}${endpoint}`, config);
        const json = await response.json();

        if (!response.ok) {
            // Handle non-2xx responses
            console.error('API Error:', json);
            return Promise.reject(json);
        }

        return json;
    } catch (error) {
        console.error('Network Error:', error);
        return Promise.reject(error);
    }
}

export function logout() {
    localStorage.removeItem('authToken');
    // Optionally, redirect to login page or home page
    // window.location.href = '/login';
}
