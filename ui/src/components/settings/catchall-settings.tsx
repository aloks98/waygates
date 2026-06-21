import {
  Button,
  Card,
  CardContent,
  CardDescription,
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
  RadioGroup,
  RadioGroupItem,
  Skeleton,
} from '@e412/rnui-react';
import { zodResolver } from '@hookform/resolvers/zod';
import { useEffect } from 'react';
import { useForm } from 'react-hook-form';
import { z } from 'zod';

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

export function CatchallSettings() {
  const { settings, isLoading, update, isUpdating } = useNotFoundSettings();

  const form = useForm<SettingsFormValues>({
    resolver: zodResolver(settingsSchema),
    defaultValues: {
      mode: 'default',
      redirect_url: '',
    },
  });

  useEffect(() => {
    if (settings) {
      form.reset({
        mode: settings.mode || 'default',
        redirect_url: settings.redirect_url || '',
      });
    }
  }, [settings, form]);

  const onSubmit = async (value: SettingsFormValues) => {
    await update({
      mode: value.mode,
      redirect_url: value.mode === 'redirect' ? value.redirect_url : '',
    });
  };

  const mode = form.watch('mode');

  if (isLoading) {
    return (
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
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>404 Page Configuration</CardTitle>
        <CardDescription>
          Configure what happens when a visitor accesses a hostname that doesn&apos;t match any
          proxy.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
            <FormField
              control={form.control}
              name="mode"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Behavior</FormLabel>
                  <FormControl>
                    <RadioGroup
                      value={field.value}
                      onValueChange={field.onChange}
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
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            {mode === 'redirect' && (
              <FormField
                control={form.control}
                name="redirect_url"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Redirect URL</FormLabel>
                    <FormControl>
                      <Input placeholder="https://example.com" {...field} />
                    </FormControl>
                    <FormDescription>
                      Visitors will be redirected to this URL when accessing an unknown hostname
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            )}

            <div className="flex justify-end">
              <Button type="submit" disabled={isUpdating}>
                {isUpdating ? 'Saving...' : 'Save Changes'}
              </Button>
            </div>
          </form>
        </Form>
      </CardContent>
    </Card>
  );
}
