import type { ACLGroup } from '@/types/acl';

const RULE_TYPE_LABELS: Record<string, string> = {
  allow: 'Allow',
  deny: 'Block',
  bypass: 'Trusted — skip auth',
};

export function getRuleTypeLabel(t: string): string {
  return RULE_TYPE_LABELS[t] ?? t;
}

const MODE_LABELS: Record<string, string> = {
  any: 'Any match',
  all: 'All required',
  ip_bypass: 'Trusted IPs first',
};

export function getModeLabel(m: string): string {
  return MODE_LABELS[m] ?? m;
}

export type AuthMethodKey = 'ip' | 'basic' | 'oauth' | 'account' | 'forward';

export interface AuthMethod {
  key: AuthMethodKey;
  label: string;
}

export const AUTH_METHODS: Record<AuthMethodKey, string> = {
  ip: 'IP',
  basic: 'Basic',
  oauth: 'OAuth',
  account: 'Account',
  forward: 'Forward',
};

// Derive which auth methods a group has configured, from the relations the
// group-list query already preloads. Order is stable (ip → basic → oauth →
// account → forward) so the pills render consistently.
export function groupAuthMethods(group: ACLGroup): AuthMethod[] {
  const out: AuthMethod[] = [];
  const has = (k: AuthMethodKey) => out.push({ key: k, label: AUTH_METHODS[k] });
  if ((group.ip_rules?.length ?? 0) > 0) has('ip');
  if ((group.basic_auth_users?.length ?? 0) > 0) has('basic');
  if ((group.waygates_auth?.allowed_providers?.length ?? 0) > 0) has('oauth');
  if (group.waygates_auth?.enabled === true) has('account');
  if ((group.external_providers?.length ?? 0) > 0) has('forward');
  return out;
}
