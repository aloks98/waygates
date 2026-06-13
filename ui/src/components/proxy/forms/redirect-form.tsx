import {
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardHeading,
  CardTitle,
  Field,
  FieldContent,
  FieldDescription,
  FieldError,
  FieldLabel,
  Input,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Switch,
} from '@e412/titanium';
import { useForm } from '@tanstack/react-form';
import { useEffect, useState } from 'react';
import { z } from 'zod';

import type { CreateRedirectRequest, ProxyConfig } from '@/types/proxy';

import { type ACLAssignment, ACLSelector } from './acl-selector';

const redirectSchema = z.object({
  name: z.string().min(1, 'Name is required').max(255, 'Name must be at most 255 characters'),
  hostname: z
    .string()
    .min(1, 'Hostname is required')
    .max(253, 'Hostname must be at most 253 characters'),
  description: z.string().max(500, 'Description must be at most 500 characters').optional(),
  ssl_enabled: z.boolean(),
  target: z.string().min(1, 'Target URL is required').url('Target must be a valid URL'),
  status_code: z.number().refine((val) => [301, 302, 307, 308].includes(val), {
    message: 'Status code must be 301, 302, 307, or 308',
  }),
  preserve_path: z.boolean(),
  preserve_query: z.boolean(),
});

type RedirectFormValues = z.infer<typeof redirectSchema>;

interface RedirectFormProps {
  initialData?: ProxyConfig | null;
  initialACLAssignments?: ACLAssignment[];
  onSubmit: (data: CreateRedirectRequest, aclAssignments?: ACLAssignment[]) => void;
  loading: boolean;
  onCancel: () => void;
}

