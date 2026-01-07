import {
  Alert,
  AlertDescription,
  Badge,
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardHeading,
  CardTitle,
  Checkbox,
  Field,
  FieldContent,
  FieldDescription,
  FieldError,
  FieldGroup,
  FieldLabel,
  Input,
  Label,
  Separator,
  Skeleton,
  Switch,
  Textarea,
} from '@e412/titanium';
import { useForm } from '@tanstack/react-form';
import {
  AlertCircle,
  Clock,
  Globe,
  KeyRound,
  Mail,
  Save,
  Shield,
  ShieldCheck,
  Users,
} from 'lucide-react';
import { useEffect, useState } from 'react';
import { z } from 'zod';
import { useConfigureWaygatesAuth, useOAuthProviders, useWaygatesAuth } from '@/hooks';

// Available OAuth providers that can be configured
const OAUTH_PROVIDERS = [
  { id: 'google', name: 'Google', description: 'Sign in with Google accounts' },
  { id: 'github', name: 'GitHub', description: 'Sign in with GitHub accounts' },
  { id: 'microsoft', name: 'Microsoft', description: 'Sign in with Microsoft accounts' },
  { id: 'gitlab', name: 'GitLab', description: 'Sign in with GitLab accounts' },
] as const;

const waygatesAuthSchema = z.object({
  enabled: z.boolean(),
  allowed_users: z.array(z.string()).optional(),
  allowed_roles: z.array(z.string()).optional(),
  allowed_email_patterns: z.array(z.string()).optional(),
  require_2fa: z.boolean(),
  session_ttl: z
    .number()
    .min(60, 'Minimum session TTL is 60 seconds')
    .max(604800, 'Maximum session TTL is 7 days'),
  // OAuth settings (independent of Waygates auth)
  allowed_emails: z.array(z.string()).optional(),
  allowed_domains: z.array(z.string()).optional(),
  allowed_providers: z.array(z.string()).optional(),
});

type WaygatesAuthFormValues = z.infer<typeof waygatesAuthSchema>;

interface WaygatesAuthTabProps {
  groupId: number;
}

