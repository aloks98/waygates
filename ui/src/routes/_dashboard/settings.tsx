import {
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  Field,
  FieldContent,
  FieldDescription,
  FieldError,
  FieldLabel,
  Input,
  RadioGroup,
  RadioGroupItem,
  Skeleton,
} from '@e412/titanium';
import { useForm } from '@tanstack/react-form';
import { useEffect } from 'react';
import { z } from 'zod';
import { AuditConfigPanel } from '@/components/audit-logs';
import { useNotFoundSettings } from '@/hooks/use-settings';

const settingsSchema = z
  .object({
    mode: z.enum(['default', 'redirect']),
    redirect_url: z.string(),
  })
  .refine(
    (data) => {
      if (data.mode === 'redirect') {
        try {
          new URL(data.redirect_url);
          return true;
        } catch {
          return false;
        }
      }
      return true;
    },
    {
      message: 'Please enter a valid URL',
      path: ['redirect_url'],
    },
  );

type SettingsFormValues = z.infer<typeof settingsSchema>;

export function SettingsPage() {
  const { settings, isLoading, update, isUpdating } = useNotFoundSettings();

  const form = useForm({
    defaultValues: {
      mode: 'default',
      redirect_url: '',
    } as SettingsFormValues,
    validators: {
      onSubmit: settingsSchema,
    },
    onSubmit: async ({ value }) => {
      await update({
        mode: value.mode,
        redirect_url: value.mode === 'redirect' ? value.redirect_url : '',
      });
    },
  });

  // Update form when settings are loaded
  useEffect(() => {
    if (settings) {
      form.setFieldValue('mode', settings.mode || 'default');
      form.setFieldValue('redirect_url', settings.redirect_url || '');
    }
  }, [settings, form.setFieldValue]);

  if (isLoading) {
    return (
      <div className="space-y-6">
        <h1 className="text-2xl font-bold">Settings</h1>
        <Card>
          <CardHeader>
            <Skeleton className="h-6 w-48" />
            <Skeleton className="h-4 w-96" />
          </CardHeader>
          <CardContent className="space-y-6">
            <Skeleton className="h-20 w-full" />
            <Skeleton className="h-10 w-full" />
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Settings</h1>

      <Card>
        <CardHeader>
          <CardTitle>404 Page Configuration</CardTitle>
          <CardDescription>
            Configure what happens when a visitor accesses a hostname that doesn&apos;t match any
            proxy.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form
            onSubmit={(e) => {
              e.preventDefault();
              e.stopPropagation();
              form.handleSubmit();
            }}
            className="space-y-6"
          >
            <form.Field name="mode">
              {(field) => (
                <Field>
                  <FieldLabel>Behavior</FieldLabel>
                  <RadioGroup
                    value={field.state.value}
                    onValueChange={(value) => field.handleChange(value as 'default' | 'redirect')}
                    className="mt-2 space-y-3"
                  >
                    <div className="flex items-start space-x-3">
                      <RadioGroupItem value="default" id="mode-default" className="mt-1" />
                      <div className="space-y-1">
                        <label
                          htmlFor="mode-default"
                          className="text-sm font-medium cursor-pointer"
                        >
                          Show Default 404 Page
                        </label>
                        <p className="text-sm text-muted-foreground">
                          Display a branded &quot;Page Not Found&quot; message
                        </p>
                      </div>
                    </div>
                    <div className="flex items-start space-x-3">
                      <RadioGroupItem value="redirect" id="mode-redirect" className="mt-1" />
                      <div className="space-y-1">
                        <label
                          htmlFor="mode-redirect"
                          className="text-sm font-medium cursor-pointer"
                        >
                          Redirect to URL
                        </label>
                        <p className="text-sm text-muted-foreground">
                          Automatically redirect visitors to a specific URL
                        </p>
                      </div>
                    </div>
                  </RadioGroup>
                </Field>
              )}
            </form.Field>

            <form.Subscribe selector={(state) => state.values.mode}>
              {(mode) =>
                mode === 'redirect' && (
                  <form.Field name="redirect_url">
                    {(field) => {
                      const hasError =
                        field.state.meta.isTouched && field.state.meta.errors.length > 0;
                      return (
                        <Field data-invalid={hasError}>
                          <FieldLabel htmlFor={field.name}>Redirect URL</FieldLabel>
                          <FieldContent>
                            <Input
                              id={field.name}
                              placeholder="https://example.com"
                              value={field.state.value}
                              onChange={(e) => field.handleChange(e.target.value)}
                              onBlur={field.handleBlur}
                              aria-invalid={hasError}
                            />
                          </FieldContent>
                          <FieldDescription>
                            Visitors will be redirected to this URL when accessing an unknown
                            hostname
                          </FieldDescription>
                          {hasError && <FieldError errors={field.state.meta.errors} />}
                        </Field>
                      );
                    }}
                  </form.Field>
                )
              }
            </form.Subscribe>

            <div className="flex justify-end">
              <Button type="submit" disabled={isUpdating}>
                {isUpdating ? 'Saving...' : 'Save Changes'}
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>

      <AuditConfigPanel />
    </div>
  );
}
