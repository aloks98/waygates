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
import { Link, useNavigate, useSearch } from '@tanstack/react-router';
import { CheckCircle2, XCircle } from 'lucide-react';
import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { z } from 'zod';

import { WaygateLogo } from '@/components/layout/waygate-logo';

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
  const searchParams = useSearch({ strict: false }) as { registered?: string };
  const { setTokens } = useAuthStore();
  const [error, setError] = useState<string | null>(null);
  const justRegistered = searchParams.registered === 'true';

  const form = useForm<LoginValues>({
    resolver: zodResolver(loginSchema),
    mode: 'onTouched',
    defaultValues: { identifier: '', password: '' },
  });
  const isSubmitting = form.formState.isSubmitting;

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
    } catch {
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

                <p className="text-center text-sm text-muted-foreground">
                  Don&apos;t have an account?{' '}
                  <Link to="/signup" className="text-primary hover:underline">
                    Sign up
                  </Link>
                </p>
              </FieldGroup>
            </form>
          </Form>
        </div>
      </div>
    </div>
  );
}
