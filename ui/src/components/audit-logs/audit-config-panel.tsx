import {
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  Switch,
} from '@e412/titanium';
import { Loader2, Save } from 'lucide-react';
import { useEffect, useState } from 'react';
import { useAuditConfig } from '@/hooks';
import type { AuditConfig } from '@/types/audit';

const eventCategories = [
  {
    key: 'proxy_events' as const,
    label: 'Proxy Events',
    description: 'Create, update, delete, enable, and disable proxy operations',
  },
  {
    key: 'auth_events' as const,
    label: 'Authentication Events',
    description: 'Login, logout, registration, password changes, and failed login attempts',
  },
  {
    key: 'settings_events' as const,
    label: 'Settings Events',
    description: 'Configuration changes and setting updates',
  },
  {
    key: 'sync_events' as const,
    label: 'Sync Events',
    description: 'Caddy synchronization operations (start, complete, fail)',
  },
  {
    key: 'system_events' as const,
    label: 'System Events',
    description: 'System startup and Caddy reload events',
  },
];

export function AuditConfigPanel() {
  const { config, isLoading, updateConfig, isUpdating } = useAuditConfig();
  const [localConfig, setLocalConfig] = useState<AuditConfig | null>(null);
  const [hasChanges, setHasChanges] = useState(false);

  useEffect(() => {
    if (config && !localConfig) {
      setLocalConfig(config);
    }
  }, [config, localConfig]);

  useEffect(() => {
    if (config && localConfig) {
      const changed = Object.keys(config).some(
        (key) => config[key as keyof AuditConfig] !== localConfig[key as keyof AuditConfig],
      );
      setHasChanges(changed);
    }
  }, [config, localConfig]);

  const handleToggle = (key: keyof AuditConfig) => {
    if (!localConfig) return;
    setLocalConfig({
      ...localConfig,
      [key]: !localConfig[key],
    });
  };

  const handleSave = async () => {
    if (!localConfig) return;
    await updateConfig(localConfig);
    setHasChanges(false);
  };

  const handleReset = () => {
    if (config) {
      setLocalConfig(config);
      setHasChanges(false);
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
          Choose which events to log. Disabled categories will not create audit log entries.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {eventCategories.map((category) => (
          <div
            key={category.key}
            className="flex items-center justify-between rounded-lg border p-4"
          >
            <div className="space-y-0.5">
              <div className="font-medium">{category.label}</div>
              <div className="text-sm text-muted-foreground">{category.description}</div>
            </div>
            <Switch
              checked={localConfig[category.key]}
              onCheckedChange={() => handleToggle(category.key)}
            />
          </div>
        ))}

        {hasChanges && (
          <div className="flex items-center justify-end gap-2 pt-4 border-t">
            <Button variant="outline" onClick={handleReset} disabled={isUpdating}>
              Reset
            </Button>
            <Button onClick={handleSave} disabled={isUpdating}>
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
        )}
      </CardContent>
    </Card>
  );
}
