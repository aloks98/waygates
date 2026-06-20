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
import { Link, useNavigate } from '@tanstack/react-router';
import { XCircle } from 'lucide-react';
import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { z } from 'zod';

import { WaygateLogo } from '@/components/layout/waygate-logo';

import { publicApi } from '../lib/api';
import type { ApiResponse } from '../types/api';
import type { User } from '../types/auth';

const signupSchema = z
  .object({
    name: z.string().min(1, 'Full name is required'),
    username: z.string().min(3, 'Username must be at least 3 characters'),
    email: z.string().email('Invalid email address'),
    password: z.string().min(8, 'Password must be at least 8 characters'),
    confirmPassword: z.string().min(1, 'Please confirm your password'),
  })
  .refine((data) => data.password === data.confirmPassword, {
    message: "Passwords don't match",
    path: ['confirmPassword'],
  });
type SignupValues = z.infer<typeof signupSchema>;

export function SignupPage() {
  const navigate = useNavigate();
  const [error, setError] = useState<string | null>(null);

  const form = useForm<SignupValues>({
    resolver: zodResolver(signupSchema),
    mode: 'onTouched',
    defaultValues: {
      name: '',
      username: '',
      email: '',
      password: '',
      confirmPassword: '',
    },
  });

  const onSubmit = async (value: SignupValues) => {
    setError(null);
    try {
      const response = await publicApi
        .post('auth/register', {
          json: {
            name: value.name,
            username: value.username,
            email: value.email,
            password: value.password,
          },
        })
        .json<ApiResponse<User>>();

      if (response.success) {
        navigate({ to: '/login', search: { registered: 'true' } });
      } else {
        setError(response.message || 'Registration failed');
      }
    } catch {
      setError('Could not create your account. Please check your details and try again.');
    }
  };

  return (
    <div className="flex min-h-screen">
      {/* Brand panel — left side on desktop */}
      <div className="hidden lg:flex lg:w-1/2 xl:w-[55%] bg-primary/[0.06] items-end justify-start p-12 relative overflow-hidden">
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

      {/* Form panel */}
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
            <h2 className="text-2xl font-bold tracking-tight">Create account</h2>
            <p className="mt-1 text-sm text-muted-foreground">Set up your Waygates admin account</p>
          </div>

          {/* Mobile subtitle */}
          <p className="text-center text-sm text-muted-foreground mb-6 lg:hidden">
            Create a new account
          </p>

          <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)}>
              <FieldGroup>
                {error && (
                  <Alert variant="destructive">
                    <XCircle className="size-4" />
                    <AlertDescription>{error}</AlertDescription>
                  </Alert>
                )}

                <FormField
                  control={form.control}
                  name="name"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Full Name</FormLabel>
                      <FormControl>
                        <Input
                          placeholder="Enter your full name"
                          autoComplete="name"
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
                  name="username"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Username</FormLabel>
                      <FormControl>
                        <Input placeholder="Choose a username" autoComplete="username" {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name="email"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Email</FormLabel>
                      <FormControl>
                        <Input
                          type="email"
                          placeholder="Enter your email"
                          autoComplete="email"
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
                          placeholder="Create a password"
                          autoComplete="new-password"
                          {...field}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name="confirmPassword"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Confirm Password</FormLabel>
                      <FormControl>
                        <Input
                          type="password"
                          placeholder="Confirm your password"
                          autoComplete="new-password"
                          {...field}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <Button type="submit" disabled={form.formState.isSubmitting} className="w-full">
                  {form.formState.isSubmitting ? 'Creating account...' : 'Create account'}
                </Button>

                <p className="text-center text-sm text-muted-foreground">
                  Already have an account?{' '}
                  <Link to="/login" className="text-primary hover:underline">
                    Sign in
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
