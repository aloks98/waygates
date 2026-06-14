import {
  Alert,
  AlertDescription,
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  Button,
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardHeading,
  CardTitle,
  Checkbox,
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
  Skeleton,
  Spinner,
} from '@e412/titanium';
import { ChevronRight, ShieldAlert, TriangleAlert } from 'lucide-react';
import { useCallback, useEffect, useMemo, useState } from 'react';

import { useAuditConfig, useAuditEventGroups } from '@/hooks';
import type { AuditConfig, AuditEventDefinition, AuditEventGroup } from '@/types/audit';

// Event groups that contain security-critical audit events
const SECURITY_CRITICAL_GROUPS = new Set(['auth', 'acl']);

// Individual events that should always be flagged as security-critical
const SECURITY_CRITICAL_EVENTS = new Set<keyof AuditConfig>([
  'auth_login',
  'auth_logout',
  'auth_login_failed',
  'auth_password_change',
  'auth_register',
  'acl_session_revoke',
  'acl_basic_auth_add',
  'acl_basic_auth_update',
  'acl_basic_auth_delete',
  'acl_waygates_auth_update',
  'acl_oauth_restriction_set',
  'acl_oauth_restriction_delete',
]);

function isSecurityGroup(group: AuditEventGroup): boolean {
  return SECURITY_CRITICAL_GROUPS.has(group.key);
}

function isSecurityEvent(key: keyof AuditConfig): boolean {
  return SECURITY_CRITICAL_EVENTS.has(key);
}

function getGroupCheckedState(
  config: AuditConfig,
  events: AuditEventDefinition[],
): boolean | 'indeterminate' {
  const enabledCount = events.filter((e) => config[e.key]).length;
  if (enabledCount === 0) return false;
  if (enabledCount === events.length) return true;
  return 'indeterminate';
}

function getGroupEnabledCount(config: AuditConfig, events: AuditEventDefinition[]): number {
  return events.filter((e) => config[e.key]).length;
}

function getTotalEnabledCount(config: AuditConfig, groups: AuditEventGroup[]): number {
  return groups.reduce((sum, g) => sum + getGroupEnabledCount(config, g.events), 0);
}

function getTotalEventCount(groups: AuditEventGroup[]): number {
  return groups.reduce((sum, g) => sum + g.events.length, 0);
}

