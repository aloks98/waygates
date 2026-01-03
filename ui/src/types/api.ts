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

export interface ProxyStats {
  total: number;
  active: number;
  inactive: number;
  by_type: Record<string, number>;
}

export interface TokenPair {
  access_token: string;
  refresh_token: string;
}
