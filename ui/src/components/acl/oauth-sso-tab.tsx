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
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  Label,
  Skeleton,
  Switch,
} from '@e412/rnui-react';
import { zodResolver } from '@hookform/resolvers/zod';
import { AlertCircle, Globe, KeyRound, Mail, Save, Settings, Shield } from 'lucide-react';
import { useCallback, useEffect, useState } from 'react';
import { useForm } from 'react-hook-form';
import { z } from 'zod';

import { TagsInput } from '@/components/ui/tags-input';
import {
  useConfigureWaygatesAuth,
  useOAuthProviderRestrictions,
  useOAuthProviders,
  useSetOAuthProviderRestriction,
  useWaygatesAuth,
} from '@/hooks';
import { usePermissions } from '@/hooks/use-permissions';
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

// Validation configurations for TagsInput
const emailTagsValidation = {
  pattern: EMAIL_REGEX,
};

const domainTagsValidation = {
  pattern: DOMAIN_REGEX,
};

const oauthSSOSchema = z.object({
  allowed_providers: z.array(z.string()).optional(),
});

type OAuthSSOFormValues = z.infer<typeof oauthSSOSchema>;

// Schema for the ProviderRestrictionModal — mirrors the prior useState validation exactly
// (no per-tag validation at the form level; TagsInput enforces EMAIL_REGEX / DOMAIN_REGEX inline)
const providerRestrictionSchema = z.object({
  enabled: z.boolean(),
  allowed_emails: z.array(z.string()),
  allowed_domains: z.array(z.string()),
});