export function AuditConfigPanel() {
  const { config, isLoading, isError, updateConfig, isUpdating } = useAuditConfig();
  const { eventGroups, isLoading: isLoadingEventGroups } = useAuditEventGroups();
  const [localConfig, setLocalConfig] = useState<AuditConfig | null>(null);
  const [savedConfig, setSavedConfig] = useState<AuditConfig | null>(null);
  const [showConfirm, setShowConfirm] = useState(false);
  const [openGroups, setOpenGroups] = useState<Set<string>>(new Set());

  // Initialize local config when server config loads
  useEffect(() => {
    if (config && !savedConfig) {
      setSavedConfig(config);
      setLocalConfig(config);
    }
  }, [config, savedConfig]);

  // Track which individual keys have changed from the saved state
  const changedKeys = useMemo(() => {
    if (!savedConfig || !localConfig) return new Set<keyof AuditConfig>();
    const keys = new Set<keyof AuditConfig>();
    for (const key of Object.keys(savedConfig) as (keyof AuditConfig)[]) {
      if (savedConfig[key] !== localConfig[key]) keys.add(key);
    }
    return keys;
  }, [savedConfig, localConfig]);

  const hasChanges = changedKeys.size > 0;

  // Detect security events being newly disabled
  const newlyDisabledSecurityEvents = useMemo(() => {
    if (!savedConfig || !localConfig) return [];
    return (Object.keys(savedConfig) as (keyof AuditConfig)[]).filter(
      (key) => isSecurityEvent(key) && savedConfig[key] && !localConfig[key],
    );
  }, [savedConfig, localConfig]);

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

  const executeSave = async () => {
    if (!localConfig) return;
    await updateConfig(localConfig);
    setSavedConfig(localConfig);
  };

  const handleSave = () => {
    if (!localConfig) return;
    if (newlyDisabledSecurityEvents.length > 0) {
      setShowConfirm(true);
    } else {
      executeSave();
    }
  };

  const handleConfirmSave = async () => {
    setShowConfirm(false);
    await executeSave();
  };

  const handleReset = () => {
    if (savedConfig) {
      setLocalConfig(savedConfig);
    }
  };

  const handleMasterToggle = useCallback(() => {
    if (!localConfig) return;
    const totalEnabled = getTotalEnabledCount(localConfig, eventGroups);
    const totalEvents = getTotalEventCount(eventGroups);
    const enableAll = totalEnabled < totalEvents;

    const updates: Partial<AuditConfig> = {};
    for (const group of eventGroups) {
      for (const event of group.events) {
        updates[event.key] = enableAll;
      }
    }
    setLocalConfig({ ...localConfig, ...updates });
  }, [localConfig, eventGroups]);

  const toggleGroupOpen = useCallback((groupKey: string) => {
    setOpenGroups((prev) => {
      const next = new Set(prev);
      if (next.has(groupKey)) {
        next.delete(groupKey);
      } else {
        next.add(groupKey);
      }
      return next;
    });
  }, []);

  const masterCheckedState = useMemo(() => {
    if (!localConfig) return false;
    const total = getTotalEventCount(eventGroups);
    const enabled = getTotalEnabledCount(localConfig, eventGroups);
    if (enabled === 0) return false;
    if (enabled === total) return true;
    return 'indeterminate' as const;
  }, [localConfig, eventGroups]);

  if (isError) {
    return (
      <Card>
        <CardHeader>
          <CardHeading>
            <CardTitle>Event Logging Configuration</CardTitle>
            <CardDescription>
              Choose which events to log. Disabled events will not create audit log entries.
            </CardDescription>
          </CardHeading>
        </CardHeader>
        <CardContent>
          <Alert variant="destructive">
            <AlertDescription>
              Failed to load audit configuration. Please try refreshing the page.
            </AlertDescription>
          </Alert>
        </CardContent>
      </Card>
    );
  }

  if (isLoading || isLoadingEventGroups || !localConfig) {
    return (
      <Card>
        <CardHeader>
          <Skeleton className="h-6 w-64 max-w-full" />
          <Skeleton className="h-4 w-96 max-w-full" />
        </CardHeader>
        <CardContent className="space-y-4">
          <Skeleton className="h-8 w-full" />
          <div className="space-y-2">
            <Skeleton className="h-16 w-full rounded-md" />
            <Skeleton className="h-16 w-full rounded-md" />
            <Skeleton className="h-16 w-full rounded-md" />
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <>
      <Card>
        <CardHeader>
          <CardHeading>
            <CardTitle>Event Logging Configuration</CardTitle>
            <CardDescription>
              Choose which events to log. Disabled events will not create audit log entries.
            </CardDescription>
          </CardHeading>
        </CardHeader>
        <CardContent className="space-y-4">
          {newlyDisabledSecurityEvents.length > 0 && (
            <Alert
              variant="destructive"
              className="border-amber-500/50 bg-amber-500/5 text-amber-700 dark:text-amber-400 [&>svg]:text-amber-600 dark:[&>svg]:text-amber-400"
            >
              <TriangleAlert className="size-4" />
              <AlertDescription>
                You are about to disable {newlyDisabledSecurityEvents.length} security-related{' '}
                {newlyDisabledSecurityEvents.length === 1 ? 'event' : 'events'}. This may impact
                compliance and audit trail completeness.
              </AlertDescription>
            </Alert>
          )}

          {/* Master toggle */}
          <div className="flex items-center justify-between border-b pb-3">
            <div className="flex items-center gap-3">
              <Checkbox
                checked={masterCheckedState}
                onCheckedChange={handleMasterToggle}
                id="master-toggle"
              />
              <label htmlFor="master-toggle" className="text-sm font-medium cursor-pointer">
                {masterCheckedState === true ? 'Disable all events' : 'Enable all events'}
              </label>
            </div>
            <span className="text-xs text-muted-foreground tabular-nums">
              {getTotalEnabledCount(localConfig, eventGroups)}/{getTotalEventCount(eventGroups)}{' '}
              enabled
            </span>
          </div>

          {/* Event groups */}
          <div className="space-y-2">
            {eventGroups.map((group) => {
              const checkedState = getGroupCheckedState(localConfig, group.events);
              const isSecurity = isSecurityGroup(group);
              const enabledCount = getGroupEnabledCount(localConfig, group.events);
              const isOpen = openGroups.has(group.key);

              return (
                <Collapsible
                  key={group.key}
                  open={isOpen}
                  onOpenChange={() => toggleGroupOpen(group.key)}
                >
                  <div
                    className={
                      isSecurity
                        ? 'rounded-md border border-amber-500/30 bg-amber-500/5 p-3'
                        : 'rounded-md border p-3'
                    }
                  >
                    <div className="flex items-center gap-3">
                      <Checkbox
                        className="shrink-0"
                        checked={checkedState}
                        onCheckedChange={() => handleGroupToggle(group.events)}
                        id={`group-${group.key}`}
                      />
                      <CollapsibleTrigger asChild>
                        <button
                          type="button"
                          className="flex flex-1 items-center gap-2 text-left cursor-pointer"
                        >
                          <ChevronRight
                            className={`size-4 shrink-0 text-muted-foreground transition-transform duration-200 ${isOpen ? 'rotate-90' : ''}`}
                          />
                          <div className="flex-1 min-w-0">
                            <div className="flex items-center gap-2">
                              <span className="text-sm font-medium">{group.label}</span>
                              {isSecurity && (
                                <span className="inline-flex items-center gap-1 rounded-full bg-amber-500/10 px-2 py-0.5 text-xs font-medium text-amber-700 dark:text-amber-400">
                                  <ShieldAlert className="size-3" />
                                  Security
                                </span>
                              )}
                            </div>
                            <p className="text-xs text-muted-foreground">{group.description}</p>
                          </div>
                          <span className="text-xs text-muted-foreground tabular-nums shrink-0">
                            {enabledCount}/{group.events.length}
                          </span>
                        </button>
                      </CollapsibleTrigger>
                    </div>

                    <CollapsibleContent>
                      <div className="ml-10 mt-3 flex flex-wrap gap-x-6 gap-y-2 border-t pt-3">
                        {group.events.map((event) => {
                          const isChanged = changedKeys.has(event.key);
                          return (
                            <div
                              key={event.key}
                              className={`flex items-center gap-2 rounded px-1.5 py-0.5 transition-colors duration-200 ${isChanged ? 'bg-primary/10' : ''}`}
                            >
                              <Checkbox
                                checked={localConfig[event.key]}
                                onCheckedChange={() => handleEventToggle(event.key)}
                                id={`event-${event.key}`}
                              />
                              <label
                                htmlFor={`event-${event.key}`}
                                className={`text-sm cursor-pointer ${isChanged ? 'text-foreground font-medium' : 'text-muted-foreground'}`}
                              >
                                {event.label}
                              </label>
                            </div>
                          );
                        })}
                      </div>
                    </CollapsibleContent>
                  </div>
                </Collapsible>
              );
            })}
          </div>
        </CardContent>
        <CardFooter className="flex items-center justify-between border-t">
          <span className="text-xs text-muted-foreground">
            {changedKeys.size > 0
              ? `${changedKeys.size} unsaved ${changedKeys.size === 1 ? 'change' : 'changes'}`
              : ''}
          </span>
          <div className="flex gap-2">
            <Button variant="outline" onClick={handleReset} disabled={isUpdating || !hasChanges}>
              Discard Changes
            </Button>
            <Button onClick={handleSave} disabled={isUpdating || !hasChanges}>
              {isUpdating ? (
                <>
                  <Spinner variant="circle" />
                  Saving...
                </>
              ) : (
                'Save Changes'
              )}
            </Button>
          </div>
        </CardFooter>
      </Card>

      <AlertDialog open={showConfirm} onOpenChange={setShowConfirm}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Disable Security Event Logging?</AlertDialogTitle>
            <AlertDialogDescription asChild>
              <div className="space-y-3">
                <p>
                  You are about to disable logging for{' '}
                  <strong>
                    {newlyDisabledSecurityEvents.length} security-related{' '}
                    {newlyDisabledSecurityEvents.length === 1 ? 'event' : 'events'}
                  </strong>
                  . These events help detect unauthorized access, brute-force attacks, and
                  credential misuse.
                </p>
                <ul className="list-disc pl-5 text-sm space-y-1">
                  {newlyDisabledSecurityEvents.map((key) => (
                    <li key={key}>{key.replaceAll('_', ' ')}</li>
                  ))}
                </ul>
                <p className="text-sm font-medium text-amber-700 dark:text-amber-400">
                  Disabling these events may impact compliance requirements and reduce your ability
                  to investigate security incidents.
                </p>
              </div>
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleConfirmSave}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              Disable & Save
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}
