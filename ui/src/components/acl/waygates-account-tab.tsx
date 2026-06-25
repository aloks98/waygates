import {
  Alert,
  AlertDescription,
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  Checkbox,
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  Input,
  Label,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Skeleton,
  Switch,
} from '@e412/rnui-react';
import { zodResolver } from '@hookform/resolvers/zod';
import { AlertCircle, Clock, Mail, Save, Shield, ShieldCheck, Users } from 'lucide-react';
import { useEffect, useState } from 'react';
import { useForm } from 'react-hook-form';
import { z } from 'zod';

import {
  SESSION_TTL_MAX,
  SESSION_TTL_MIN,
  durationToSeconds,
  secondsToDuration,
} from '@/components/acl/session-duration';
import type { DurationUnit } from '@/components/acl/session-duration';
import { TagsInput } from '@/components/ui/tags-input';
import { useConfigureWaygatesAuth, useWaygatesAuth } from '@/hooks';
import { usePermissions } from '@/hooks/use-permissions';

// Email pattern validation - allows wildcards like *@domain.com or specific emails
const EMAIL_PATTERN_REGEX = /^(\*|[^\s@]+)@[^\s@]+\.[^\s@]+$/;

// Known roles from backend RBAC configuration (see backend/rbac.yaml)
const KNOWN_ROLES = [
  { value: 'admin', label: 'Administrator', description: 'Full access to all features' },
  { value: 'operator', label: 'Operator', description: 'Manage proxies & settings' },
  { value: 'viewer', label: 'Viewer', description: 'Read-only access' },
];

const emailPatternTagsValidation = {
  pattern: EMAIL_PATTERN_REGEX,
};

const waygatesAccountSchema = z.object({
  enabled: z.boolean(),
  allowed_users: z.array(z.string()).optional(),
  allowed_roles: z.array(z.string()).optional(),
  allowed_email_patterns: z.array(z.string()).optional(),
  require_2fa: z.boolean(),
  session_ttl: z
    .number()
    .min(SESSION_TTL_MIN, 'Minimum session TTL is 60 seconds')
    .max(SESSION_TTL_MAX, 'Maximum session TTL is 7 days'),
});

type WaygatesAccountFormValues = z.infer<typeof waygatesAccountSchema>;

