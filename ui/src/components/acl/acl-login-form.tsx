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
  email: z.string().min(1, 'Email is required').email('Please enter a valid email'),
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
      email: '',
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
              email: value.email,
              password: value.password,
              redirect: redirectUrl,
            },
          })
          .json<ApiResponse<ACLLoginResponse>>();

        if (response.success && response.data) {
          onSuccess?.();
          // If backend provides a redirect URL, navigate to it
          if (response.data.redirect_url) {
            window.location.href = response.data.redirect_url;
          } else if (redirectUrl) {
            window.location.href = redirectUrl;
          }
        } else {
          setError(response.message || 'Login failed');
        }
      } catch {
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

        <form.Field name="email">
          {(field) => {
            const hasError = field.state.meta.isTouched && field.state.meta.errors.length > 0;
            return (
              <Field data-invalid={hasError}>
                <FieldLabel htmlFor={field.name}>Email</FieldLabel>
                <Input
                  id={field.name}
                  type="email"
                  placeholder="Enter your email"
                  value={field.state.value}
                  onChange={(e) => field.handleChange(e.target.value)}
                  onBlur={field.handleBlur}
                  aria-invalid={hasError}
                  autoComplete="email"
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
