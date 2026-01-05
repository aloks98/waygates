import {
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  Checkbox,
} from '@e412/titanium';
import { Loader2, Save } from 'lucide-react';
import { useEffect, useMemo, useState } from 'react';
import { useAuditConfig } from '@/hooks';
import type { AuditConfig } from '@/types/audit';

interface EventGroup {
  key: string;
  label: string;
  description: string;
  events: { key: keyof AuditConfig; label: string }[];
}

const eventGroups: EventGroup[] = [
  {
    key: 'proxy',
    label: 'Proxy Events',
    description: 'Proxy configuration changes',
    events: [
      { key: 'proxy_create', label: 'Create' },
      { key: 'proxy_update', label: 'Update' },
      { key: 'proxy_delete', label: 'Delete' },
      { key: 'proxy_enable', label: 'Enable' },
      { key: 'proxy_disable', label: 'Disable' },
    ],
  },
  {
    key: 'auth',
    label: 'Authentication Events',
    description: 'User authentication activities',
    events: [
      { key: 'auth_login', label: 'Login' },
      { key: 'auth_logout', label: 'Logout' },
      { key: 'auth_register', label: 'Register' },
      { key: 'auth_password_change', label: 'Password Change' },
      { key: 'auth_login_failed', label: 'Failed Login' },
    ],
  },
  {
    key: 'settings',
    label: 'Settings Events',
    description: 'Configuration changes',
    events: [{ key: 'settings_update', label: 'Settings Update' }],
  },
  {
    key: 'sync',
    label: 'Sync Events',
    description: 'Caddy synchronization operations',
    events: [
      { key: 'sync_started', label: 'Started' },
      { key: 'sync_completed', label: 'Completed' },
      { key: 'sync_failed', label: 'Failed' },
    ],
  },
  {
    key: 'system',
    label: 'System Events',
    description: 'System and Caddy operations',
    events: [
      { key: 'system_startup', label: 'System Startup' },
      { key: 'caddy_reload', label: 'Caddy Reload' },
    ],
  },
];

function getGroupCheckedState(
  config: AuditConfig,
  events: { key: keyof AuditConfig }[],
): boolean | 'indeterminate' {
  const enabledCount = events.filter((e) => config[e.key]).length;
  if (enabledCount === 0) return false;
  if (enabledCount === events.length) return true;
  return 'indeterminate';
}

export function AuditConfigPanel() {
  const { config, isLoading, updateConfig, isUpdating } = useAuditConfig();
  const [localConfig, setLocalConfig] = useState<AuditConfig | null>(null);
  const [savedConfig, setSavedConfig] = useState<AuditConfig | null>(null);

  // Initialize local config when server config loads
  useEffect(() => {
    if (config && !savedConfig) {
      setSavedConfig(config);
      setLocalConfig(config);
    }
  }, [config, savedConfig]);

  const hasChanges = useMemo(() => {
    if (!savedConfig || !localConfig) return false;
    return (Object.keys(savedConfig) as (keyof AuditConfig)[]).some(
      (key) => savedConfig[key] !== localConfig[key],
    );
  }, [localConfig, savedConfig]);

  const handleEventToggle = (key: keyof AuditConfig) => {
    if (!localConfig) return;
    setLocalConfig({
      ...localConfig,
      [key]: !localConfig[key],
    });
  };

  const handleGroupToggle = (events: { key: keyof AuditConfig }[]) => {
    if (!localConfig) return;
    const checkedState = getGroupCheckedState(localConfig, events);
    // If all are checked, uncheck all. Otherwise, check all.
    const newValue = checkedState !== true;

    const updates: Partial<AuditConfig> = {};
    for (const event of events) {
      updates[event.key] = newValue;
    }

    setLocalConfig({
      ...localConfig,
      ...updates,
    });
  };

  const handleSave = async () => {
    if (!localConfig) return;
    await updateConfig(localConfig);
    // After successful save, update savedConfig to match
    setSavedConfig(localConfig);
  };

  const handleReset = () => {
    if (savedConfig) {
      setLocalConfig(savedConfig);
    }
  };

  if (isLoading || !localConfig) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Event Logging Configuration</CardTitle>
          <CardDescription>Loading configuration...</CardDescription>
        </CardHeader>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Event Logging Configuration</CardTitle>
        <CardDescription>
          Choose which events to log. Disabled events will not create audit log entries.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        {eventGroups.map((group) => {
          const checkedState = getGroupCheckedState(localConfig, group.events);

          return (
            <div key={group.key} className="space-y-3">
              <div className="flex items-start gap-3">
                <Checkbox
                  checked={checkedState}
                  onCheckedChange={() => handleGroupToggle(group.events)}
                  id={`group-${group.key}`}
                />
                <div className="space-y-1">
                  <label
                    htmlFor={`group-${group.key}`}
                    className="text-sm font-medium cursor-pointer"
                  >
                    {group.label}
                  </label>
                  <p className="text-xs text-muted-foreground">{group.description}</p>
                </div>
              </div>

              <div className="ml-7 flex flex-wrap gap-x-6 gap-y-2">
                {group.events.map((event) => (
                  <div key={event.key} className="flex items-center gap-2">
                    <Checkbox
                      checked={localConfig[event.key]}
                      onCheckedChange={() => handleEventToggle(event.key)}
                      id={`event-${event.key}`}
                    />
                    <label
                      htmlFor={`event-${event.key}`}
                      className="text-sm text-muted-foreground cursor-pointer"
                    >
                      {event.label}
                    </label>
                  </div>
                ))}
              </div>
            </div>
          );
        })}

        <div className="flex items-center justify-end gap-2 pt-4 border-t">
          <Button variant="outline" onClick={handleReset} disabled={isUpdating || !hasChanges}>
            Reset
          </Button>
          <Button onClick={handleSave} disabled={isUpdating || !hasChanges}>
            {isUpdating ? (
              <>
                <Loader2 className="mr-2 size-4 animate-spin" />
                Saving...
              </>
            ) : (
              <>
                <Save className="mr-2 size-4" />
                Save Changes
              </>
            )}
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}
