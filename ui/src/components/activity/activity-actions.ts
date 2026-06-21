import {
  Ban,
  CheckCircle2,
  Edit,
  KeyRound,
  LogIn,
  type LucideIcon,
  Plus,
  Power,
  PowerOff,
  RefreshCw,
  Server,
  Settings as SettingsIcon,
  Shield,
  Trash2,
} from 'lucide-react';

import { getActionLabel } from '@/lib/dashboard-format';

export type ActionTone =
  | 'create'
  | 'update'
  | 'delete'
  | 'auth'
  | 'sync'
  | 'system'
  | 'acl'
  | 'neutral';

export interface ActionMeta {
  label: string;
  icon: LucideIcon;
  tone: ActionTone;
}

// Icon per action. Unknown actions fall back (in getActionMeta) to Shield for
// acl.* actions and Edit for everything else.
const ICONS: Record<string, LucideIcon> = {
  'proxy.create': Plus,
  'proxy.update': Edit,
  'proxy.delete': Trash2,
  'proxy.enable': Power,
  'proxy.disable': PowerOff,
  'auth.login': LogIn,
  'auth.logout': LogIn,
  'auth.register': Plus,
  'auth.password_change': KeyRound,
  'auth.login_failed': Ban,
  'settings.update': SettingsIcon,
  'sync.started': RefreshCw,
  'sync.completed': CheckCircle2,
  'sync.failed': Ban,
  'system.startup': Server,
  'caddy.reload': RefreshCw,
};

function deriveTone(action: string): ActionTone {
  if (action.startsWith('acl')) return 'acl';
  if (action.startsWith('auth')) return 'auth';
  if (action.startsWith('sync')) return 'sync';
  if (action === 'system.startup' || action === 'caddy.reload') return 'system';
  if (action.includes('create') || action.includes('.add') || action.includes('enable'))
    return 'create';
  if (action.includes('delete') || action.includes('disable') || action.includes('revoke'))
    return 'delete';
  if (action.includes('update') || action.includes('.set')) return 'update';
  return 'neutral';
}

const TONE_CLASS: Record<ActionTone, string> = {
  create: 'text-green-600 dark:text-green-500',
  update: 'text-blue-600 dark:text-blue-500',
  delete: 'text-destructive',
  auth: 'text-violet-600 dark:text-violet-400',
  sync: 'text-amber-600 dark:text-amber-500',
  system: 'text-muted-foreground',
  acl: 'text-cyan-600 dark:text-cyan-400',
  neutral: 'text-muted-foreground',
};

export function getActionMeta(action: string): ActionMeta {
  // Default ACL icon is Shield; everything else falls back to a generic edit glyph.
  const fallback = action.startsWith('acl') ? Shield : Edit;
  return {
    label: getActionLabel(action),
    icon: ICONS[action] ?? fallback,
    tone: deriveTone(action),
  };
}

export function toneTextClass(tone: ActionTone): string {
  return TONE_CLASS[tone];
}