type ProviderRestrictionFormValues = z.infer<typeof providerRestrictionSchema>;

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
  const form = useForm<ProviderRestrictionFormValues>({
    resolver: zodResolver(providerRestrictionSchema),
    defaultValues: {
      enabled: true,
      allowed_emails: [],
      allowed_domains: [],
    },
  });

  // Seed from the provider's existing restriction when modal opens
  useEffect(() => {
    if (open) {
      if (restriction) {
        form.reset({
          enabled: restriction.enabled,
          allowed_emails: restriction.allowed_emails || [],
          allowed_domains: restriction.allowed_domains || [],
        });
      } else {
        form.reset({
          enabled: true,
          allowed_emails: [],
          allowed_domains: [],
        });
      }
    }
  }, [open, restriction, form]);

  const onSubmit = async (value: ProviderRestrictionFormValues) => {
    await onSave({
      allowed_emails: value.allowed_emails.length > 0 ? value.allowed_emails : undefined,
      allowed_domains: value.allowed_domains.length > 0 ? value.allowed_domains : undefined,
      enabled: value.enabled,
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
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)}>
            <div className="grid gap-4 py-2 space-y-6">
              <FormField
                control={form.control}
                name="enabled"
                render={({ field }) => (
                  <FormItem className="flex flex-row items-center justify-between">
                    <div className="space-y-0.5">
                      <FormLabel>Enable Restrictions</FormLabel>
                      <FormDescription>
                        When enabled, only users matching the restrictions below can authenticate
                        via {providerName}
                      </FormDescription>
                    </div>
                    <FormControl>
                      <Switch checked={field.value} onCheckedChange={field.onChange} />
                    </FormControl>
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="allowed_emails"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel className="flex items-center gap-2">
                      <Mail className="size-4" />
                      Allowed Emails
                    </FormLabel>
                    <FormControl>
                      <TagsInput
                        value={field.value ?? []}
                        onValueChange={field.onChange}
                        placeholder="Add email..."
                        delimiters={['Enter', ',', ' ']}
                        validation={emailTagsValidation}
                      />
                    </FormControl>
                    <FormDescription>
                      Specific email addresses allowed via {providerName}. Press Enter or comma to
                      add.
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="allowed_domains"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel className="flex items-center gap-2">
                      <Globe className="size-4" />
                      Allowed Domains
                    </FormLabel>
                    <FormControl>
                      <TagsInput
                        value={field.value ?? []}
                        onValueChange={field.onChange}
                        placeholder="Add domain (e.g., @company.com)..."
                        delimiters={['Enter', ',', ' ']}
                        validation={domainTagsValidation}
                      />
                    </FormControl>
                    <FormDescription>
                      Email domains allowed. Users with emails ending in these domains can
                      authenticate.
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={isSaving}>
                <Save className="size-4" />
                {isSaving ? 'Saving...' : 'Save Restrictions'}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
}

export function OAuthSSOTab({ groupId }: { groupId: number }) {
  const { config, isLoading } = useWaygatesAuth(groupId);
  const { configureAuth, isConfiguring } = useConfigureWaygatesAuth();
  const { restrictions, isLoading: isLoadingRestrictions } = useOAuthProviderRestrictions(groupId);
  const { setRestriction, isSetting } = useSetOAuthProviderRestriction();
  const { providers: backendProviders, isLoading: isLoadingProviders } = useOAuthProviders();
  const { canUpdateAccess } = usePermissions();

  // Modal state for provider restriction configuration
  const [restrictionModalOpen, setRestrictionModalOpen] = useState(false);
  const [selectedProvider, setSelectedProvider] = useState<{
    id: string;
    name: string;
  } | null>(null);

  const form = useForm<OAuthSSOFormValues>({
    resolver: zodResolver(oauthSSOSchema),
    defaultValues: {
      allowed_providers: [] as string[],
    },
  });

  // Seed this tab's slice from config on load (load-merge-save: form holds only the OAuth slice)
  useEffect(() => {
    if (config) {
      form.reset({ allowed_providers: config.allowed_providers || [] });
    }
  }, [config, form]);

  const onSubmit = async (value: OAuthSSOFormValues) => {
    await configureAuth({
      groupId,
      data: {
        // preserve the account fields owned by the other tab
        enabled: config?.enabled ?? false,
        allowed_users: config?.allowed_users,
        allowed_roles: config?.allowed_roles,
        allowed_email_patterns: config?.allowed_email_patterns,
        require_2fa: config?.require_2fa ?? false,
        session_ttl: config?.session_ttl ?? 3600,
        // this tab's slice:
        allowed_providers: value.allowed_providers?.length ? value.allowed_providers : undefined,
      },
    });
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

  // Get provider list for rendering - only show providers that are available (env vars configured)
  const providerList = backendProviders
    .filter((p) => p.available)
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
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <KeyRound className="size-5" />
              OAuth Providers
            </CardTitle>
            <CardDescription>
              Allow users to authenticate using external OAuth providers. Users don't need a
              Waygates account - they can sign in directly with their Google, GitHub, or other OAuth
              accounts.
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
              <FormField
                control={form.control}
                name="allowed_providers"
                render={({ field }) => {
                  const selectedProviders: string[] = field.value ?? [];
                  const toggle = (providerId: string, checked: boolean) => {
                    field.onChange(
                      checked
                        ? [...selectedProviders, providerId]
                        : selectedProviders.filter((p) => p !== providerId),
                    );
                  };
                  return (
                    <FormItem>
                      <FormLabel className="flex items-center gap-2">
                        <Shield className="size-4" />
                        Enabled OAuth Providers
                      </FormLabel>
                      <FormDescription className="mb-3">
                        Select which OAuth providers users can use to authenticate. Click the
                        configure button to set email and domain restrictions for each provider.
                      </FormDescription>
                      <div className="grid gap-3 sm:grid-cols-2">
                        {providerList.map((provider) => {
                          const isSelected = selectedProviders.includes(provider.id);
                          const restrictionSummary = getRestrictionSummary(provider.id);

                          return (
                            <div
                              key={provider.id}
                              className={`flex items-start gap-3 p-3 rounded-lg border ${
                                isSelected
                                  ? 'border-primary bg-primary/5'
                                  : 'border-border bg-muted/30'
                              }`}
                            >
                              <Checkbox
                                id={`provider-${provider.id}`}
                                checked={isSelected}
                                onCheckedChange={(checked) =>
                                  toggle(provider.id, checked as boolean)
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
                                {isSelected && canUpdateAccess && (
                                  <Button
                                    type="button"
                                    variant="ghost"
                                    size="sm"
                                    className="mt-2 h-7 px-2 text-xs"
                                    onClick={() =>
                                      handleConfigureProvider(provider.id, provider.name)
                                    }
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
                      <FormMessage />
                    </FormItem>
                  );
                }}
              />
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
    </Form>
  );
}
