import {
  Button,
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  Input,
  Skeleton,
  Spinner,
  Switch,
} from '@e412/rnui-react';
import { zodResolver } from '@hookform/resolvers/zod';
import { useEffect } from 'react';
import { useForm } from 'react-hook-form';
import { z } from 'zod';

import { TagsInput } from '@/components/ui/tags-input';
import { usePermissions } from '@/hooks/use-permissions';
import { useMetricsPublishSettings } from '@/hooks/use-settings';
import type { MetricsPublishSettings } from '@/types/metrics-publish';

// IPv4 CIDR: e.g. 10.0.0.0/8
// IPv6 CIDR: e.g. 2001:db8::/32
const CIDR_PATTERN =
  /^(?:(?:\d{1,3}\.){3}\d{1,3}\/(?:[0-9]|[1-2]\d|3[0-2])|[0-9a-fA-F:]+\/(?:\d|[1-9]\d|1[01]\d|12[0-8]))$/;

const metricsPublishSchema = z
  .object({
    enabled: z.boolean(),
    host: z.string(),
    path: z.string(),
    basic_auth_user: z.string(),
    basic_auth_password: z.string(),
    has_basic_auth: z.boolean(),
    allowed_cidrs: z.array(z.string()),
  })
  .superRefine((data, ctx) => {
    if (!data.enabled) return;

    if (!data.host.trim()) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        message: 'Host is required when metrics publishing is enabled',
        path: ['host'],
      });
    }

    if (!data.basic_auth_user.trim()) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        message: 'Username is required when metrics publishing is enabled',
        path: ['basic_auth_user'],
      });
    }

    if (!data.has_basic_auth && !data.basic_auth_password) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        message: 'Password is required when no password is currently set',
        path: ['basic_auth_password'],
      });
    }

    for (const cidr of data.allowed_cidrs) {
      if (!CIDR_PATTERN.test(cidr)) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          message: `"${cidr}" is not a valid CIDR (e.g. 10.0.0.0/8 or 2001:db8::/32)`,
          path: ['allowed_cidrs'],
        });
        break;
      }
    }
  });

type MetricsPublishFormValues = z.infer<typeof metricsPublishSchema>;

function settingsToFormValues(s: MetricsPublishSettings): MetricsPublishFormValues {
  return {
    enabled: s.enabled,
    host: s.host,
    path: s.path || '/metrics',
    basic_auth_user: s.basic_auth_user,
    basic_auth_password: '',
    has_basic_auth: s.has_basic_auth,
    allowed_cidrs: s.allowed_cidrs ?? [],
  };
}

