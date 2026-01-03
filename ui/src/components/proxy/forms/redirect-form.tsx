import {
  Button,
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
import { useEffect } from 'react';
import { z } from 'zod';
import type { CreateRedirectRequest, Proxy } from '@/types/proxy';

const redirectSchema = z.object({
  name: z.string().min(1, 'Name is required').max(255, 'Name must be at most 255 characters'),
  hostname: z
    .string()
    .min(1, 'Hostname is required')
    .max(253, 'Hostname must be at most 253 characters'),
  description: z.string().max(500, 'Description must be at most 500 characters').optional(),
  target: z.string().min(1, 'Target URL is required').url('Target must be a valid URL'),
  status_code: z.number().refine((val) => [301, 302, 307, 308].includes(val), {
    message: 'Status code must be 301, 302, 307, or 308',
  }),
  preserve_path: z.boolean(),
  preserve_query: z.boolean(),
});

type RedirectFormValues = z.infer<typeof redirectSchema>;

interface RedirectFormProps {
  initialData?: Proxy | null;
  onSubmit: (data: CreateRedirectRequest) => void;
  loading: boolean;
  onCancel: () => void;
}

export function RedirectForm({ initialData, onSubmit, loading, onCancel }: RedirectFormProps) {
  const form = useForm({
    defaultValues: {
      name: '',
      hostname: '',
      description: '',
      target: '',
      status_code: 301,
      preserve_path: true,
      preserve_query: true,
    } as RedirectFormValues,
    validators: {
      onChange: redirectSchema,
    },
    onSubmit: async ({ value }) => {
      const data: CreateRedirectRequest = {
        type: 'redirect',
        name: value.name,
        hostname: value.hostname,
        description: value.description || undefined,
        redirect: {
          target: value.target,
          status_code: value.status_code as 301 | 302 | 307 | 308,
          preserve_path: value.preserve_path,
          preserve_query: value.preserve_query,
        },
      };

      onSubmit(data);
    },
  });

  useEffect(() => {
    if (initialData) {
      form.setFieldValue('name', initialData.name || '');
      form.setFieldValue('hostname', initialData.hostname || '');
      form.setFieldValue('description', initialData.description || '');
      form.setFieldValue('target', initialData.redirect?.target || '');
      form.setFieldValue('status_code', initialData.redirect?.status_code || 301);
      form.setFieldValue('preserve_path', initialData.redirect?.preserve_path ?? true);
      form.setFieldValue('preserve_query', initialData.redirect?.preserve_query ?? true);
    }
  }, [initialData]);

  return (
    <form
      onSubmit={(e) => {
        e.preventDefault();
        e.stopPropagation();
        form.handleSubmit();
      }}
      className="space-y-6"
    >
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
                onValueChange={(val) => field.handleChange(parseInt(val))}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="301">301 - Permanent Redirect</SelectItem>
                  <SelectItem value="302">302 - Temporary Redirect</SelectItem>
                  <SelectItem value="307">307 - Temporary Redirect (preserve method)</SelectItem>
                  <SelectItem value="308">308 - Permanent Redirect (preserve method)</SelectItem>
                </SelectContent>
              </Select>
              <FieldDescription>
                301/308 are permanent (cached by browsers), 302/307 are temporary
              </FieldDescription>
              {hasError && <FieldError errors={field.state.meta.errors} />}
            </Field>
          );
        }}
      </form.Field>

      <div className="space-y-4">
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
      </div>

      <div className="flex justify-end gap-2">
        <Button type="button" variant="outline" onClick={onCancel}>
          Cancel
        </Button>
        <Button type="submit" disabled={loading}>
          {loading ? 'Saving...' : 'Save'}
        </Button>
      </div>
    </form>
  );
}
