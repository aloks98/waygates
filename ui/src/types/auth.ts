export interface User {
  id: number;
  name: string;
  username: string;
  email: string;
  role?: string;
  permissions?: string[];
}

export interface AuthState {
  accessToken: string | null;
  refreshToken: string | null;
  user: User | null;
  isAuthenticated: boolean;
}
