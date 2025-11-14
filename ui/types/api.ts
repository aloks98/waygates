export interface SuccessResponse<T = any> {
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
    details?: any;
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
    email: string;
    password: string;
}

export interface LoginResponseData {
    token: string; // Assuming the API returns a token upon successful login
    // Add any other relevant user data returned by the login API
}
