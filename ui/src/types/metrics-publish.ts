export interface MetricsPublishSettings {
  enabled: boolean;
  host: string;
  path: string;
  basic_auth_user: string;
  has_basic_auth: boolean;
  allowed_cidrs: string[];
}

export interface UpdateMetricsPublishRequest {
  enabled: boolean;
  host: string;
  path: string;
  basic_auth_user: string;
  basic_auth_password: string;
  allowed_cidrs: string[];
}
