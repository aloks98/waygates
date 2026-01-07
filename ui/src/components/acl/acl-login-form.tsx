import {
  Alert,
  AlertDescription,
  Button,
  Field,
  FieldError,
  FieldGroup,
  FieldLabel,
  Input,
} from '@e412/titanium';
import { useForm } from '@tanstack/react-form';
import { XCircle } from 'lucide-react';
import { useState } from 'react';
import { z } from 'zod';
import { publicApi } from '@/lib/api';
import type { ApiResponse } from '@/types/api';

const loginSchema = z.object({
  identifier: z.string().min(1, 'Username or email is required'),
  password: z.string().min(1, 'Password is required'),
});

interface ACLLoginResponse {
  success: boolean;
  redirect_url?: string;
}

interface ACLLoginFormProps {
  redirectUrl?: string;
  onSuccess?: () => void;
  primaryColor?: string;
}

export function ACLLoginForm({ redirectUrl, onSuccess, primaryColor }: ACLLoginFormProps) {
  const [error, setError] = useState<string | null>(null);

  const form = useForm({
    defaultValues: {
      identifier: '',
      password: '',
    },
    validators: {
      onChange: loginSchema,
    },
    onSubmit: async ({ value }) => {
      setError(null);
      try {
        const response = await publicApi
          .post('auth/acl/login', {
            json: {
              email: value.identifier,
              password: value.password,
              redirect: redirectUrl,
            },
          })
          .json<ApiResponse<ACLLoginResponse>>();

        if (response.success && response.data) {
          onSuccess?.();
          // Determine where to redirect after login
          const targetUrl = response.data.redirect_url || redirectUrl;
          if (targetUrl) {
            // Small delay to ensure cookie is set before redirect
            setTimeout(() => {
              window.location.href = targetUrl;
            }, 100);
          } else {
            // If no redirect URL, reload to show authenticated state
            window.location.reload();
          }
        } else {
          setError(response.message || 'Login failed');
        }
      } catch (err) {
        console.error('ACL login error:', err);
        setError('Invalid credentials');
      }
    },
  });

  // Generate button style with primary color
  const buttonStyle = primaryColor
    ? ({
        '--primary': primaryColor,
        backgroundColor: primaryColor,
      } as React.CSSProperties)
    : undefined;

  return (
    <form
      onSubmit={(e) => {
        e.preventDefault();
        e.stopPropagation();
        form.handleSubmit();
      }}
    >
      <FieldGroup>
        {error && (
          <Alert variant="destructive">
            <XCircle className="size-4" />
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}

        <form.Field name="identifier">
          {(field) => {
            const hasError = field.state.meta.isTouched && field.state.meta.errors.length > 0;
            return (
              <Field data-invalid={hasError}>
                <FieldLabel htmlFor={field.name}>Username or Email</FieldLabel>
                <Input
                  id={field.name}
                  type="text"
                  placeholder="Enter your username or email"
                  value={field.state.value}
                  onChange={(e) => field.handleChange(e.target.value)}
                  onBlur={field.handleBlur}
                  aria-invalid={hasError}
                  autoComplete="username"
                />
                {hasError && <FieldError errors={field.state.meta.errors} />}
              </Field>
            );
          }}
        </form.Field>

        <form.Field name="password">
          {(field) => {
            const hasError = field.state.meta.isTouched && field.state.meta.errors.length > 0;
            return (
              <Field data-invalid={hasError}>
                <FieldLabel htmlFor={field.name}>Password</FieldLabel>
                <Input
                  id={field.name}
                  type="password"
                  placeholder="Enter your password"
                  value={field.state.value}
                  onChange={(e) => field.handleChange(e.target.value)}
                  onBlur={field.handleBlur}
                  aria-invalid={hasError}
                  autoComplete="current-password"
                />
                {hasError && <FieldError errors={field.state.meta.errors} />}
              </Field>
            );
          }}
        </form.Field>

        <form.Subscribe selector={(state) => state.isSubmitting}>
          {(isSubmitting) => (
            <Button type="submit" disabled={isSubmitting} className="w-full" style={buttonStyle}>
              {isSubmitting ? 'Signing in...' : 'Sign in'}
            </Button>
          )}
        </form.Subscribe>
      </FieldGroup>
    </form>
  );
}
