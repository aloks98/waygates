export interface ApiResponse<T> {
  success: boolean;
  message?: string;
  data?: T;
  error?: string;
  details?: Record<string, string>;
}

export interface PaginatedResponse<T> {
  items: T[];
  total: number;
  page: number;
  limit: number;
  total_pages: number;
}

export interface LoginRequest {
  identifier: string;
  password: string;
}

export interface RegisterRequest {
  name: string;
  username: string;
  email: string;
  password: string;
}

export interface TokenPair {
  access_token: string;
  refresh_token: string;
}

export interface RefreshTokenRequest {
  refresh_token: string;
}