export function RedirectForm({
  initialData,
  initialACLAssignments,
  onSubmit,
  loading,
  onCancel,
}: RedirectFormProps) {
  const [aclAssignments, setAclAssignments] = useState<ACLAssignment[]>(
    initialACLAssignments ?? [],
  );

  const form = useForm({
    defaultValues: {
      name: '',
      hostname: '',
      description: '',
      ssl_enabled: true,
      target: '',
      status_code: 301,
      preserve_path: true,
      preserve_query: true,
    } as RedirectFormValues,
    validators: {
      onSubmit: redirectSchema,
    },
    onSubmit: async ({ value }) => {
      const data: CreateRedirectRequest = {
        type: 'redirect',
        name: value.name,
        hostname: value.hostname,
        description: value.description || undefined,
        ssl_enabled: value.ssl_enabled,
        redirect: {
          target: value.target,
          status_code: value.status_code as 301 | 302 | 307 | 308,
          preserve_path: value.preserve_path,
          preserve_query: value.preserve_query,
        },
      };

      onSubmit(data, aclAssignments.length > 0 ? aclAssignments : undefined);
    },
  });

  useEffect(() => {
    if (initialData) {
      form.setFieldValue('name', initialData.name || '');
      form.setFieldValue('hostname', initialData.hostname || '');
      form.setFieldValue('description', initialData.description || '');
      form.setFieldValue('ssl_enabled', initialData.ssl_enabled ?? true);
      form.setFieldValue('target', initialData.redirect?.target || '');
      form.setFieldValue('status_code', initialData.redirect?.status_code || 301);
      form.setFieldValue('preserve_path', initialData.redirect?.preserve_path ?? true);
      form.setFieldValue('preserve_query', initialData.redirect?.preserve_query ?? true);
    }
  }, [initialData, form.setFieldValue]);

  // Update ACL assignments when initialACLAssignments changes (async load)
  useEffect(() => {
    if (initialACLAssignments) {
      setAclAssignments(initialACLAssignments);
    }
  }, [initialACLAssignments]);

  return (
    <form
      onSubmit={(e) => {
        e.preventDefault();
        e.stopPropagation();
        form.handleSubmit();
      }}
      className="space-y-6"
    >
      {/* Basic Information */}
      <Card>
        <CardHeader>
          <CardHeading>
            <CardTitle>Basic Information</CardTitle>
            <CardDescription>General settings for this redirect</CardDescription>
          </CardHeading>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid gap-4 sm:grid-cols-2">
            <form.Field name="name">
              {(field) => {
                const hasError = field.state.meta.isTouched && field.state.meta.errors.length > 0;
                return (
                  <Field data-invalid={hasError}>
                    <FieldLabel htmlFor={field.name}>Name</FieldLabel>
                    <Input
                      id={field.name}
                      placeholder="Blog Redirect"
                      value={field.state.value}
                      onChange={(e) => field.handleChange(e.target.value)}
                      onBlur={field.handleBlur}
                      aria-invalid={hasError}
                    />
                    {hasError && <FieldError errors={field.state.meta.errors} />}
                  </Field>
                );
              }}
            </form.Field>

            <form.Field name="hostname">
              {(field) => {
                const hasError = field.state.meta.isTouched && field.state.meta.errors.length > 0;
                return (
                  <Field data-invalid={hasError}>
                    <FieldLabel htmlFor={field.name}>Hostname</FieldLabel>
                    <Input
                      id={field.name}
                      placeholder="old.example.com"
                      value={field.state.value}
                      onChange={(e) => field.handleChange(e.target.value)}
                      onBlur={field.handleBlur}
                      aria-invalid={hasError}
                    />
                    {hasError && <FieldError errors={field.state.meta.errors} />}
                  </Field>
                );
              }}
            </form.Field>
          </div>

          <form.Field name="description">
            {(field) => {
              const hasError = field.state.meta.isTouched && field.state.meta.errors.length > 0;
              return (
                <Field data-invalid={hasError}>
                  <FieldLabel htmlFor={field.name}>Description (optional)</FieldLabel>
                  <Input
                    id={field.name}
                    placeholder="Redirect old domain to new"
                    value={field.state.value}
                    onChange={(e) => field.handleChange(e.target.value)}
                    onBlur={field.handleBlur}
                    aria-invalid={hasError}
                  />
                  {hasError && <FieldError errors={field.state.meta.errors} />}
                </Field>
              );
            }}
          </form.Field>
        </CardContent>
      </Card>

      {/* Redirect Configuration + Options side by side */}
      <div className="grid gap-6 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <CardHeading>
              <CardTitle>Redirect Target</CardTitle>
              <CardDescription>Where to redirect visitors</CardDescription>
            </CardHeading>
          </CardHeader>
          <CardContent className="space-y-4">
            <form.Field name="target">
              {(field) => {
                const hasError = field.state.meta.isTouched && field.state.meta.errors.length > 0;
                return (
                  <Field data-invalid={hasError}>
                    <FieldLabel htmlFor={field.name}>Target URL</FieldLabel>
                    <Input
                      id={field.name}
                      placeholder="https://new.example.com"
                      value={field.state.value}
                      onChange={(e) => field.handleChange(e.target.value)}
                      onBlur={field.handleBlur}
                      aria-invalid={hasError}
                    />
                    <FieldDescription>The URL to redirect visitors to</FieldDescription>
                    {hasError && <FieldError errors={field.state.meta.errors} />}
                  </Field>
                );
              }}
            </form.Field>

            <form.Field name="status_code">
              {(field) => {
                const hasError = field.state.meta.isTouched && field.state.meta.errors.length > 0;
                return (
                  <Field data-invalid={hasError}>
                    <FieldLabel>Redirect Type</FieldLabel>
                    <Select
                      value={String(field.state.value)}
                      onValueChange={(val) => field.handleChange(parseInt(val, 10))}
                    >
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="301">301 - Permanent</SelectItem>
                        <SelectItem value="302">302 - Temporary</SelectItem>
                        <SelectItem value="307">307 - Temporary (preserve method)</SelectItem>
                        <SelectItem value="308">308 - Permanent (preserve method)</SelectItem>
                      </SelectContent>
                    </Select>
                    <FieldDescription>
                      301/308 are cached by browsers, 302/307 are temporary
                    </FieldDescription>
                    {hasError && <FieldError errors={field.state.meta.errors} />}
                  </Field>
                );
              }}
            </form.Field>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardHeading>
              <CardTitle>Options</CardTitle>
              <CardDescription>SSL and redirect behavior</CardDescription>
            </CardHeading>
          </CardHeader>
          <CardContent className="space-y-4">
            <form.Field name="ssl_enabled">
              {(field) => (
                <Field orientation="horizontal">
                  <FieldContent>
                    <FieldLabel>Enable HTTPS</FieldLabel>
                    <FieldDescription>
                      Automatically obtain and manage SSL/TLS certificates
                    </FieldDescription>
                  </FieldContent>
                  <Switch checked={field.state.value} onCheckedChange={field.handleChange} />
                </Field>
              )}
            </form.Field>

            <form.Field name="preserve_path">
              {(field) => (
                <Field orientation="horizontal">
                  <FieldContent>
                    <FieldLabel>Preserve Path</FieldLabel>
                    <FieldDescription>Append the original path to the target URL</FieldDescription>
                  </FieldContent>
                  <Switch checked={field.state.value} onCheckedChange={field.handleChange} />
                </Field>
              )}
            </form.Field>

            <form.Field name="preserve_query">
              {(field) => (
                <Field orientation="horizontal">
                  <FieldContent>
                    <FieldLabel>Preserve Query String</FieldLabel>
                    <FieldDescription>
                      Append the original query parameters to the target URL
                    </FieldDescription>
                  </FieldContent>
                  <Switch checked={field.state.value} onCheckedChange={field.handleChange} />
                </Field>
              )}
            </form.Field>
          </CardContent>
        </Card>
      </div>

      {/* Access Control */}
      <ACLSelector value={aclAssignments} onChange={setAclAssignments} disabled={loading} />

      {/* Actions */}
      <div className="flex justify-end gap-2 pt-4 border-t">
        <Button type="button" variant="outline" onClick={onCancel}>
          Cancel
        </Button>
        <Button type="submit" disabled={loading}>
          {loading ? 'Saving...' : initialData ? 'Save Changes' : 'Create Proxy'}
        </Button>
      </div>
    </form>
  );
}
