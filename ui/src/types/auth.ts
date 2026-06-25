export interface User {
  id: number;
  name: string;
  username: string;
  email: string;
  role?: string;
  permissions?: string[];
  must_change_password?: boolean;
}
