import {
  Alert,
  AlertDescription,
  Badge,
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  Checkbox,
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  Field,
  FieldContent,
  FieldDescription,
  FieldError,
  FieldGroup,
  FieldLabel,
  Input,
  Label,
  Skeleton,
  Switch,
} from '@e412/rnui-react';
import { useForm } from '@tanstack/react-form';
import {
  AlertCircle,
  Clock,
  Globe,
  KeyRound,
  Mail,
  Save,
  Settings,
  Shield,
  ShieldCheck,
  Users,
} from 'lucide-react';
import { useCallback, useEffect, useState } from 'react';
import { z } from 'zod';

import { TagsInput } from '@/components/ui/tags-input';
import {
  useConfigureWaygatesAuth,
  useOAuthProviderRestrictions,
  useOAuthProviders,
  useSetOAuthProviderRestriction,
  useWaygatesAuth,
} from '@/hooks';
import type { ACLOAuthProviderRestriction } from '@/types/acl';

// Available OAuth providers that can be configured
const OAUTH_PROVIDERS = [
  { id: 'google', name: 'Google', description: 'Sign in with Google accounts' },
  { id: 'github', name: 'GitHub', description: 'Sign in with GitHub accounts' },
  { id: 'microsoft', name: 'Microsoft', description: 'Sign in with Microsoft accounts' },
  { id: 'gitlab', name: 'GitLab', description: 'Sign in with GitLab accounts' },
] as const;

// Email validation regex (RFC 5322 simplified)
const EMAIL_REGEX = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

// Domain validation - accepts @domain.com or domain.com formats
const DOMAIN_REGEX = /^@?[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?(\.[a-zA-Z]{2,})+$/;

// Known roles from backend RBAC configuration (see backend/rbac.yaml)
const KNOWN_ROLES = [
  { value: 'admin', label: 'Administrator', description: 'Full access to all features' },
  { value: 'operator', label: 'Operator', description: 'Manage proxies & settings' },
  { value: 'viewer', label: 'Viewer', description: 'Read-only access' },
];

// Email pattern validation - allows wildcards like *@domain.com or specific emails
const EMAIL_PATTERN_REGEX = /^(\*|[^\s@]+)@[^\s@]+\.[^\s@]+$/;

// Validation configurations for TagsInput
const emailTagsValidation = {
  pattern: EMAIL_REGEX,
};

const domainTagsValidation = {
  pattern: DOMAIN_REGEX,
};

const emailPatternTagsValidation = {
  pattern: EMAIL_PATTERN_REGEX,
};

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
  // OAuth providers list (not restrictions - those are per-provider now)
  allowed_providers: z.array(z.string()).optional(),
});

type WaygatesAuthFormValues = z.infer<typeof waygatesAuthSchema>;

// Per-provider restriction state
interface ProviderRestrictionState {
  allowed_emails: string[];
  allowed_domains: string[];
  enabled: boolean;
}

interface WaygatesAuthTabProps {
  groupId: number;
}

// Modal component for provider restriction settings
interface ProviderRestrictionModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  providerId: string;
  providerName: string;
  restriction: ACLOAuthProviderRestriction | undefined;
  onSave: (data: {
    allowed_emails?: string[];
    allowed_domains?: string[];
    enabled: boolean;
  }) => Promise<void>;
  isSaving: boolean;
}

