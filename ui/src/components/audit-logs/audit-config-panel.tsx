import {
  Button,
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
  Checkbox,
  Spinner,
} from '@e412/titanium';
import { useEffect, useMemo, useState } from 'react';
import { useAuditConfig, useAuditEventGroups } from '@/hooks';
import type { AuditConfig, AuditEventDefinition } from '@/types/audit';

function getGroupCheckedState(
  config: AuditConfig,
  events: AuditEventDefinition[],
): boolean | 'indeterminate' {
  const enabledCount = events.filter((e) => config[e.key]).length;
  if (enabledCount === 0) return false;
  if (enabledCount === events.length) return true;
  return 'indeterminate';
}

export function AuditConfigPanel() {
  const { config, isLoading, updateConfig, isUpdating } = useAuditConfig();
  const { eventGroups, isLoading: isLoadingEventGroups } = useAuditEventGroups();
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

  const handleGroupToggle = (events: AuditEventDefinition[]) => {
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

  if (isLoading || isLoadingEventGroups || !localConfig) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Event Logging Configuration</CardTitle>
          <CardDescription>
            Choose which events to log. Disabled events will not create audit log entries.
          </CardDescription>
        </CardHeader>
        <CardContent className="flex justify-center items-center min-h-64">
          <div className="flex items-center gap-2">
            <Spinner variant={"circle"} />
            Loading...
          </div>
        </CardContent>
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
                  className="mt-1"
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

      </CardContent>
      <CardFooter className="flex justify-end gap-2">
        <Button variant="outline" onClick={handleReset} disabled={isUpdating || !hasChanges}>
          Reset
        </Button>
        <Button onClick={handleSave} disabled={isUpdating || !hasChanges}>
          {isUpdating ? (
            <>
              <Spinner variant="circle" />
              Saving...
            </>
          ) : (
            <>
              Save Changes
            </>
          )}
        </Button>
      </CardFooter>
    </Card>
  );
}
