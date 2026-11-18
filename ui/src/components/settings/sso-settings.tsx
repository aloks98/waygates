import { zodResolver } from '@hookform/resolvers/zod';
import { Check, Copy, X } from 'lucide-react';
import { useEffect, useState } from 'react';
import { useForm } from 'react-hook-form';
import { z } from 'zod';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectGroup,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Skeleton } from '@/components/ui/skeleton';
import { Spinner } from '@/components/ui/spinner';
import { Switch } from '@/components/ui/switch';
import { usePermissions } from '@/hooks/use-permissions';
import { useSsoSettings } from '@/hooks/use-settings';
import type { SsoConfig } from '@/hooks/use-settings';
import { api } from '@/lib/api';
import type { ApiResponse } from '@/types/api';

const ssoSchema = z
  .object({
    enabled: z.boolean(),
    issuer: z.string(),
    client_id: z.string(),
    client_secret: z.string(),
    has_client_secret: z.boolean(),
    auto_provision: z.boolean(),
    default_role: z.string(),
    button_label: z.string(),
    base_url: z.string(),
  })
  .superRefine((data, ctx) => {
    if (!data.enabled) return;

    if (!data.issuer.trim()) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        message: 'Issuer URL is required when SSO is enabled',
        path: ['issuer'],
      });
    }

    if (!data.client_id.trim()) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        message: 'Client ID is required when SSO is enabled',
        path: ['client_id'],
      });
    }

    if (!data.has_client_secret && !data.client_secret) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        message: 'Client secret is required when no secret is currently set',
        path: ['client_secret'],
      });
    }
  });

type SsoFormValues = z.infer<typeof ssoSchema>;

const ROLE_OPTIONS = [
  { value: 'viewer', label: 'Viewer' },
  { value: 'operator', label: 'Operator' },
  { value: 'admin', label: 'Admin' },
] as const;

function settingsToFormValues(s: SsoConfig): SsoFormValues {
  return {
    enabled: s.enabled,
    issuer: s.issuer,
    client_id: s.client_id,
    client_secret: '',
    has_client_secret: s.has_client_secret,
    auto_provision: s.auto_provision,
    default_role: s.default_role || 'viewer',
    button_label: s.button_label,
    base_url: s.base_url,
  };
}

interface TestResult {
  ok: boolean;
  error?: string;
}

