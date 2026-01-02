export interface SuccessResponse<T = never> {
    success: boolean;
    message?: string;
    data?: T;
}

export interface ErrorResponse {
    success: boolean;
    error: ErrorDetail;
}

export interface ErrorDetail {
    code: string;
    message: string;
    details?: never;
}

export interface StatusResponseData {
    caddy_status: string;
    user_setup_complete: boolean;
}

export interface RegisterRequest {
    name: string;
    username: string;
    email: string;
    password: string;
}

export interface LoginRequest {
    identifier: string;
    password: string;
}

export interface LoginResponseData {
    access_token: string;
    refresh_token: string;
}

export interface User {
    id: number;
    name: string;
    username: string;
    email: string;
    created_at: string;
    updated_at: string;
}

export interface RegisterResponseData {
    user: User;
}

export interface RefreshTokenRequest {
    refresh_token: string;
}

export interface RefreshTokenResponse {
    access_token: string;
}
