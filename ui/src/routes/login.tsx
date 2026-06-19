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
      onChange: loginSchema,
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
    <div className="flex min-h-screen">
      {/* Brand panel — left side on desktop, hidden on mobile */}
      <div className="hidden lg:flex lg:w-1/2 xl:w-[55%] bg-primary/[0.06] items-end justify-start p-12 relative overflow-hidden">
        {/* Decorative circles */}
        <div className="absolute -top-24 -right-24 size-96 rounded-full border border-primary/10" />
        <div className="absolute -top-12 -right-12 size-72 rounded-full border border-primary/[0.06]" />
        <div className="absolute bottom-32 right-24 size-40 rounded-full bg-primary/[0.04]" />

        <div className="relative z-10 max-w-lg animate-fade-up">
          <div className="flex size-14 items-center justify-center rounded bg-primary text-primary-foreground">
            <WaygateLogo className="size-8" />
          </div>
          <h1
            className="mt-6 text-5xl font-bold tracking-tight leading-[1.1]"
            style={{ fontFamily: '"Bricolage Grotesque", system-ui, sans-serif' }}
          >
            Waygates
          </h1>
          <p className="mt-4 text-lg text-muted-foreground max-w-sm">
            Fast travel for your network.
          </p>
        </div>
      </div>

      {/* Form panel — right side on desktop, full screen on mobile */}
      <div className="flex flex-1 items-center justify-center px-6 py-12 bg-background">
        <div className="w-full max-w-sm animate-fade-up" style={{ animationDelay: '100ms' }}>
          {/* Mobile-only brand header */}
          <div className="flex flex-col items-center gap-2 mb-8 lg:hidden">
            <div className="flex size-12 items-center justify-center rounded bg-primary text-primary-foreground">
              <WaygateLogo className="size-7" />
            </div>
            <h2
              className="text-2xl font-bold tracking-tight"
              style={{ fontFamily: '"Bricolage Grotesque", system-ui, sans-serif' }}
            >
              Waygates
            </h2>
          </div>

          {/* Desktop heading */}
          <div className="hidden lg:block mb-8">
            <h2 className="text-2xl font-bold tracking-tight">Sign in</h2>
            <p className="mt-1 text-sm text-muted-foreground">Enter your credentials to continue</p>
          </div>

          {/* Mobile subtitle */}
          <p className="text-center text-sm text-muted-foreground mb-6 lg:hidden">
            Sign in to your account
          </p>

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
        </div>
      </div>
    </div>
  );
}