export function SSOSettings() {
  const { settings, isLoading, update, isUpdating } = useSsoSettings();
  const { canWriteSettings } = usePermissions();
  const [testResult, setTestResult] = useState<TestResult | null>(null);
  const [isTesting, setIsTesting] = useState(false);
  const [copied, setCopied] = useState(false);

  const form = useForm<SsoFormValues>({
    resolver: zodResolver(ssoSchema),
    defaultValues: {
      enabled: false,
      issuer: '',
      client_id: '',
      client_secret: '',
      has_client_secret: false,
      auto_provision: false,
      default_role: 'viewer',
      button_label: 'Login with SSO',
      base_url: '',
    },
  });

  const { isDirty, isValid } = form.formState;

  useEffect(() => {
    if (settings) {
      form.reset(settingsToFormValues(settings));
    }
  }, [settings, form]);

  const onSubmit = async (values: SsoFormValues) => {
    const sentSecret = values.client_secret ?? '';

    await update({
      enabled: values.enabled,
      issuer: values.issuer.trim(),
      client_id: values.client_id.trim(),
      client_secret: sentSecret,
      auto_provision: values.auto_provision,
      default_role: values.default_role,
      button_label: values.button_label.trim(),
      base_url: values.base_url.trim(),
    });

    setTestResult(null);

    form.reset({
      enabled: values.enabled,
      issuer: values.issuer.trim(),
      client_id: values.client_id.trim(),
      client_secret: '',
      has_client_secret: sentSecret !== '' || values.has_client_secret,
      auto_provision: values.auto_provision,
      default_role: values.default_role,
      button_label: values.button_label.trim(),
      base_url: values.base_url.trim(),
    });
  };

  const handleTest = async () => {
    const issuer = form.getValues('issuer').trim();
    if (!issuer) return;

    setIsTesting(true);
    setTestResult(null);
    try {
      const response = await api
        .post('auth/sso/test', { json: { issuer } })
        .json<ApiResponse<TestResult>>();
      setTestResult(response.data ?? { ok: false, error: 'No response data' });
    } catch {
      setTestResult({ ok: false, error: 'Connection test failed' });
    } finally {
      setIsTesting(false);
    }
  };

  const handleCopyRedirectUri = async () => {
    if (!settings?.redirect_uri) return;
    await navigator.clipboard.writeText(settings.redirect_uri);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const enabled = form.watch('enabled');
  const hasClientSecret = form.watch('has_client_secret');
  const currentIssuer = form.watch('issuer');

  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <Skeleton className="h-6 w-56" />
          <Skeleton className="h-4 w-96" />
        </CardHeader>
        <CardContent className="space-y-6">
          <Skeleton className="h-8 w-full" />
          <Skeleton className="h-10 w-full" />
          <Skeleton className="h-10 w-full" />
          <Skeleton className="h-10 w-full" />
          <Skeleton className="h-10 w-full" />
          <Skeleton className="h-10 w-full" />
          <Skeleton className="h-10 w-full" />
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Single Sign-On</CardTitle>
      </CardHeader>
      <Form {...form}>
        <form onSubmit={form.handleSubmit(onSubmit)}>
          <CardContent className="space-y-6">
            {/* Enable SSO toggle */}
            <FormField
              control={form.control}
              name="enabled"
              render={({ field }) => (
                <FormItem
                  orientation="horizontal"
                  className="flex items-center justify-between rounded-lg border p-4"
                >
                  <div className="space-y-0.5">
                    <FormLabel className="text-base">Enable Single Sign-On</FormLabel>
                  </div>
                  <FormControl>
                    <Switch checked={field.value} onCheckedChange={field.onChange} />
                  </FormControl>
                </FormItem>
              )}
            />

            <div className="space-y-4">
              {/* Issuer URL */}
              <FormField
                control={form.control}
                name="issuer"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      Issuer URL
                      {enabled && <span className="ml-1 text-destructive">*</span>}
                    </FormLabel>
                    <FormControl>
                      <Input
                        placeholder="https://accounts.example.com"
                        disabled={!enabled}
                        {...field}
                      />
                    </FormControl>

                    <FormMessage />
                  </FormItem>
                )}
              />

              {/* Client ID */}
              <FormField
                control={form.control}
                name="client_id"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      Client ID
                      {enabled && <span className="ml-1 text-destructive">*</span>}
                    </FormLabel>
                    <FormControl>
                      <Input
                        placeholder="waygates"
                        autoComplete="off"
                        disabled={!enabled}
                        {...field}
                      />
                    </FormControl>

                    <FormMessage />
                  </FormItem>
                )}
              />

              {/* Client Secret */}
              <FormField
                control={form.control}
                name="client_secret"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      Client Secret
                      {enabled && !hasClientSecret && (
                        <span className="ml-1 text-destructive">*</span>
                      )}
                    </FormLabel>
                    <FormControl>
                      <Input
                        type="password"
                        placeholder={
                          hasClientSecret ? 'Leave blank to keep current secret' : 'Enter secret'
                        }
                        autoComplete="new-password"
                        disabled={!enabled}
                        {...field}
                      />
                    </FormControl>

                    <FormMessage />
                  </FormItem>
                )}
              />

              {/* Test Connection */}
              <div className="flex items-center gap-3">
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  disabled={isTesting || !currentIssuer.trim() || !enabled}
                  onClick={handleTest}
                >
                  {isTesting ? (
                    <>
                      <Spinner />
                      Testing...
                    </>
                  ) : (
                    'Test Connection'
                  )}
                </Button>
                {testResult !== null && (
                  <span
                    className={`flex items-center gap-1 text-sm ${testResult.ok ? 'text-green-600' : 'text-destructive'}`}
                  >
                    {testResult.ok ? (
                      <>
                        <Check className="size-4" />
                        Connected
                      </>
                    ) : (
                      <>
                        <X className="size-4" />
                        {testResult.error ?? 'Connection failed'}
                      </>
                    )}
                  </span>
                )}
              </div>

              {/* Auto Provision */}
              <FormField
                control={form.control}
                name="auto_provision"
                render={({ field }) => (
                  <FormItem
                    orientation="horizontal"
                    className="flex items-center justify-between rounded-lg border p-4"
                  >
                    <div className="space-y-0.5">
                      <FormLabel className="text-base">Auto-provision Users</FormLabel>
                    </div>
                    <FormControl>
                      <Switch
                        checked={field.value}
                        onCheckedChange={field.onChange}
                        disabled={!enabled}
                      />
                    </FormControl>
                  </FormItem>
                )}
              />

              {/* Default Role */}
              <FormField
                control={form.control}
                name="default_role"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Default Role</FormLabel>
                    <Select
                      items={ROLE_OPTIONS}
                      value={field.value}
                      onValueChange={field.onChange}
                      disabled={!enabled}
                    >
                      <FormControl>
                        <SelectTrigger>
                          <SelectValue placeholder="Select a role" />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent>
                        <SelectGroup>
                          {ROLE_OPTIONS.map((o) => (
                            <SelectItem key={o.value} value={o.value}>
                              {o.label}
                            </SelectItem>
                          ))}
                        </SelectGroup>
                      </SelectContent>
                    </Select>

                    <FormMessage />
                  </FormItem>
                )}
              />

              {/* Button Label */}
              <FormField
                control={form.control}
                name="button_label"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Login Button Label</FormLabel>
                    <FormControl>
                      <Input placeholder="Login with SSO" disabled={!enabled} {...field} />
                    </FormControl>

                    <FormMessage />
                  </FormItem>
                )}
              />

              {/* Redirect URI (read-only) */}
              {settings?.redirect_uri && (
                <div className="space-y-1.5">
                  <p className="text-sm font-medium">Redirect URI</p>
                  <div className="flex items-center gap-2 rounded-md border bg-muted/40 px-3 py-2">
                    <span className="min-w-0 flex-1 break-all font-mono text-sm">
                      {settings.redirect_uri}
                    </span>
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      className="shrink-0"
                      onClick={handleCopyRedirectUri}
                    >
                      {copied ? (
                        <Check className="size-4 text-green-600" />
                      ) : (
                        <Copy className="size-4" />
                      )}
                    </Button>
                  </div>
                  <p className="text-xs text-muted-foreground">
                    Register this URL as the redirect/callback URI in your identity provider.
                  </p>
                </div>
              )}
            </div>
          </CardContent>

          <CardFooter className="flex justify-end gap-2 border-t">
            <Button
              type="button"
              variant="outline"
              onClick={() => {
                if (settings) form.reset(settingsToFormValues(settings));
                setTestResult(null);
              }}
              disabled={isUpdating || !isDirty}
            >
              Discard Changes
            </Button>
            <Button
              type="submit"
              disabled={!canWriteSettings || isUpdating || !isDirty || !isValid}
            >
              {isUpdating ? (
                <>
                  <Spinner />
                  Saving...
                </>
              ) : (
                'Save Changes'
              )}
            </Button>
          </CardFooter>
        </form>
      </Form>
    </Card>
  );
}
