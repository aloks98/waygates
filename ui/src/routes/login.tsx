import {
  Alert,
  AlertDescription,
  Button,
  FieldGroup,
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  Input,
} from '@e412/rnui-react';
import { zodResolver } from '@hookform/resolvers/zod';
import { useQuery } from '@tanstack/react-query';
import { Link, useNavigate, useSearch } from '@tanstack/react-router';
import { HTTPError } from 'ky';
import { CheckCircle2, KeyRound, XCircle } from 'lucide-react';
import { useEffect, useState } from 'react';
import { useForm } from 'react-hook-form';
import { z } from 'zod';

import { WaygateLogo } from '@/components/layout/waygate-logo';
import { ssoErrorMessage } from '@/lib/sso';

import { publicApi } from '../lib/api';
import { useAuthStore } from '../stores/auth';
import type { ApiResponse, TokenPair } from '../types/api';

const loginSchema = z.object({
  identifier: z.string().min(1, 'Username or email is required'),
  password: z.string().min(1, 'Password is required'),
});
type LoginValues = z.infer<typeof loginSchema>;

export function LoginPage() {
  const navigate = useNavigate();
  const search = useSearch({ strict: false }) as { registered?: string; sso_error?: string };
  const { setTokens } = useAuthStore();
  const [error, setError] = useState<string | null>(null);
  const justRegistered = search.registered === 'true';

  // Gate signup link on registration status
  const { data: regStatus } = useQuery({
    queryKey: ['auth', 'registration-status'],
    queryFn: () => publicApi.get('auth/registration-status').json<ApiResponse<{ open: boolean }>>(),
    staleTime: 60 * 1000,
  });
  const registrationOpen = regStatus?.data?.open ?? false;

  const { data: ssoStatus } = useQuery({
    queryKey: ['auth', 'sso-status'],
    queryFn: () =>
      publicApi.get('auth/sso/status').json<ApiResponse<{ enabled: boolean; label: string }>>(),
    staleTime: 60 * 1000,
  });
  const ssoEnabled = ssoStatus?.data?.enabled ?? false;
  const ssoLabel = ssoStatus?.data?.label?.trim() || 'Sign in with SSO';

  const [step, setStep] = useState<'identifier' | 'password'>('identifier');
  const [checking, setChecking] = useState(false);

  // surface sso_error from the callback redirect
  useEffect(() => {
    if (search.sso_error) setError(ssoErrorMessage(search.sso_error));
  }, [search.sso_error]);

  const form = useForm<LoginValues>({
    resolver: zodResolver(loginSchema),
    mode: 'onTouched',
    defaultValues: { identifier: '', password: '' },
  });
  const isSubmitting = form.formState.isSubmitting;

  const onContinue = async () => {
    setError(null);
    const identifier = form.getValues('identifier').trim();
    if (!identifier) {
      form.setError('identifier', { message: 'Username or email is required' });
      return;
    }
    // Only an email can route to SSO; usernames fall through to password.
    if (ssoEnabled && identifier.includes('@')) {
      setChecking(true);
      try {
        const res = await publicApi
          .post('auth/sso/lookup', { json: { email: identifier } })
          .json<ApiResponse<{ method: string }>>();
        if (res.data?.method === 'sso') {
          window.location.href = `/api/auth/sso/login?login_hint=${encodeURIComponent(identifier)}`;
          return;
        }
      } catch {
        // fall through to password step on lookup failure
      } finally {
        setChecking(false);
      }
    }
    setStep('password');
  };

  const onSubmit = async (value: LoginValues) => {
    setError(null);
    try {
      const response = await publicApi
        .post('auth/login', { json: value })
        .json<ApiResponse<TokenPair>>();
      if (response.success && response.data) {
        setTokens(response.data);
        navigate({ to: '/' });
      } else {
        setError(response.message || 'Login failed');
      }
    } catch (err) {
      if (err instanceof HTTPError) {
        try {
          const body = (await err.response.json()) as {
            error?: { message?: string };
            message?: string;
          };
          const msg = body?.error?.message || body?.message || '';
          if (err.response.status === 403 && msg.toLowerCase().includes('disabled')) {
            setError('Your account has been disabled. Contact your administrator.');
            return;
          }
        } catch {
          // fall through to generic message
        }
      }
      setError('Wrong username or password. Check your details and try again.');
    }
  };

  return (
    <div className="flex min-h-screen">
      {/* Brand panel — left side on desktop, hidden on mobile */}
      <div className="hidden lg:flex lg:w-1/2 xl:w-[55%] bg-primary/[0.06] items-end justify-start p-12 relative overflow-hidden">
        {/* Decorative circles */}
        <div className="absolute -top-24 -right-24 size-96 rounded-none border border-primary/10" />
        <div className="absolute -top-12 -right-12 size-72 rounded-none border border-primary/[0.06]" />
        <div className="absolute bottom-32 right-24 size-40 rounded-none bg-primary/[0.04]" />

        <div className="relative z-10 max-w-lg animate-fade-up">
          <WaygateLogo className="size-16" />
          <h1
            className="mt-6 text-6xl tracking-wide leading-[1.1]"
            style={{ fontFamily: 'var(--font-display)' }}
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
            <WaygateLogo className="size-12" />
            <h2 className="text-4xl tracking-wide" style={{ fontFamily: 'var(--font-display)' }}>
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

          <Form {...form}>
            {ssoEnabled ? (
              /* Two-step SSO flow */
              <form
                onSubmit={
                  step === 'password'
                    ? form.handleSubmit(onSubmit)
                    : (e) => {
                        e.preventDefault();
                        void onContinue();
                      }
                }
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

                  <FormField
                    control={form.control}
                    name="identifier"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Username or Email</FormLabel>
                        <FormControl>
                          <Input
                            placeholder="Enter your username or email"
                            autoComplete="username"
                            autoFocus={step === 'identifier'}
                            readOnly={step === 'password'}
                            className={
                              step === 'password' ? 'bg-muted/50 text-muted-foreground' : undefined
                            }
                            {...field}
                          />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />

                  {step === 'password' && (
                    <>
                      <FormField
                        control={form.control}
                        name="password"
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>Password</FormLabel>
                            <FormControl>
                              <Input
                                type="password"
                                placeholder="Enter your password"
                                autoComplete="current-password"
                                autoFocus
                                {...field}
                              />
                            </FormControl>
                            <FormMessage />
                          </FormItem>
                        )}
                      />
                      <button
                        type="button"
                        className="self-start text-sm text-muted-foreground hover:text-foreground"
                        onClick={() => {
                          setStep('identifier');
                          setError(null);
                          form.setValue('password', '');
                          form.clearErrors();
                        }}
                      >
                        ← Use a different account
                      </button>
                    </>
                  )}

                  {step === 'identifier' ? (
                    <Button
                      type="button"
                      disabled={checking}
                      className="w-full glow-primary"
                      onClick={onContinue}
                    >
                      {checking ? 'Checking...' : 'Continue'}
                    </Button>
                  ) : (
                    <Button type="submit" disabled={isSubmitting} className="w-full glow-primary">
                      {isSubmitting ? 'Signing in...' : 'Sign in'}
                    </Button>
                  )}

                  <Button
                    type="button"
                    variant="outline"
                    className="w-full"
                    onMouseDown={(e) => e.preventDefault()}
                    onClick={() => {
                      window.location.href = '/api/auth/sso/login';
                    }}
                  >
                    <KeyRound className="size-4" />
                    {ssoLabel}
                  </Button>

                  {registrationOpen && (
                    <p className="text-center text-sm text-muted-foreground">
                      Don&apos;t have an account?{' '}
                      <Link to="/signup" className="text-primary hover:underline">
                        Sign up
                      </Link>
                    </p>
                  )}
                </FieldGroup>
              </form>
            ) : (
              /* Single-step form — original behavior when SSO disabled */
              <form onSubmit={form.handleSubmit(onSubmit)}>
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

                  <FormField
                    control={form.control}
                    name="identifier"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Username or Email</FormLabel>
                        <FormControl>
                          <Input
                            placeholder="Enter your username or email"
                            autoComplete="username"
                            autoFocus
                            {...field}
                          />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />

                  <FormField
                    control={form.control}
                    name="password"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Password</FormLabel>
                        <FormControl>
                          <Input
                            type="password"
                            placeholder="Enter your password"
                            autoComplete="current-password"
                            {...field}
                          />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />

                  <Button type="submit" disabled={isSubmitting} className="w-full glow-primary">
                    {isSubmitting ? 'Signing in...' : 'Sign in'}
                  </Button>

                  {registrationOpen && (
                    <p className="text-center text-sm text-muted-foreground">
                      Don&apos;t have an account?{' '}
                      <Link to="/signup" className="text-primary hover:underline">
                        Sign up
                      </Link>
                    </p>
                  )}
                </FieldGroup>
              </form>
            )}
          </Form>
        </div>
      </div>
    </div>
  );
}