function ProviderRestrictionModal({
  open,
  onOpenChange,
  providerName,
  restriction,
  onSave,
  isSaving,
}: ProviderRestrictionModalProps) {
  const [state, setState] = useState<ProviderRestrictionState>({
    allowed_emails: [],
    allowed_domains: [],
    enabled: true,
  });

  // Initialize state from restriction when modal opens
  useEffect(() => {
    if (open) {
      if (restriction) {
        setState({
          allowed_emails: restriction.allowed_emails || [],
          allowed_domains: restriction.allowed_domains || [],
          enabled: restriction.enabled,
        });
      } else {
        setState({
          allowed_emails: [],
          allowed_domains: [],
          enabled: true,
        });
      }
    }
  }, [open, restriction]);

  const handleEmailsChange = (value: string[]) => {
    setState((prev) => ({ ...prev, allowed_emails: value }));
  };

  const handleDomainsChange = (value: string[]) => {
    setState((prev) => ({ ...prev, allowed_domains: value }));
  };

  const handleEnabledChange = (enabled: boolean) => {
    setState((prev) => ({ ...prev, enabled }));
  };

  const handleSave = async () => {
    await onSave({
      allowed_emails: state.allowed_emails.length > 0 ? state.allowed_emails : undefined,
      allowed_domains: state.allowed_domains.length > 0 ? state.allowed_domains : undefined,
      enabled: state.enabled,
    });
    onOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Settings className="size-5" />
            {providerName} Restrictions
          </DialogTitle>
        </DialogHeader>
        <div className="grid gap-4 py-2 space-y-6">
          <Field orientation="horizontal">
            <FieldContent>
              <FieldLabel>Enable Restrictions</FieldLabel>
              <FieldDescription>
                When enabled, only users matching the restrictions below can authenticate via{' '}
                {providerName}
              </FieldDescription>
            </FieldContent>
            <Switch checked={state.enabled} onCheckedChange={handleEnabledChange} />
          </Field>

          <Field>
            <FieldLabel className="flex items-center gap-2">
              <Mail className="size-4" />
              Allowed Emails
            </FieldLabel>
            <TagsInput
              value={state.allowed_emails}
              onValueChange={handleEmailsChange}
              placeholder="Add email..."
              delimiters={['Enter', ',', ' ']}
              validation={emailTagsValidation}
            />
            <FieldDescription>
              Specific email addresses allowed via {providerName}. Press Enter or comma to add.
            </FieldDescription>
          </Field>

          <Field>
            <FieldLabel className="flex items-center gap-2">
              <Globe className="size-4" />
              Allowed Domains
            </FieldLabel>
            <TagsInput
              value={state.allowed_domains}
              onValueChange={handleDomainsChange}
              placeholder="Add domain (e.g., @company.com)..."
              delimiters={['Enter', ',', ' ']}
              validation={domainTagsValidation}
            />
            <FieldDescription>
              Email domains allowed. Users with emails ending in these domains can authenticate.
            </FieldDescription>
          </Field>
        </div>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button type="button" onClick={handleSave} disabled={isSaving}>
            <Save className="size-4" />
            {isSaving ? 'Saving...' : 'Save Restrictions'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export function WaygatesAuthTab({ groupId }: WaygatesAuthTabProps) {
  const { config, isLoading } = useWaygatesAuth(groupId);
  const { configureAuth, isConfiguring } = useConfigureWaygatesAuth();
  const { restrictions, isLoading: isLoadingRestrictions } = useOAuthProviderRestrictions(groupId);
  const { setRestriction, isSetting } = useSetOAuthProviderRestriction();

  // Modal state for provider restriction configuration
  const [restrictionModalOpen, setRestrictionModalOpen] = useState(false);
  const [selectedProvider, setSelectedProvider] = useState<{
    id: string;
    name: string;
  } | null>(null);

  // Fetch OAuth providers from the backend (with availability status)
  const { providers: backendProviders, isLoading: isLoadingProviders } = useOAuthProviders();

  const form = useForm({
    defaultValues: {
      enabled: false,
      allowed_users: [] as string[],
      allowed_roles: [] as string[],
      allowed_email_patterns: [] as string[],
      require_2fa: false,
      session_ttl: 3600,
      // OAuth providers list
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
          // OAuth providers
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
      // OAuth providers
      form.setFieldValue('allowed_providers', config.allowed_providers || []);
    }
  }, [config, form.setFieldValue]);

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

  // Open restriction modal for a provider
  const handleConfigureProvider = (providerId: string, providerName: string) => {
    setSelectedProvider({ id: providerId, name: providerName });
    setRestrictionModalOpen(true);
  };

  // Save provider restriction
  const handleSaveProviderRestriction = useCallback(
    async (data: { allowed_emails?: string[]; allowed_domains?: string[]; enabled: boolean }) => {
      if (!selectedProvider) return;
      await setRestriction({
        groupId,
        provider: selectedProvider.id,
        data,
      });
    },
    [groupId, selectedProvider, setRestriction],
  );

  // Get restriction for a specific provider
  const getRestrictionForProvider = useCallback(
    (providerId: string): ACLOAuthProviderRestriction | undefined => {
      return restrictions.find((r) => r.provider === providerId);
    },
    [restrictions],
  );

  // Get restriction summary for display
  const getRestrictionSummary = (providerId: string): string | null => {
    const restriction = getRestrictionForProvider(providerId);
    if (!restriction) return null;

    const emailCount = restriction.allowed_emails?.length || 0;
    const domainCount = restriction.allowed_domains?.length || 0;
    const total = emailCount + domainCount;

    if (total === 0) return null;
    return `${total} restriction${total > 1 ? 's' : ''}`;
  };

  const formatTTL = (seconds: number): string => {
    if (seconds < 60) return `${seconds} seconds`;
    if (seconds < 3600) return `${Math.round(seconds / 60)} minutes`;
    if (seconds < 86400) return `${Math.round(seconds / 3600)} hours`;
    return `${Math.round(seconds / 86400)} days`;
  };

  // Get provider list for rendering - only show providers that are available (env vars configured)
  const providerList = backendProviders
    .filter((p) => p.available) // Only show providers with env vars configured
    .map((p) => ({
      id: p.id,
      name: p.name,
      available: p.available,
      description:
        OAUTH_PROVIDERS.find((op) => op.id === p.id)?.description || `Sign in with ${p.name}`,
    }));

  // Check if no OAuth providers are configured
  const hasNoAvailableProviders = !isLoadingProviders && providerList.length === 0;

  if (isLoading || isLoadingProviders) {
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
          <CardTitle className="flex items-center gap-2">
            <KeyRound className="size-5" />
            OAuth Providers
          </CardTitle>
          <CardDescription>
            Allow users to authenticate using external OAuth providers. Users don't need a Waygates
            account - they can sign in directly with their Google, GitHub, or other OAuth accounts.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-6">
          {isLoadingProviders ? (
            <div className="grid gap-3 sm:grid-cols-2">
              <Skeleton className="h-20 w-full rounded-lg" />
              <Skeleton className="h-20 w-full rounded-lg" />
            </div>
          ) : hasNoAvailableProviders ? (
            <Alert>
              <AlertCircle className="size-4" />
              <AlertDescription>
                No OAuth providers are configured on the server. To enable OAuth authentication,
                configure the following environment variables on the backend:
                <ul className="list-disc list-inside mt-2 space-y-1 text-sm">
                  <li>
                    <strong>Google:</strong> GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET
                  </li>
                  <li>
                    <strong>GitHub:</strong> GITHUB_CLIENT_ID, GITHUB_CLIENT_SECRET
                  </li>
                  <li>
                    <strong>Microsoft:</strong> MICROSOFT_CLIENT_ID, MICROSOFT_CLIENT_SECRET
                  </li>
                  <li>
                    <strong>GitLab:</strong> GITLAB_CLIENT_ID, GITLAB_CLIENT_SECRET
                  </li>
                </ul>
                <p className="mt-2">After configuring, restart the backend server.</p>
              </AlertDescription>
            </Alert>
          ) : (
            <Field>
              <FieldLabel className="flex items-center gap-2">
                <Shield className="size-4" />
                Enabled OAuth Providers
              </FieldLabel>
              <FieldDescription className="mb-3">
                Select which OAuth providers users can use to authenticate. Click the configure
                button to set email and domain restrictions for each provider.
              </FieldDescription>
              <form.Subscribe selector={(state) => state.values.allowed_providers}>
                {(selectedProviders) => (
                  <div className="grid gap-3 sm:grid-cols-2">
                    {providerList.map((provider) => {
                      const isSelected = selectedProviders?.includes(provider.id) || false;
                      const restrictionSummary = getRestrictionSummary(provider.id);

                      return (
                        <div
                          key={provider.id}
                          className={`flex items-start gap-3 p-3 rounded-lg border ${
                            isSelected ? 'border-primary bg-primary/5' : 'border-border bg-muted/30'
                          }`}
                        >
                          <Checkbox
                            id={`provider-${provider.id}`}
                            checked={isSelected}
                            onCheckedChange={(checked) =>
                              handleProviderToggle(provider.id, checked as boolean)
                            }
                          />
                          <div className="flex-1 min-w-0">
                            <div className="flex items-center gap-2">
                              <Label
                                htmlFor={`provider-${provider.id}`}
                                className="text-sm font-medium cursor-pointer"
                              >
                                {provider.name}
                              </Label>
                              {restrictionSummary && (
                                <Badge variant="outline" className="text-xs">
                                  {restrictionSummary}
                                </Badge>
                              )}
                            </div>
                            <p className="text-xs text-muted-foreground mt-0.5">
                              {provider.description}
                            </p>
                            {isSelected && (
                              <Button
                                type="button"
                                variant="ghost"
                                size="sm"
                                className="mt-2 h-7 px-2 text-xs"
                                onClick={() => handleConfigureProvider(provider.id, provider.name)}
                                disabled={isLoadingRestrictions}
                              >
                                <Settings className="size-3 mr-1" />
                                Configure Restrictions
                              </Button>
                            )}
                          </div>
                        </div>
                      );
                    })}
                  </div>
                )}
              </form.Subscribe>
            </Field>
          )}
        </CardContent>
      </Card>

      {/* Waygates Authentication Section - Separate from OAuth */}
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
                <FieldGroup className="space-y-6">
                  <Alert>
                    <AlertCircle className="size-4" />
                    <AlertDescription>
                      Leave all restriction fields empty to allow any authenticated Waygates user.
                      Add restrictions to limit access to specific users, roles, or email patterns.
                    </AlertDescription>
                  </Alert>

                  <form.Field name="allowed_roles">
                    {(field) => {
                      const selectedRoles = field.state.value || [];
                      const handleRoleToggle = (roleValue: string, checked: boolean) => {
                        if (checked) {
                          field.handleChange([...selectedRoles, roleValue]);
                        } else {
                          field.handleChange(selectedRoles.filter((r: string) => r !== roleValue));
                        }
                      };

                      return (
                        <Field>
                          <FieldLabel className="flex items-center gap-2">
                            <Users className="size-4" />
                            Allowed Roles
                          </FieldLabel>
                          <FieldDescription className="mb-3">
                            Select which roles can access this resource
                          </FieldDescription>
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
                                    handleRoleToggle(role.value, checked as boolean)
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
                        </Field>
                      );
                    }}
                  </form.Field>

                  <form.Field name="allowed_email_patterns">
                    {(field) => (
                      <Field>
                        <FieldLabel className="flex items-center gap-2">
                          <Mail className="size-4" />
                          Allowed Email Patterns
                        </FieldLabel>
                        <TagsInput
                          value={field.state.value || []}
                          onValueChange={field.handleChange}
                          placeholder="Add pattern (e.g., *@company.com)..."
                          delimiters={['Enter', ',']}
                          validation={emailPatternTagsValidation}
                        />
                        <FieldDescription>
                          Email patterns. Use * as wildcard. Press Enter or comma to add.
                        </FieldDescription>
                      </Field>
                    )}
                  </form.Field>

                  {/* TODO: 2FA support coming in future release */}

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

      {/* Provider Restriction Modal */}
      {selectedProvider && (
        <ProviderRestrictionModal
          open={restrictionModalOpen}
          onOpenChange={setRestrictionModalOpen}
          providerId={selectedProvider.id}
          providerName={selectedProvider.name}
          restriction={getRestrictionForProvider(selectedProvider.id)}
          onSave={handleSaveProviderRestriction}
          isSaving={isSetting}
        />
      )}
    </form>
  );
}
