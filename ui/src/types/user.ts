export type Role = 'admin' | 'operator' | 'viewer';

export interface ManagedUser {
  id: number;
  name: string;
  username: string;
  email: string;
  role: Role;
  active: boolean;
  must_change_password: boolean;
  last_login_at: string | null;
  created_at: string;
}

export interface CreateUserRequest {
  name: string;
  username: string;
  email: string;
  role: Role;
  password: string;
  must_change_password: boolean;
}

export interface UpdateUserRequest {
  name: string;
  email: string;
  role: Role;
  active: boolean;
}

export interface ResetPasswordRequest {
  password: string;
  must_change_password: boolean;
}