export function MetricsPublishSettings() {
  const { settings, isLoading, update, isUpdating } = useMetricsPublishSettings();
  const { canWriteSettings } = usePermissions();

  const form = useForm<MetricsPublishFormValues>({
    resolver: zodResolver(metricsPublishSchema),
    defaultValues: {
      enabled: false,
      host: '',
      path: '/metrics',
      basic_auth_user: '',
      basic_auth_password: '',
      has_basic_auth: false,
      allowed_cidrs: [],
    },
  });

  const { isDirty, isValid } = form.formState;

  useEffect(() => {
    if (settings) {
      form.reset(settingsToFormValues(settings));
    }
  }, [settings, form]);

  const onSubmit = async (values: MetricsPublishFormValues) => {
    const trimmedHost = values.host.trim();
    const trimmedPath = values.path.trim() || '/metrics';
    const trimmedUser = values.basic_auth_user.trim();
    const sentPassword = values.basic_auth_password ?? '';

    await update({
      enabled: values.enabled,
      host: trimmedHost,
      path: trimmedPath,
      basic_auth_user: trimmedUser,
      basic_auth_password: sentPassword,
      allowed_cidrs: values.allowed_cidrs,
    });

    // Reset to the values that represent the saved state:
    // use trimmed fields (what was actually sent), blank password, and
    // has_basic_auth true if a password was just set OR one was already set.
    form.reset({
      enabled: values.enabled,
      host: trimmedHost,
      path: trimmedPath,
      basic_auth_user: trimmedUser,
      basic_auth_password: '',
      has_basic_auth: sentPassword !== '' || values.has_basic_auth,
      allowed_cidrs: values.allowed_cidrs,
    });
  };

  const enabled = form.watch('enabled');
  const host = form.watch('host');
  const path = form.watch('path');
  const hasBasicAuth = form.watch('has_basic_auth');

  const scrapeUrl = host.trim() ? `https://${host.trim()}${path.trim() || '/metrics'}` : null;

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
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Metrics Publishing</CardTitle>
        <CardDescription>
          Expose a Prometheus-compatible metrics endpoint so external scrapers can collect traffic
          statistics.
        </CardDescription>
      </CardHeader>
      <Form {...form}>
        <form onSubmit={form.handleSubmit(onSubmit)}>
          <CardContent className="space-y-6">
            {/* Enable toggle */}
            <FormField
              control={form.control}
              name="enabled"
              render={({ field }) => (
                <FormItem className="flex items-center justify-between rounded-lg border p-4">
                  <div className="space-y-0.5">
                    <FormLabel className="text-base">Enable Metrics Publishing</FormLabel>
                    <FormDescription>
                      Allow external services to scrape Prometheus metrics from this instance.
                    </FormDescription>
                  </div>
                  <FormControl>
                    <Switch checked={field.value} onCheckedChange={field.onChange} />
                  </FormControl>
                </FormItem>
              )}
            />

            {/* Fields that are only meaningful when enabled — always rendered so validation fires */}
            <div className="space-y-4">
              {/* Host */}
              <FormField
                control={form.control}
                name="host"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      Host
                      {enabled && <span className="ml-1 text-destructive">*</span>}
                    </FormLabel>
                    <FormControl>
                      <Input placeholder="metrics.example.com" disabled={!enabled} {...field} />
                    </FormControl>
                    <FormDescription>
                      The hostname under which the metrics endpoint is exposed.
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              {/* Path */}
              <FormField
                control={form.control}
                name="path"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Path</FormLabel>
                    <FormControl>
                      <Input placeholder="/metrics" disabled={!enabled} {...field} />
                    </FormControl>
                    <FormDescription>
                      URL path for the metrics endpoint. Defaults to{' '}
                      <code className="font-mono">/metrics</code>.
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              {/* Scrape URL hint */}
              {scrapeUrl && enabled && (
                <div className="rounded-md border bg-muted/40 px-4 py-3">
                  <p className="text-xs text-muted-foreground">Scrape URL</p>
                  <p className="mt-0.5 font-mono text-sm break-all">{scrapeUrl}</p>
                </div>
              )}

              {/* Basic auth user */}
              <FormField
                control={form.control}
                name="basic_auth_user"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      Username
                      {enabled && <span className="ml-1 text-destructive">*</span>}
                    </FormLabel>
                    <FormControl>
                      <Input
                        placeholder="prometheus"
                        autoComplete="username"
                        disabled={!enabled}
                        {...field}
                      />
                    </FormControl>
                    <FormDescription>
                      HTTP basic auth username scrapers must use to authenticate.
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              {/* Basic auth password */}
              <FormField
                control={form.control}
                name="basic_auth_password"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      Password
                      {enabled && !hasBasicAuth && <span className="ml-1 text-destructive">*</span>}
                    </FormLabel>
                    <FormControl>
                      <Input
                        type="password"
                        placeholder={
                          hasBasicAuth ? 'Leave blank to keep current password' : 'Enter password'
                        }
                        autoComplete="new-password"
                        disabled={!enabled}
                        {...field}
                      />
                    </FormControl>
                    <FormDescription>
                      {hasBasicAuth
                        ? 'A password is already set. Leave blank to keep it unchanged.'
                        : 'Password for HTTP basic authentication.'}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              {/* Allowed CIDRs */}
              <FormField
                control={form.control}
                name="allowed_cidrs"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Allowed CIDRs</FormLabel>
                    <FormControl>
                      <TagsInput
                        value={field.value}
                        onValueChange={field.onChange}
                        placeholder="10.0.0.0/8 — press Enter to add"
                        validation={{ pattern: CIDR_PATTERN }}
                        delimiters={['Enter', ',']}
                      />
                    </FormControl>
                    <FormDescription>
                      Restrict scraper access to these IP ranges. Leave empty to allow any source.
                      Enter each CIDR and press Enter.
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>
          </CardContent>

          <CardFooter className="flex justify-end gap-2 border-t">
            <Button
              type="button"
              variant="outline"
              onClick={() => {
                if (settings) form.reset(settingsToFormValues(settings));
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
                  <Spinner variant="circle" />
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