export function WaygatesAccountTab({ groupId }: { groupId: number }) {
  const { config, isLoading } = useWaygatesAuth(groupId);
  const { configureAuth, isConfiguring } = useConfigureWaygatesAuth();
  const { canUpdateAccess } = usePermissions();

  // Duration picker unit state — initialised from config once loaded
  const [durationUnit, setDurationUnit] = useState<DurationUnit>('hours');

  const form = useForm<WaygatesAccountFormValues>({
    resolver: zodResolver(waygatesAccountSchema),
    defaultValues: {
      enabled: false,
      allowed_users: [] as string[],
      allowed_roles: [] as string[],
      allowed_email_patterns: [] as string[],
      require_2fa: false,
      session_ttl: 3600,
    },
  });

  useEffect(() => {
    if (config) {
      form.reset({
        enabled: config.enabled,
        allowed_users: config.allowed_users || [],
        allowed_roles: config.allowed_roles || [],
        allowed_email_patterns: config.allowed_email_patterns || [],
        require_2fa: config.require_2fa,
        session_ttl: config.session_ttl,
      });
      // Initialise the unit from the stored seconds value
      setDurationUnit(secondsToDuration(config.session_ttl).unit);
    }
  }, [config, form]);

  const onSubmit = async (value: WaygatesAccountFormValues) => {
    await configureAuth({
      groupId,
      data: {
        enabled: value.enabled,
        allowed_users: value.allowed_users?.length ? value.allowed_users : undefined,
        allowed_roles: value.allowed_roles?.length ? value.allowed_roles : undefined,
        allowed_email_patterns: value.allowed_email_patterns?.length
          ? value.allowed_email_patterns
          : undefined,
        require_2fa: value.require_2fa,
        session_ttl: value.session_ttl,
        // preserve the OAuth slice owned by the other tab
        allowed_providers: config?.allowed_providers?.length ? config.allowed_providers : undefined,
      },
    });
  };

  const enabled = form.watch('enabled');

  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <Skeleton className="h-6 w-48" />
          <Skeleton className="h-4 w-full mt-2" />
        </CardHeader>
        <CardContent className="space-y-4">
          <Skeleton className="h-12 w-full" />
          <Skeleton className="h-12 w-full" />
        </CardContent>
      </Card>
    );
  }

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Users className="size-5" />
              Waygates Authentication
            </CardTitle>
            <CardDescription>
              Allow users with Waygates accounts to authenticate using their platform credentials.
              This is separate from OAuth - users need an account in Waygates to use this method.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-6">
            <FormField
              control={form.control}
              name="enabled"
              render={({ field }) => (
                <FormItem className="flex flex-row items-center justify-between">
                  <div className="space-y-0.5">
                    <FormLabel className="flex items-center gap-2">
                      {field.value ? (
                        <ShieldCheck className="size-4 text-green-500" />
                      ) : (
                        <Shield className="size-4 text-muted-foreground" />
                      )}
                      Enable Waygates Authentication
                    </FormLabel>
                    <FormDescription>
                      Allow users to sign in with their Waygates account credentials
                    </FormDescription>
                  </div>
                  <FormControl>
                    <Switch checked={field.value} onCheckedChange={field.onChange} />
                  </FormControl>
                </FormItem>
              )}
            />

            {enabled && (
              <div className="space-y-6">
                <Alert>
                  <AlertCircle className="size-4" />
                  <AlertDescription>
                    Leave all restriction fields empty to allow any authenticated Waygates user. Add
                    restrictions to limit access to specific users, roles, or email patterns.
                  </AlertDescription>
                </Alert>

                <FormField
                  control={form.control}
                  name="allowed_roles"
                  render={({ field }) => {
                    const selectedRoles: string[] = field.value ?? [];
                    const toggle = (roleValue: string, checked: boolean) => {
                      field.onChange(
                        checked
                          ? [...selectedRoles, roleValue]
                          : selectedRoles.filter((r) => r !== roleValue),
                      );
                    };

                    return (
                      <FormItem>
                        <FormLabel className="flex items-center gap-2">
                          <Users className="size-4" />
                          Allowed Roles
                        </FormLabel>
                        <FormDescription className="mb-3">
                          Select which roles can access this resource
                        </FormDescription>
                        <div className="space-y-2">
                          {KNOWN_ROLES.map((role) => (
                            <div
                              key={role.value}
                              className={`flex items-start gap-3 p-3 rounded-lg border ${
                                selectedRoles.includes(role.value)
                                  ? 'border-primary bg-primary/5'
                                  : 'border-border bg-muted/30'
                              }`}
                            >
                              <Checkbox
                                id={`role-${role.value}`}
                                checked={selectedRoles.includes(role.value)}
                                onCheckedChange={(checked) =>
                                  toggle(role.value, checked as boolean)
                                }
                              />
                              <div className="flex-1">
                                <Label
                                  htmlFor={`role-${role.value}`}
                                  className="text-sm font-medium cursor-pointer"
                                >
                                  {role.label}
                                </Label>
                                <p className="text-xs text-muted-foreground mt-0.5">
                                  {role.description}
                                </p>
                              </div>
                            </div>
                          ))}
                        </div>
                        <FormMessage />
                      </FormItem>
                    );
                  }}
                />

                <FormField
                  control={form.control}
                  name="allowed_email_patterns"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel className="flex items-center gap-2">
                        <Mail className="size-4" />
                        Allowed Email Patterns
                      </FormLabel>
                      <FormControl>
                        <TagsInput
                          value={field.value ?? []}
                          onValueChange={field.onChange}
                          placeholder="Add pattern (e.g., *@company.com)..."
                          delimiters={['Enter', ',']}
                          validation={emailPatternTagsValidation}
                        />
                      </FormControl>
                      <FormDescription>
                        Email patterns. Use * as wildcard. Press Enter or comma to add.
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                {/* TODO: 2FA support coming in future release */}

                <FormField
                  control={form.control}
                  name="session_ttl"
                  render={({ field }) => {
                    const duration = secondsToDuration(field.value);

                    const handleValueChange = (e: React.ChangeEvent<HTMLInputElement>) => {
                      const numVal = parseInt(e.target.value, 10) || 1;
                      field.onChange(durationToSeconds(numVal, durationUnit));
                    };

                    const handleUnitChange = (unit: string) => {
                      const newUnit = unit as DurationUnit;
                      setDurationUnit(newUnit);
                      field.onChange(durationToSeconds(duration.value, newUnit));
                    };

                    return (
                      <FormItem>
                        <FormLabel className="flex items-center gap-2">
                          <Clock className="size-4" />
                          Session Duration
                        </FormLabel>
                        <div className="flex items-center gap-2">
                          <Input
                            type="number"
                            min={1}
                            value={duration.value}
                            onChange={handleValueChange}
                            onBlur={field.onBlur}
                            className="w-24"
                          />
                          <Select value={durationUnit} onValueChange={handleUnitChange}>
                            <SelectTrigger className="w-32">
                              <SelectValue />
                            </SelectTrigger>
                            <SelectContent>
                              <SelectItem value="minutes">Minutes</SelectItem>
                              <SelectItem value="hours">Hours</SelectItem>
                              <SelectItem value="days">Days</SelectItem>
                            </SelectContent>
                          </Select>
                        </div>
                        <FormDescription>
                          How long the session remains valid (60 seconds to 7 days)
                        </FormDescription>
                        <FormMessage />
                      </FormItem>
                    );
                  }}
                />
              </div>
            )}
          </CardContent>
        </Card>

        {canUpdateAccess && (
          <div className="flex justify-end">
            <Button type="submit" disabled={isConfiguring}>
              <Save className="size-4" />
              {isConfiguring ? 'Saving...' : 'Save Configuration'}
            </Button>
          </div>
        )}
      </form>
    </Form>
  );
}