export function WaygatesAuthTab({ groupId }: WaygatesAuthTabProps) {
  const { config, isLoading } = useWaygatesAuth(groupId);
  const { configureAuth, isConfiguring } = useConfigureWaygatesAuth();

  const [rolesInput, setRolesInput] = useState('');
  const [emailPatternsInput, setEmailPatternsInput] = useState('');
  // OAuth restriction inputs
  const [allowedEmailsInput, setAllowedEmailsInput] = useState('');
  const [allowedDomainsInput, setAllowedDomainsInput] = useState('');

  // Fetch available OAuth providers from the backend
  const { providers: availableProviders } = useOAuthProviders();

  const form = useForm({
    defaultValues: {
      enabled: false,
      allowed_users: [] as string[],
      allowed_roles: [] as string[],
      allowed_email_patterns: [] as string[],
      require_2fa: false,
      session_ttl: 3600,
      // OAuth settings
      allowed_emails: [] as string[],
      allowed_domains: [] as string[],
      allowed_providers: [] as string[],
    } as WaygatesAuthFormValues,
    validators: {
      onSubmit: waygatesAuthSchema,
    },
    onSubmit: async ({ value }) => {
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
          // OAuth settings
          allowed_emails: value.allowed_emails?.length ? value.allowed_emails : undefined,
          allowed_domains: value.allowed_domains?.length ? value.allowed_domains : undefined,
          allowed_providers: value.allowed_providers?.length ? value.allowed_providers : undefined,
        },
      });
    },
  });

  useEffect(() => {
    if (config) {
      form.setFieldValue('enabled', config.enabled);
      form.setFieldValue('allowed_users', config.allowed_users || []);
      form.setFieldValue('allowed_roles', config.allowed_roles || []);
      form.setFieldValue('allowed_email_patterns', config.allowed_email_patterns || []);
      form.setFieldValue('require_2fa', config.require_2fa);
      form.setFieldValue('session_ttl', config.session_ttl);
      // OAuth settings
      form.setFieldValue('allowed_emails', config.allowed_emails || []);
      form.setFieldValue('allowed_domains', config.allowed_domains || []);
      form.setFieldValue('allowed_providers', config.allowed_providers || []);

      setRolesInput((config.allowed_roles || []).join(', '));
      setEmailPatternsInput((config.allowed_email_patterns || []).join(', '));
      // OAuth restriction inputs
      setAllowedEmailsInput((config.allowed_emails || []).join(', '));
      setAllowedDomainsInput((config.allowed_domains || []).join(', '));
    }
  }, [config, form.setFieldValue]);

  const handleRolesChange = (value: string) => {
    setRolesInput(value);
    const roles = value
      .split(',')
      .map((r) => r.trim())
      .filter(Boolean);
    form.setFieldValue('allowed_roles', roles);
  };

  const handleEmailPatternsChange = (value: string) => {
    setEmailPatternsInput(value);
    const patterns = value
      .split(',')
      .map((p) => p.trim())
      .filter(Boolean);
    form.setFieldValue('allowed_email_patterns', patterns);
  };

  // OAuth restriction handlers
  const handleAllowedEmailsChange = (value: string) => {
    setAllowedEmailsInput(value);
    const emails = value
      .split(',')
      .map((e) => e.trim())
      .filter(Boolean);
    form.setFieldValue('allowed_emails', emails);
  };

  const handleAllowedDomainsChange = (value: string) => {
    setAllowedDomainsInput(value);
    const domains = value
      .split(',')
      .map((d) => d.trim())
      .filter(Boolean);
    form.setFieldValue('allowed_domains', domains);
  };

  // Toggle OAuth provider selection
  const handleProviderToggle = (providerId: string, checked: boolean) => {
    const currentProviders = form.getFieldValue('allowed_providers') || [];
    if (checked) {
      form.setFieldValue('allowed_providers', [...currentProviders, providerId]);
    } else {
      form.setFieldValue(
        'allowed_providers',
        currentProviders.filter((p: string) => p !== providerId),
      );
    }
  };

  const formatTTL = (seconds: number): string => {
    if (seconds < 60) return `${seconds} seconds`;
    if (seconds < 3600) return `${Math.round(seconds / 60)} minutes`;
    if (seconds < 86400) return `${Math.round(seconds / 3600)} hours`;
    return `${Math.round(seconds / 86400)} days`;
  };

  // Get provider list for rendering
  const providerList =
    availableProviders.length > 0
      ? availableProviders.map((p) => ({
          id: p.id,
          name: p.name,
          enabled: p.enabled,
          description:
            OAUTH_PROVIDERS.find((op) => op.id === p.id)?.description || `Sign in with ${p.name}`,
        }))
      : OAUTH_PROVIDERS.map((p) => ({
          ...p,
          enabled: true,
        }));

  if (isLoading) {
    return (
      <div className="space-y-6">
        <Card>
          <CardHeader>
            <Skeleton className="h-6 w-48" />
            <Skeleton className="h-4 w-full mt-2" />
          </CardHeader>
          <CardContent className="space-y-6">
            <Skeleton className="h-12 w-full" />
            <Skeleton className="h-12 w-full" />
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <Skeleton className="h-6 w-48" />
            <Skeleton className="h-4 w-full mt-2" />
          </CardHeader>
          <CardContent className="space-y-6">
            <Skeleton className="h-12 w-full" />
            <Skeleton className="h-12 w-full" />
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <form
      onSubmit={(e) => {
        e.preventDefault();
        e.stopPropagation();
        form.handleSubmit();
      }}
      className="space-y-6"
    >
      {/* OAuth Providers Section - Independent of Waygates Auth */}
      <Card>
        <CardHeader>
          <CardHeading>
            <CardTitle className="flex items-center gap-2">
              <KeyRound className="size-5" />
              OAuth Providers
            </CardTitle>
            <CardDescription>
              Allow users to authenticate using external OAuth providers. Users don't need a
              Waygates account - they can sign in directly with their Google, GitHub, or other
              OAuth accounts.
            </CardDescription>
          </CardHeading>
        </CardHeader>
        <CardContent className="space-y-6">
          <Field>
            <FieldLabel className="flex items-center gap-2">
              <Shield className="size-4" />
              Enabled OAuth Providers
            </FieldLabel>
            <FieldDescription className="mb-3">
              Select which OAuth providers users can use to authenticate. Leave all unchecked to
              disable OAuth login entirely.
            </FieldDescription>
            <form.Subscribe selector={(state) => state.values.allowed_providers}>
              {(selectedProviders) => (
                <div className="grid gap-3 sm:grid-cols-2">
                  {providerList.map((provider) => (
                    <div
                      key={provider.id}
                      className={`flex items-start gap-3 p-3 rounded-lg border ${
                        selectedProviders?.includes(provider.id)
                          ? 'border-primary bg-primary/5'
                          : 'border-border bg-muted/30'
                      } ${!provider.enabled ? 'opacity-50' : ''}`}
                    >
                      <Checkbox
                        id={`provider-${provider.id}`}
                        checked={selectedProviders?.includes(provider.id) || false}
                        onCheckedChange={(checked) =>
                          handleProviderToggle(provider.id, checked as boolean)
                        }
                        disabled={!provider.enabled}
                      />
                      <div className="flex-1 min-w-0">
                        <Label
                          htmlFor={`provider-${provider.id}`}
                          className="text-sm font-medium cursor-pointer"
                        >
                          {provider.name}
                          {!provider.enabled && (
                            <Badge variant="outline" className="ml-2 text-xs">
                              Not Configured
                            </Badge>
                          )}
                        </Label>
                        <p className="text-xs text-muted-foreground mt-0.5">
                          {provider.description}
                        </p>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </form.Subscribe>
          </Field>

          <form.Subscribe selector={(state) => state.values.allowed_providers}>
            {(selectedProviders) =>
              selectedProviders &&
              selectedProviders.length > 0 && (
                <>
                  <Separator />
                  <div className="space-y-4">
                    <div className="flex items-center gap-2">
                      <Shield className="size-4 text-muted-foreground" />
                      <span className="text-sm font-medium">OAuth User Restrictions</span>
                    </div>
                    <p className="text-sm text-muted-foreground">
                      Optionally restrict which OAuth users can access this resource. Leave empty to
                      allow any user who authenticates via the enabled providers.
                    </p>
                  </div>

                  <Field>
                    <FieldLabel className="flex items-center gap-2">
                      <Mail className="size-4" />
                      Allowed Emails
                    </FieldLabel>
                    <Textarea
                      placeholder="e.g., john@example.com, jane@company.com"
                      value={allowedEmailsInput}
                      onChange={(e) => handleAllowedEmailsChange(e.target.value)}
                      rows={2}
                    />
                    <FieldDescription>
                      Comma-separated list of specific email addresses allowed to access
                    </FieldDescription>
                    <form.Subscribe selector={(state) => state.values.allowed_emails}>
                      {(emails) =>
                        emails &&
                        emails.length > 0 && (
                          <div className="flex flex-wrap gap-1 mt-2">
                            {emails.map((email) => (
                              <Badge key={email} variant="secondary" className="font-mono text-xs">
                                {email}
                              </Badge>
                            ))}
                          </div>
                        )
                      }
                    </form.Subscribe>
                  </Field>

                  <Field>
                    <FieldLabel className="flex items-center gap-2">
                      <Globe className="size-4" />
                      Allowed Domains
                    </FieldLabel>
                    <Input
                      placeholder="e.g., @company.com, @example.org"
                      value={allowedDomainsInput}
                      onChange={(e) => handleAllowedDomainsChange(e.target.value)}
                    />
                    <FieldDescription>
                      Comma-separated email domains (e.g., @company.com). Users with emails ending
                      in these domains will be allowed.
                    </FieldDescription>
                    <form.Subscribe selector={(state) => state.values.allowed_domains}>
                      {(domains) =>
                        domains &&
                        domains.length > 0 && (
                          <div className="flex flex-wrap gap-1 mt-2">
                            {domains.map((domain) => (
                              <Badge key={domain} variant="outline" className="font-mono text-xs">
                                {domain}
                              </Badge>
                            ))}
                          </div>
                        )
                      }
                    </form.Subscribe>
                  </Field>
                </>
              )
            }
          </form.Subscribe>
        </CardContent>
      </Card>

      {/* Waygates Authentication Section - Separate from OAuth */}
      <Card>
        <CardHeader>
          <CardHeading>
            <CardTitle className="flex items-center gap-2">
              <Users className="size-5" />
              Waygates Authentication
            </CardTitle>
            <CardDescription>
              Allow users with Waygates accounts to authenticate using their platform credentials.
              This is separate from OAuth - users need an account in Waygates to use this method.
            </CardDescription>
          </CardHeading>
        </CardHeader>
        <CardContent className="space-y-6">
          <form.Field name="enabled">
            {(field) => (
              <Field orientation="horizontal">
                <FieldContent>
                  <FieldLabel className="flex items-center gap-2">
                    {field.state.value ? (
                      <ShieldCheck className="size-4 text-green-500" />
                    ) : (
                      <Shield className="size-4 text-muted-foreground" />
                    )}
                    Enable Waygates Authentication
                  </FieldLabel>
                  <FieldDescription>
                    Allow users to sign in with their Waygates account credentials
                  </FieldDescription>
                </FieldContent>
                <Switch checked={field.state.value} onCheckedChange={field.handleChange} />
              </Field>
            )}
          </form.Field>

          <form.Subscribe selector={(state) => state.values.enabled}>
            {(enabled) =>
              enabled && (
                <FieldGroup className="space-y-6 pl-6 border-l-2 border-primary/20">
                  <Alert>
                    <AlertCircle className="size-4" />
                    <AlertDescription>
                      Leave all restriction fields empty to allow any authenticated Waygates user.
                      Add restrictions to limit access to specific users, roles, or email patterns.
                    </AlertDescription>
                  </Alert>

                  <Field>
                    <FieldLabel className="flex items-center gap-2">
                      <Users className="size-4" />
                      Allowed Roles
                    </FieldLabel>
                    <Input
                      placeholder="e.g., admin, developer, viewer"
                      value={rolesInput}
                      onChange={(e) => handleRolesChange(e.target.value)}
                    />
                    <FieldDescription>
                      Comma-separated list of roles that can access this resource
                    </FieldDescription>
                    <form.Subscribe selector={(state) => state.values.allowed_roles}>
                      {(roles) =>
                        roles &&
                        roles.length > 0 && (
                          <div className="flex flex-wrap gap-1 mt-2">
                            {roles.map((role) => (
                              <Badge key={role} variant="secondary">
                                {role}
                              </Badge>
                            ))}
                          </div>
                        )
                      }
                    </form.Subscribe>
                  </Field>

                  <Field>
                    <FieldLabel className="flex items-center gap-2">
                      <Mail className="size-4" />
                      Allowed Email Patterns
                    </FieldLabel>
                    <Textarea
                      placeholder="e.g., *@company.com, john@example.com"
                      value={emailPatternsInput}
                      onChange={(e) => handleEmailPatternsChange(e.target.value)}
                      rows={2}
                    />
                    <FieldDescription>
                      Comma-separated email patterns. Use * as wildcard (e.g., *@company.com)
                    </FieldDescription>
                    <form.Subscribe selector={(state) => state.values.allowed_email_patterns}>
                      {(patterns) =>
                        patterns &&
                        patterns.length > 0 && (
                          <div className="flex flex-wrap gap-1 mt-2">
                            {patterns.map((pattern) => (
                              <Badge key={pattern} variant="outline" className="font-mono text-xs">
                                {pattern}
                              </Badge>
                            ))}
                          </div>
                        )
                      }
                    </form.Subscribe>
                  </Field>

                  <form.Field name="require_2fa">
                    {(field) => (
                      <Field orientation="horizontal">
                        <FieldContent>
                          <FieldLabel>Require Two-Factor Authentication</FieldLabel>
                          <FieldDescription>
                            Only allow users with 2FA enabled on their account
                          </FieldDescription>
                        </FieldContent>
                        <Switch checked={field.state.value} onCheckedChange={field.handleChange} />
                      </Field>
                    )}
                  </form.Field>

                  <form.Field name="session_ttl">
                    {(field) => {
                      const hasError =
                        field.state.meta.isTouched && field.state.meta.errors.length > 0;
                      return (
                        <Field data-invalid={hasError}>
                          <FieldLabel className="flex items-center gap-2">
                            <Clock className="size-4" />
                            Session Duration
                          </FieldLabel>
                          <div className="flex items-center gap-4">
                            <Input
                              type="number"
                              min={60}
                              max={604800}
                              value={field.state.value}
                              onChange={(e) =>
                                field.handleChange(parseInt(e.target.value, 10) || 3600)
                              }
                              onBlur={field.handleBlur}
                              className="w-32"
                              aria-invalid={hasError}
                            />
                            <span className="text-sm text-muted-foreground">
                              seconds ({formatTTL(field.state.value)})
                            </span>
                          </div>
                          <FieldDescription>
                            How long the session remains valid (60 seconds to 7 days)
                          </FieldDescription>
                          {hasError && <FieldError errors={field.state.meta.errors} />}
                        </Field>
                      );
                    }}
                  </form.Field>
                </FieldGroup>
              )
            }
          </form.Subscribe>
        </CardContent>
      </Card>

      <div className="flex justify-end">
        <Button type="submit" disabled={isConfiguring}>
          <Save className="size-4" />
          {isConfiguring ? 'Saving...' : 'Save Configuration'}
        </Button>
      </div>
    </form>
  );
}
