import {
  Alert,
  AlertDescription,
  Button,
  Card,
  CardContent,
  Field,
  FieldError,
  FieldGroup,
  FieldLabel,
  Input,
} from '@e412/titanium';
import { useForm } from '@tanstack/react-form';
import { Link, useNavigate, useSearch } from '@tanstack/react-router';
import { CheckCircle2, XCircle } from 'lucide-react';
import { useState } from 'react';
import { z } from 'zod';
import { WaygateLogo } from '@/components/layout/waygate-logo';
import { publicApi } from '../lib/api';
import { useAuthStore } from '../stores/auth';
import type { ApiResponse, TokenPair } from '../types/api';

const loginSchema = z.object({
  identifier: z.string().min(1, 'Username or email is required'),
  password: z.string().min(1, 'Password is required'),
});

export function LoginPage() {
  const navigate = useNavigate();
  const searchParams = useSearch({ strict: false }) as { registered?: string };
  const { setTokens } = useAuthStore();
  const [error, setError] = useState<string | null>(null);
  const justRegistered = searchParams.registered === 'true';

  const form = useForm({
    defaultValues: {
      identifier: '',
      password: '',
    },
    validators: {
      onBlur: loginSchema,
    },
    onSubmit: async ({ value }) => {
      setError(null);
      try {
        const response = await publicApi
          .post('auth/login', { json: value })
          .json<ApiResponse<TokenPair>>();

        if (response.success && response.data) {
          setTokens(response.data);
          navigate({ to: '/dashboard' });
        } else {
          setError(response.message || 'Login failed');
        }
      } catch {
        setError('Wrong username or password. Check your details and try again.');
      }
    },
  });

  return (
    <div className="flex min-h-screen items-center justify-center px-4 bg-gradient-to-br from-background via-background to-accent/30">
      <Card className="w-full max-w-md animate-fade-up">
        <div className="flex flex-col items-center gap-2 px-5 pt-6 pb-2">
          <div className="flex size-10 items-center justify-center rounded bg-primary text-primary-foreground">
            <WaygateLogo className="size-6" />
          </div>
          <h2
            className="text-2xl font-semibold tracking-tight"
            style={{ fontFamily: '"Bricolage Grotesque", system-ui, sans-serif' }}
          >
            Waygates
          </h2>
          <p className="text-sm text-muted-foreground">Sign in to your account</p>
        </div>
        <CardContent>
          <form
            onSubmit={(e) => {
              e.preventDefault();
              e.stopPropagation();
              form.handleSubmit();
            }}
          >
            <FieldGroup>
              {justRegistered && !error && (
                <Alert variant="success">
                  <CheckCircle2 className="size-4" />
                  <AlertDescription>Account created! Sign in to get started.</AlertDescription>
                </Alert>
              )}

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
                        placeholder="Enter your username or email"
                        autoComplete="username"
                        autoFocus
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
                        autoComplete="current-password"
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

              <form.Subscribe selector={(state) => state.isSubmitting}>
                {(isSubmitting) => (
                  <Button type="submit" disabled={isSubmitting} className="w-full">
                    {isSubmitting ? 'Signing in...' : 'Sign in'}
                  </Button>
                )}
              </form.Subscribe>

              <p className="text-center text-sm text-muted-foreground">
                Don&apos;t have an account?{' '}
                <Link to="/signup" className="text-primary hover:underline">
                  Sign up
                </Link>
              </p>
            </FieldGroup>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
