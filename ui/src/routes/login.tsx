import { Link, useNavigate } from '@tanstack/react-router';
import { useForm } from 'react-hook-form';
import {
  Button,
  Input,
  Card,
  CardHeader,
  CardContent,
  CardTitle,
  CardDescription,
  Form,
  FormField,
  FormItem,
  FormLabel,
  FormControl,
  FormMessage,
} from '@e412/titanium';
import { publicApi } from '../lib/api';
import { useAuthStore } from '../stores/auth';
import type { ApiResponse, TokenPair } from '../types/api';

interface LoginFormValues {
  identifier: string;
  password: string;
}

export function LoginPage() {
  const navigate = useNavigate();
  const { setTokens } = useAuthStore();

  const form = useForm<LoginFormValues>({
    defaultValues: {
      identifier: '',
      password: '',
    },
  });

  const onSubmit = async (values: LoginFormValues) => {
    try {
      const response = await publicApi
        .post('auth/login', {
          json: values,
        })
        .json<ApiResponse<TokenPair>>();

      if (response.success && response.data) {
        setTokens(response.data);
        navigate({ to: '/dashboard' });
      } else {
        form.setError('root', { message: response.message || 'Login failed' });
      }
    } catch {
      form.setError('root', { message: 'Invalid credentials' });
    }
  };

  return (
    <div className="flex min-h-screen items-center justify-center px-4">
      <Card className="w-full max-w-md">
        <CardHeader className="text-center">
          <CardTitle className="text-2xl">Homelab Proxy</CardTitle>
          <CardDescription>Sign in to your account</CardDescription>
        </CardHeader>
        <CardContent>
          <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
              {form.formState.errors.root && (
                <div className="rounded-md bg-destructive/10 p-3 text-sm text-destructive">
                  {form.formState.errors.root.message}
                </div>
              )}

              <FormField
                control={form.control}
                name="identifier"
                rules={{ required: 'Username or email is required' }}
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Username or Email</FormLabel>
                    <FormControl>
                      <Input
                        placeholder="Enter your username or email"
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
                rules={{ required: 'Password is required' }}
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Password</FormLabel>
                    <FormControl>
                      <Input
                        type="password"
                        placeholder="Enter your password"
                        {...field}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <Button
                type="submit"
                disabled={form.formState.isSubmitting}
                className="w-full"
              >
                {form.formState.isSubmitting ? 'Signing in...' : 'Sign in'}
              </Button>

              <p className="text-center text-sm text-muted-foreground">
                Don&apos;t have an account?{' '}
                <Link to="/signup" className="text-primary hover:underline">
                  Sign up
                </Link>
              </p>
            </form>
          </Form>
        </CardContent>
      </Card>
    </div>
  );
}
